package types

import (
	"context"
	"mime/multipart"

	"forge/biz/entity"
)

// GenerateMindMapProParams 批量生成导图参数
type GenerateMindMapProParams struct {
	Text     string                `json:"text"`
	File     *multipart.FileHeader `json:"-"`
	Count    int                   `json:"count"`    // 生成数量 3-5
	Strategy int                   `json:"strategy"` // 1=并行+内容多样化, 2=单次多样
}

// IGenerationService 生成服务接口
type IGenerationService interface {
	// GetBatchWithResults 获取批次及其结果
	GetBatchWithResults(ctx context.Context, batchID string) (*entity.GenerationBatch, []*entity.GenerationResult, error)

	// ListUserBatches 获取用户批次列表
	ListUserBatches(ctx context.Context, userID string, page, pageSize int) ([]*entity.GenerationBatch, int64, error)

	// LabelResult 标记结果
	LabelResult(ctx context.Context, resultID string, label int) error

	// LabelResultWithSave 标记结果并可能保存导图
	LabelResultWithSave(ctx context.Context, resultID string, label int) (*entity.MindMap, error)

	// ExportDPOData 导出DPO数据
	ExportDPOData(ctx context.Context, startDate, endDate, userID string) (string, error)

	// ExportSFTDataToFile 导出SFT数据到文件（支持loss_weight筛选）
	// 返回：jsonlData, filename, error
	ExportSFTDataToFile(ctx context.Context, startDate, endDate, userID string, minLossWeight float64) (string, string, error)

	// SaveSelectedMindMap 保存选中的导图到正式系统
	SaveSelectedMindMap(ctx context.Context, resultID string) (*entity.MindMap, error)

	// SaveGenerationBatch 保存批次和结果（事务操作）
	SaveGenerationBatch(ctx context.Context, batch *entity.GenerationBatch, results []*entity.GenerationResult, conversations []*entity.Conversation) error
}
