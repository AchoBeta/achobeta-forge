package storage

import (
	"context"
	"fmt"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/infra/database"
	"forge/infra/storage/po"

	"gorm.io/gorm"
)

type userPersistence struct {
	db *gorm.DB
}

var up *userPersistence

func InitUserStorage() {
	db := database.ForgeDB()

	// 自动迁移用户表
	if err := db.AutoMigrate(&po.UserPO{}); err != nil {
		panic(fmt.Sprintf("failed to auto migrate user table: %v", err))
	}

	up = &userPersistence{
		db: db,
	}
}

func GetUserPersistence() repo.UserRepo {
	return up
}

// CreateUser 创建用户
func (u *userPersistence) CreateUser(ctx context.Context, user *entity.User) error {
	userPO := CastUserDO2PO(user)
	err := u.db.WithContext(ctx).Create(&userPO).Error
	if err != nil {
		//todo 这里如何让上游更好地感知到错误类型，甚至前端感知到错误类型呢？
		return err
	}
	return nil
}

// 其他仓储

// UpdateUser 更新用户信息 - 统一的更新接口
func (u *userPersistence) UpdateUser(ctx context.Context, updateInfo *repo.UserUpdateInfo) error {
	if updateInfo == nil || updateInfo.UserID == "" {
		return fmt.Errorf("invalid update info: userID is required") // 需要id定位用户
	}

	updates := map[string]any{}

	// 基础信息
	if updateInfo.UserName != nil {
		updates["username"] = *updateInfo.UserName
	}
	if updateInfo.Avatar != nil {
		updates["avatar"] = *updateInfo.Avatar
	}

	// 联系方式
	if updateInfo.Phone != nil {
		updates["phone"] = *updateInfo.Phone
	}
	if updateInfo.Email != nil {
		updates["email"] = *updateInfo.Email
	}

	// 密码
	if updateInfo.Password != nil {
		updates["password"] = *updateInfo.Password
	}

	// 状态信息
	if updateInfo.Status != nil {
		updates["status"] = *updateInfo.Status
	}
	if updateInfo.PhoneVerified != nil {
		updates["phone_verified"] = *updateInfo.PhoneVerified
	}
	if updateInfo.EmailVerified != nil {
		updates["email_verified"] = *updateInfo.EmailVerified
	}

	// 第三方登录
	if updateInfo.GithubID != nil {
		updates["github_id"] = *updateInfo.GithubID
	}
	if updateInfo.GithubLogin != nil {
		updates["github_login"] = *updateInfo.GithubLogin
	}
	if updateInfo.WechatOpenID != nil {
		updates["wechat_open_id"] = *updateInfo.WechatOpenID
	}
	if updateInfo.WechatUnionID != nil {
		updates["wechat_union_id"] = *updateInfo.WechatUnionID
	}

	// 时间信息
	if updateInfo.LastLoginAt != nil {
		updates["last_login_at"] = *updateInfo.LastLoginAt
	}

	if len(updates) == 0 {
		return nil
	}

	return u.db.WithContext(ctx).Model(&po.UserPO{}).Where("user_id = ?", updateInfo.UserID).Updates(updates).Error
}

// GetUser 用户查询接口，根据查询条件获取用户
func (u *userPersistence) GetUser(ctx context.Context, query repo.UserQuery) (*entity.User, error) {
	var userPO po.UserPO

	// 根据查询条件构建查询
	db := u.db.WithContext(ctx)

	// 唯一标识直接查
	if query.UserID != "" {
		if err := db.Where("user_id = ? AND is_deleted = 0", query.UserID).First(&userPO).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}
			return nil, err
		}
		return CastUserPO2DO(&userPO), nil
	}

	var hasCond bool
	if query.UserName != "" {
		db = db.Where("username = ?", query.UserName)
		hasCond = true
	}
	if query.Phone != "" {
		db = db.Where("phone = ?", query.Phone)
		hasCond = true
	}
	if query.Email != "" {
		db = db.Where("email = ?", query.Email)
		hasCond = true
	}
	if query.GithubID != "" {
		db = db.Where("github_id = ?", query.GithubID)
		hasCond = true
	}
	if query.WechatOpenID != "" {
		db = db.Where("wechat_open_id = ?", query.WechatOpenID)
		hasCond = true
	}

	if !hasCond {
		return nil, fmt.Errorf("invalid user query: no query field provided")
	}

	// 未删除条件
	db = db.Where("is_deleted = 0")

	if err := db.First(&userPO).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return CastUserPO2DO(&userPO), nil
}
