package po

import (
	"time"

	"gorm.io/gorm"
)

type UserPO struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID   string `gorm:"column:user_id" json:"user_id"`
	UserName string `gorm:"column:username" json:"username"`
	Password string `gorm:"column:password" json:"password"`

	Avatar string `gorm:"column:avatar" json:"avatar"`
	Phone  string `gorm:"column:phone" json:"phone"`
	Email  string `gorm:"column:email" json:"email"`

	// 状态信息
	Status        int  `gorm:"column:status;default:1" json:"status"`
	PhoneVerified bool `gorm:"column:phone_verified;default:false" json:"phone_verified"`
	EmailVerified bool `gorm:"column:email_verified;default:false" json:"email_verified"`

	// 第三方登录
	GithubID      string `gorm:"column:github_id" json:"github_id"`
	GithubLogin   string `gorm:"column:github_login" json:"github_login"`
	WechatOpenID  string `gorm:"column:wechat_open_id" json:"wechat_open_id"`
	WechatUnionID string `gorm:"column:wechat_union_id" json:"wechat_union_id"`

	CreatedAt   *time.Time `gorm:"column:created_at" json:"create_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at" json:"updated_at"`
	IsDeleted   int8       `gorm:"column:is_deleted" json:"is_deleted"` // 已删除：1
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	//Extra     string     `gorm:"column:extra" json:"extra,omitempty"`
}

func (UserPO) TableName() string {
	return "achobeta_forge_user"
}

func (u *UserPO) BeforeCreate(tx *gorm.DB) error {
	return nil
}
