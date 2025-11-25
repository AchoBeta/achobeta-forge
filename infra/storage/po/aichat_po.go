package po

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type ConversationPO struct {
	ID             uint64         `gorm:"column:id;primary_key;autoIncrement"`
	ConversationID string         `gorm:"column:conversation_id;unique"`
	UserID         string         `gorm:"column:user_id;not null"`
	MapID          string         `gorm:"column:map_id;not null"`
	Title          string         `gorm:"column:title;not null"`
	Text           string         `gorm:"column:text"`
	Messages       datatypes.JSON `gorm:"column:messages;type:json"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
}

func (ConversationPO) TableName() string {
	return "achobeta_forge_conversation"
}

func (m *ConversationPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
	return nil
}

func (m *ConversationPO) BeforeUpdate(tx *gorm.DB) error {
	now := time.Now()
	m.UpdatedAt = now
	return nil
}
