package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"forge/biz/cosservice"
	"forge/biz/userservice"

	// "forge/constant"
	"forge/interface/def"
	"forge/interface/handler"
	"forge/pkg/log/zlog"

	// "forge/pkg/loop"
	"forge/pkg/response"
	"forge/util"
)

// 这里就是gin框架的相关接触代码
// 因为解耦的缘故，框架层面的更换不会对内部代码造成任何影响
// router 与hander 应该是一个一对一的关系，有可能会有多对一的关系

// handleHandlerResponse 统一处理 handler 的响应和错误
func handleHandlerResponse(gCtx *gin.Context, rsp interface{}, err error, emptyResp interface{}) {
	r := response.NewResponse(gCtx)
	if err != nil {
		msgCode := mapServiceErrorToMsgCode(err)
		gCtx.JSON(http.StatusOK, response.JsonMsgResult{
			Code:    msgCode.Code,
			Message: msgCode.Msg,
			Data:    emptyResp,
		})
		return
	}
	r.Success(rsp)
}

// abortWithError 辅助函数：封装错误响应逻辑，减少代码重复
func abortWithError(gCtx *gin.Context, ctx context.Context, msgCode response.MsgCode, err error) {
	logMsg := err.Error()
	if err == nil {
		logMsg = msgCode.Msg
	}
	zlog.CtxErrorf(ctx, "error: %s", logMsg)
	gCtx.JSON(http.StatusOK, response.JsonMsgResult{
		Code:    msgCode.Code,
		Message: msgCode.Msg,
		Data:    def.UpdateAvatarResp{Success: false},
	})
}

// mapServiceErrorToMsgCode 根据应用层返回的错误映射到相应的错误码
func mapServiceErrorToMsgCode(err error) response.MsgCode {
	if err == nil {
		return response.SUCCESS
	}

	// 对应 code_der.go
	// 使用 errors.Is 进行哨兵错误匹配，更加健壮  避免通过字符串匹配来判断
	if errors.Is(err, userservice.ErrUserNotFound) {
		return response.USER_ACCOUNT_NOT_EXIST
	}

	if errors.Is(err, userservice.ErrUserAlreadyExists) {
		return response.USER_ACCOUNT_ALREADY_EXIST
	}

	if errors.Is(err, userservice.ErrAccountAlreadyInUse) {
		return response.ACCOUNT_ALREADY_IN_USE
	}

	if errors.Is(err, userservice.ErrPasswordRequired) {
		return response.PASSWORD_REQUIRED
	}

	if errors.Is(err, userservice.ErrInvalidParams) {
		return response.PARAM_NOT_VALID
	}

	if errors.Is(err, userservice.ErrCannotUnbindOnlyContact) {
		return response.ACCOUNT_LAST_CONTACT
	}

	if errors.Is(err, userservice.ErrPasswordMismatch) {
		return response.USER_PASSWORD_DIFFERENT
	}

	if errors.Is(err, userservice.ErrCredentialsIncorrect) {
		return response.USER_CREDENTIALS_ERROR
	}

	if errors.Is(err, userservice.ErrUnsupportedAccountType) {
		return response.PARAM_NOT_VALID
	}

	if errors.Is(err, userservice.ErrInternalError) {
		return response.INTERNAL_ERROR
	}

	if errors.Is(err, userservice.ErrPermissionDenied) {
		return response.INSUFFICENT_PERMISSIONS
	}

	// 验证码错误
	if errors.Is(err, userservice.ErrVerificationCodeIncorrect) {
		return response.CAPTCHA_ERROR
	}

	// 密码强度校验错误
	if errors.Is(err, util.ErrPasswordTooShort) {
		return response.PARAM_NOT_VALID
	}
	if errors.Is(err, util.ErrPasswordTooWeak) {
		return response.PARAM_NOT_VALID
	}
	if errors.Is(err, util.ErrPasswordTooLong) {
		return response.PARAM_NOT_VALID
	}

	// COS相关错误
	if errors.Is(err, cosservice.ErrInvalidParams) {
		return response.PARAM_NOT_VALID
	}
	if errors.Is(err, cosservice.ErrPermissionDenied) {
		return response.INSUFFICENT_PERMISSIONS
	}
	if errors.Is(err, cosservice.ErrInternalError) {
		return response.INTERNAL_FILE_UPLOAD_ERROR
	}

	// 默认返回通用错误
	return response.COMMON_FAIL
}

// Login
//
//	@Description:[POST] /api/biz/v1/user/login
//	@return gin.HandlerFunc
func Login() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.LoginReq{}
		ctx := gCtx.Request.Context()

		// 绑定JSON请求体
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.LoginResp{Success: false},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "login", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().Login(ctx, req)
		// loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
		zlog.CtxAllInOne(ctx, "login", req, rsp, err)

		// 统一处理响应和错误
		handleHandlerResponse(gCtx, rsp, err, def.LoginResp{Success: false})
	}
}

// Register
//
//	@Description:[POST] /api/biz/v1/user/register
//	@return gin.HandlerFunc
func Register() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.RegisterReq{}
		// 统一从 gin 上下文取出 request 的 context，供后续业务调用使用
		ctx := gCtx.Request.Context()
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.RegisterResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().Register(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.RegisterResp{Success: false})
	}
}

// SendCode
//
//	@Description:[POST] /api/biz/v1/user/send_code
//	@return gin.HandlerFunc
func SendCode() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.SendVerificationCodeReq{}
		// 统一从 gin 上下文取出 request 的 context
		ctx := gCtx.Request.Context()
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.SendVerificationCodeResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().SendCode(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.SendVerificationCodeResp{Success: false})
	}
}

// ResetPassword
//
//	@Description:[POST] /api/biz/v1/user/reset_password
//	@return gin.HandlerFunc
func ResetPassword() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.ResetPasswordReq{}
		// 统一从 gin 上下文取出 request 的 context
		ctx := gCtx.Request.Context()
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ResetPasswordResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().ResetPassword(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.ResetPasswordResp{Success: false})
	}
}

// GetHome
//
//	@Description:[GET] /api/biz/v1/user/home
//	@return gin.HandlerFunc
func GetHome() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		rsp, err := handler.GetHandler().GetHome(ctx)
		handleHandlerResponse(gCtx, rsp, err, def.GetHomeResp{})
	}
}

// UpdateAccount
//
//	@Description:[POST] /api/biz/v1/user/account
//	@return gin.HandlerFunc
func UpdateAccount() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.UpdateAccountReq{}
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.UpdateAccountResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().UpdateAccount(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.UpdateAccountResp{Success: false})
	}
}

// UnbindAccount
//
//	@Description:[DELETE] /api/biz/v1/user/contact
//	@return gin.HandlerFunc
func UnbindAccount() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.UnbindAccountReq{}
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.UnbindAccountResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().UnbindAccount(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.UnbindAccountResp{Success: false})
	}
}

// UpdateAvatar
//
//	@Description:[POST] /api/biz/v1/user/avatar
//	@return gin.HandlerFunc
func UpdateAvatar() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		// 设置文件大小限制（8MB）
		gCtx.Request.ParseMultipartForm(8 << 20)

		// 接收文件
		file, err := gCtx.FormFile("avatar") // "avatar" 是前端表单字段名
		if err != nil {
			// 检查是否是文件大小错误
			if strings.Contains(err.Error(), "too large") || strings.Contains(err.Error(), "request body too large") {
				abortWithError(gCtx, ctx, response.PARAM_FILE_SIZE_TOO_BIG, err)
				return
			}
			abortWithError(gCtx, ctx, response.PARAM_NOT_VALID, err)
			return
		}

		// 检查文件大小
		if file.Size > 5*1024*1024 { // 5MB
			abortWithError(gCtx, ctx, response.PARAM_FILE_SIZE_TOO_BIG, fmt.Errorf("file size too large: %d bytes", file.Size))
			return
		}

		// 打开文件
		src, err := file.Open()
		if err != nil {
			abortWithError(gCtx, ctx, response.INTERNAL_FILE_UPLOAD_ERROR, err)
			return
		}
		defer src.Close() // 确保关闭

		// 读取文件内容
		fileData, err := io.ReadAll(src)
		if err != nil {
			abortWithError(gCtx, ctx, response.INTERNAL_FILE_UPLOAD_ERROR, err)
			return
		}

		// 构建请求对象
		req := &def.UpdateAvatarReq{
			FileData: fileData,
			Filename: file.Filename,
		}

		// 调用handler
		rsp, err := handler.GetHandler().UpdateAvatar(ctx, req)
		r := response.NewResponse(gCtx)

		if err != nil {
			msgCode := mapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.UpdateAvatarResp{Success: false},
			})
			return
		}
		r.Success(rsp)
	}
}

// UpdateUserName
//
//	@Description:[POST] /api/biz/v1/user/username
//	@return gin.HandlerFunc
func UpdateUserName() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.UpdateUserNameReq{}
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.UpdateUserNameResp{Success: false},
			})
			return
		}

		rsp, err := handler.GetHandler().UpdateUserName(ctx, req)
		handleHandlerResponse(gCtx, rsp, err, def.UpdateUserNameResp{Success: false})
	}
}

// loadOAuthService 加载 OAuth 第三方登录路由
func loadOAuthService(r *gin.RouterGroup) {
	// 获取 UserService（从 handler 中获取）
	h := handler.GetHandler()
	// 使用类型断言获取 UserService
	hHandler, ok := h.(*handler.Handler)
	if !ok || hHandler == nil {
		// 如果类型断言失败或 handler 为 nil，这是严重的服务初始化问题，应该直接 panic
		zlog.Panicf("CRITICAL: Handler is not initialized or is not of type *handler.Handler")
	}
	if hHandler.UserService == nil {
		// 如果 UserService 为 nil，这是严重的服务初始化问题，应该直接 panic
		zlog.Panicf("CRITICAL: UserService is not initialized in handler")
	}
	userService := hHandler.UserService

	// GitHub OAuth
	// [GET] /api/biz/v1/auth/github
	r.Handle(GET, "github", handler.OAuthBegin("github"))
	// [GET] /api/biz/v1/auth/github/callback
	r.Handle(GET, "github/callback", handler.OAuthCallback(userService, "github"))

	// 微信 OAuth
	// [GET] /api/biz/v1/auth/wechat
	r.Handle(GET, "wechat", handler.OAuthBegin("wechat"))
	// [GET] /api/biz/v1/auth/wechat/callback
	r.Handle(GET, "wechat/callback", handler.OAuthCallback(userService, "wechat"))

	// 获取可用的 OAuth 提供商列表
	// [GET] /api/biz/v1/auth/providers
	r.Handle(GET, "providers", handler.GetOAuthProviders())
}
