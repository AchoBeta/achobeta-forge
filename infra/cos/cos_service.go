package cos

import (
	"bytes"
	"context"
	"fmt"
	"forge/biz/adapter"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"net/http"
	"net/url"

	"github.com/tencentyun/cos-go-sdk-v5"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

type cosServiceImpl struct {
	config    configs.COSConfig
	stsClient *sts.Client // 老大原先的 用于获取临时凭证
	cosClient *cos.Client // COS上传客户端
}

// NewCOSService 创建COS服务实例（依赖注入模式）
// 通过构造函数接收配置，返回接口类型，便于测试和依赖注入
func NewCOSService(cfg configs.COSConfig) adapter.COSService {
	stsClient := sts.NewClient(
		cfg.SecretID,
		cfg.SecretKey,
		nil,
	)

	// 创建COS上传客户端
	// 腾讯云COS的正确格式：{bucket}-{app_id}.cos.{region}.myqcloud.com    cfg.Bucket, cfg.AppID, cfg.Region
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.AppID, cfg.Region))
	if err != nil {
		zlog.Errorf("invalid bucket URL: %v", err)
		panic(fmt.Sprintf("invalid bucket URL: %v", err))
	}
	cosClient := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})

	service := &cosServiceImpl{
		config:    cfg,
		stsClient: stsClient,
		cosClient: cosClient,
	}

	zlog.Infof("COS service created successfully, region: %s, bucket: %s", cfg.Region, cfg.Bucket)
	return service
}

// GetTemporaryCredentials 获取COS临时凭证
func (c *cosServiceImpl) GetTemporaryCredentials(resourcePath string, durationSeconds int64) (*sts.CredentialResult, error) {
	// 构建对象级别的资源ARN（用于具体对象的操作）
	resourceArn := fmt.Sprintf(
		"qcs::cos:%s:uid/%s:%s-%s/%s",
		c.config.Region,
		c.config.AppID,
		c.config.Bucket,
		c.config.AppID,
		resourcePath,
	)

	// 构建存储桶级别的资源ARN（用于存储桶级别的操作，如 ListMultipartUploads）
	bucketArn := fmt.Sprintf(
		"qcs::cos:%s:uid/%s:%s-%s/",
		c.config.Region,
		c.config.AppID,
		c.config.Bucket,
		c.config.AppID,
	)

	// 配置STS策略
	// 注意：ListMultipartUploads 需要存储桶级别权限，必须单独授权到存储桶ARN
	opt := &sts.CredentialOptions{
		DurationSeconds: durationSeconds,
		Region:          c.config.Region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					// Statement 1: 对象级别的操作（文件上传、下载、分片上传等）
					Action: []string{
						// 简单上传操作
						"name/cos:PostObject",
						"name/cos:PutObject",
						"name/cos:GetObject",
						"name/cos:HeadObject",
						// 分片上传操作（对象级别）
						"name/cos:InitiateMultipartUpload",
						"name/cos:ListParts",
						"name/cos:UploadPart",
						"name/cos:CompleteMultipartUpload",
					},
					Effect: "allow",
					Resource: []string{
						resourceArn,
					},
				},
				{
					// Statement 2: 存储桶级别的操作（列出所有分片上传任务）
					Action: []string{
						"name/cos:ListMultipartUploads",
					},
					Effect: "allow",
					Resource: []string{
						bucketArn,
					},
				},
			},
		},
	}

	// 请求临时凭证
	result, err := c.stsClient.GetCredential(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to get COS STS credentials: %w", err)
	}

	return result, nil
}

// UploadFile 上传文件到COS
func (c *cosServiceImpl) UploadFile(ctx context.Context, resourcePath string, fileData []byte, contentType string) (string, error) {
	// 上传文件
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: contentType,
		},
	}

	_, err := c.cosClient.Object.Put(ctx, resourcePath, bytes.NewReader(fileData), opt)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to upload file to COS, path: %s, error: %v", resourcePath, err)
		return "", fmt.Errorf("failed to upload file to COS: %w", err)
	}

	// 构建完整URL（使用url.JoinPath正确处理URL拼接，避免双斜杠问题）
	fullURL, err := url.JoinPath(c.config.BaseURL, resourcePath)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to construct file URL: %v", err)
		return "", fmt.Errorf("failed to construct file URL: %w", err)
	}

	zlog.CtxInfof(ctx, "file uploaded successfully to COS, path: %s", resourcePath)
	return fullURL, nil
}
