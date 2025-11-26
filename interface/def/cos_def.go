package def

// GetOSSCredentialsReq 获取OSS凭证请求
type GetOSSCredentialsReq struct {
	ResourcePath    string `json:"resource_path" binding:"required"` // 资源路径，如 user/123/avatar/profile.jpg
	DurationSeconds int64  `json:"duration_seconds"`                 // 有效期（秒），可选，默认3600，范围900-7200（最短15分钟，最长2小时）
}

// GetOSSCredentialsResp 获取OSS凭证响应
type GetOSSCredentialsResp struct {
	AccessKeyID     string `json:"access_key_id"`     // 临时访问密钥ID
	SecretAccessKey string `json:"secret_access_key"` // 临时访问密钥
	SessionToken    string `json:"session_token"`     // 会话令牌
	Expiration      int64  `json:"expiration"`        // 过期时间（Unix时间戳）
	Region          string `json:"region"`            // OSS地域
	BucketName      string `json:"bucket_name"`       // 存储桶名称
	Provider        string `json:"provider"`          // 提供商（cos）
	BaseURL         string `json:"base_url"`          // 访问基础URL
	AccountID       string `json:"account_id"`        // 账户ID
}
