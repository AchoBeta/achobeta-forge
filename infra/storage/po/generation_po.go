package po

import (
	"time"

	"gorm.io/gorm"
)

// GenerationBatchPO 批次持久化对象
type GenerationBatchPO struct {
	ID                 uint64    `gorm:"column:id;primary_key;autoIncrement"`
	BatchID            string    `gorm:"column:batch_id;unique;not null"`
	UserID             string    `gorm:"column:user_id;not null"`
	InputText          string    `gorm:"column:input_text;type:longtext;not null"`
	GenerationCount    int       `gorm:"column:generation_count;default:3"`
	GenerationStrategy int       `gorm:"column:generation_strategy;default:1"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (GenerationBatchPO) TableName() string {
	return "achobeta_forge_generation_batch"
}

// GenerationResultPO 结果持久化对象
type GenerationResultPO struct {
	ID             uint64     `gorm:"column:id;primary_key;autoIncrement"`
	ResultID       string     `gorm:"column:result_id;unique;not null"`
	BatchID        string     `gorm:"column:batch_id;not null;index"`
	ConversationID string     `gorm:"column:conversation_id;not null;index"`
	MapJSON        string     `gorm:"column:map_json;type:longtext;not null"`
	Label          int        `gorm:"column:label;default:0;index"`
	LabeledAt      *time.Time `gorm:"column:labeled_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	// AI生成参数（用于训练优化）
	Strategy     *int    `gorm:"column:strategy"`                // 生成策略 1=并行+内容多样化, 2=单次多样
	ErrorMessage *string `gorm:"column:error_message;type:text"` // 错误信息
}

func (GenerationResultPO) TableName() string {
	return "achobeta_forge_generation_result"
}

func (po *GenerationBatchPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	po.CreatedAt = now
	po.UpdatedAt = now
	return nil
}

func (po *GenerationBatchPO) BeforeUpdate(tx *gorm.DB) error {
	po.UpdatedAt = time.Now()
	return nil
}

func (po *GenerationResultPO) BeforeCreate(tx *gorm.DB) error {
	po.CreatedAt = time.Now()
	return nil
}
