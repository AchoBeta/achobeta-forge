package caster

import (
	"forge/biz/entity"
	"forge/biz/types"
	"forge/interface/def"
)

func CastProcessUserMessageReq2Params(req *def.ProcessUserMessageRequest) *types.ProcessUserMessageParams {
	if req == nil {
		return nil
	}

	return &types.ProcessUserMessageParams{
		ConversationID: req.ConversationID,
		Message:        req.Content,
		MapData:        req.MapData,
	}
}

func CastSaveNewConversationReq2Params(req *def.SaveNewConversationRequest) *types.SaveNewConversationParams {
	if req == nil {
		return nil
	}
	return &types.SaveNewConversationParams{
		Title:   req.Title,
		MapID:   req.MapID,
		MapData: req.MapData,
	}
}

func CastGetConversationListReq2Params(req *def.GetConversationListRequest) *types.GetConversationListParams {
	if req == nil {
		return nil
	}
	return &types.GetConversationListParams{
		MapID: req.MapID,
	}
}

func CastConversationsDOs2Resp(conversations []*entity.Conversation) []def.ConversationData {
	if conversations == nil {
		return nil
	}

	conversationsData := make([]def.ConversationData, len(conversations))

	for i, conversation := range conversations {
		conversationsData[i] = def.ConversationData{
			ConversationID: conversation.ConversationID,
			Title:          conversation.Title,
			CreatedAt:      conversation.CreatedAt,
			UpdatedAt:      conversation.UpdatedAt,
		}
	}

	return conversationsData
}

func CastDelConversationReq2Params(req *def.DelConversationRequest) *types.DelConversationParams {
	if req == nil {
		return nil
	}

	return &types.DelConversationParams{
		ConversationID: req.ConversationID,
	}
}

func CastGetConversationReq2Params(req *def.GetConversationRequest) *types.GetConversationParams {
	if req == nil {
		return nil
	}
	return &types.GetConversationParams{
		ConversationID: req.ConversationID,
	}
}

func CastUpdateConversationTitleReq2Params(req *def.UpdateConversationTitleRequest) *types.UpdateConversationTitleParams {
	if req == nil {
		return nil
	}
	return &types.UpdateConversationTitleParams{
		Title:          req.Title,
		ConversationID: req.ConversationID,
	}
}

func CastGenerateMindMapReq2Params(req *def.GenerateMindMapRequest) *types.GenerateMindMapParams {
	if req == nil {
		return nil
	}
	return &types.GenerateMindMapParams{
		Text: req.Text,
		File: req.File,
	}
}

// CastTabCompletionReq2Params 转换Tab补全请求参数
func CastTabCompletionReq2Params(req *def.TabCompletionRequest) *types.TabCompletionParams {
	if req == nil {
		return nil
	}
	return &types.TabCompletionParams{
		ConversationID: req.ConversationID,
		UserInput:      req.UserInput,
		MapData:        req.MapData,
	}
}

// CastExportQualityDataReq2Params 转换质量数据导出请求参数
func CastExportQualityDataReq2Params(req *def.ExportQualityDataRequest) *types.ExportQualityDataParams {
	if req == nil {
		return nil
	}
	return &types.ExportQualityDataParams{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
	}
}
