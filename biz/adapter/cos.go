package adapter

import (
	"context"

	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

// COSService 定义COS服务的业务接口
type COSService interface {
	GetTemporaryCredentials(resourcePath string, durationSeconds int64) (*sts.CredentialResult, error)

	// UploadFile 上传文件到COS
	// resourcePath: 存储路径，如 "user/123/avatar/avatar.jpg"
	// fileData: 文件内容
	// contentType: 文件类型，如 "image/jpeg"
	// 返回: 完整URL
	UploadFile(ctx context.Context, resourcePath string, fileData []byte, contentType string) (string, error)
}
