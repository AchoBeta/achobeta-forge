package adapter

import "context"

// CodeService 验证码服务接口，支持邮件与短信
type CodeService interface {
	// SendEmailCode 发送邮件验证码
	SendEmailCode(ctx context.Context, email, code string) error
	// SendSMSCode 发送短信验证码
	SendSMSCode(ctx context.Context, phone, code string) error
}
