package router

import (
	"errors"
	"fmt"
	"forge/biz/aichatservice"
	"forge/interface/def"
	"forge/interface/handler"
	"forge/interface/outputPort"
	"forge/pkg/log/zlog"
	"forge/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func aiChatServiceErrorToMsgCode(err error) response.MsgCode {
	if err == nil {
		return response.SUCCESS
	}

	if errors.Is(err, aichatservice.CONVERSATION_ID_NOT_NULL) {
		return response.CONVERSATION_ID_NOT_NULL
	}
	if errors.Is(err, aichatservice.USER_ID_NOT_NULL) {
		return response.USER_ID_NOT_NULL
	}
	if errors.Is(err, aichatservice.MAP_ID_NOT_NULL) {
		return response.MAP_ID_NOT_NULL
	}
	if errors.Is(err, aichatservice.CONVERSATION_TITLE_NOT_NULL) {
		return response.CONVERSATION_TITLE_NOT_NULL
	}
	if errors.Is(err, aichatservice.CONVERSATION_NOT_EXIST) {
		return response.CONVERSATION_NOT_EXIST
	}
	if errors.Is(err, aichatservice.AI_CHAT_PERMISSION_DENIED) {
		return response.AI_CHAT_PERMISSION_DENIED
	}
	if errors.Is(err, aichatservice.MIND_MAP_NOT_EXIST) {
		return response.MIND_MAP_NOT_EXIST
	}
	if errors.Is(err, aichatservice.AI_CHAT_MESSAGE_MAX) {
		return response.AI_CHAT_MESSAGE_MAX
	}

	return response.COMMON_FAIL
}

// SendMessage 基础ai对话
func SendMessage() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ProcessUserMessageRequest
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.ProcessUserMessageResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().SendMessage(ctx, &req)

		zlog.CtxAllInOne(ctx, "send_message", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ProcessUserMessageResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// 流式对话输出
func SendMessageStream() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ProcessUserMessageRequest
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.ProcessUserMessageResponse{Success: false},
			})
			return
		}

		// 设置SSE响应头
		gCtx.Header("Content-Type", "text/event-stream; charset=utf-8")
		gCtx.Header("Cache-Control", "no-cache, no-store, must-revalidate") // 禁用所有缓存
		gCtx.Header("Connection", "keep-alive")
		gCtx.Header("X-Accel-Buffering", "no")           // Nginx禁用缓冲
		gCtx.Header("X-Content-Type-Options", "nosniff") // 避免浏览器解析干扰
		gCtx.Header("Transfer-Encoding", "chunked")      // 显式启用分块传输（部分环境需要）

		writer := &outputPort.GinSSEWriter{Ctx: gCtx}

		resp, err := handler.GetHandler().SendMessageStream(ctx, &req, writer)

		zlog.CtxAllInOne(ctx, "send_message", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ProcessUserMessageResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// SaveNewConversation 保存新的会话
func SaveNewConversation() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.SaveNewConversationRequest
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.SaveNewConversationResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().SaveNewConversation(ctx, &req)

		zlog.CtxAllInOne(ctx, "save_new_conversation", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.SaveNewConversationResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// GetConversationList 获取某导图的所有会话
func GetConversationList() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.GetConversationListRequest
		ctx := gCtx.Request.Context()

		req.MapID = gCtx.Query("map_id")

		if req.MapID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.GetConversationListResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().GetConversationList(ctx, &req)

		zlog.CtxAllInOne(ctx, "get_conversation_list", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GetConversationListResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// DelConversation 删除某个会话
func DelConversation() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.DelConversationRequest
		ctx := gCtx.Request.Context()
		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.DelConversationResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().DelConversation(ctx, &req)
		zlog.CtxAllInOne(ctx, "del_conversation", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.DelConversationResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// GetConversation 获取某个会话的详细信息
func GetConversation() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.GetConversationRequest
		ctx := gCtx.Request.Context()

		req.ConversationID = gCtx.Query("conversation_id")

		if req.ConversationID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.GetConversationResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().GetConversation(ctx, &req)
		zlog.CtxAllInOne(ctx, "get_conversation", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GetConversationResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

func UpdateConversationTitle() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.UpdateConversationTitleRequest
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.UpdateConversationTitleResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().UpdateConversationTitle(ctx, &req)
		zlog.CtxAllInOne(ctx, "update_conversation_title", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.UpdateConversationTitleResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

func GenerateMindMap() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.GenerateMindMapRequest
		ctx := gCtx.Request.Context()

		contentType := gCtx.ContentType()

		if contentType == "application/json" {
			if err := gCtx.ShouldBindJSON(&req); err != nil {
				gCtx.JSON(http.StatusOK, response.JsonMsgResult{
					Code:    response.PARAM_NOT_COMPLETE.Code,
					Message: response.PARAM_NOT_COMPLETE.Msg,
					Data:    def.GenerateMindMapResponse{Success: false},
				})
				return
			}
		} else if contentType == "multipart/form-data" {
			file, err := gCtx.FormFile("file")
			if err != nil {
				gCtx.JSON(http.StatusOK, response.JsonMsgResult{
					Code:    response.INTERNAL_FILE_UPLOAD_ERROR.Code,
					Message: response.INTERNAL_FILE_UPLOAD_ERROR.Msg + err.Error(),
					Data:    def.GenerateMindMapResponse{Success: false},
				})
				return
			}
			req.File = file
		} else {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_CONTENT_TYPE.Code,
				Message: response.INVALID_CONTENT_TYPE.Msg,
				Data:    def.GenerateMindMapResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().GenerateMindMap(ctx, &req)
		zlog.CtxAllInOne(ctx, "generate_mind_map", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GenerateMindMapResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}

	}
}

// TabComplete Tab补全路由处理
func TabComplete() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.TabCompletionRequest
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.TabCompletionResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().TabComplete(ctx, &req)

		zlog.CtxAllInOne(ctx, "tab_complete", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.TabCompletionResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// ExportQualityData 导出质量数据路由处理
func ExportQualityData() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ExportQualityDataRequest
		ctx := gCtx.Request.Context()

		// 从查询参数绑定
		if err := gCtx.ShouldBindQuery(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.ExportQualityDataResponse{Success: false},
			})
			return
		}

		resp, err := handler.GetHandler().ExportQualityData(ctx, &req)

		zlog.CtxAllInOne(ctx, "export_quality_data", map[string]interface{}{"req": req}, resp, err)

		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ExportQualityDataResponse{Success: false},
			})
			return
		}

		// 对于导出功能，设置下载响应头
		gCtx.Header("Content-Type", "application/x-ndjson")
		gCtx.Header("Content-Disposition", "attachment; filename=\"tab_completion_training_data.jsonl\"")
		gCtx.Header("Content-Length", fmt.Sprintf("%d", len(resp.Data)))

		// 直接返回JSONL内容
		gCtx.String(http.StatusOK, resp.Data)
	}
}

// TriggerQualityAssessment 手动触发质量评估路由处理
func TriggerQualityAssessment() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.TriggerQualityAssessmentRequest
		ctx := gCtx.Request.Context()

		// 从查询参数或JSON绑定
		if gCtx.ContentType() == "application/json" {
			if err := gCtx.ShouldBindJSON(&req); err != nil {
				gCtx.JSON(http.StatusOK, response.JsonMsgResult{
					Code:    response.PARAM_NOT_COMPLETE.Code,
					Message: response.PARAM_NOT_COMPLETE.Msg,
					Data:    def.TriggerQualityAssessmentResponse{Success: false},
				})
				return
			}
		} else {
			if err := gCtx.ShouldBindQuery(&req); err != nil {
				gCtx.JSON(http.StatusOK, response.JsonMsgResult{
					Code:    response.PARAM_NOT_COMPLETE.Code,
					Message: response.PARAM_NOT_COMPLETE.Msg,
					Data:    def.TriggerQualityAssessmentResponse{Success: false},
				})
				return
			}
		}

		resp, err := handler.GetHandler().TriggerQualityAssessment(ctx, &req)

		zlog.CtxAllInOne(ctx, "trigger_quality_assessment", map[string]interface{}{"req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := aiChatServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.TriggerQualityAssessmentResponse{Success: false},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}
