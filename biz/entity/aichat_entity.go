package entity

import (
	"context"
	"fmt"
	"forge/infra/configs"
	"forge/util"
	"time"

	"github.com/cloudwego/eino/schema"
)

var (
	SYSTEM    = "system"
	USER      = "user"
	ASSISTANT = "assistant"
	TOOL      = "tool"
)

// SFT训练数据的MapID标识符
const (
	SFT_BATCH_GENERATION   = "BATCH_GENERATION"
	SFT_FEWSHOT_GENERATION = "FEWSHOT_GENERATION"
)

type aiChatCtxKey struct{}

type Message struct {
	ID           string // 消息唯一ID
	Content      string
	Role         string
	ToolCallID   string
	ToolCalls    []schema.ToolCall
	Timestamp    time.Time
	QualityScore int // 0=未评估，1=高质量，-1=低质量
}

type Conversation struct {
	ConversationID string
	UserID         string
	MapID          string
	Title          string
	MapData        string
	Messages       []*Message
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewConversation(userID, mapID, title, mapData string) (*Conversation, error) {
	newID, err := util.GenerateStringID()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	messages := make([]*Message, 0)

	return &Conversation{
		MapData:        mapData,
		ConversationID: newID,
		UserID:         userID,
		MapID:          mapID,
		Title:          title,
		Messages:       messages,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (c *Conversation) AddMessage(content, role, ToolCallID string, ToolCalls []schema.ToolCall) (*Message, error) {
	now := time.Now()

	// 生成消息唯一ID
	messageID, err := util.GenerateStringID()
	if err != nil {
		// 记录错误但继续执行，使用时间戳作为备选方案
		messageID = fmt.Sprintf("%s_%d", c.ConversationID, now.UnixNano())
		// 返回警告，但不阻断流程
		err = fmt.Errorf("ID生成失败，使用备选方案: %w", err)
	}

	message := &Message{
		ID:         messageID,
		Content:    content,
		Role:       role,
		ToolCallID: ToolCallID,
		ToolCalls:  ToolCalls,
		Timestamp:  now,
	}

	c.Messages = append(c.Messages, message)
	c.UpdatedAt = now
	return message, err // 返回消息和可能的警告错误
}

func (c *Conversation) UpdateTitle(title string) {
	c.Title = title
}

func (c *Conversation) UpdateMapData(mapData string) {
	c.MapData = mapData
}

// 处理系统提示词
func (c *Conversation) ProcessSystemPrompt() {
	version := len(c.Messages)

	text := fmt.Sprintf(configs.Config().GetAiChatConfig().SystemPrompt, version, version, c.MapData)
	if len(c.Messages) == 0 {
		c.AddMessage(text, SYSTEM, "", nil)
	} else {
		c.Messages[0] = &Message{
			Content:   text,
			Role:      SYSTEM,
			Timestamp: time.Now(),
		}
	}
}

func WithConversation(ctx context.Context, conversation *Conversation) context.Context {
	ctx = context.WithValue(ctx, aiChatCtxKey{}, conversation)
	return ctx
}

func GetConversation(ctx context.Context) (*Conversation, bool) {
	v, ok := ctx.Value(aiChatCtxKey{}).(*Conversation)
	return v, ok
}

// IsRealUserConversation 判断是否为真实用户对话（非SFT训练数据）
func (c *Conversation) IsRealUserConversation() bool {
	return c.MapID != SFT_BATCH_GENERATION && c.MapID != SFT_FEWSHOT_GENERATION
}

// IsSFTTrainingData 判断是否为SFT训练数据
func (c *Conversation) IsSFTTrainingData() bool {
	return c.MapID == SFT_BATCH_GENERATION || c.MapID == SFT_FEWSHOT_GENERATION
}

// Tab补全相关实体
type TabCompletionRequest struct {
	ConversationID string
	UserInput      string
	MapData        string
}

type TabCompletionResponse struct {
	CompletedText string
	Success       bool
}

// 质量评估队列任务
type QualityAssessmentTask struct {
	MessageID      string
	MessageContent string
	ConversationID string
	MapData        string
}

// JSONL导出相关实体
type JSONLMessage struct {
	Role    string
	Content string
}

type JSONLRecord struct {
	Messages []JSONLMessage
}

// 质量数据导出请求
type ExportQualityDataRequest struct {
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}

type ExportQualityDataResponse struct {
	Success bool
	Count   int
	Data    string // JSONL格式的数据
	Message string
}
