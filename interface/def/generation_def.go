package def

import (
	"mime/multipart"
	"time"
)

// GenerateMindMapProReq 批量生成请求
type GenerateMindMapProReq struct {
	Text     *string               `json:"text,omitempty" form:"text"`
	File     *multipart.FileHeader `json:"-" form:"file"`            // 文件上传
	Count    int                   `json:"count" form:"count"`       // 生成数量 1-5
	Strategy int                   `json:"strategy" form:"strategy"` // 1=SFT训练数据(带推理过程), 2=DPO训练数据(质量对比)
}

// GenerateMindMapProResp 批量生成响应
type GenerateMindMapProResp struct {
	BatchID string `json:"batch_id"`
	Success bool   `json:"success"`
}

// GetGenerationBatchResp 获取批次响应
type GetGenerationBatchResp struct {
	Batch   *GenerationBatchDTO    `json:"batch"`
	Results []*GenerationResultDTO `json:"results"`
}

// GenerationBatchDTO 批次DTO
type GenerationBatchDTO struct {
	BatchID            string    `json:"batch_id"`
	UserID             string    `json:"user_id"`
	InputText          string    `json:"input_text"`
	GenerationCount    int       `json:"generation_count"`
	GenerationStrategy int       `json:"generation_strategy"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// GenerationResultDTO 结果DTO
type GenerationResultDTO struct {
	ResultID       string     `json:"result_id"`
	BatchID        string     `json:"batch_id"`
	ConversationID string     `json:"conversation_id"`
	MapJSON        string     `json:"map_json"`
	Label          int        `json:"label"`
	LabeledAt      *time.Time `json:"labeled_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// LabelGenerationResultReq 标记结果请求
type LabelGenerationResultReq struct {
	Label int `json:"label"` // -1=负样本, 0=未标记, 1=正样本
}

// LabelGenerationResultResp 标记结果响应
type LabelGenerationResultResp struct {
	Success       bool    `json:"success"`
	SavedMapID    *string `json:"saved_map_id,omitempty"`    // 当标记为正值时返回保存的导图ID
	SavedMapTitle *string `json:"saved_map_title,omitempty"` // 保存的导图标题
}

// ListUserGenerationBatchesReq 获取用户批次列表请求
type ListUserGenerationBatchesReq struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// ListUserGenerationBatchesResp 获取用户批次列表响应
type ListUserGenerationBatchesResp struct {
	Batches []*GenerationBatchDTO `json:"batches"`
	Total   int64                 `json:"total"`
	Page    int                   `json:"page"`
	Success bool                  `json:"success"`
}

// ExportSFTDataReq 导出SFT数据请求
type ExportSFTDataReq struct {
	StartDate     string  `json:"start_date" form:"start_date"`                                 // YYYY-MM-DD
	EndDate       string  `json:"end_date" form:"end_date"`                                     // YYYY-MM-DD
	UserID        string  `json:"user_id" form:"user_id"`                                       // 可选，管理员权限
	MinLossWeight float64 `json:"min_loss_weight" form:"min_loss_weight" binding:"min=0,max=1"` // loss_weight筛选阈值（1.0=只人工标注，0.5=人工+部分AI，0.0=全部）
}

// ExportSFTDataToFileResp 导出SFT数据到文件响应
type ExportSFTDataToFileResp struct {
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url,omitempty"` // 可选的下载链接
	Success     bool   `json:"success"`
}
