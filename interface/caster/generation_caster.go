package caster

import (
	"forge/biz/entity"
	"forge/biz/types"
	"forge/interface/def"
)

// CastGenerateMindMapProReq2Params 请求转参数
func CastGenerateMindMapProReq2Params(req *def.GenerateMindMapProReq) *types.GenerateMindMapProParams {
	params := &types.GenerateMindMapProParams{
		Count:    req.Count,
		Strategy: req.Strategy,
		File:     req.File, // 修复：添加文件字段转换
	}

	if req.Text != nil {
		params.Text = *req.Text
	}

	return params
}

// CastGenerationBatchDO2DTO 实体转DTO
func CastGenerationBatchDO2DTO(batch *entity.GenerationBatch) *def.GenerationBatchDTO {
	return &def.GenerationBatchDTO{
		BatchID:            batch.BatchID,
		UserID:             batch.UserID,
		InputText:          batch.InputText,
		GenerationCount:    batch.GenerationCount,
		GenerationStrategy: batch.GenerationStrategy,
		CreatedAt:          batch.CreatedAt,
		UpdatedAt:          batch.UpdatedAt,
	}
}

// CastGenerationResultDO2DTO 实体转DTO
func CastGenerationResultDO2DTO(result *entity.GenerationResult) *def.GenerationResultDTO {
	return &def.GenerationResultDTO{
		ResultID: result.ResultID,
		BatchID:  result.BatchID,
		//		ConversationID: result.ConversationID,
		MapJSON:   result.MapJSON,
		Label:     result.Label,
		LabeledAt: result.LabeledAt,
		CreatedAt: result.CreatedAt,
	}
}

// CastGenerationResultDOs2DTOs 批量实体转DTO
func CastGenerationResultDOs2DTOs(results []*entity.GenerationResult) []*def.GenerationResultDTO {
	dtos := make([]*def.GenerationResultDTO, 0, len(results))
	for _, result := range results {
		dtos = append(dtos, CastGenerationResultDO2DTO(result))
	}
	return dtos
}

// CastGenerationBatchDOs2DTOs 批量实体转DTO
func CastGenerationBatchDOs2DTOs(batches []*entity.GenerationBatch) []*def.GenerationBatchDTO {
	dtos := make([]*def.GenerationBatchDTO, 0, len(batches))
	for _, batch := range batches {
		dtos = append(dtos, CastGenerationBatchDO2DTO(batch))
	}
	return dtos
}
