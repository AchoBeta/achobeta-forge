package po

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OutboxEventPO 事务发件箱事件
type OutboxEventPO struct {
	ID          uint64         `gorm:"column:id;primary_key;autoIncrement"`
	EventID     string         `gorm:"column:event_id;unique;not null"`
	EventType   string         `gorm:"column:event_type;not null"`
	AggregateID string         `gorm:"column:aggregate_id;not null"` // 关联的业务ID（如ConversationID）
	Payload     datatypes.JSON `gorm:"column:payload;type:json"`
	Status      string         `gorm:"column:status;not null;default:'pending'"` // pending, processing, completed, failed
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	ProcessedAt *time.Time     `gorm:"column:processed_at"`
	RetryCount  int            `gorm:"column:retry_count;default:0"`
	LastError   string         `gorm:"column:last_error"`
}

func (OutboxEventPO) TableName() string {
	return "achobeta_forge_outbox_event"
}

func (o *OutboxEventPO) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	o.CreatedAt = now
	o.UpdatedAt = now
	return nil
}

func (o *OutboxEventPO) BeforeUpdate(tx *gorm.DB) error {
	o.UpdatedAt = time.Now()
	return nil
}
