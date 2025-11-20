package handler

import (
	"context"
	"fmt"
	"forge/constant"
	"forge/interface/caster"
	"forge/interface/def"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
)

func (h *Handler) SendMessage(ctx context.Context, req *def.ProcessUserMessageRequest) (resp *def.ProcessUserMessageResponse, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.send_message", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.send_message", req, resp, err)
		loop.SetSpanAllInOne(ctx, sp, req, resp, err)
	}()

	//转 biz层 参数
	params := caster.CastProcessUserMessageReq2Params(req)

	aiMsg, err := h.AiChatService.ProcessUserMessage(ctx, params)
	if err != nil {
		return nil, err
	}

	resp = &def.ProcessUserMessageResponse{
		Content:    aiMsg.Content,
		NewMapJson: aiMsg.NewMapJson,
		Success:    true,
	}

	return resp, nil
}

func (h *Handler) SaveNewConversation(ctx context.Context, req *def.SaveNewConversationRequest) (*def.SaveNewConversationResponse, error) {
	params := caster.CastSaveNewConversationReq2Params(req)

	conversationID, err := h.AiChatService.SaveNewConversation(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &def.SaveNewConversationResponse{
		ConversationID: conversationID,
		Success:        true,
	}
	return resp, nil
}

func (h *Handler) GetConversationList(ctx context.Context, req *def.GetConversationListRequest) (*def.GetConversationListResponse, error) {
	params := caster.CastGetConversationListReq2Params(req)

	conversations, err := h.AiChatService.GetConversationList(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &def.GetConversationListResponse{
		Success: true,
		List:    caster.CastConversationsDOs2Resp(conversations),
	}

	return resp, nil

}

func (h *Handler) DelConversation(ctx context.Context, req *def.DelConversationRequest) (*def.DelConversationResponse, error) {
	params := caster.CastDelConversationReq2Params(req)

	err := h.AiChatService.DelConversation(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &def.DelConversationResponse{
		Success: true,
	}
	return resp, nil
}

func (h *Handler) GetConversation(ctx context.Context, req *def.GetConversationRequest) (*def.GetConversationResponse, error) {
	params := caster.CastGetConversationReq2Params(req)

	conversation, err := h.AiChatService.GetConversation(ctx, params)

	if err != nil {
		return nil, err
	}

	resp := &def.GetConversationResponse{
		Success:        true,
		Title:          conversation.Title,
		Messages:       conversation.Messages,
		ConversationID: conversation.ConversationID,
	}

	return resp, nil
}

func (h *Handler) UpdateConversationTitle(ctx context.Context, req *def.UpdateConversationTitleRequest) (*def.UpdateConversationTitleResponse, error) {
	params := caster.CastUpdateConversationTitleReq2Params(req)

	err := h.AiChatService.UpdateConversationTitle(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &def.UpdateConversationTitleResponse{
		Success: true,
	}
	return resp, nil
}

func (h *Handler) GenerateMindMap(ctx context.Context, req *def.GenerateMindMapRequest) (resp *def.GenerateMindMapResponse, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.generate_mindmap", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.generate_mindmap", req, resp, err)
		loop.SetSpanAllInOne(ctx, sp, req, resp, err)
	}()

	params := caster.CastGenerateMindMapReq2Params(req)

	res, err := h.AiChatService.GenerateMindMap(ctx, params)
	if err != nil {
		return nil, err
	}

	resp = &def.GenerateMindMapResponse{
		Success: true,
		MapJson: res,
	}
	return resp, nil
}

// TabComplete 处理Tab补全请求
func (h *Handler) TabComplete(ctx context.Context, req *def.TabCompletionRequest) (resp *def.TabCompletionResponse, err error) {
	// 链路追踪
	ctx, sp := loop.GetNewSpan(ctx, "handler.tab_complete", constant.LoopSpanType_Handle)
	defer func() {
		zlog.CtxAllInOne(ctx, "handler.tab_complete", req, resp, err)
		loop.SetSpanAllInOne(ctx, sp, req, resp, err)
	}()

	// 转换参数
	params := caster.CastTabCompletionReq2Params(req)

	// 调用服务层
	completedText, err := h.AiChatService.ProcessTabCompletion(ctx, params)
	if err != nil {
		return nil, err
	}

	resp = &def.TabCompletionResponse{
		CompletedText: completedText,
		Success:       true,
	}

	return resp, nil
}

// ExportQualityData 导出质量数据
func (h *Handler) ExportQualityData(ctx context.Context, req *def.ExportQualityDataRequest) (*def.ExportQualityDataResponse, error) {
	// 转换参数
	params := caster.CastExportQualityDataReq2Params(req)

	// 调用服务层
	jsonlData, count, err := h.AiChatService.ExportQualityConversations(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &def.ExportQualityDataResponse{
		Success: true,
		Count:   count,
		Data:    jsonlData,
	}

	return resp, nil
}

// TriggerQualityAssessment 手动触发质量评估
func (h *Handler) TriggerQualityAssessment(ctx context.Context, req *def.TriggerQualityAssessmentRequest) (*def.TriggerQualityAssessmentResponse, error) {
	// 调用服务层
	totalCount, processedCount, errorCount, err := h.AiChatService.TriggerQualityAssessment(ctx, req.Date)
	if err != nil {
		return nil, err
	}

	resp := &def.TriggerQualityAssessmentResponse{
		Success:        true,
		TotalCount:     totalCount,
		ProcessedCount: processedCount,
		ErrorCount:     errorCount,
		Message:        fmt.Sprintf("质量评估完成，处理了 %d/%d 条消息", processedCount, totalCount),
	}

	return resp, nil
}
