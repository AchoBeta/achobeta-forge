package entity

import (
	"context"
	"fmt"
	"forge/infra/configs"
	"forge/util"
	"github.com/cloudwego/eino/schema"
	"time"
)

var (
	SYSTEM    = "system"
	USER      = "user"
	ASSISTANT = "assistant"
	TOOL      = "tool"
)

type aiChatCtxKey struct{}

type Message struct {
	Content    string            `json:"content"`
	Role       string            `json:"role"`
	ToolCallID string            `json:"tool_call_id"`
	ToolCalls  []schema.ToolCall `json:"tool_calls"`
	Timestamp  time.Time         `json:"timestamp"`
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

func (c *Conversation) AddMessage(content, role, ToolCallID string, ToolCalls []schema.ToolCall) *Message {
	now := time.Now()

	message := &Message{
		Content:    content,
		Role:       role,
		ToolCallID: ToolCallID,
		ToolCalls:  ToolCalls,
		Timestamp:  now,
	}

	c.Messages = append(c.Messages, message)
	c.UpdatedAt = now

	return message
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
