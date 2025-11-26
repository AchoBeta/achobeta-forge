package repo

import (
	"context"
	"errors"
	"forge/biz/entity"
)

var (
	ErrGenerationBatchNotFound  = errors.New("生成批次未找到")
	ErrGenerationResultNotFound = errors.New("生成结果未找到")
)

// IGenerationRepo 生成数据存储接口
type IGenerationRepo interface {
	// CreateGenerationBatch 创建生成批次
	CreateGenerationBatch(ctx context.Context, batch *entity.GenerationBatch) error

	// GetGenerationBatch 获取生成批次
	GetGenerationBatch(ctx context.Context, batchID, userID string) (*entity.GenerationBatch, error)

	// ListUserGenerationBatches 获取用户的批次列表
	ListUserGenerationBatches(ctx context.Context, userID string, page, pageSize int) ([]*entity.GenerationBatch, int64, error)

	// CreateGenerationResults 批量创建生成结果
	CreateGenerationResults(ctx context.Context, results []*entity.GenerationResult) error

	// GetGenerationResultsByBatchID 根据批次ID获取所有结果
	GetGenerationResultsByBatchID(ctx context.Context, batchID string) ([]*entity.GenerationResult, error)

	// GetGenerationResult 获取单个生成结果
	GetGenerationResult(ctx context.Context, resultID string) (*entity.GenerationResult, error)

	// UpdateGenerationResultLabel 更新结果标签
	UpdateGenerationResultLabel(ctx context.Context, resultID string, label int) error

	// UpdateGenerationResult 更新生成结果
	UpdateGenerationResult(ctx context.Context, result *entity.GenerationResult) error

	// GetLabeledResults 获取已标记的结果（用于SFT导出）
	GetLabeledResults(ctx context.Context, userID, startDate, endDate string) ([]*entity.GenerationResult, error)

	// SaveGenerationBatch 保存批次和结果（事务操作）
	SaveGenerationBatch(ctx context.Context, batch *entity.GenerationBatch, results []*entity.GenerationResult, conversations []*entity.Conversation) error
}
