package entity

import (
	"context"
	"forge/pkg/log/zlog"
	"time"

	"go.uber.org/zap"
)

// 实体关注业务 无需gorm tag
type User struct {
	UserID   string `json:"user_id"`   // 用户ID
	UserName string `json:"user_name"` // 用户名
	Password string `json:"-"`         //密码 无json
	Avatar   string `json:"avatar"`    // 头像URL

	//登录方式  二选一
	Phone string `json:"phone"` // 手机号
	Email string `json:"email"` // 邮箱

	//用户状态 1：正常 0：禁用
	Status int `json:"status"`

	// 时间信息
	CreatedAt   time.Time  `json:"created_at"`    // 创建时间
	UpdatedAt   time.Time  `json:"updated_at"`    // 更新时间
	LastLoginAt *time.Time `json:"last_login_at"` // 最后登录时间

	PhoneVerified bool `json:"phone_verified"` // 手机号是否已验证
	EmailVerified bool `json:"email_verified"` // 邮箱是否已验证

	// 第三方登录
	GithubID      string `json:"github_id,omitempty"`      // GitHub 用户 ID
	GithubLogin   string `json:"github_login,omitempty"`   // GitHub 用户名
	WechatOpenID  string `json:"wechat_open_id,omitempty"` // 微信 openid
	WechatUnionID string `json:"wechat_union_id,omitempty"` // 微信 unionid（开放平台）

	Dogs []*Dog
	// ... ex
}

// 用户状态常量
const (
	UserStatusActive   = 1 // 正常
	UserStatusDisabled = 0 // 禁用
)

type Dog struct {
	DogID   string
	DogName string
}

type userCtxKey struct{}

func WithUser(ctx context.Context, user *User) context.Context {
	// 设置用户链路 todo可以考虑在jwt层设置
	ctx = zlog.WithLogKey(ctx, zap.String("user_id", user.UserID))
	// 存储指针，避免值拷贝
	ctx = context.WithValue(ctx, userCtxKey{}, user)
	return ctx
}

func GetUser(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userCtxKey{}).(*User)
	return user, ok
}
