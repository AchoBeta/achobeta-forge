package entity

import (
	"errors"
	"time"
)

// GenerationBatch 生成批次实体
type GenerationBatch struct {
	BatchID            string
	UserID             string
	InputText          string // 存储解析后的文本内容（无论原始输入是文本还是文件）
	GenerationCount    int    // 生成数量3-5个
	GenerationStrategy int    // 1=并行+内容多样化, 2=单次多样
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// GenerationResult 生成结果实体
type GenerationResult struct {
	ResultID       string
	BatchID        string
	ConversationID string // 关联对话数据
	MapJSON        string // 导图JSON
	Label          int    // 0=未标记, 1=正样本, -1=负样本
	LabeledAt      *time.Time
	CreatedAt      time.Time
	// AI生成参数（用于训练优化）
	Strategy     *int    `json:"strategy,omitempty"`      // 生成策略 1=并行+内容多样化, 2=单次多样
	ErrorMessage *string `json:"error_message,omitempty"` // 错误信息（格式错误时）
}

// Validate 批次实体校验
func (gb *GenerationBatch) Validate() error {
	if gb.InputText == "" {
		return errors.New("输入文本不能为空")
	}
	if gb.GenerationCount < 1 || gb.GenerationCount > 5 {
		return errors.New("生成数量必须在1-5个之间")
	}
	if gb.GenerationStrategy != 1 && gb.GenerationStrategy != 2 {
		return errors.New("生成策略必须是1或2")
	}
	return nil
}

// SetLabel 设置标签
func (gr *GenerationResult) SetLabel(label int) error {
	if label != -1 && label != 0 && label != 1 {
		return errors.New("标签值必须是-1、0或1")
	}
	gr.Label = label
	if label != 0 {
		now := time.Now()
		gr.LabeledAt = &now
	}
	return nil
}
