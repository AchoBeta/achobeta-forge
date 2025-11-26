package types

import (
	"context"
)

type ICOSService interface {
	GetOSSCredentials(ctx context.Context, req *GetOSSCredentialsParams) (*OSSCredentials, error)

	// UploadAvatar 上传用户头像
	// 返回: 上传后的完整URL
	UploadAvatar(ctx context.Context, userID string, fileData []byte, filename string) (string, error)
}

// GetOSSCredentialsParams 获取OSS凭证参数
type GetOSSCredentialsParams struct {
	ResourcePath    string // 资源路径，如 user/123/avatar/profile.jpg
	DurationSeconds int64  // 有效期（秒），范围900-7200（最短15分钟，最长2小时）
}

// OSSCredentials OSS凭证信息
type OSSCredentials struct {
	AccessKeyID     string // 临时访问密钥ID
	SecretAccessKey string // 临时访问密钥
	SessionToken    string // 会话令牌
	Expiration      int64  // 过期时间（Unix时间戳）
	Region          string // OSS地域
	BucketName      string // 存储桶名称
	Provider        string // 提供商（cos）
	BaseURL         string // 访问基础URL
	AccountID       string // 账户ID
}
