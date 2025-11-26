package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"

	"forge/biz/types"
	"forge/interface/def"
	"forge/pkg/log/zlog"
	"forge/pkg/response"
)

// OAuthBegin 开始 OAuth 登录流程
// 重定向到第三方登录页面
func OAuthBegin(provider string) gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		// gothic 从 URL 参数中获取 provider
		// 确保 URL 参数中包含 provider
		q := gCtx.Request.URL.Query()
		q.Set("provider", provider)
		gCtx.Request.URL.RawQuery = q.Encode()

		// 直接使用 gothic.BeginAuthHandler，它会处理 session 保存和重定向
		// 注意：BeginAuthHandler 内部会调用 http.Redirect，所以不能在这之后再写响应
		gothic.BeginAuthHandler(gCtx.Writer, gCtx.Request)

		zlog.CtxInfof(ctx, "oauth begin: provider=%s", provider)
	}
}

// OAuthCallback OAuth 回调处理
// 处理第三方登录回调，获取用户信息并登录
func OAuthCallback(userService types.IUserService, provider string) gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		// 记录回调请求信息（用于调试）
		zlog.CtxInfof(ctx, "oauth callback received: provider=%s, url=%s, cookies count=%d",
			provider, gCtx.Request.URL.String(), len(gCtx.Request.Cookies()))

		// gothic 从 URL 参数中获取 provider
		// 确保 URL 参数中包含 provider（GitHub 回调时可能没有）
		q := gCtx.Request.URL.Query()
		if q.Get("provider") == "" {
			q.Set("provider", provider)
			gCtx.Request.URL.RawQuery = q.Encode()
		}

		// 使用 gothic 完成 OAuth 流程，获取用户信息
		gothUser, err := gothic.CompleteUserAuth(gCtx.Writer, gCtx.Request)
		if err != nil {
			zlog.CtxErrorf(ctx, "oauth callback failed: provider=%s, error=%v, url=%s, cookies=%v",
				provider, err, gCtx.Request.URL.String(), gCtx.Request.Cookies())
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INTERNAL_ERROR.Code,
				Message: fmt.Sprintf("第三方登录失败: %v", err),
				Data:    def.OAuthCallbackResp{Success: false},
			})
			return
		}

		// 调用 Service 层进行登录或注册
		user, token, err := userService.OAuthLogin(ctx, provider, &gothUser)
		if err != nil {
			zlog.CtxErrorf(ctx, "oauth login failed: provider=%s, error=%v", provider, err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INTERNAL_ERROR.Code,
				Message: fmt.Sprintf("登录失败: %v", err),
				Data:    def.OAuthCallbackResp{Success: false},
			})
			return
		}

		zlog.CtxInfof(ctx, "oauth login success: provider=%s, userID=%s", provider, user.UserID)

		// 返回成功响应，包含 token 和用户信息
		// 前端可以通过 token 进行后续请求
		gCtx.JSON(http.StatusOK, response.JsonMsgResult{
			Code:    response.SUCCESS.Code,
			Message: response.SUCCESS.Msg,
			Data: def.OAuthCallbackResp{
				Success:  true,
				Token:    token,
				UserID:   user.UserID,
				UserName: user.UserName,
				Avatar:   user.Avatar,
				Phone:    user.Phone,
				Email:    user.Email,
			},
		})
	}
}

// GetOAuthProviders 获取可用的 OAuth 提供商列表
func GetOAuthProviders() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		providers := goth.GetProviders()
		providerList := make([]def.OAuthProvider, 0, len(providers))
		for name := range providers {
			providerList = append(providerList, def.OAuthProvider{
				Name: name,
			})
		}

		zlog.CtxInfof(ctx, "get oauth providers: count=%d", len(providerList))

		gCtx.JSON(http.StatusOK, response.JsonMsgResult{
			Code:    response.SUCCESS.Code,
			Message: response.SUCCESS.Msg,
			Data: def.GetOAuthProvidersResp{
				Providers: providerList,
			},
		})
	}
}
