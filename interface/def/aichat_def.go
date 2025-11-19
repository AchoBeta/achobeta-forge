package def

import (
	"forge/biz/entity"
	"mime/multipart"
	"time"
)

// 请求体
type ProcessUserMessageRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	Content        string `json:"content" binding:"required"`
	MapData        string `json:"map_data"`
}

type ProcessUserMessageResponse struct {
	NewMapJson string `json:"new_map_json"`
	Content    string `json:"content"`
	Success    bool   `json:"success"`
}

type SaveNewConversationRequest struct {
	Title   string `json:"title" binding:"required"`
	MapID   string `json:"map_id" binding:"required"`
	MapData string `json:"map_data"`
}

type SaveNewConversationResponse struct {
	ConversationID string `json:"conversation_id"`
	Success        bool   `json:"success"`
}

type GetConversationListRequest struct {
	MapID string `json:"map_id" binding:"required"`
}

type ConversationData struct {
	ConversationID string    `json:"conversation_id"`
	Title          string    `json:"title"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GetConversationListResponse struct {
	List    []ConversationData `json:"list"`
	Success bool               `json:"success"`
}

type DelConversationRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

type DelConversationResponse struct {
	Success bool `json:"success"`
}

type GetConversationRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

type GetConversationResponse struct {
	Title          string            `json:"title"`
	Messages       []*entity.Message `json:"messages"`
	ConversationID string            `json:"conversation_id"`
	Success        bool              `json:"success"`
}

type UpdateConversationTitleRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	Title          string `json:"title" binding:"required"`
}

type UpdateConversationTitleResponse struct {
	Success bool `json:"success"`
}

type GenerateMindMapRequest struct {
	Text string `json:"text"` //预留文本字段
	File *multipart.FileHeader
}

type GenerateMindMapResponse struct {
	Success bool   `json:"success"`
	MapJson string `json:"map_json"`
}

// Tab补全相关定义
type TabCompletionRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	UserInput      string `json:"user_input" binding:"required"`
	MapData        string `json:"map_data"`
}

type TabCompletionResponse struct {
	CompletedText string `json:"completed_text"`
	Success       bool   `json:"success"`
}

// 质量数据导出相关定义
type ExportQualityDataRequest struct {
	StartDate *string `json:"start_date"` // 格式: "2006-01-02"
	EndDate   *string `json:"end_date"`   // 格式: "2006-01-02"
	Limit     int     `json:"limit"`
}

type ExportQualityDataResponse struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Data    string `json:"data"` // JSONL格式的数据
	Message string `json:"message,omitempty"`
}

// 手动触发质量评估相关定义
type TriggerQualityAssessmentRequest struct {
	Date string `json:"date" form:"date"` // 格式: "2006-01-02"，为空则评估昨天
}

type TriggerQualityAssessmentResponse struct {
	Success        bool   `json:"success"`
	ProcessedCount int    `json:"processed_count"`
	TotalCount     int    `json:"total_count"`
	ErrorCount     int    `json:"error_count"`
	Message        string `json:"message,omitempty"`
}
