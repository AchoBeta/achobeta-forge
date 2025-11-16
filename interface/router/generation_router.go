package router

import (
	"fmt"
	"time"

	"forge/interface/def"
	"forge/interface/handler"

	"github.com/gin-gonic/gin"
)

// loadGenerationService 加载生成相关路由
func loadGenerationService(r *gin.RouterGroup) {
	// 批量生成导图
	// [POST] /api/biz/v1/mindmap/generation/pro
	r.Handle(POST, "generation/pro", GenerateMindMapPro())

	// 获取批次详情
	// [GET] /api/biz/v1/mindmap/generation/batch?batch_id=xxx
	r.Handle(GET, "generation/batch", GetGenerationBatch())

	// 标记结果
	// [POST] /api/biz/v1/mindmap/generation/result/:result_id/label
	r.Handle(POST, "generation/result/:result_id/label", LabelGenerationResult())

	// 获取用户批次列表
	// [GET] /api/biz/v1/mindmap/generation/batches
	r.Handle(GET, "generation/batches", ListUserGenerationBatches())

	// 导出SFT数据到文件
	// [GET] /api/biz/v1/mindmap/generation/export-sft-file
	r.Handle(GET, "generation/export-sft-file", ExportSFTDataToFile())

	// 导出DPO数据
	// [GET] /api/biz/v1/mindmap/generation/export-dpo
	r.Handle(GET, "generation/export-dpo", ExportDPOData())

}

// GenerateMindMapPro 批量生成导图路由处理
func GenerateMindMapPro() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req def.GenerateMindMapProReq

		// 处理文件上传
		if file, err := c.FormFile("file"); err == nil {
			req.File = file
		}

		// 绑定其他参数
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": err.Error()})
			return
		}

		// 调用Handler
		resp, err := handler.GetHandler().GenerateMindMapPro(c.Request.Context(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		c.JSON(200, resp)
	}
}

// GetGenerationBatch 获取批次详情路由处理
func GetGenerationBatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		batchID := c.Query("batch_id")
		if batchID == "" {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": "batch_id is required"})
			return
		}

		resp, err := handler.GetHandler().GetGenerationBatch(c.Request.Context(), batchID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		c.JSON(200, resp)
	}
}

// LabelGenerationResult 标记结果路由处理
func LabelGenerationResult() gin.HandlerFunc {
	return func(c *gin.Context) {
		resultID := c.Param("result_id")
		if resultID == "" {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": "result_id is required"})
			return
		}

		var req def.LabelGenerationResultReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": err.Error()})
			return
		}

		resp, err := handler.GetHandler().LabelGenerationResult(c.Request.Context(), resultID, &req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		c.JSON(200, resp)
	}
}

// ListUserGenerationBatches 获取用户批次列表路由处理
func ListUserGenerationBatches() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req def.ListUserGenerationBatchesReq
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": err.Error()})
			return
		}

		resp, err := handler.GetHandler().ListUserGenerationBatches(c.Request.Context(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		c.JSON(200, resp)
	}
}

// ExportSFTDataToFile 导出SFT数据到文件路由处理
func ExportSFTDataToFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req def.ExportSFTDataReq
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": err.Error()})
			return
		}

		// 调用Handler导出SFT数据（返回JSONL数据和文件名）
		jsonlData, filename, err := handler.GetHandler().ExportSFTDataToFile(c.Request.Context(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		// 设置响应头
		c.Header("Content-Type", "application/x-ndjson")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Length", fmt.Sprintf("%d", len(jsonlData)))

		// 直接返回JSONL内容
		c.String(200, jsonlData)
	}
}

// ExportDPOData 导出DPO数据路由处理
func ExportDPOData() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req def.ExportSFTDataReq
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid parameters", "message": err.Error()})
			return
		}

		// 调用Handler导出DPO数据
		jsonlData, err := handler.GetHandler().ExportDPOData(c.Request.Context(), &req)
		if err != nil {
			c.JSON(500, gin.H{"error": "Internal server error", "message": err.Error()})
			return
		}

		// 生成文件名
		filename := fmt.Sprintf("DPO_Text_Sample_%s.jsonl", time.Now().Format("20060102_150405"))

		// 设置响应头
		c.Header("Content-Type", "application/x-ndjson")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Length", fmt.Sprintf("%d", len(jsonlData)))

		// 直接返回JSONL内容
		c.String(200, jsonlData)
	}
}
