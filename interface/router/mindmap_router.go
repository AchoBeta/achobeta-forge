package router

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"forge/biz/mindmapservice"
	// "forge/constant"
	"forge/interface/def"
	"forge/interface/handler"
	"forge/pkg/log/zlog"

	// "forge/pkg/loop"
	"forge/pkg/response"
)

// mapMindMapServiceErrorToMsgCode 根据服务层返回的错误映射到相应的错误码
func mapMindMapServiceErrorToMsgCode(err error) response.MsgCode {
	if err == nil {
		return response.SUCCESS
	}

	// 使用 errors.Is 进行哨兵错误匹配
	if errors.Is(err, mindmapservice.ErrMindMapNotFound) {
		return response.MINDMAP_NOT_FOUND
	}

	if errors.Is(err, mindmapservice.ErrMindMapAlreadyExists) {
		return response.MINDMAP_ALREADY_EXISTS
	}

	if errors.Is(err, mindmapservice.ErrInvalidParams) {
		return response.PARAM_NOT_VALID
	}

	if errors.Is(err, mindmapservice.ErrPermissionDenied) {
		return response.MINDMAP_PERMISSION_DENIED
	}

	if errors.Is(err, mindmapservice.ErrInternalError) {
		return response.INTERNAL_ERROR
	}

	// 默认返回通用错误
	return response.COMMON_FAIL
}

// CreateMindMap
//
//	@Description:[POST] /api/biz/v1/mindmap
//	@return gin.HandlerFunc
func CreateMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.CreateMindMapReq{}
		ctx := gCtx.Request.Context()

		// 绑定JSON请求体
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.CreateMindMapResp{},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "create_mindmap", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().CreateMindMap(ctx, req)
		// loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
		zlog.CtxAllInOne(ctx, "create_mindmap", req, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.CreateMindMapResp{},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}

// GetMindMap
//
//	@Description:[GET] /api/biz/v1/mindmap/:id
//	@return gin.HandlerFunc
func GetMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		mapID := gCtx.Param("id")
		ctx := gCtx.Request.Context()

		// 参数校验
		if mapID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_VALID.Code,
				Message: response.PARAM_NOT_VALID.Msg,
				Data:    def.GetMindMapResp{},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "get_mindmap", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().GetMindMap(ctx, mapID)
		// loop.SetSpanAllInOne(ctx, sp, mapID, rsp, err)
		zlog.CtxAllInOne(ctx, "get_mindmap", mapID, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GetMindMapResp{},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}

// ListMindMaps
//
//	@Description:[GET] /api/biz/v1/mindmap/list
//	@return gin.HandlerFunc
func ListMindMaps() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.ListMindMapsReq{}
		ctx := gCtx.Request.Context()

		// 绑定查询参数
		if err := gCtx.ShouldBindQuery(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ListMindMapsResp{},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "list_mindmaps", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().ListMindMaps(ctx, req)
		// loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
		zlog.CtxAllInOne(ctx, "list_mindmaps", req, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ListMindMapsResp{},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}

// UpdateMindMap
//
//	@Description:[PUT] /api/biz/v1/mindmap/:id
//	@return gin.HandlerFunc
func UpdateMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		mapID := gCtx.Param("id")
		req := &def.UpdateMindMapReq{}
		ctx := gCtx.Request.Context()

		// 参数校验
		if mapID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_VALID.Code,
				Message: response.PARAM_NOT_VALID.Msg,
				Data:    def.UpdateMindMapResp{Success: false},
			})
			return
		}

		// 绑定JSON请求体
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.UpdateMindMapResp{Success: false},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "update_mindmap", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().UpdateMindMap(ctx, mapID, req)
		// loop.SetSpanAllInOne(ctx, sp, map[string]interface{}{"mapID": mapID, "req": req}, rsp, err)
		zlog.CtxAllInOne(ctx, "update_mindmap", map[string]interface{}{"mapID": mapID, "req": req}, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.UpdateMindMapResp{Success: false},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}

// DeleteMindMap
//
//	@Description:[DELETE] /api/biz/v1/mindmap/:id
//	@return gin.HandlerFunc
func DeleteMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		mapID := gCtx.Param("id")
		ctx := gCtx.Request.Context()

		// 参数校验
		if mapID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_VALID.Code,
				Message: response.PARAM_NOT_VALID.Msg,
				Data:    def.DeleteMindMapResp{Success: false},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "delete_mindmap", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().DeleteMindMap(ctx, mapID)
		// loop.SetSpanAllInOne(ctx, sp, mapID, rsp, err)
		zlog.CtxAllInOne(ctx, "delete_mindmap", mapID, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.DeleteMindMapResp{Success: false},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}

// BatchDeleteMindMap
//
//	@Description:[POST] /api/biz/v1/mindmap/batch_delete
//	@return gin.HandlerFunc
func BatchDeleteMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		req := &def.BatchDeleteMindMapReq{}
		ctx := gCtx.Request.Context()

		// 绑定JSON请求体
		if err := gCtx.ShouldBindJSON(req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.BatchDeleteMindMapResp{Success: false},
			})
			return
		}

		// TODO: cozeloop配置好后启用
		// ctx, sp := loop.GetNewSpan(ctx, "batch_delete_mindmap", constant.LoopSpanType_Root)
		rsp, err := handler.GetHandler().BatchDeleteMindMap(ctx, req)
		// loop.SetSpanAllInOne(ctx, sp, req, rsp, err)
		zlog.CtxAllInOne(ctx, "batch_delete_mindmap", req, rsp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapMindMapServiceErrorToMsgCode(err)
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.BatchDeleteMindMapResp{Success: false},
			})
			return
		} else {
			r.Success(rsp)
		}
	}
}
