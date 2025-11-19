package storage

import (
	"context"
	"errors"
	"fmt"
	"forge/biz/aichatservice"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/infra/database"
	"forge/infra/storage/po"

	"gorm.io/gorm"
)

type aiChatPersistence struct {
	db *gorm.DB
}

var cp *aiChatPersistence

func InitAiChatStorage() {
	db := database.ForgeDB()

	if err := db.AutoMigrate(&po.ConversationPO{}); err != nil {
		panic(fmt.Sprintf("自动建表失败 :%v", err))
	}

	cp = &aiChatPersistence{db: db}
}

func GetAiChatPersistence() repo.AiChatRepo { return cp }

func (a *aiChatPersistence) GetConversation(ctx context.Context, conversationID, userID string) (*entity.Conversation, error) {
	if conversationID == "" {
		return nil, aichatservice.CONVERSATION_ID_NOT_NULL
	}

	var conversationPO po.ConversationPO
	query := a.db.WithContext(ctx).Model(&po.ConversationPO{}).Where("conversation_id = ?", conversationID)

	// userID 为空时不进行用户ID过滤（用于导出场景）
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&conversationPO).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, aichatservice.CONVERSATION_NOT_EXIST
		}
		return nil, fmt.Errorf("数据库出错 :%w", err)

	}

	return CastConversationPO2DO(&conversationPO)
}

func (a *aiChatPersistence) GetMapAllConversation(ctx context.Context, mapID, userID string) ([]*entity.Conversation, error) {

	if mapID == "" {
		return nil, aichatservice.MAP_ID_NOT_NULL
	} else if userID == "" {
		return nil, aichatservice.USER_ID_NOT_NULL
	}

	check, err := checkMapIsExist(ctx, a, mapID)
	if err != nil {
		return nil, err
	} else if !check {
		return nil, aichatservice.MIND_MAP_NOT_EXIST
	}

	var conversationPOs []po.ConversationPO
	if err := a.db.WithContext(ctx).Model(&po.ConversationPO{}).Where("map_id = ? AND user_id = ?", mapID, userID).Find(&conversationPOs).Error; err != nil {
		return nil, fmt.Errorf("获取导图会话时 数据库出错 %w", err)
	}

	return CastConversationPOs2DOs(conversationPOs)
}

func (a *aiChatPersistence) SaveConversation(ctx context.Context, conversation *entity.Conversation) error {

	if conversation.ConversationID == "" {
		return aichatservice.CONVERSATION_ID_NOT_NULL
	} else if conversation.UserID == "" {
		return aichatservice.USER_ID_NOT_NULL
	} else if conversation.MapID == "" {
		return aichatservice.MAP_ID_NOT_NULL
	} else if conversation.Title == "" {
		return aichatservice.CONVERSATION_TITLE_NOT_NULL
	}

	check, err := checkMapIsExist(ctx, a, conversation.MapID)
	if err != nil {
		return err
	} else if !check {
		return aichatservice.MIND_MAP_NOT_EXIST
	}

	conversationPO, err := CastConversationDO2PO(conversation)
	if err != nil {
		return err
	}
	err = a.db.WithContext(ctx).Model(&po.ConversationPO{}).Create(&conversationPO).Error
	if err != nil {
		return fmt.Errorf("保存会话时，数据库出错 %w", err)
	}
	return nil
}

func (a *aiChatPersistence) UpdateConversationMessage(ctx context.Context, conversation *entity.Conversation) error {

	if conversation.UserID == "" {
		return aichatservice.USER_ID_NOT_NULL
	} else if conversation.MapID == "" {
		return aichatservice.MAP_ID_NOT_NULL
	}

	check, err := checkConversationIsExist(ctx, a, conversation.ConversationID)
	if err != nil {
		return err
	} else if !check {
		return aichatservice.CONVERSATION_NOT_EXIST
	}

	conversationPO, err := CastConversationDO2PO(conversation)
	if err != nil {
		return err
	}

	Updates := make(map[string]interface{})
	if conversationPO.Messages != nil {
		Updates["messages"] = conversationPO.Messages
	}

	err = a.db.WithContext(ctx).Model(&po.ConversationPO{}).Where("conversation_id = ? AND user_id = ?", conversationPO.ConversationID, conversationPO.UserID).Updates(Updates).Error
	if err != nil {
		return fmt.Errorf("更新会话时 数据库出错 %w", err)
	}
	return nil
}

func (a *aiChatPersistence) UpdateConversationTitle(ctx context.Context, conversation *entity.Conversation) error {

	if conversation.UserID == "" {
		return aichatservice.USER_ID_NOT_NULL
	} else if conversation.MapID == "" {
		return aichatservice.MAP_ID_NOT_NULL
	}

	check, err := checkConversationIsExist(ctx, a, conversation.ConversationID)

	if err != nil {
		return err
	} else if !check {
		return aichatservice.CONVERSATION_NOT_EXIST
	}

	conversationPO, err := CastConversationDO2PO(conversation)
	if err != nil {
		return err
	}
	Updates := make(map[string]interface{})
	if conversationPO.Title != "" {
		Updates["title"] = conversationPO.Title
	}

	err = a.db.WithContext(ctx).Model(&po.ConversationPO{}).Where("conversation_id = ? AND user_id = ?", conversationPO.ConversationID, conversationPO.UserID).Updates(Updates).Error
	if err != nil {
		return fmt.Errorf("更新会话时 数据库出错 %w", err)
	}
	return nil
}

func (a *aiChatPersistence) DeleteConversation(ctx context.Context, conversationID, userID string) error {
	if conversationID == "" {
		return aichatservice.CONVERSATION_ID_NOT_NULL
	} else if userID == "" {
		return aichatservice.USER_ID_NOT_NULL
	}

	result := a.db.WithContext(ctx).Model(&po.ConversationPO{}).Where("conversation_id = ? AND user_id = ?", conversationID, userID).Delete(&po.ConversationPO{})
	if result.RowsAffected == 0 {
		return aichatservice.CONVERSATION_NOT_EXIST
	}
	if result.Error != nil {
		return fmt.Errorf("删除会话时出错 %w", result.Error)
	}
	return nil
}

func checkMapIsExist(ctx context.Context, a *aiChatPersistence, checkMapID string) (bool, error) {
	var id uint64
	err := a.db.WithContext(ctx).Model(&po.MindMapPO{}).Select("id").Where("map_id = ?", checkMapID).Take(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("查询失败 数据库错误 %w", err)
	} else {
		return true, nil
	}
}

func checkConversationIsExist(ctx context.Context, a *aiChatPersistence, checkConversationID string) (bool, error) {
	var id uint64
	err := a.db.WithContext(ctx).Model(&po.ConversationPO{}).Select("id").Where("conversation_id = ?", checkConversationID).Take(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("查询失败 数据库错误 %w", err)
	} else {
		return true, nil
	}
}

// GetQualityConversations 获取高质量的对话数据用于导出
// 注意：只获取真实用户对话，排除SFT训练数据
func (a *aiChatPersistence) GetQualityConversations(ctx context.Context, startDate, endDate *string, limit int) ([]*entity.Conversation, error) {
	var conversationPOs []po.ConversationPO
	query := a.db.WithContext(ctx).Model(&po.ConversationPO{})

	// 关键：排除SFT训练数据，只获取真实用户对话
	query = query.Where("map_id NOT IN (?, ?)", entity.SFT_BATCH_GENERATION, entity.SFT_FEWSHOT_GENERATION)

	// 添加时间范围过滤
	if startDate != nil && *startDate != "" {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil && *endDate != "" {
		query = query.Where("created_at <= ?", *endDate)
	}

	// 添加限制
	if limit > 0 {
		query = query.Limit(limit)
	}

	// 按创建时间排序
	query = query.Order("created_at DESC")

	if err := query.Find(&conversationPOs).Error; err != nil {
		return nil, fmt.Errorf("获取质量对话时数据库出错: %w", err)
	}

	return CastConversationPOs2DOs(conversationPOs)
}

// UpdateMessageQuality 更新特定消息的质量评分
func (a *aiChatPersistence) UpdateMessageQuality(ctx context.Context, conversationID string, messageID string, qualityScore int) error {
	// 由于消息存储在JSON字段中，我们需要先获取对话，更新消息，然后保存回去
	conversation, err := a.GetConversation(ctx, conversationID, "")
	if err != nil {
		return err
	}

	// 查找并更新对应ID的消息
	updated := false
	for _, message := range conversation.Messages {
		if message.ID == messageID {
			message.QualityScore = qualityScore
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("未找到ID为 %s 的消息", messageID)
	}

	// 更新对话
	return a.UpdateConversationMessage(ctx, conversation)
}
