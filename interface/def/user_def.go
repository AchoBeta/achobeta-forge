package def

// 这个是DTO层，会暴露给前端 主要是接口定义

type User struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Avatar   string `json:"avatar,omitempty"`

	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`

	Dogs []*Dog `json:"dogs"`
}

type Dog struct {
	DogID   string `json:"dog_id"`
	DogName string `json:"dog_name"`
}

// ---------登录相关----------
type LoginReq struct {
	Account     string `json:"account"`      // 账号（手机号或邮箱）
	AccountType string `json:"account_type"` // 账号类型：phone（手机号）或 email（邮箱）
	Password    string `json:"password"`     // 密码
}

type LoginResp struct {
	Token    string `json:"token,omitempty"`     // JWT token
	UserID   string `json:"user_id,omitempty"`   // 用户ID
	UserName string `json:"user_name,omitempty"` // 用户名
	Avatar   string `json:"avatar,omitempty"`    // 头像
	Phone    string `json:"phone,omitempty"`     // 手机号
	Email    string `json:"email,omitempty"`     // 邮箱
	Success  bool   `json:"success"`             // 登录是否成功
}

//---------注册相关------------
// 注册：用户名 + 手机号/邮箱 + 验证码 + 设置密码
type RegisterReq struct {
	UserName    string `json:"user_name"`
	Account     string `json:"account"`
	AccountType string `json:"account_type"` // 手机号或邮箱
	Code        string `json:"code"`
	Password    string `json:"password"`
}

type RegisterResp struct {
	Success bool `json:"success"` // 注册是否成功
}

//---------重置密码-----------
type ResetPasswordReq struct {
	Account         string `json:"account"`
	AccountType     string `json:"account_type"` // 手机号或邮箱
	Code            string `json:"code"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type ResetPasswordResp struct {
	Success bool `json:"success"`
}

//---------更新头像-----------
type UpdateAvatarReq struct {
	FileData []byte `json:"-"`        // 文件内容
	Filename string `json:"filename"` // 文件名
}

type UpdateAvatarResp struct {
	AvatarURL string `json:"avatar_url"` // 返回上传后的URL
	Success   bool   `json:"success"`    // 更新是否成功
}

//---------发送验证码-----------
type SendVerificationCodeReq struct {
	Account     string `json:"account"`      // 账号（手机号或邮箱）  目前只支持邮箱 邮件收取验证码
	AccountType string `json:"account_type"` // 账号类型：phone（手机号）或 email（邮箱）
	Purpose     string `json:"purpose"`      // 使用场景：register（注册）、reset_password（重置密码）、change_account（换绑联系方式，手机号/邮箱）  // 控制验证
}

type SendVerificationCodeResp struct {
	Success bool `json:"success"` // 发送是否成功
}

//---------个人主页-----------
type GetHomeResp struct {
	UserName    string `json:"user_name"`        // 用户名
	Avatar      string `json:"avatar,omitempty"` // 头像URL
	Phone       string `json:"phone,omitempty"`  // 手机号
	Email       string `json:"email,omitempty"`  // 邮箱
	HasPassword bool   `json:"has_password"`     // 是否有密码
	// 第三方登录信息
	HasGithub   bool   `json:"has_github"`             // 是否绑定 GitHub
	GithubLogin string `json:"github_login,omitempty"` // GitHub 用户名
	HasWechat   bool   `json:"has_wechat"`             // 是否绑定微信
}

//---------更新联系方式（绑定/换绑）-----------
type UpdateAccountReq struct {
	Account     string `json:"account"`      // 新手机号/邮箱
	AccountType string `json:"account_type"` // 账号类型：phone（手机号）或 email（邮箱）
	Code        string `json:"code"`         // 验证码
	Password    string `json:"password"`     // 密码（如果用户没有密码则必填，如果有密码则可选）
}

type UpdateAccountResp struct {
	Success bool   `json:"success"` // 更新是否成功
	Account string `json:"account"` // 更新后的联系方式
}

//---------解绑联系方式-----------
type UnbindAccountReq struct {
	Account     string `json:"account,omitempty"` // 需要解绑的手机号/邮箱（解绑第三方账号时可为空）
	AccountType string `json:"account_type"`      // 账号类型：phone（手机号）、email（邮箱）、github（GitHub）、wechat（微信）
}

type UnbindAccountResp struct {
	Success bool `json:"success"` // 解绑是否成功
}

//---------更新用户名-----------
type UpdateUserNameReq struct {
	UserName string `json:"user_name"` // 新用户名
}

type UpdateUserNameResp struct {
	Success  bool   `json:"success"`
	UserName string `json:"user_name"` // 更新后的用户名
}

//---------第三方登录相关-----------
type OAuthCallbackResp struct {
	Success  bool   `json:"success"`             // 登录是否成功
	Token    string `json:"token,omitempty"`     // JWT token
	UserID   string `json:"user_id,omitempty"`   // 用户ID
	UserName string `json:"user_name,omitempty"` // 用户名
	Avatar   string `json:"avatar,omitempty"`    // 头像
	Phone    string `json:"phone,omitempty"`     // 手机号
	Email    string `json:"email,omitempty"`     // 邮箱
}

type OAuthProvider struct {
	Name string `json:"name"` // Provider 名称（github、wechat 等）
}

type GetOAuthProvidersResp struct {
	Providers []OAuthProvider `json:"providers"` // 可用的 OAuth 提供商列表
}

//---------第三方--------- 暂时先不做
