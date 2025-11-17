package userservice

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"path"
	"strings"
	"time"

	"forge/biz/adapter"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/constant"
	"forge/infra/cache"
	"forge/pkg/log/zlog"
	"forge/util"

	"github.com/markbates/goth"
)

var (
	// ErrUserNotFound 表示用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists 表示账号已存在
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidParams 表示参数无效
	ErrInvalidParams = errors.New("invalid params")
	// ErrPasswordMismatch 表示密码不一致
	ErrPasswordMismatch = errors.New("password mismatch")
	// ErrCredentialsIncorrect 表示账号或密码错误
	ErrCredentialsIncorrect = errors.New("credentials incorrect")
	// ErrUnsupportedAccountType 表示不支持的账号类型
	ErrUnsupportedAccountType = errors.New("unsupported account type")
	// ErrInternalError 表示内部错误
	ErrInternalError = errors.New("internal error")
	// ErrPermissionDenied 表示权限被拒绝
	ErrPermissionDenied = errors.New("permission denied")
	// ErrVerificationCodeIncorrect 表示验证码错误
	ErrVerificationCodeIncorrect = errors.New("verification code incorrect")
	// ErrAccountAlreadyInUse 表示账号（手机号/邮箱）已被使用
	ErrAccountAlreadyInUse = errors.New("account already in use")
	ErrEmailAlreadyInUse   = ErrAccountAlreadyInUse
	// ErrPasswordRequired 表示密码必填
	ErrPasswordRequired        = errors.New("password required")
	ErrCannotUnbindOnlyContact = errors.New("cannot unbind only contact")
)

// 最好的设计方案：
// infra的所有函数都是通过接口来用的

type UserServiceImpl struct {
	userRepo    repo.UserRepo
	cozeService adapter.CozeService
	jwtUtil     *util.JWTUtil
	codeService adapter.CodeService
}

func NewUserServiceImpl(
	userRepo repo.UserRepo,
	cozeService adapter.CozeService,
	jwtUtil *util.JWTUtil,
	codeService adapter.CodeService) *UserServiceImpl {
	return &UserServiceImpl{
		userRepo:    userRepo,
		cozeService: cozeService,
		jwtUtil:     jwtUtil,
		codeService: codeService,
	}
}

// Login 登录：根据账号和密码进行登录
func (u *UserServiceImpl) Login(ctx context.Context, account, accountType, password string) (*entity.User, string, error) {
	// 参数校验
	if account == "" || accountType == "" || password == "" {
		zlog.CtxErrorf(ctx, "invalid params for login: account, accountType or password is empty")
		return nil, "", ErrInvalidParams
	}

	// 根据账号类型查找用户
	user, err := u.findUserByAccount(ctx, account, accountType)
	if err != nil {
		// 如果用户不存在，返回错误
		if errors.Is(err, ErrUserNotFound) {
			zlog.CtxErrorf(ctx, "user not found: %s", account)
			return nil, "", ErrCredentialsIncorrect
		}
		// 其他错误（数据库错误等）
		return nil, "", err
	}

	// 验证密码
	match, err := util.ComparePassword(user.Password, password)
	if err != nil {
		zlog.CtxErrorf(ctx, "compare password failed: %v", err)
		return nil, "", ErrInternalError
	}
	if !match {
		zlog.CtxErrorf(ctx, "password incorrect for user: %s", user.UserID)
		return nil, "", ErrCredentialsIncorrect
	}

	// 生成JWT token
	token, err := u.jwtUtil.GenerateToken(user.UserID)
	if err != nil {
		zlog.CtxErrorf(ctx, "generate token failed: %v", err)
		return nil, "", ErrInternalError
	}

	// 方法一  通过注入的 cozeService 接口调用
	//result, err := u.cozeService.RunWorkflow(ctx, &adapter.RunWorkflowReq{})
	//if err != nil {
	//	zlog.CtxErrorf(ctx, "run workflow failed: %v", err)
	//} else {
	//	zlog.CtxInfof(ctx, "result:%v", result)
	//}

	// 方法二
	// result, err = coze.GetCozeService().RunWorkflow(ctx, &adapter.RunWorkflowReq{})
	// if err != nil {
	// 	zlog.CtxErrorf(ctx, "run workflow failed: %v", err)
	// 	return nil, "", err
	// }
	// zlog.CtxInfof(ctx, "result:%v", result)
	// ============================================================

	// 更新最后登录时间（可选）
	// lastLoginAt := time.Now()
	// updateInfo := &repo.UserUpdateInfo{
	// 	UserID:     user.UserID,
	// 	LastLoginAt: &lastLoginAt,
	// }
	// _ = u.userRepo.UpdateUser(ctx, updateInfo)

	zlog.CtxInfof(ctx, "login success for user: %s", user.UserID)
	return user, token, nil
}

// Register 基于手机号/邮箱进行注册
func (u *UserServiceImpl) Register(ctx context.Context, req *types.RegisterParams) (*entity.User, error) {
	// 基本校验
	if req.Account == "" || req.AccountType == "" || req.Password == "" {
		zlog.CtxErrorf(ctx, "invalid params for register")
		return nil, ErrInvalidParams
	}

	// 检查账号是否已存在
	existUser, err := u.findUserByAccount(ctx, req.Account, req.AccountType)
	if err != nil {
		// 账号不存在，可以继续注册
		if errors.Is(err, ErrUserNotFound) {
			// 用户不存在，继续注册流程
		} else {
			// 其他错误，直接返回
			return nil, err
		}
	} else if existUser != nil {
		// 用户已存在，返回错误
		var accountField string
		if req.AccountType == types.AccountTypePhone {
			accountField = "phone"
		} else {
			accountField = "email"
		}
		zlog.CtxErrorf(ctx, "%s already registered: %s", accountField, req.Account)
		return nil, ErrUserAlreadyExists
	}

	// 校验验证码 code（短信/邮箱）
	if err := u.VerifyCode(ctx, req.Account, req.AccountType, req.Code); err != nil {
		return nil, err
	}

	//------------------------------------------------

	// 验证密码强度  按照常规要求设置
	if err := util.ValidatePasswordStrength(req.Password); err != nil {
		zlog.CtxErrorf(ctx, "password strength validation failed: %v", err)
		return nil, err
	}

	// 生成用户ID  snowflake雪花id
	userID, err := util.GenerateStringID()
	if err != nil {
		zlog.CtxErrorf(ctx, "generate user id failed: %v", err)
		return nil, ErrInternalError
	}
	//

	// 加密密码
	hash, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, ErrInternalError
	}

	// 组装实体 仓储接口写入数据库持久化
	user := &entity.User{
		UserID:   userID,
		UserName: req.UserName,
		Password: hash,
		// 根据 accountType 填写登录方式字段
	}
	switch req.AccountType {
	case types.AccountTypePhone:
		user.Phone = req.Account
		user.PhoneVerified = true
	case types.AccountTypeEmail:
		user.Email = req.Account
		user.EmailVerified = true
	}

	if err := u.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// findUserByAccount 根据账号类型查找用户 抽离重复判断逻辑
// 返回值说明：
//   - 如果返回错误不为nil，表示数据库查询出错（内部错误）或账号类型不支持
//   - 如果用户为nil且错误为nil，表示用户不存在，返回"user not found"错误
//   - 如果用户不为nil，表示找到用户，正常返回
func (u *UserServiceImpl) findUserByAccount(ctx context.Context, account, accountType string) (*entity.User, error) {
	var query repo.UserQuery
	var accountField string

	switch accountType {
	case types.AccountTypePhone:
		query = repo.NewUserQueryByPhone(account)
		accountField = "phone"
	case types.AccountTypeEmail:
		query = repo.NewUserQueryByEmail(account)
		accountField = "email"
	default:
		zlog.CtxErrorf(ctx, "unsupported accountType: %s", accountType)
		return nil, ErrUnsupportedAccountType
	}

	user, err := u.userRepo.GetUser(ctx, query)
	if err != nil {
		// 数据库查询错误，返回内部错误
		zlog.CtxErrorf(ctx, "failed to get user by %s: %v", accountField, err)
		return nil, ErrInternalError
	}

	if user == nil {
		// 用户不存在
		return nil, ErrUserNotFound
	}

	return user, nil
}

// ResetPassword 重置密码
func (u *UserServiceImpl) ResetPassword(ctx context.Context, req *types.ResetPasswordParams) error {
	// 参数校验
	if req == nil {
		zlog.CtxErrorf(ctx, "reset password request is nil")
		return ErrInvalidParams
	}
	if req.Account == "" || req.AccountType == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		zlog.CtxErrorf(ctx, "invalid params for reset password: missing required fields")
		return ErrInvalidParams
	}

	// 校验两次密码一致性
	if req.NewPassword != req.ConfirmPassword {
		zlog.CtxErrorf(ctx, "password and confirm password do not match")
		return ErrPasswordMismatch
	}

	// 根据账号类型查找用户
	user, err := u.findUserByAccount(ctx, req.Account, req.AccountType)
	if err != nil {
		return err
	}

	// 校验验证码 code（短信/邮箱）
	if err := u.VerifyCode(ctx, req.Account, req.AccountType, req.Code); err != nil {
		return err
	}

	// 验证新密码强度
	if err := util.ValidatePasswordStrength(req.NewPassword); err != nil {
		zlog.CtxErrorf(ctx, "password strength validation failed: %v", err)
		return err
	}

	// 加密新密码
	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		zlog.CtxErrorf(ctx, "hash password failed: %v", err)
		return ErrInternalError
	}

	// 更新用户密码
	password := hash
	updateInfo := &repo.UserUpdateInfo{
		UserID:   user.UserID,
		Password: &password,
	}
	if err := u.userRepo.UpdateUser(ctx, updateInfo); err != nil {
		zlog.CtxErrorf(ctx, "update password failed: %v", err)
		return ErrInternalError
	}

	zlog.CtxInfof(ctx, "reset password successfully for user: %s", user.UserID)
	return nil
}

// GetUserByID 根据用户ID获取用户信息（用于JWT鉴权等场景）
func (u *UserServiceImpl) GetUserByID(ctx context.Context, userID string) (*entity.User, error) {
	// 参数校验
	if userID == "" {
		zlog.CtxErrorf(ctx, "userID is required")
		return nil, ErrInvalidParams
	}

	// 通过repo查询用户
	query := repo.NewUserQueryByID(userID)
	user, err := u.userRepo.GetUser(ctx, query)
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to get user by ID: %v", err)
		return nil, ErrInternalError
	}

	if user == nil {
		zlog.CtxWarnf(ctx, "user not found: %s", userID)
		return nil, ErrUserNotFound
	}

	// 检查用户状态（业务逻辑应该在service层）
	if user.Status != entity.UserStatusActive {
		zlog.CtxWarnf(ctx, "user is disabled: %s", userID)
		return nil, ErrPermissionDenied
	}

	return user, nil
}

// SendVerificationCode 发送验证码
func (u *UserServiceImpl) SendVerificationCode(ctx context.Context, account, accountType, purpose string) error {
	// 参数校验
	if account == "" || accountType == "" {
		zlog.CtxErrorf(ctx, "invalid params for send verification code")
		return ErrInvalidParams
	}

	// 根据使用场景进行账号验证
	// 注册 换绑需要提供未被使用的账号   重置密码需要提供用户自己的 存在的账号
	switch purpose {
	case types.PurposeRegister:
		// 注册场景：账号应该不存在，如果已存在则返回错误
		_, err := u.findUserByAccount(ctx, account, accountType)
		if err != nil {
			// 如果是用户不存在的错误，说明账号未被使用，可以继续发送验证码
			if !errors.Is(err, ErrUserNotFound) {
				// 其他错误（数据库错误等），返回内部错误
				zlog.CtxErrorf(ctx, "failed to check if account exists: %v", err)
				return ErrInternalError
			}
			// ErrUserNotFound 表示账号未被使用，可以继续
		} else {
			// 账号已被使用，返回错误
			// 当 err == nil 时，说明找到了用户（findUserByAccount 保证）
			zlog.CtxWarnf(ctx, "account already in use for register: %s (type: %s)", account, accountType)
			return ErrAccountAlreadyInUse
		}

	case types.PurposeResetPassword:
		// 重置密码场景：账号应该存在，如果不存在则返回错误
		_, err := u.findUserByAccount(ctx, account, accountType)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				// 用户不存在，返回错误
				zlog.CtxWarnf(ctx, "user not found for reset password: %s (type: %s)", account, accountType)
				return ErrUserNotFound
			}
			// 其他错误（数据库错误等），返回内部错误
			zlog.CtxErrorf(ctx, "failed to check if account exists: %v", err)
			return ErrInternalError
		}
		// err == nil 时，说明用户存在（findUserByAccount 保证）

	case types.PurposeChangeAccount:
		// 换绑联系方式场景：需要从context获取当前用户，检查新账号是否被其他用户使用
		currentUser, ok := entity.GetUser(ctx)
		if !ok {
			zlog.CtxErrorf(ctx, "user not found in context for change account")
			return ErrPermissionDenied
		}
		if err := u.checkAccountAvailabilityForUpdate(ctx, currentUser, account, accountType); err != nil {
			return err
		}

	default:
		// 未指定场景或未知场景，不进行验证（向后兼容）
		zlog.CtxWarnf(ctx, "unknown purpose for send verification code: %s, skipping validation", purpose)
	}

	// 生成6位随机验证码
	code := generateVerificationCode()

	// 先将验证码存储到 Redis，并设置过期时间
	key := fmt.Sprintf(constant.REDIS_VERIFICATION_CODE_KEY, account)
	// TODO: 建议将过期时间（10分钟）配置化
	expiration := 10 * time.Minute
	if err := cache.SetRedis(ctx, key, code, expiration); err != nil {
		zlog.CtxErrorf(ctx, "存储验证码到Redis失败: %v", err)
		return ErrInternalError
	}

	var (
		sendFunc func(context.Context, string, string) error
		errorLog string
	)

	switch accountType {
	case types.AccountTypeEmail:
		sendFunc = u.codeService.SendEmailCode
		errorLog = "send verification code failed"
	case types.AccountTypePhone:
		sendFunc = u.codeService.SendSMSCode
		errorLog = "send sms verification code failed"
	default:
		zlog.CtxErrorf(ctx, "unsupported account type for verification: %s", accountType)
		return ErrUnsupportedAccountType
	}

	if err := sendFunc(ctx, account, code); err != nil {
		zlog.CtxErrorf(ctx, "%s: %v", errorLog, err)
		if delErr := cache.DelRedis(ctx, key); delErr != nil {
			zlog.CtxErrorf(ctx, "删除Redis中未发送成功的验证码失败: %v", delErr)
		}
		return ErrInternalError
	}

	return nil
}

// VerifyCode 校验验证码
func (u *UserServiceImpl) VerifyCode(ctx context.Context, account, accountType, code string) error {
	if account == "" || code == "" {
		return ErrInvalidParams
	}

	// 从Redis获取验证码
	key := fmt.Sprintf(constant.REDIS_VERIFICATION_CODE_KEY, account)
	storedCode, err := cache.GetRedis(ctx, key)
	if err != nil {
		zlog.CtxErrorf(ctx, "get verification code from redis failed: %v", err)
		return ErrInternalError
	}

	if storedCode == "" {
		zlog.CtxWarnf(ctx, "verification code not found or expired for: %s", account)
		return ErrVerificationCodeIncorrect
	}

	if storedCode != code {
		zlog.CtxWarnf(ctx, "verification code mismatch for: %s", account)
		return ErrVerificationCodeIncorrect
	}

	// 校验成功后删除验证码（一次性使用）
	if err := cache.DelRedis(ctx, key); err != nil {
		zlog.CtxErrorf(ctx, "delete verification code from redis failed: %v", err)
		// 不返回错误，因为验证码已经校验成功
	}

	return nil
}

// checkAccountAvailabilityForUpdate 检查账号是否可用于更新（换绑/绑定）
// 检查新账号是否被其他用户使用，如果是当前用户自己的账号则允许
func (u *UserServiceImpl) checkAccountAvailabilityForUpdate(ctx context.Context, currentUser *entity.User, account, accountType string) error {
	existingUser, err := u.findUserByAccount(ctx, account, accountType)
	if err != nil {
		// 如果是用户不存在的错误，说明新账号未被使用，可以继续
		if !errors.Is(err, ErrUserNotFound) {
			// 其他错误（数据库错误等），返回内部错误
			zlog.CtxErrorf(ctx, "failed to check if account exists: %v", err)
			return ErrInternalError
		}
		// ErrUserNotFound 表示新账号未被使用，可以继续
		return nil
	}

	// 找到用户，检查是否是当前用户自己的账号
	// 当 err == nil 时，existingUser 一定不为 nil（findUserByAccount 保证）
	if existingUser.UserID != currentUser.UserID {
		// 被其他用户使用，返回错误
		zlog.CtxWarnf(ctx, "account already in use by another user: %s (type: %s)", account, accountType)
		return ErrAccountAlreadyInUse
	}
	// 是自己的账号，可以继续（允许用户重新验证自己的账号）

	return nil
}

// UpdateAccount 更新联系方式（绑定/换绑手机号或邮箱）
func (u *UserServiceImpl) UpdateAccount(ctx context.Context, req *types.UpdateAccountParams) (string, error) {
	// 参数校验
	if req == nil {
		zlog.CtxErrorf(ctx, "update account request is nil")
		return "", ErrInvalidParams
	}
	if req.Account == "" || req.AccountType == "" || req.Code == "" {
		zlog.CtxErrorf(ctx, "invalid params for update account: missing required fields")
		return "", ErrInvalidParams
	}

	// 从context获取当前用户（JWT中间件已注入）
	currentUser, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "user not found in context, this should not happen if JWT middleware works correctly")
		return "", ErrPermissionDenied
	}

	// 判断用户是否有密码
	hasPassword := currentUser.Password != ""
	if !hasPassword && req.Password == "" {
		zlog.CtxErrorf(ctx, "password required for user without password: %s", currentUser.UserID)
		return "", ErrPasswordRequired
	}

	// 验证验证码（验证发送到新联系方式的验证码）
	if err := u.VerifyCode(ctx, req.Account, req.AccountType, req.Code); err != nil {
		return "", err
	}

	// 检查新联系方式是否被其他用户使用
	if err := u.checkAccountAvailabilityForUpdate(ctx, currentUser, req.Account, req.AccountType); err != nil {
		return "", err
	}

	// 准备更新信息
	updateInfo := &repo.UserUpdateInfo{
		UserID: currentUser.UserID,
	}

	// 更新联系方式
	trueValue := true
	switch req.AccountType {
	case types.AccountTypePhone:
		updateInfo.Phone = &req.Account
		updateInfo.PhoneVerified = &trueValue
	case types.AccountTypeEmail:
		updateInfo.Email = &req.Account
		updateInfo.EmailVerified = &trueValue
	default:
		zlog.CtxErrorf(ctx, "unsupported account type: %s", req.AccountType)
		return "", ErrUnsupportedAccountType
	}

	// 如果传了密码，更新密码
	if req.Password != "" {
		// 验证密码强度
		if err := util.ValidatePasswordStrength(req.Password); err != nil {
			zlog.CtxErrorf(ctx, "password strength validation failed: %v", err)
			return "", err
		}

		// 加密密码
		hash, err := util.HashPassword(req.Password)
		if err != nil {
			zlog.CtxErrorf(ctx, "hash password failed: %v", err)
			return "", ErrInternalError
		}
		updateInfo.Password = &hash
	}

	// 更新用户信息
	if err := u.userRepo.UpdateUser(ctx, updateInfo); err != nil {
		zlog.CtxErrorf(ctx, "update account failed: %v", err)
		return "", ErrInternalError
	}

	zlog.CtxInfof(ctx, "account updated successfully, userID: %s, new account: %s", currentUser.UserID, req.Account)
	return req.Account, nil
}

// UnbindAccount 解绑联系方式（手机号/邮箱/第三方账号）
func (u *UserServiceImpl) UnbindAccount(ctx context.Context, req *types.UnbindAccountParams) error {
	// 参数校验
	if req == nil {
		zlog.CtxErrorf(ctx, "unbind account request is nil")
		return ErrInvalidParams
	}
	if req.AccountType == "" {
		zlog.CtxErrorf(ctx, "invalid params for unbind account: account_type is required")
		return ErrInvalidParams
	}

	// 获取当前用户
	currentUser, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "user not found in context for unbind account")
		return ErrPermissionDenied
	}

	// 使用 var 定义变量来减少重复逻辑
	var (
		currentBoundValue string // 当前绑定的值（手机号/邮箱/第三方ID）
		accountLabel      string // 账号标签（用于日志）
		needAccountParam  bool   // 是否需要 account 参数（手机号和邮箱需要，第三方不需要）
	)

	// 准备更新信息
	updateInfo := &repo.UserUpdateInfo{
		UserID: currentUser.UserID,
	}
	falseValue := false
	emptyString := ""

	// 根据账号类型设置变量和更新字段（合并到一个 switch 中）
	switch req.AccountType {
	case types.AccountTypePhone:
		currentBoundValue = currentUser.Phone
		accountLabel = "phone"
		needAccountParam = true
		updateInfo.Phone = &emptyString
		updateInfo.PhoneVerified = &falseValue
	case types.AccountTypeEmail:
		currentBoundValue = currentUser.Email
		accountLabel = "email"
		needAccountParam = true
		updateInfo.Email = &emptyString
		updateInfo.EmailVerified = &falseValue
	case types.AccountTypeGithub:
		currentBoundValue = currentUser.GithubID
		accountLabel = "github"
		needAccountParam = false
		updateInfo.GithubID = &emptyString
		updateInfo.GithubLogin = &emptyString
	case types.AccountTypeWechat:
		currentBoundValue = currentUser.WechatOpenID
		accountLabel = "wechat"
		needAccountParam = false
		updateInfo.WechatOpenID = &emptyString
		updateInfo.WechatUnionID = &emptyString
	default:
		zlog.CtxErrorf(ctx, "unsupported account type for unbind: %s", req.AccountType)
		return ErrUnsupportedAccountType
	}

	// 统一参数校验：需要 account 参数的类型
	if needAccountParam {
		if req.Account == "" {
			zlog.CtxErrorf(ctx, "account is required for unbind %s", accountLabel)
			return ErrInvalidParams
		}
		if req.Account != currentBoundValue {
			zlog.CtxErrorf(ctx, "%s mismatch for unbind, userID: %s, request %s: %s", accountLabel, currentUser.UserID, accountLabel, req.Account)
			return ErrInvalidParams
		}
	}

	// 统一检查：是否已绑定
	if currentBoundValue == "" {
		zlog.CtxErrorf(ctx, "%s is not bound, userID: %s", accountLabel, currentUser.UserID)
		return ErrInvalidParams
	}

	// 统一检查：是否还有其他登录方式
	if !u.hasOtherLoginMethod(currentUser, req.AccountType) {
		zlog.CtxErrorf(ctx, "cannot unbind %s, no other login method available, userID: %s", accountLabel, currentUser.UserID)
		return ErrCannotUnbindOnlyContact
	}

	// 执行更新
	if err := u.userRepo.UpdateUser(ctx, updateInfo); err != nil {
		zlog.CtxErrorf(ctx, "unbind account failed: %v", err)
		return ErrInternalError
	}

	zlog.CtxInfof(ctx, "account unbound successfully, userID: %s, accountType: %s", currentUser.UserID, req.AccountType)
	return nil
}

// hasOtherLoginMethod 检查用户是否还有其他登录方式（排除指定的账号类型）
func (u *UserServiceImpl) hasOtherLoginMethod(user *entity.User, excludeAccountType string) bool {
	// 直接检查所有登录方式，排除当前要解绑的账号类型
	return (user.Phone != "" && excludeAccountType != types.AccountTypePhone) ||
		(user.Email != "" && excludeAccountType != types.AccountTypeEmail) ||
		user.Password != "" ||
		(user.GithubID != "" && excludeAccountType != types.AccountTypeGithub) ||
		(user.WechatOpenID != "" && excludeAccountType != types.AccountTypeWechat)
}

// generateVerificationCode 生成6位随机验证码
func generateVerificationCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// crypto/rand 的失败是一个罕见且严重的事件，表明系统的熵源存在问题。
		// 在这种情况下，记录严重错误并 panic 是一个合理的做法。
		panic(fmt.Sprintf("failed to generate cryptographically secure random number for verification code: %v", err))
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// UpdateAvatar 更新用户头像
func (u *UserServiceImpl) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	// 参数校验
	if userID == "" || avatarURL == "" {
		zlog.CtxErrorf(ctx, "invalid params for update avatar: userID or avatarURL is empty")
		return ErrInvalidParams
	}

	// URL验证
	if err := validateAvatarURL(ctx, avatarURL); err != nil {
		zlog.CtxErrorf(ctx, "avatar URL validation failed: %v", err)
		// 包装错误以保留详细验证信息，同时仍可用 errors.Is 检查错误类型
		return fmt.Errorf("%w: %v", ErrInvalidParams, err) // 保留详细错误
	}

	// 检查用户是否存在（GetUserByID 包含状态检查）
	_, err := u.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// 更新头像
	updateInfo := &repo.UserUpdateInfo{
		UserID: userID,
		Avatar: &avatarURL,
	}
	if err := u.userRepo.UpdateUser(ctx, updateInfo); err != nil {
		zlog.CtxErrorf(ctx, "update avatar failed: %v", err)
		return ErrInternalError
	}

	zlog.CtxInfof(ctx, "update avatar successfully for user: %s", userID)
	return nil
}

// UpdateUserName 更新用户名
func (u *UserServiceImpl) UpdateUserName(ctx context.Context, userID, newUserName string) error {
	// 参数校验
	if userID == "" {
		zlog.CtxErrorf(ctx, "invalid params for update username: userID is empty")
		return ErrInvalidParams
	}
	if newUserName == "" {
		zlog.CtxErrorf(ctx, "invalid params for update username: newUserName is empty")
		return ErrInvalidParams
	}

	// 检查用户名长度
	if len(newUserName) > 50 {
		zlog.CtxErrorf(ctx, "username too long: %s", newUserName)
		return ErrInvalidParams
	}

	// 检查用户是否存在
	_, err := u.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// 更新用户名
	updateInfo := &repo.UserUpdateInfo{
		UserID:   userID,
		UserName: &newUserName,
	}
	if err := u.userRepo.UpdateUser(ctx, updateInfo); err != nil {
		zlog.CtxErrorf(ctx, "update username failed: %v", err)
		return ErrInternalError
	}

	zlog.CtxInfof(ctx, "username updated successfully, userID: %s, newUserName: %s", userID, newUserName)
	return nil
}

// validateAvatarURL URL验证函数
// 注意：移除了路径格式强制检查（原 /user/{userID}/avatar/），允许使用外部服务
// 如果需要对自有存储路径进行限制，应该在存储访问层（COS IAM策略）实现
func validateAvatarURL(ctx context.Context, avatarURL string) error {
	// 1. URL长度限制（防止过长的URL）
	const maxURLLength = 2048 // RFC 7230 建议的最大URL长度
	if len(avatarURL) > maxURLLength {
		return fmt.Errorf("avatar URL too long: exceeds %d characters", maxURLLength)
	}

	// 2. 使用标准库解析URL
	parsedURL, err := url.Parse(avatarURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// 3. 验证协议（只允许http和https）
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("invalid URL scheme: only http and https are allowed, got %s", scheme)
	}

	// 4. 验证Host不为空
	if parsedURL.Host == "" {
		return fmt.Errorf("invalid URL: host is required")
	}

	// 5. 验证Host格式（不能包含危险字符）
	// 注意：移除了对 ".." 的检查，因为主机名中的 ".." 不是安全问题（路径遍历发生在路径部分）
	// 虽然 url.Parse 通常会处理 "//"，但保留检查以防格式错误
	if strings.Contains(parsedURL.Host, "//") {
		return fmt.Errorf("invalid URL: host contains invalid characters")
	}

	// 6. SSRF 防护：禁止访问内网/私有IP地址
	// 使用 Hostname() 方法提取主机名，自动处理端口和 IPv6 方括号
	host := parsedURL.Hostname()

	// 解析 IP 地址
	ip := net.ParseIP(host)
	if ip != nil {
		// 如果是 IP 地址，检查是否为私有/保留地址
		if isPrivateIP(ip) {
			return fmt.Errorf("invalid URL: private/internal IP addresses are not allowed for security reasons")
		}
	} else {
		// 如果是域名，解析为 IP 并检查
		ips, err := net.LookupIP(host)
		if err != nil {
			// 域名解析失败，拒绝URL（可能是恶意域名或网络问题）
			zlog.CtxErrorf(ctx, "failed to resolve host %s: %v", host, err)
			return fmt.Errorf("invalid URL: failed to resolve host %s", host)
		}

		// 检查所有解析出的 IP 地址
		if len(ips) == 0 {
			return fmt.Errorf("invalid URL: host %s resolves to no IP addresses", host)
		}

		for _, resolvedIP := range ips {
			if isPrivateIP(resolvedIP) {
				return fmt.Errorf("invalid URL: host %s resolves to private/internal IP address", host)
			}
		}
	}

	// 7. 验证路径中不能包含危险字符（防止路径遍历攻击）
	if strings.Contains(parsedURL.Path, "..") || strings.Contains(parsedURL.Path, "//") {
		return fmt.Errorf("invalid URL path: contains dangerous characters")
	}

	// 8. 允许查询参数（外部服务如 Gravatar、CDN 需要查询参数）
	// 但禁止锚点（Fragment），因为锚点不会发送到服务器
	if parsedURL.Fragment != "" {
		return fmt.Errorf("invalid URL: fragment is not allowed")
	}

	// 9. 验证URL路径或查询参数中是否包含图片格式标识
	// 支持多种常见格式：
	// - 直接路径：https://example.com/avatar.jpg
	// - 查询参数：https://gravatar.com/avatar/xxx?s=200&d=identicon
	// - 路径+查询：https://cdn.example.com/user123.jpg?width=200

	// 从路径中提取可能的文件名
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	var fileName string
	if len(pathParts) > 0 {
		fileName = pathParts[len(pathParts)-1]
	}

	// 检查路径中的文件扩展名
	hasValidExtension := false
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}
	// 允许的图片格式（不带点，用于查询参数）- 从allowedExtensions自动生成，避免重复维护
	validImageFormats := make([]string, len(allowedExtensions))
	for i, ext := range allowedExtensions {
		validImageFormats[i] = strings.TrimPrefix(ext, ".")
	}

	if fileName != "" {
		// 使用 path.Ext 提取真正的文件扩展名，避免被恶意文件名绕过（如 avatar.jpg.exe）
		fileExt := strings.ToLower(path.Ext(fileName))
		for _, ext := range allowedExtensions {
			if fileExt == ext {
				hasValidExtension = true
				break
			}
		}
	}

	// 如果路径中没有有效的扩展名，检查查询参数中是否有图片相关的标识
	// 例如：?format=png, ?type=image 等（某些服务使用查询参数指定格式）
	if !hasValidExtension && parsedURL.RawQuery != "" {
		// 解析查询参数，避免误判（如 ?some_other_param=format=png 不应该被识别）
		// url.Values.Get() 只返回指定键的值，不会因为参数值中包含字符串而误判
		query := parsedURL.Query()

		// 检查 format 参数（如 ?format=png）
		if format := strings.ToLower(query.Get("format")); format != "" {
			for _, validFormat := range validImageFormats {
				if format == validFormat {
					hasValidExtension = true
					break
				}
			}
		}

		// 检查 type 参数（如 ?type=image）
		if !hasValidExtension && strings.ToLower(query.Get("type")) == "image" {
			hasValidExtension = true
		}

		// 检查 mime 参数（如 ?mime=image/png）
		if !hasValidExtension && strings.Contains(strings.ToLower(query.Get("mime")), "image") {
			hasValidExtension = true
		}

		// 检查 ext 参数（如 ?ext=png）
		if !hasValidExtension {
			if ext := strings.ToLower(query.Get("ext")); ext != "" {
				for _, validExt := range validImageFormats {
					if ext == validExt {
						hasValidExtension = true
						break
					}
				}
			}
		}
	}

	// 如果既没有路径扩展名，也没有查询参数标识，允许通过但记录警告
	// 因为某些服务可能通过 Content-Type 响应头来标识图片，而不是URL
	if !hasValidExtension {
		zlog.CtxWarnf(ctx, "avatar URL does not contain explicit image format identifier: %s", avatarURL)
		// 不返回错误，允许通过，因为某些合法的图片URL可能没有扩展名
	}

	// 10. 如果路径中有文件名，验证文件名格式
	if fileName != "" {
		// 验证文件名长度（防止过长的文件名）
		const maxFileNameLength = 255
		if len(fileName) > maxFileNameLength {
			return fmt.Errorf("invalid filename: too long, exceeds %d characters", maxFileNameLength)
		}

		// 验证文件名不能包含明显的危险字符
		// 注意：这里不禁止 : 和 ?，因为它们可能在合法的URL中出现
		dangerousChars := []string{"<", ">", "|", "\"", "*", "\\", "\x00"}
		for _, char := range dangerousChars {
			if strings.Contains(fileName, char) {
				return fmt.Errorf("invalid filename: contains dangerous character '%s'", char)
			}
		}
	}

	return nil
}

// isPrivateIP 检查 IP 地址是否为私有/保留地址（用于 SSRF 防护）
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// 使用标准库函数检查常见的私有/保留地址范围（同时支持 IPv4 和 IPv6）
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() || ip.IsMulticast() {
		return true
	}

	// 标准库的 IsUnspecified() 只检查单个地址（0.0.0.0 或 ::），但对于 SSRF 防护，
	// 我们应该拒绝整个 0.0.0.0/8 范围（0.0.0.0 到 0.255.255.255）
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 0
	}

	// 对于 IPv6，IsUnspecified() 已足够检查未指定地址（::）
	return ip.IsUnspecified()
}

// OAuthLogin 第三方登录（GitHub/微信等）
func (u *UserServiceImpl) OAuthLogin(ctx context.Context, provider string, gothUser *goth.User) (*entity.User, string, error) {
	// 参数校验
	if provider == "" || gothUser == nil {
		zlog.CtxErrorf(ctx, "invalid params for oauth login: provider or gothUser is empty")
		return nil, "", ErrInvalidParams
	}

	if gothUser.UserID == "" {
		zlog.CtxErrorf(ctx, "gothUser.UserID is empty")
		return nil, "", ErrInvalidParams
	}

	var user *entity.User
	var err error

	// 根据 provider 类型查询用户
	switch provider {
	case "github":
		// 根据 GithubID 查询
		query := repo.NewUserQueryByGithubID(gothUser.UserID)
		user, err = u.userRepo.GetUser(ctx, query)
	case "wechat":
		// 根据 WechatOpenID 查询
		query := repo.NewUserQueryByWechatOpenID(gothUser.UserID)
		user, err = u.userRepo.GetUser(ctx, query)
	default:
		zlog.CtxErrorf(ctx, "unsupported oauth provider: %s", provider)
		return nil, "", ErrUnsupportedAccountType
	}

	// GetUser 返回错误表示数据库查询错误
	if err != nil {
		zlog.CtxErrorf(ctx, "failed to query user by %s: %v", provider, err)
		return nil, "", ErrInternalError
	}
	// user == nil 表示用户不存在，需要创建新用户

	// 用户不存在，创建新用户
	if user == nil {
		userID, err := util.GenerateStringID()
		if err != nil {
			zlog.CtxErrorf(ctx, "generate user id failed: %v", err)
			return nil, "", ErrInternalError
		}

		// 确定用户名：优先使用 Name，如果为空则使用 NickName，如果还是为空则使用默认值
		userName := gothUser.Name
		if userName == "" {
			userName = gothUser.NickName
		}
		if userName == "" {
			// 如果 Name 和 NickName 都为空，使用 provider 和 UserID 生成默认用户名
			// 安全地截取 UserID（最多取前8个字符，如果长度不足则使用全部）
			userIDPrefix := gothUser.UserID
			if len(userIDPrefix) > 8 {
				userIDPrefix = userIDPrefix[:8]
			}
			userName = fmt.Sprintf("%s_user_%s", provider, userIDPrefix)
			zlog.CtxWarnf(ctx, "gothUser.Name and NickName are both empty, using default username: %s", userName)
		}

		user = &entity.User{
			UserID:   userID,
			UserName: userName,
			Avatar:   gothUser.AvatarURL,
			Status:   entity.UserStatusActive,
		}

		// 根据 provider 设置对应字段
		switch provider {
		case "github":
			user.GithubID = gothUser.UserID
			// GitHub 的 NickName 是 login（用户名），Name 是显示名称
			if gothUser.NickName != "" {
				user.GithubLogin = gothUser.NickName
			} else if gothUser.Name != "" {
				user.GithubLogin = gothUser.Name
			} else {
				// 如果都没有，使用 UserID 的一部分作为 login
				user.GithubLogin = gothUser.UserID
			}
			// GitHub 用户可能有邮箱
			if gothUser.Email != "" {
				user.Email = gothUser.Email
				user.EmailVerified = true
			}
		case "wechat":
			user.WechatOpenID = gothUser.UserID
			// 如果有 unionid，也存储
			if unionid, ok := gothUser.RawData["unionid"].(string); ok && unionid != "" {
				user.WechatUnionID = unionid
			}
		}

		// 创建用户
		if err := u.userRepo.CreateUser(ctx, user); err != nil {
			zlog.CtxErrorf(ctx, "create user failed: %v", err)
			return nil, "", ErrInternalError
		}

		zlog.CtxInfof(ctx, "oauth user created: userID=%s, provider=%s", userID, provider)
	} else {
		// 用户已存在，更新最后登录时间和相关信息
		now := time.Now()

		// 关键更新：最后登录时间和第三方ID（这些是登录流程的核心数据）
		criticalUpdateInfo := &repo.UserUpdateInfo{
			UserID:      user.UserID,
			LastLoginAt: &now,
		}

		// 更新第三方账号ID（如果变化）- 这是关键信息
		switch provider {
		case "github":
			if gothUser.UserID != user.GithubID {
				criticalUpdateInfo.GithubID = &gothUser.UserID
			}
		case "wechat":
			if gothUser.UserID != user.WechatOpenID {
				criticalUpdateInfo.WechatOpenID = &gothUser.UserID
			}
			// unionid 也是关键信息，用于跨应用识别
			if unionid, ok := gothUser.RawData["unionid"].(string); ok && unionid != "" && unionid != user.WechatUnionID {
				criticalUpdateInfo.WechatUnionID = &unionid
			}
		}

		// 先更新关键信息，如果失败则返回错误
		if err := u.userRepo.UpdateUser(ctx, criticalUpdateInfo); err != nil {
			zlog.CtxErrorf(ctx, "update critical user info failed (lastLoginAt/thirdPartyID): %v", err)
			return nil, "", ErrInternalError
		}

		// 非关键更新：头像、用户名、邮箱等（这些失败不影响登录）
		nonCriticalUpdateInfo := &repo.UserUpdateInfo{
			UserID: user.UserID,
		}

		// 更新头像（如果第三方头像更新了）
		if gothUser.AvatarURL != "" && gothUser.AvatarURL != user.Avatar {
			nonCriticalUpdateInfo.Avatar = &gothUser.AvatarURL
		}

		// 更新第三方账号的非关键信息（如果变化）
		switch provider {
		case "github":
			githubLogin := gothUser.NickName
			if githubLogin == "" {
				githubLogin = gothUser.Name
			}
			if githubLogin != "" && githubLogin != user.GithubLogin {
				nonCriticalUpdateInfo.GithubLogin = &githubLogin
			}
			// 如果 GitHub 用户有邮箱且用户当前没有邮箱，更新邮箱
			if gothUser.Email != "" && user.Email == "" {
				nonCriticalUpdateInfo.Email = &gothUser.Email
				emailVerified := true
				nonCriticalUpdateInfo.EmailVerified = &emailVerified
			}
		}

		// 检查是否有非关键信息需要更新
		hasNonCriticalUpdate := nonCriticalUpdateInfo.Avatar != nil ||
			nonCriticalUpdateInfo.GithubLogin != nil ||
			nonCriticalUpdateInfo.Email != nil

		// 更新非关键信息，失败时只记录警告，不影响登录
		if hasNonCriticalUpdate {
			if err := u.userRepo.UpdateUser(ctx, nonCriticalUpdateInfo); err != nil {
				zlog.CtxWarnf(ctx, "update non-critical user info failed (avatar/username/email): %v", err)
				// 不返回错误，继续登录流程
			}
		}

		zlog.CtxInfof(ctx, "oauth user login: userID=%s, provider=%s", user.UserID, provider)
	}

	// 生成 JWT token
	token, err := u.jwtUtil.GenerateToken(user.UserID)
	if err != nil {
		zlog.CtxErrorf(ctx, "generate token failed: %v", err)
		return nil, "", ErrInternalError
	}

	return user, token, nil
}
