package storage

import (
	"context"
	"fmt"
	"time"

	"forge/biz/entity"
	"forge/biz/repo"
	"forge/infra/database"
	"forge/infra/storage/po"
	"forge/pkg/log/zlog"

	"gorm.io/gorm"
)

type generationPersistence struct {
	db *gorm.DB
}

var gp *generationPersistence

func InitGenerationStorage() {
	db := database.ForgeDB()

	// 自动迁移生成相关表
	if err := db.AutoMigrate(&po.GenerationBatchPO{}, &po.GenerationResultPO{}); err != nil {
		panic(fmt.Sprintf("failed to auto migrate generation tables: %v", err))
	}

	gp = &generationPersistence{
		db: db,
	}
}

func GetGenerationPersistence() repo.IGenerationRepo {
	return gp
}

// CreateGenerationBatch 创建生成批次
func (g *generationPersistence) CreateGenerationBatch(ctx context.Context, batch *entity.GenerationBatch) error {
	batchPO := CastGenerationBatchDO2PO(batch)
	if err := g.db.WithContext(ctx).Create(batchPO).Error; err != nil {
		return fmt.Errorf("create generation batch failed: %w", err)
	}
	return nil
}

// GetGenerationBatch 获取生成批次
func (g *generationPersistence) GetGenerationBatch(ctx context.Context, batchID, userID string) (*entity.GenerationBatch, error) {
	var batchPO po.GenerationBatchPO

	db := g.db.WithContext(ctx)
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}

	if err := db.Where("batch_id = ?", batchID).First(&batchPO).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrGenerationBatchNotFound
		}
		return nil, fmt.Errorf("get generation batch failed: %w", err)
	}

	return CastGenerationBatchPO2DO(&batchPO), nil
}

// ListUserGenerationBatches 获取用户的批次列表
func (g *generationPersistence) ListUserGenerationBatches(ctx context.Context, userID string, page, pageSize int) ([]*entity.GenerationBatch, int64, error) {
	var batchPOs []po.GenerationBatchPO
	var total int64

	db := g.db.WithContext(ctx).Where("user_id = ?", userID)

	// 统计总数
	if err := db.Model(&po.GenerationBatchPO{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count generation batches failed: %w", err)
	}

	// 分页查询
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	// 按创建时间倒序
	if err := db.Order("created_at DESC").Find(&batchPOs).Error; err != nil {
		return nil, 0, fmt.Errorf("list generation batches failed: %w", err)
	}

	// 转换为实体
	batches := make([]*entity.GenerationBatch, 0, len(batchPOs))
	for _, po := range batchPOs {
		batches = append(batches, CastGenerationBatchPO2DO(&po))
	}

	return batches, total, nil
}

// CreateGenerationResults 批量创建生成结果
func (g *generationPersistence) CreateGenerationResults(ctx context.Context, results []*entity.GenerationResult) error {
	resultPOs := CastGenerationResultDOs2POs(results)
	if err := g.db.WithContext(ctx).Create(&resultPOs).Error; err != nil {
		return fmt.Errorf("create generation results failed: %w", err)
	}
	return nil
}

// GetGenerationResultsByBatchID 根据批次ID获取所有结果
func (g *generationPersistence) GetGenerationResultsByBatchID(ctx context.Context, batchID string) ([]*entity.GenerationResult, error) {
	var resultPOs []po.GenerationResultPO

	if err := g.db.WithContext(ctx).Where("batch_id = ?", batchID).Order("created_at ASC").Find(&resultPOs).Error; err != nil {
		return nil, fmt.Errorf("get generation results by batch id failed: %w", err)
	}

	return CastGenerationResultPOs2DOs(resultPOs), nil
}

// GetGenerationResult 获取单个生成结果
func (g *generationPersistence) GetGenerationResult(ctx context.Context, resultID string) (*entity.GenerationResult, error) {
	var resultPO po.GenerationResultPO

	if err := g.db.WithContext(ctx).Where("result_id = ?", resultID).First(&resultPO).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repo.ErrGenerationResultNotFound
		}
		return nil, fmt.Errorf("get generation result failed: %w", err)
	}

	return CastGenerationResultPO2DO(&resultPO), nil
}

// UpdateGenerationResultLabel 更新结果标签
func (g *generationPersistence) UpdateGenerationResultLabel(ctx context.Context, resultID string, label int) error {
	updates := make(map[string]interface{})
	updates["label"] = label
	if label != 0 {
		updates["labeled_at"] = time.Now()
	} else {
		updates["labeled_at"] = nil
	}

	result := g.db.WithContext(ctx).Model(&po.GenerationResultPO{}).Where("result_id = ?", resultID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update generation result label failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return repo.ErrGenerationResultNotFound
	}

	return nil
}

// UpdateGenerationResult 更新生成结果
func (g *generationPersistence) UpdateGenerationResult(ctx context.Context, result *entity.GenerationResult) error {
	po := CastGenerationResultDO2PO(result)

	dbResult := g.db.WithContext(ctx).Model(&po).Where("result_id = ?", result.ResultID).Updates(po)
	if dbResult.Error != nil {
		return fmt.Errorf("update generation result failed: %w", dbResult.Error)
	}

	if dbResult.RowsAffected == 0 {
		return repo.ErrGenerationResultNotFound
	}

	return nil
}

// GetLabeledResults 获取已标记的结果（用于SFT导出）
// userID 为空时导出所有用户的数据
func (g *generationPersistence) GetLabeledResults(ctx context.Context, userID, startDate, endDate string) ([]*entity.GenerationResult, error) {
	var resultPOs []po.GenerationResultPO

	// 使用GORM的Model和Joins，让GORM自动处理表名
	batchTable := po.GenerationBatchPO{}.TableName()
	resultTable := po.GenerationResultPO{}.TableName()

	db := g.db.WithContext(ctx).
		Table(resultTable).
		Joins(fmt.Sprintf("JOIN %s ON %s.batch_id COLLATE utf8mb4_unicode_ci = %s.batch_id COLLATE utf8mb4_unicode_ci", batchTable, resultTable, batchTable)).
		Where(fmt.Sprintf("%s.label != 0", resultTable)) // 只获取已标记的数据

	// 用户ID过滤（如果提供）
	if userID != "" {
		db = db.Where(fmt.Sprintf("%s.user_id = ?", batchTable), userID)
	}

	// 时间范围过滤
	if startDate != "" {
		db = db.Where(fmt.Sprintf("%s.created_at >= ?", resultTable), startDate)
	}
	if endDate != "" {
		db = db.Where(fmt.Sprintf("%s.created_at <= ?", resultTable), endDate)
	}

	if err := db.Order(fmt.Sprintf("%s.created_at ASC", resultTable)).Find(&resultPOs).Error; err != nil {
		return nil, fmt.Errorf("get labeled results failed: %w", err)
	}

	return CastGenerationResultPOs2DOs(resultPOs), nil
}

// SaveGenerationBatch 保存批次和结果（事务操作）
func (g *generationPersistence) SaveGenerationBatch(ctx context.Context, batch *entity.GenerationBatch, results []*entity.GenerationResult, conversations []*entity.Conversation) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 创建批次
		batchPO := CastGenerationBatchDO2PO(batch)
		if err := tx.Create(batchPO).Error; err != nil {
			return fmt.Errorf("create batch failed: %w", err)
		}

		// 2. 创建对话记录 (批量生成专用保存逻辑)
		for _, conversation := range conversations {
			if err := g.saveBatchGenerationConversation(tx, conversation); err != nil {
				zlog.CtxWarnf(ctx, "save batch conversation failed: %v", err)
				// 继续处理其他对话，不中断事务
			}
		}

		// 3. 创建结果记录
		if len(results) > 0 {
			resultPOs := CastGenerationResultDOs2POs(results)
			if err := tx.Create(&resultPOs).Error; err != nil {
				return fmt.Errorf("create results failed: %w", err)
			}
		}

		return nil
	})
}

// saveBatchGenerationConversation 保存批量生成的对话记录（绕过MapID检查）
func (g *generationPersistence) saveBatchGenerationConversation(tx *gorm.DB, conversation *entity.Conversation) error {
	if conversation.ConversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}
	if conversation.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if conversation.Title == "" {
		return fmt.Errorf("conversation title is required")
	}

	conversationPO, err := CastConversationDO2PO(conversation)
	if err != nil {
		return fmt.Errorf("cast conversation to PO failed: %w", err)
	}

	if err := tx.Create(&conversationPO).Error; err != nil {
		return fmt.Errorf("create conversation failed: %w", err)
	}

	return nil
}

// CastGenerationBatchDO2PO 实体转PO
func CastGenerationBatchDO2PO(batch *entity.GenerationBatch) *po.GenerationBatchPO {
	return &po.GenerationBatchPO{
		BatchID:            batch.BatchID,
		UserID:             batch.UserID,
		InputText:          batch.InputText,
		GenerationCount:    batch.GenerationCount,
		GenerationStrategy: batch.GenerationStrategy,
		CreatedAt:          batch.CreatedAt,
		UpdatedAt:          batch.UpdatedAt,
	}
}

// CastGenerationBatchPO2DO PO转实体
func CastGenerationBatchPO2DO(po *po.GenerationBatchPO) *entity.GenerationBatch {
	return &entity.GenerationBatch{
		BatchID:            po.BatchID,
		UserID:             po.UserID,
		InputText:          po.InputText,
		GenerationCount:    po.GenerationCount,
		GenerationStrategy: po.GenerationStrategy,
		CreatedAt:          po.CreatedAt,
		UpdatedAt:          po.UpdatedAt,
	}
}

// CastGenerationResultDO2PO 实体转PO
func CastGenerationResultDO2PO(result *entity.GenerationResult) *po.GenerationResultPO {
	return &po.GenerationResultPO{
		ResultID:       result.ResultID,
		BatchID:        result.BatchID,
		ConversationID: result.ConversationID,
		MapJSON:        result.MapJSON,
		Label:          result.Label,
		LabeledAt:      result.LabeledAt,
		CreatedAt:      result.CreatedAt,
		Strategy:       result.Strategy,
		ErrorMessage:   result.ErrorMessage,
	}
}

// CastGenerationResultPO2DO PO转实体
func CastGenerationResultPO2DO(po *po.GenerationResultPO) *entity.GenerationResult {
	return &entity.GenerationResult{
		ResultID:       po.ResultID,
		BatchID:        po.BatchID,
		ConversationID: po.ConversationID,
		MapJSON:        po.MapJSON,
		Label:          po.Label,
		LabeledAt:      po.LabeledAt,
		CreatedAt:      po.CreatedAt,
		Strategy:       po.Strategy,
		ErrorMessage:   po.ErrorMessage,
	}
}

// CastGenerationResultDOs2POs 批量实体转PO
func CastGenerationResultDOs2POs(results []*entity.GenerationResult) []po.GenerationResultPO {
	pos := make([]po.GenerationResultPO, 0, len(results))
	for _, result := range results {
		pos = append(pos, *CastGenerationResultDO2PO(result))
	}
	return pos
}

// CastGenerationResultPOs2DOs 批量PO转实体
func CastGenerationResultPOs2DOs(pos []po.GenerationResultPO) []*entity.GenerationResult {
	results := make([]*entity.GenerationResult, 0, len(pos))
	for _, po := range pos {
		results = append(results, CastGenerationResultPO2DO(&po))
	}
	return results
}
