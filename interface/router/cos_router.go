package router

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"forge/biz/cosservice"
	"forge/interface/def"
	"forge/interface/handler"
	"forge/pkg/log/zlog"
	"forge/pkg/response"
)

// mapCOSServiceErrorToMsgCode 根据服务层返回的错误映射到相应的错误码
func mapCOSServiceErrorToMsgCode(err error) response.MsgCode {
	if err == nil {
		return response.SUCCESS
	}

	// 使用 errors.Is 进行哨兵错误匹配
	if errors.Is(err, cosservice.ErrInvalidParams) {
		return response.PARAM_NOT_VALID
	}

	if errors.Is(err, cosservice.ErrPermissionDenied) {
		return response.COS_PERMISSION_DENIED
	}

	if errors.Is(err, cosservice.ErrInvalidResourcePath) {
		return response.COS_INVALID_RESOURCE_PATH
	}

	if errors.Is(err, cosservice.ErrInvalidDuration) {
		return response.COS_INVALID_DURATION
	}

	if errors.Is(err, cosservice.ErrInternalError) {
		return response.COS_GET_CREDENTIALS_FAILED
	}

	// 默认返回通用错误
	return response.COMMON_FAIL
}

// GetOSSCredentials
//
//	@Description:[POST] /api/biz/v1/cos/sts/credentials
//	@return gin.HandlerFunc
func GetOSSCredentials() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.GetOSSCredentialsReq{}
		ctx := gCtx.Request.Context()

		// 绑定JSON请求体
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.GetOSSCredentialsResp{},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "get_oss_credentials", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().GetOSSCredentials(ctx, req)
		// loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
		zlog.CtxAllInOne(ctx, "get_oss_credentials", req, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapCOSServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GetOSSCredentialsResp{},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}
