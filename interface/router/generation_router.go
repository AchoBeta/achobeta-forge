package router

import (
	"fmt"
	"net/http"
	"time"

	"forge/interface/def"
	"forge/interface/handler"
	"forge/pkg/log/zlog"
	"forge/pkg/response"

	"github.com/gin-gonic/gin"
)

// mapGenerationServiceErrorToMsgCode 根据服务层返回的错误映射到相应的错误码
func mapGenerationServiceErrorToMsgCode(err error) response.MsgCode {
	if err == nil {
		return response.SUCCESS
	}

	// generation service 没有定义标准错误类型，使用通用错误码
	// 可以根据错误信息进行更精确的匹配（如果需要）
	return response.COMMON_FAIL
}

// GenerateMindMapPro 批量生成导图路由处理
func GenerateMindMapPro() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.GenerateMindMapProReq
		ctx := gCtx.Request.Context()

		// 处理文件上传
		if file, err := gCtx.FormFile("file"); err == nil {
			req.File = file
		}

		// 绑定其他参数
		if err := gCtx.ShouldBind(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.GenerateMindMapProResp{},
			})
			return
		}

		// 调用Handler
		resp, err := handler.GetHandler().GenerateMindMapPro(ctx, &req)
		zlog.CtxAllInOne(ctx, "generate_mindmap_pro", req, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GenerateMindMapProResp{},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// GetGenerationBatch 获取批次详情路由处理
func GetGenerationBatch() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		batchID := gCtx.Query("batch_id")
		ctx := gCtx.Request.Context()

		if batchID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.GetGenerationBatchResp{},
			})
			return
		}

		resp, err := handler.GetHandler().GetGenerationBatch(ctx, batchID)
		zlog.CtxAllInOne(ctx, "get_generation_batch", batchID, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.GetGenerationBatchResp{},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// LabelGenerationResult 标记结果路由处理
func LabelGenerationResult() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		resultID := gCtx.Param("result_id")
		ctx := gCtx.Request.Context()

		if resultID == "" {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.PARAM_NOT_COMPLETE.Code,
				Message: response.PARAM_NOT_COMPLETE.Msg,
				Data:    def.LabelGenerationResultResp{},
			})
			return
		}

		var req def.LabelGenerationResultReq
		if err := gCtx.ShouldBindJSON(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.LabelGenerationResultResp{},
			})
			return
		}

		resp, err := handler.GetHandler().LabelGenerationResult(ctx, resultID, &req)
		zlog.CtxAllInOne(ctx, "label_generation_result", map[string]interface{}{"result_id": resultID, "req": req}, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.LabelGenerationResultResp{},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// ListUserGenerationBatches 获取用户批次列表路由处理
func ListUserGenerationBatches() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ListUserGenerationBatchesReq
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindQuery(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ListUserGenerationBatchesResp{},
			})
			return
		}

		resp, err := handler.GetHandler().ListUserGenerationBatches(ctx, &req)
		zlog.CtxAllInOne(ctx, "list_user_generation_batches", req, resp, err)

		r := response.NewResponse(gCtx)
		if err != nil {
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ListUserGenerationBatchesResp{},
			})
			return
		} else {
			r.Success(resp)
		}
	}
}

// ExportSFTDataToFile 导出SFT数据到文件路由处理
func ExportSFTDataToFile() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ExportSFTDataReq
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindQuery(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		// 调用Handler导出SFT数据（返回JSONL数据和文件名）
		jsonlData, filename, err := handler.GetHandler().ExportSFTDataToFile(ctx, &req)
		zlog.CtxAllInOne(ctx, "export_sft_data_to_file", req, map[string]interface{}{"filename": filename, "data_length": len(jsonlData)}, err)

		if err != nil {
			// 错误时返回JSON格式的错误响应
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		// 成功时设置响应头并返回文件流
		gCtx.Header("Content-Type", "application/x-ndjson")
		gCtx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		gCtx.Header("Content-Length", fmt.Sprintf("%d", len(jsonlData)))

		// 直接返回JSONL内容
		gCtx.String(http.StatusOK, jsonlData)
	}
}

// ExportSFTSessionDataToFile 导出session格式SFT数据
func ExportSFTSessionDataToFile() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ExportSFTDataReq
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindQuery(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		jsonlData, filename, err := handler.GetHandler().ExportSFTSessionDataToFile(ctx, &req)
		zlog.CtxAllInOne(ctx, "export_sft_session_data_to_file", req, map[string]interface{}{"filename": filename, "data_length": len(jsonlData)}, err)

		if err != nil {
			// 错误时返回JSON格式的错误响应
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		// 成功时设置响应头并返回文件流
		gCtx.Header("Content-Type", "application/x-ndjson")
		gCtx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		gCtx.Header("Content-Length", fmt.Sprintf("%d", len(jsonlData)))
		gCtx.String(http.StatusOK, jsonlData)
	}
}

// ExportDPOData 导出DPO数据路由处理
func ExportDPOData() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		var req def.ExportSFTDataReq
		ctx := gCtx.Request.Context()

		if err := gCtx.ShouldBindQuery(&req); err != nil {
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    response.INVALID_PARAMS.Code,
				Message: response.INVALID_PARAMS.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		// 调用Handler导出DPO数据
		jsonlData, err := handler.GetHandler().ExportDPOData(ctx, &req)

		// 生成文件名
		filename := fmt.Sprintf("DPO_Text_Sample_%s.jsonl", time.Now().Format("20060102_150405"))
		zlog.CtxAllInOne(ctx, "export_dpo_data", req, map[string]interface{}{"filename": filename, "data_length": len(jsonlData)}, err)

		if err != nil {
			// 错误时返回JSON格式的错误响应
			msgCode := mapGenerationServiceErrorToMsgCode(err)
			if msgCode == response.COMMON_FAIL {
				msgCode.Msg = err.Error()
			}
			gCtx.JSON(http.StatusOK, response.JsonMsgResult{
				Code:    msgCode.Code,
				Message: msgCode.Msg,
				Data:    def.ExportSFTDataToFileResp{},
			})
			return
		}

		// 成功时设置响应头并返回文件流
		gCtx.Header("Content-Type", "application/x-ndjson")
		gCtx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		gCtx.Header("Content-Length", fmt.Sprintf("%d", len(jsonlData)))

		// 直接返回JSONL内容
		gCtx.String(http.StatusOK, jsonlData)
	}
}
