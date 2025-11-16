package handler

import (
	"context"
	"errors"
	"strings"

	"forge/biz/entity"
	"forge/interface/caster"
	"forge/interface/def"
	"forge/pkg/log/zlog"
)

var (
	ErrInvalidParams    = errors.New("参数错误")
	ErrPermissionDenied = errors.New("权限不足")
)

// GenerateMindMapPro 批量生成导图
func (h *Handler) GenerateMindMapPro(ctx context.Context, req *def.GenerateMindMapProReq) (rsp *def.GenerateMindMapProResp, err error) {
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.generate_mindmap_pro", req, rsp, err)
	}()

	// 参数验证
	if req.Count < 1 || req.Count > 5 {
		err = ErrInvalidParams
		return
	}

	if req.Strategy != 1 && req.Strategy != 2 {
		err = ErrInvalidParams
		return
	}

	// 修复：检查Text和File至少提供一个
	if (req.Text == nil || strings.TrimSpace(*req.Text) == "") && req.File == nil {
		err = ErrInvalidParams
		return
	}

	// 参数转换
	params := caster.CastGenerateMindMapProReq2Params(req)

	// 调用AI服务层生成
	batch, results, conversations, err := h.AiChatService.GenerateMindMapPro(ctx, params)
	if err != nil {
		return nil, err
	}

	// 调用Generation服务层保存数据
	err = h.GenerationService.SaveGenerationBatch(ctx, batch, results, conversations)
	if err != nil {
		zlog.CtxErrorf(ctx, "保存批次数据失败: %v", err)
		return nil, err
	}

	// 组装响应
	rsp = &def.GenerateMindMapProResp{
		BatchID: batch.BatchID,
		Success: true,
	}
	return rsp, nil
}

// GetGenerationBatch 获取批次详情
func (h *Handler) GetGenerationBatch(ctx context.Context, batchID string) (rsp *def.GetGenerationBatchResp, err error) {
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.get_generation_batch", batchID, rsp, err)
	}()

	// 调用服务层获取批次详情
	batch, results, err := h.GenerationService.GetBatchWithResults(ctx, batchID)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.GetGenerationBatchResp{
		Batch:   caster.CastGenerationBatchDO2DTO(batch),
		Results: caster.CastGenerationResultDOs2DTOs(results),
	}
	return rsp, nil
}

// LabelGenerationResult 标记生成结果
func (h *Handler) LabelGenerationResult(ctx context.Context, resultID string, req *def.LabelGenerationResultReq) (rsp *def.LabelGenerationResultResp, err error) {
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.label_generation_result", map[string]interface{}{"resultID": resultID, "req": req}, rsp, err)
	}()

	// 参数验证
	if req.Label != -1 && req.Label != 0 && req.Label != 1 {
		err = ErrInvalidParams
		return
	}

	// 调用服务层标记并可能保存导图
	savedMindMap, err := h.GenerationService.LabelResultWithSave(ctx, resultID, req.Label)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.LabelGenerationResultResp{
		Success: true,
	}

	// 如果保存了导图，返回导图信息
	if savedMindMap != nil {
		rsp.SavedMapID = &savedMindMap.MapID
		rsp.SavedMapTitle = &savedMindMap.Title
	}

	return rsp, nil
}

// ListUserGenerationBatches 获取用户批次列表
func (h *Handler) ListUserGenerationBatches(ctx context.Context, req *def.ListUserGenerationBatchesReq) (rsp *def.ListUserGenerationBatchesResp, err error) {
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.list_user_generation_batches", req, rsp, err)
	}()

	// 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		err = ErrPermissionDenied
		return
	}

	// 默认分页参数
	page := req.Page
	if page <= 0 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 调用服务层
	batches, total, err := h.GenerationService.ListUserBatches(ctx, user.UserID, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 组装响应
	rsp = &def.ListUserGenerationBatchesResp{
		Batches: caster.CastGenerationBatchDOs2DTOs(batches),
		Total:   total,
		Page:    page,
		Success: true,
	}
	return rsp, nil
}

// ExportSFTDataToFile 导出SFT数据到文件（直接返回JSONL内容用于下载）
func (h *Handler) ExportSFTDataToFile(ctx context.Context, req *def.ExportSFTDataReq) (jsonlData string, filename string, err error) {
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.export_sft_data_to_file", req, map[string]interface{}{"filename": filename, "dataLen": len(jsonlData)}, err)
	}()

	// 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		err = ErrPermissionDenied
		return
	}

	userID := user.UserID
	if req.UserID != "" && req.UserID != userID {
		userID = req.UserID
	}

	// 默认minLossWeight=1.0（只导出人工标注）
	minLossWeight := req.MinLossWeight
	if minLossWeight == 0 {
		minLossWeight = 1.0
	}

	// 调用服务层导出（返回JSONL数据和文件名）
	jsonlData, filename, err = h.GenerationService.ExportSFTDataToFile(ctx, req.StartDate, req.EndDate, userID, minLossWeight)
	if err != nil {
		return "", "", err
	}

	return jsonlData, filename, nil
}

// ExportDPOData 导出DPO数据
func (h *Handler) ExportDPOData(ctx context.Context, req *def.ExportSFTDataReq) (string, error) {
	// 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		return "", ErrPermissionDenied
	}

	userID := user.UserID
	if req.UserID != "" && req.UserID != userID {
		userID = req.UserID
	}

	// 调用服务层导出
	return h.GenerationService.ExportDPOData(ctx, req.StartDate, req.EndDate, userID)
}
