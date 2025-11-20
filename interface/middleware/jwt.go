package middleware

import (
	"errors"
	"forge/biz/entity"
	"forge/biz/types"
	"forge/biz/userservice"
	"forge/pkg/log/zlog"
	"forge/pkg/response"
	"forge/util"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth JWT鉴权中间件
// 从请求头获取token，验证token，提取用户信息并注入到context中
func JWTAuth(jwtUtil *util.JWTUtil, userService types.IUserService) gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		ctx := gCtx.Request.Context()

		// 从请求头获取token
		authHeader := gCtx.GetHeader("Authorization")
		if authHeader == "" {
			zlog.CtxWarnf(ctx, "missing authorization header")
			r := response.NewResponse(gCtx)
			r.Error(response.USER_NOT_LOGIN)
			gCtx.Abort()
			return
		}

		// 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			zlog.CtxWarnf(ctx, "invalid authorization header format")
			r := response.NewResponse(gCtx)
			r.Error(response.USER_NOT_LOGIN)
			gCtx.Abort()
			return
		}

		tokenString := parts[1]

		// 验证token
		claims, err := jwtUtil.ValidateToken(tokenString)
		if err != nil {
			zlog.CtxWarnf(ctx, "invalid token: %v", err)
			var msgCode response.MsgCode
			if errors.Is(err, util.ErrTokenExpired) {
				msgCode = response.TOKEN_IS_EXPIRED
			} else {
				msgCode = response.USER_NOT_LOGIN
			}
			
			
			r := response.NewResponse(gCtx)
			r.Error(msgCode)
			gCtx.Abort()
			return
		}

		// 从token中提取userID
		userID := claims.UserID
		if userID == "" {
			zlog.CtxWarnf(ctx, "empty userID in token")
			r := response.NewResponse(gCtx)
			r.Error(response.USER_NOT_LOGIN)
			gCtx.Abort()
			return
		}

		// 通过service层获取用户信息（包含状态检查等业务逻辑）
		user, err := userService.GetUserByID(ctx, userID)
		if err != nil {
			var msgCode response.MsgCode
			if errors.Is(err, userservice.ErrUserNotFound) {
				msgCode = response.USER_ACCOUNT_NOT_EXIST
			} else if errors.Is(err, userservice.ErrPermissionDenied) {
				msgCode = response.INSUFFICENT_PERMISSIONS
			} else {
				msgCode = response.INTERNAL_ERROR
			}

			zlog.CtxWarnf(ctx, "failed to get user by ID: %v", err)
			r := response.NewResponse(gCtx)
			r.Error(msgCode)
			gCtx.Abort()
			return
		}

		// 将用户信息注入到context中
		ctx = entity.WithUser(ctx, user)
		// 更新gin context中的request context
		gCtx.Request = gCtx.Request.WithContext(ctx)

		gCtx.Next()
	}
}
