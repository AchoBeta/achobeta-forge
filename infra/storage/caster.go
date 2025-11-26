package storage

import (
	"encoding/json"
	"fmt"
	"forge/biz/entity"
	"forge/infra/storage/po"

	"gorm.io/datatypes"
)

// CastUserDO2PO
//
//	@Description: 实体与存储互转
//	@param user
//	@return *po.UserPO
func CastUserDO2PO(user *entity.User) *po.UserPO {
	if user == nil {
		return nil
	}
	return &po.UserPO{
		UserID:        user.UserID,
		UserName:      user.UserName,
		Avatar:        user.Avatar,
		Password:      user.Password,
		Phone:         user.Phone,
		Email:         user.Email,
		Status:        user.Status,
		PhoneVerified: user.PhoneVerified,
		EmailVerified: user.EmailVerified,
		GithubID:      user.GithubID,
		GithubLogin:   user.GithubLogin,
		WechatOpenID:  user.WechatOpenID,
		WechatUnionID: user.WechatUnionID,
		LastLoginAt:   user.LastLoginAt,
	}
}

// CastUserPO2DO 存储转实体
func CastUserPO2DO(userPO *po.UserPO) *entity.User {
	if userPO == nil {
		return nil
	}
	user := &entity.User{
		UserID:        userPO.UserID,
		UserName:      userPO.UserName,
		Avatar:        userPO.Avatar,
		Password:      userPO.Password,
		Phone:         userPO.Phone,
		Email:         userPO.Email,
		Status:        userPO.Status,
		PhoneVerified: userPO.PhoneVerified,
		EmailVerified: userPO.EmailVerified,
		GithubID:      userPO.GithubID,
		GithubLogin:   userPO.GithubLogin,
		WechatOpenID:  userPO.WechatOpenID,
		WechatUnionID: userPO.WechatUnionID,
		LastLoginAt:   userPO.LastLoginAt,
	}

	// 处理时间字段：如果 PO 中为 nil，Entity 中保持零值；否则解引用
	if userPO.CreatedAt != nil {
		user.CreatedAt = *userPO.CreatedAt
	}
	if userPO.UpdatedAt != nil {
		user.UpdatedAt = *userPO.UpdatedAt
	}

	return user
}

// CastMindMapDO2PO 领域对象转持久化对象
func CastMindMapDO2PO(mindmap *entity.MindMap) (*po.MindMapPO, error) {
	if mindmap == nil {
		return nil, nil
	}

	// 序列化Data为JSON字符串
	dataBytes, err := json.Marshal(mindmap.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mindmap data: %w", err)
	}

	mindmapPO := &po.MindMapPO{
		MapID:  mindmap.MapID,
		UserID: mindmap.UserID,
		Title:  mindmap.Title,
		Desc:   mindmap.Desc,
		Data:   string(dataBytes),
		Layout: mindmap.Layout,
	}

	// 处理时间字段
	if !mindmap.CreatedAt.IsZero() {
		mindmapPO.CreatedAt = &mindmap.CreatedAt
	}
	if !mindmap.UpdatedAt.IsZero() {
		mindmapPO.UpdatedAt = &mindmap.UpdatedAt
	}

	return mindmapPO, nil
}

// CastMindMapPO2DO 持久化对象转领域对象
func CastMindMapPO2DO(mindmapPO *po.MindMapPO) (*entity.MindMap, error) {
	if mindmapPO == nil {
		return nil, nil
	}

	// 反序列化JSON数据
	var data entity.MindMapData
	if err := json.Unmarshal([]byte(mindmapPO.Data), &data); err != nil {
		return nil, fmt.Errorf("unmarshal data failed: %w", err)
	}

	mindmap := &entity.MindMap{
		MapID:  mindmapPO.MapID,
		UserID: mindmapPO.UserID,
		Title:  mindmapPO.Title,
		Desc:   mindmapPO.Desc,
		Data:   data,
		Layout: mindmapPO.Layout,
	}

	// 处理时间字段
	if mindmapPO.CreatedAt != nil {
		mindmap.CreatedAt = *mindmapPO.CreatedAt
	}
	if mindmapPO.UpdatedAt != nil {
		mindmap.UpdatedAt = *mindmapPO.UpdatedAt
	}

	return mindmap, nil
}

func CastConversationPO2DO(conversationPO *po.ConversationPO) (*entity.Conversation, error) {
	if conversationPO == nil {
		return nil, nil
	}

	var messages []*entity.Message
	if err := json.Unmarshal(conversationPO.Messages, &messages); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &entity.Conversation{
		ConversationID: conversationPO.ConversationID,
		UserID:         conversationPO.UserID,
		MapID:          conversationPO.MapID,
		Title:          conversationPO.Title,
		Messages:       messages,
		CreatedAt:      conversationPO.CreatedAt,
		UpdatedAt:      conversationPO.UpdatedAt,
	}, nil

}

func CastConversationPOs2DOs(conversationPOs []po.ConversationPO) ([]*entity.Conversation, error) {
	if conversationPOs == nil {
		return nil, nil
	}

	var res []*entity.Conversation

	for _, conversationPO := range conversationPOs {
		conversation, err := CastConversationPO2DO(&conversationPO)
		if err != nil {
			return nil, err
		}
		res = append(res, conversation)
	}
	return res, nil
}

func CastConversationDO2PO(conversation *entity.Conversation) (*po.ConversationPO, error) {
	if conversation == nil {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(conversation.Messages)
	if err != nil {
		return nil, fmt.Errorf("json序列化失败: %w", err)
	}

	conversationPO := &po.ConversationPO{
		ConversationID: conversation.ConversationID,
		UserID:         conversation.UserID,
		MapID:          conversation.MapID,
		Title:          conversation.Title,
		Messages:       datatypes.JSON(jsonBytes),
		CreatedAt:      conversation.CreatedAt,
		UpdatedAt:      conversation.UpdatedAt,
	}
	return conversationPO, nil

}
