package cosservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"forge/biz/adapter"
	"forge/biz/entity"
	"forge/biz/types"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"forge/util"
)

const (
	// MinSTSDuration 腾讯云STS临时密钥最短有效期（秒）
	MinSTSDuration = 900 // 15分钟
	// MaxSTSDuration 腾讯云STS临时密钥最长有效期（秒）
	MaxSTSDuration = 7200 // 2小时
	// MaxAvatarSize 头像文件最大大小（5MB）
	MaxAvatarSize = 5 * 1024 * 1024
)

// 错误定义
var (
	ErrInvalidParams       = errors.New("参数无效")
	ErrInternalError       = errors.New("内部错误")
	ErrPermissionDenied    = errors.New("权限不足")
	ErrInvalidResourcePath = errors.New("无效的资源路径")
	ErrInvalidDuration     = errors.New("无效的有效期")
)

// COSServiceImpl COS服务实现
type COSServiceImpl struct {
	cosService adapter.COSService
	config     configs.COSConfig
}

func NewCOSServiceImpl(cosService adapter.COSService, cfg configs.COSConfig) *COSServiceImpl {
	return &COSServiceImpl{
		cosService: cosService,
		config:     cfg,
	}
}

// GetOSSCredentials 获取OSS临时凭证
func (s *COSServiceImpl) GetOSSCredentials(ctx context.Context, req *types.GetOSSCredentialsParams) (*types.OSSCredentials, error) {
	// 从JWT token上下文中获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "failed to get user from context")
		return nil, ErrPermissionDenied
	}

	// 参数校验：资源路径
	if req.ResourcePath == "" {
		zlog.CtxErrorf(ctx, "resource path is required")
		return nil, ErrInvalidResourcePath
	}

	// 权限校验：确保资源路径以该用户的路径开头，后续可调整
	expectedPrefix := fmt.Sprintf("user/%s/", user.UserID)
	if !strings.HasPrefix(req.ResourcePath, expectedPrefix) {
		zlog.CtxWarnf(ctx, "resource path does not match user, path: %s, userID: %s", req.ResourcePath, user.UserID)
		return nil, ErrPermissionDenied
	}

	// 参数校验：有效期
	durationSeconds := req.DurationSeconds
	if durationSeconds == 0 {
		durationSeconds = s.config.STSDuration
	}
	if durationSeconds < MinSTSDuration || durationSeconds > MaxSTSDuration {
		zlog.CtxErrorf(ctx, "invalid duration seconds: %d, must be between %d and %d", durationSeconds, MinSTSDuration, MaxSTSDuration)
		return nil, ErrInvalidDuration
	}

	// 调用基础设施层获取临时凭证
	result, err := s.cosService.GetTemporaryCredentials(req.ResourcePath, durationSeconds)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to get credentials: %v", err)
		return nil, ErrInternalError
	}

	// 解析过期时间
	expiration, err := parseExpiration(result.Expiration)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to parse expiration: %v", err)
		return nil, ErrInternalError
	}

	// 构建响应
	credentials := &types.OSSCredentials{
		AccessKeyID:     result.Credentials.TmpSecretID,
		SecretAccessKey: result.Credentials.TmpSecretKey,
		SessionToken:    result.Credentials.SessionToken,
		Expiration:      expiration,
		Region:          s.config.Region,
		BucketName:      s.config.Bucket,
		Provider:        "cos",
		BaseURL:         s.config.BaseURL,
		AccountID:       s.config.AppID,
	}

	zlog.CtxInfof(ctx, "OSS credentials retrieved successfully, userID: %s, resource: %s", user.UserID, req.ResourcePath)
	return credentials, nil
}

// parseExpiration 解析过期时间字符串为Unix时间戳
func parseExpiration(expiration string) (int64, error) {
	t, err := time.Parse(time.RFC3339, expiration)
	if err != nil {
		return 0, fmt.Errorf("failed to parse expiration time '%s': %w", expiration, err)
	}
	return t.Unix(), nil
}

// UploadAvatar 上传用户头像到COS
func (s *COSServiceImpl) UploadAvatar(ctx context.Context, userID string, fileData []byte, filename string) (string, error) {
	// 从JWT token上下文中获取用户信息（双重验证）
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "failed to get user from context")
		return "", ErrPermissionDenied
	}

	// 验证用户ID是否匹配
	if user.UserID != userID {
		zlog.CtxErrorf(ctx, "userID mismatch, userID: %s, param userID: %s", user.UserID, userID)
		return "", ErrPermissionDenied
	}

	// 参数校验
	if len(fileData) == 0 {
		zlog.CtxErrorf(ctx, "file data is empty")
		return "", ErrInvalidParams
	}
	if filename == "" {
		zlog.CtxErrorf(ctx, "filename is empty")
		return "", ErrInvalidParams
	}

	// 文件大小限制
	if len(fileData) > MaxAvatarSize {
		zlog.CtxErrorf(ctx, "file size too large: %d bytes, max: %d", len(fileData), MaxAvatarSize)
		return "", fmt.Errorf("%w: file size exceeds 5MB", ErrInvalidParams)
	}

	// 验证文件类型（包含文件内容验证）
	contentType, err := validateImageType(fileData, filename)
	if err != nil {
		zlog.CtxErrorf(ctx, "invalid image type: %v", err)
		return "", fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	// 清理文件名（防止路径注入）
	sanitizedFilename, err := sanitizeFilename(filename)
	if err != nil {
		zlog.CtxErrorf(ctx, "invalid filename: %v", err)
		return "", fmt.Errorf("%w: invalid filename", ErrInvalidParams)
	}

	// 生成唯一文件名（避免覆盖）
	// 使用雪花ID保证唯一性，同时包含时间信息
	avatarID, err := util.GenerateStringID()
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to generate avatar ID: %v", err)
		return "", ErrInternalError
	}
	uniqueFilename := fmt.Sprintf("%s_%s", avatarID, sanitizedFilename)

	// 构建存储路径（使用path.Join防止路径注入）
	resourcePath := path.Join("user", userID, "avatar", uniqueFilename)

	// 调用基础设施层上传文件
	zlog.CtxInfof(ctx, "uploading avatar, userID: %s, resourcePath: %s, filename: %s", userID, resourcePath, sanitizedFilename)
	avatarURL, err := s.cosService.UploadFile(ctx, resourcePath, fileData, contentType)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to upload avatar, userID: %s, resourcePath: %s, error: %v", userID, resourcePath, err)
		return "", ErrInternalError
	}

	zlog.CtxInfof(ctx, "avatar uploaded successfully, userID: %s", userID)
	return avatarURL, nil
}

// validateImageType 验证是否为有效的图片类型（包含文件内容验证）
func validateImageType(fileData []byte, filename string) (string, error) {
	if len(fileData) < 8 {
		return "", fmt.Errorf("file too small")
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
	}

	expectedContentType, ok := validExts[ext]
	if !ok {
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}

	// 验证文件真实类型（魔数检查）
	var isValid bool
	switch {
	case bytes.HasPrefix(fileData, []byte{0xFF, 0xD8, 0xFF}):
		// JPEG: FF D8 FF
		isValid = (expectedContentType == "image/jpeg")
	case bytes.HasPrefix(fileData, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}):
		// PNG: 89 50 4E 47 0D 0A 1A 0A
		isValid = (expectedContentType == "image/png")
	case bytes.HasPrefix(fileData, []byte{0x47, 0x49, 0x46, 0x38}):
		// GIF: 47 49 46 38 (GIF8)
		isValid = (expectedContentType == "image/gif")
	case len(fileData) >= 12 && bytes.HasPrefix(fileData[8:], []byte("WEBP")):
		// WebP: RIFF....WEBP
		isValid = (expectedContentType == "image/webp")
	default:
		return "", fmt.Errorf("invalid image file: unrecognized file format")
	}

	if !isValid {
		return "", fmt.Errorf("file extension mismatch: expected %s but file content does not match", ext)
	}

	return expectedContentType, nil
}

// sanitizeFilename 清理文件名，防止路径注入
func sanitizeFilename(filename string) (string, error) {
	// 移除路径分隔符（只保留文件名部分）
	filename = filepath.Base(filename)

	// 移除所有非字母数字、点、下划线、连字符
	var clean strings.Builder
	for _, r := range filename {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '.' || r == '_' || r == '-' {
			clean.WriteRune(r)
		}
	}

	cleanName := clean.String()
	if cleanName == "" || cleanName == "." || cleanName == ".." {
		return "", fmt.Errorf("invalid filename")
	}

	// 限制文件名长度
	if len(cleanName) > 255 {
		cleanName = cleanName[:255]
	}

	return cleanName, nil
}
