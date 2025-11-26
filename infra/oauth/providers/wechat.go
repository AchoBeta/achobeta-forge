package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/markbates/goth"
	"golang.org/x/oauth2"
)

const (
	wechatAuthURL      = "https://open.weixin.qq.com/connect/qrconnect"
	wechatTokenURL     = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wechatUserInfoURL  = "https://api.weixin.qq.com/sns/userinfo"
	wechatRefreshURL   = "https://api.weixin.qq.com/sns/oauth2/refresh_token"
	wechatProviderName = "wechat"
)

// WechatProvider 微信 OAuth Provider
type WechatProvider struct {
	ClientKey    string
	Secret       string
	CallbackURL  string
	HTTPClient   *http.Client
	providerName string
}

// NewWechat 创建微信 Provider
func NewWechat(clientKey, secret, callbackURL string) *WechatProvider {
	return &WechatProvider{
		ClientKey:    clientKey,
		Secret:       secret,
		CallbackURL:  callbackURL,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
		providerName: wechatProviderName,
	}
}

// Name 返回 Provider 名称
func (p *WechatProvider) Name() string {
	return p.providerName
}

// SetName 设置 Provider 名称
func (p *WechatProvider) SetName(name string) {
	p.providerName = name
}

// BeginAuth 开始认证流程
func (p *WechatProvider) BeginAuth(state string) (goth.Session, error) {
	params := map[string]string{
		"appid":         p.ClientKey,
		"redirect_uri":  p.CallbackURL,
		"response_type": "code",
		"scope":         "snsapi_login",
		"state":         state,
	}

	authURL := buildURL(wechatAuthURL, params) + "#wechat_redirect"

	return &WechatSession{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// FetchUser 获取用户信息
func (p *WechatProvider) FetchUser(session goth.Session) (goth.User, error) {
	wechatSession := session.(*WechatSession)

	// gothic 会从 URL 参数中提取 code 并设置到 session 中
	// 如果 session 中没有 code，尝试从 AuthURL 中解析（备用方案）
	if wechatSession.Code == "" {
		return goth.User{}, fmt.Errorf("微信授权码为空")
	}

	// 1. 用 code 换取 access_token
	token, err := p.getAccessToken(wechatSession.Code)
	if err != nil {
		return goth.User{}, err
	}

	// 2. 用 access_token 获取用户信息
	userInfo, err := p.getUserInfo(token.AccessToken, token.OpenID)
	if err != nil {
		return goth.User{}, err
	}

	// 3. 转换为 goth.User
	return p.toGothUser(userInfo, token), nil
}

// RefreshTokenAvailable 是否支持刷新 token
func (p *WechatProvider) RefreshTokenAvailable() bool {
	return true
}

// RefreshToken 刷新 token
func (p *WechatProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	params := map[string]string{
		"appid":         p.ClientKey,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	var tokenResp WechatTokenResponse
	if err := p.httpGet(wechatRefreshURL, params, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("微信API错误: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	// 返回 oauth2.Token
	return &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    "Bearer",
	}, nil
}

// Debug 设置调试模式
func (p *WechatProvider) Debug(debug bool) {
	// 微信 Provider 暂不支持调试模式
}

// UnmarshalSession 反序列化 session
func (p *WechatProvider) UnmarshalSession(data string) (goth.Session, error) {
	sess := &WechatSession{}
	err := json.Unmarshal([]byte(data), sess)
	return sess, err
}

// getAccessToken 用 code 换取 access_token
func (p *WechatProvider) getAccessToken(code string) (*WechatTokenResponse, error) {
	params := map[string]string{
		"appid":      p.ClientKey,
		"secret":     p.Secret,
		"code":       code,
		"grant_type": "authorization_code",
	}

	var tokenResp WechatTokenResponse
	if err := p.httpGet(wechatTokenURL, params, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("微信API错误: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	return &tokenResp, nil
}

// getUserInfo 获取用户信息
func (p *WechatProvider) getUserInfo(accessToken, openID string) (*WechatUserInfo, error) {
	params := map[string]string{
		"access_token": accessToken,
		"openid":       openID,
	}

	var userInfo WechatUserInfo
	if err := p.httpGet(wechatUserInfoURL, params, &userInfo); err != nil {
		return nil, err
	}

	if userInfo.ErrCode != 0 {
		return nil, fmt.Errorf("微信API错误: %d - %s", userInfo.ErrCode, userInfo.ErrMsg)
	}

	return &userInfo, nil
}

// httpGet 通用 HTTP GET 请求方法（减少重复逻辑）
func (p *WechatProvider) httpGet(baseURL string, params map[string]string, result interface{}) error {
	reqURL := buildURL(baseURL, params)

	resp, err := p.HTTPClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP状态码错误: %d, 响应: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	return nil
}

// buildURL 构建带参数的 URL（一次性设置所有参数）
func buildURL(baseURL string, params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return baseURL + "?" + values.Encode()
}

// toGothUser 转换为 goth.User
func (p *WechatProvider) toGothUser(userInfo *WechatUserInfo, token *WechatTokenResponse) goth.User {
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	rawData := map[string]interface{}{
		"openid":     userInfo.OpenID,
		"nickname":   userInfo.Nickname,
		"headimgurl": userInfo.HeadImgURL,
	}
	if userInfo.UnionID != "" {
		rawData["unionid"] = userInfo.UnionID
	}

	return goth.User{
		RawData:      rawData,
		Provider:     p.providerName,
		UserID:       userInfo.OpenID,
		Name:         userInfo.Nickname,
		NickName:     userInfo.Nickname,
		AvatarURL:    userInfo.HeadImgURL,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    expiresAt,
	}
}

// WechatSession 微信 Session
type WechatSession struct {
	AuthURL      string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	State        string
	Code         string
}

// GetAuthURL 获取授权 URL
func (s *WechatSession) GetAuthURL() (string, error) {
	return s.AuthURL, nil
}

// Marshal 序列化
func (s *WechatSession) Marshal() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Unmarshal 反序列化
func (s *WechatSession) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), s)
}

// Authorize 完成授权（gothic 会调用此方法）
func (s *WechatSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	// gothic 会从回调 URL 中提取 code 和 state
	// goth.Params 是 url.Values 类型，可以直接获取参数
	if code := params.Get("code"); code != "" {
		s.Code = code
	}
	// 返回授权 URL（gothic 需要）
	return s.GetAuthURL()
}

// WechatTokenResponse 微信 Token 响应
type WechatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

// WechatUserInfo 微信用户信息
type WechatUserInfo struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid"`
	ErrCode    int      `json:"errcode"`
	ErrMsg     string   `json:"errmsg"`
}
