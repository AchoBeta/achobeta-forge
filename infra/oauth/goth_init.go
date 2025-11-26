package oauth

import (
	"crypto/sha256"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"

	"forge/infra/configs"
	wechat "forge/infra/oauth/providers"
	"forge/pkg/log/zlog"
)

// InitGoth 初始化 goth OAuth 服务
func InitGoth(oauthConfig configs.OAuthConfig) {
	// 注册 GitHub Provider
	if oauthConfig.GitHubClientID != "" && oauthConfig.GitHubClientSecret != "" {
		goth.UseProviders(
			github.New(
				oauthConfig.GitHubClientID,
				oauthConfig.GitHubClientSecret,
				oauthConfig.GitHubCallbackURL,
			),
		)
		zlog.Infof("GitHub OAuth Provider 初始化成功")
	}

	// 注册微信 Provider
	if oauthConfig.WechatAppID != "" && oauthConfig.WechatAppSecret != "" {
		goth.UseProviders(
			wechat.NewWechat(
				oauthConfig.WechatAppID,
				oauthConfig.WechatAppSecret,
				oauthConfig.WechatCallbackURL,
			),
		)
		zlog.Infof("微信 OAuth Provider 初始化成功")
	}

	// 获取应用环境配置（用于安全检查）
	appConfig := configs.Config().GetAppConfig()

	// 配置 Session Store
	if oauthConfig.SessionSecret == "" {
		// 生产环境必须配置 session_secret，否则 panic（安全漏洞）
		if appConfig.Env != "dev" {
			zlog.Panicf("CRITICAL: OAuth session_secret is not configured in a non-development environment (env=%s). This is a security vulnerability.", appConfig.Env)
		}
		// 开发环境允许使用默认值（但不安全）
		zlog.Warnf("OAuth session_secret is not configured, using a default insecure key. This is for development only (env=%s).", appConfig.Env)
		oauthConfig.SessionSecret = "default-session-secret-change-in-production-min-32-chars"
	}

	// 使用哈希函数从提供的 secret 生成固定长度的强密钥
	// 为认证和加密派生不同的密钥以提高安全性
	// 这样可以确保无论用户提供的密钥长度如何，都能生成固定长度（32字节）的强密钥
	secretBytes := []byte(oauthConfig.SessionSecret)

	// 为认证密钥生成 SHA-256 哈希（32字节）
	authKeyHash := sha256.Sum256(secretBytes)

	// 为加密密钥生成不同的 SHA-256 哈希（通过在原始密钥后追加特定字符串）
	// 这样可以确保认证和加密使用不同的密钥，提高安全性
	encKeyBytes := append(secretBytes, []byte("-encryption")...)
	encKeyHash := sha256.Sum256(encKeyBytes)

	// 如果用户提供的密钥长度不足 32 字节，记录警告
	if len(secretBytes) < 32 {
		zlog.Warnf("OAuth session_secret 长度不足32字节（当前: %d字节），已使用 SHA-256 哈希生成强密钥", len(secretBytes))
	}

	// 使用密钥创建 CookieStore
	// 传入两个密钥：第一个用于认证，第二个用于加密
	// 每个密钥都是 32 字节的 SHA-256 哈希值，安全性更高
	store := sessions.NewCookieStore(authKeyHash[:], encKeyHash[:])
	store.MaxAge(86400 * 30) // 30天
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	// 生产环境必须使用 HTTPS，Secure 设置为 true
	store.Options.Secure = appConfig.Env != "dev"
	store.Options.SameSite = http.SameSiteLaxMode // Lax 模式允许 GET 请求跨站发送 Cookie（适合 OAuth 回调）

	gothic.Store = store
	zlog.Infof("OAuth Session Store 初始化成功")
}

// GetProvider 获取指定名称的 Provider
func GetProvider(provider string) (goth.Provider, error) {
	return goth.GetProvider(provider)
}
