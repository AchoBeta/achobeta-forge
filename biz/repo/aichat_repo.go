package repo

import (
	"context"
	"forge/biz/entity"
	"forge/biz/types"
)

type AiChatRepo interface {
	//获取某个会话
	GetConversation(ctx context.Context, conversationID, userID string) (*entity.Conversation, error)

	//获取某个导图的所有会话
	GetMapAllConversation(ctx context.Context, mapID, userID string) ([]*entity.Conversation, error)

	//保存某个会话实体
	SaveConversation(ctx context.Context, conversation *entity.Conversation) error

	//更新某个会话的聊天记录
	UpdateConversationMessage(ctx context.Context, conversation *entity.Conversation) error

	//更新某个会话的标题
	UpdateConversationTitle(ctx context.Context, conversation *entity.Conversation) error

	//删除某个会话
	DeleteConversation(ctx context.Context, conversationID, userID string) error

	//获取高质量的对话数据用于导出
	GetQualityConversations(ctx context.Context, startDate, endDate *string, limit int) ([]*entity.Conversation, error)

	//更新特定消息的质量评分
	UpdateMessageQuality(ctx context.Context, conversationID string, messageID string, qualityScore int) error
}

type EinoServer interface {
	//向ai发送消息
	SendMessage(ctx context.Context, messages []*entity.Message) (types.AgentResponse, error)

	//生成导图
	GenerateMindMap(ctx context.Context, text, userID string) (string, error)

	//批量生成导图
	GenerateMindMapBatch(ctx context.Context, text, userID string, strategy int, count int) ([]string, []*entity.Conversation, error)
}
