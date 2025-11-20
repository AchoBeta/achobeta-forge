// TODO: 事务发件箱存储层 - 未来高并发场景下可启用
// 提供强一致性的事件发布机制，避免分布式事务问题
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"forge/biz/entity"
	"forge/infra/database"
	"forge/infra/storage/po"
	"forge/util"
	"time"

	"gorm.io/gorm"
)

type OutboxRepo interface {
	// 在事务中保存发件箱事件
	SaveEventInTx(ctx context.Context, tx *gorm.DB, event *entity.OutboxEvent) error

	// 获取待处理的事件
	GetPendingEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error)

	// 更新事件状态
	UpdateEventStatus(ctx context.Context, eventID string, status string, processedAt *time.Time, lastError string) error

	// 增加重试次数
	IncrementRetryCount(ctx context.Context, eventID string) error
}

type outboxPersistence struct {
	db *gorm.DB
}

var outboxRepo *outboxPersistence

func InitOutboxStorage() {
	db := database.ForgeDB()

	if err := db.AutoMigrate(&po.OutboxEventPO{}); err != nil {
		panic(fmt.Sprintf("发件箱表自动建表失败: %v", err))
	}

	outboxRepo = &outboxPersistence{db: db}
}

func GetOutboxRepo() OutboxRepo {
	return outboxRepo
}

// SaveEventInTx 在事务中保存发件箱事件
func (o *outboxPersistence) SaveEventInTx(ctx context.Context, tx *gorm.DB, event *entity.OutboxEvent) error {
	// 生成事件ID
	if event.EventID == "" {
		eventID, err := util.GenerateStringID()
		if err != nil {
			return fmt.Errorf("生成事件ID失败: %w", err)
		}
		event.EventID = eventID
	}

	// 序列化Payload
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("序列化事件载荷失败: %w", err)
	}

	eventPO := &po.OutboxEventPO{
		EventID:     event.EventID,
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     payloadBytes,
		Status:      entity.OUTBOX_STATUS_PENDING,
		RetryCount:  0,
	}

	if err := tx.WithContext(ctx).Create(eventPO).Error; err != nil {
		return fmt.Errorf("保存发件箱事件失败: %w", err)
	}

	return nil
}

// GetPendingEvents 获取待处理的事件
func (o *outboxPersistence) GetPendingEvents(ctx context.Context, limit int) ([]*entity.OutboxEvent, error) {
	var eventPOs []po.OutboxEventPO

	query := o.db.WithContext(ctx).
		Where("status = ?", entity.OUTBOX_STATUS_PENDING).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&eventPOs).Error; err != nil {
		return nil, fmt.Errorf("获取待处理事件失败: %w", err)
	}

	events := make([]*entity.OutboxEvent, 0, len(eventPOs))
	for _, eventPO := range eventPOs {
		event, err := castOutboxEventPO2DO(&eventPO)
		if err != nil {
			continue // 跳过转换失败的事件
		}
		events = append(events, event)
	}

	return events, nil
}

// UpdateEventStatus 更新事件状态
func (o *outboxPersistence) UpdateEventStatus(ctx context.Context, eventID string, status string, processedAt *time.Time, lastError string) error {
	updates := map[string]interface{}{
		"status":     status,
		"last_error": lastError,
	}

	if processedAt != nil {
		updates["processed_at"] = *processedAt
	}

	result := o.db.WithContext(ctx).
		Model(&po.OutboxEventPO{}).
		Where("event_id = ?", eventID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新事件状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("事件不存在: %s", eventID)
	}

	return nil
}

// IncrementRetryCount 增加重试次数
func (o *outboxPersistence) IncrementRetryCount(ctx context.Context, eventID string) error {
	result := o.db.WithContext(ctx).
		Model(&po.OutboxEventPO{}).
		Where("event_id = ?", eventID).
		Update("retry_count", gorm.Expr("retry_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("增加重试次数失败: %w", result.Error)
	}

	return nil
}

// castOutboxEventPO2DO 转换PO到DO
func castOutboxEventPO2DO(eventPO *po.OutboxEventPO) (*entity.OutboxEvent, error) {
	if eventPO == nil {
		return nil, nil
	}

	// 反序列化Payload
	var payload interface{}
	if len(eventPO.Payload) > 0 {
		if err := json.Unmarshal(eventPO.Payload, &payload); err != nil {
			return nil, fmt.Errorf("反序列化事件载荷失败: %w", err)
		}
	}

	return &entity.OutboxEvent{
		EventID:     eventPO.EventID,
		EventType:   eventPO.EventType,
		AggregateID: eventPO.AggregateID,
		Payload:     payload,
		Status:      eventPO.Status,
		CreatedAt:   eventPO.CreatedAt,
		UpdatedAt:   eventPO.UpdatedAt,
		ProcessedAt: eventPO.ProcessedAt,
		RetryCount:  eventPO.RetryCount,
		LastError:   eventPO.LastError,
	}, nil
}
