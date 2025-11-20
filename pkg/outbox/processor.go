// TODO: 事务发件箱模式 - 未来高并发场景下可启用
// 当前采用简化的随机沉睡+重试方案，如果需要更强的数据一致性保证，可以启用此模块
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"forge/biz/entity"
	"forge/infra/storage"
	"forge/pkg/log/zlog"
	"forge/pkg/queue"
	"math/rand"
	"time"

	"github.com/panjf2000/ants/v2"
)

// OutboxProcessor 发件箱事件处理器
type OutboxProcessor struct {
	outboxRepo   storage.OutboxRepo
	qualityQueue *queue.QualityQueue
	workerPool   *ants.Pool
	stopChan     chan struct{}
	ticker       *time.Ticker
}

// NewOutboxProcessor 创建发件箱处理器
func NewOutboxProcessor(outboxRepo storage.OutboxRepo, qualityQueue *queue.QualityQueue) *OutboxProcessor {
	pool, err := ants.NewPool(5) // 5个并发处理器
	if err != nil {
		panic(fmt.Sprintf("创建发件箱处理器协程池失败: %v", err))
	}

	return &OutboxProcessor{
		outboxRepo:   outboxRepo,
		qualityQueue: qualityQueue,
		workerPool:   pool,
		stopChan:     make(chan struct{}),
		ticker:       time.NewTicker(5 * time.Second), // 每5秒扫描一次
	}
}

// Start 启动发件箱处理器
func (p *OutboxProcessor) Start() {
	zlog.Infof("发件箱处理器启动")
	go p.processLoop()
}

// Stop 停止发件箱处理器
func (p *OutboxProcessor) Stop() {
	close(p.stopChan)
	p.ticker.Stop()
	p.workerPool.Release()
	zlog.Infof("发件箱处理器已停止")
}

// processLoop 处理循环
func (p *OutboxProcessor) processLoop() {
	for {
		select {
		case <-p.stopChan:
			return
		case <-p.ticker.C:
			p.processPendingEvents()
		}
	}
}

// processPendingEvents 处理待处理的事件
func (p *OutboxProcessor) processPendingEvents() {
	ctx := context.Background()

	// 获取待处理的事件（批量处理，每次最多50个）
	events, err := p.outboxRepo.GetPendingEvents(ctx, 50)
	if err != nil {
		zlog.Errorf("获取待处理发件箱事件失败: %v", err)
		return
	}

	if len(events) == 0 {
		return // 没有待处理事件
	}

	zlog.Infof("发现 %d 个待处理发件箱事件", len(events))

	// 并发处理事件
	for _, event := range events {
		event := event // 避免闭包问题

		err := p.workerPool.Submit(func() {
			p.processEvent(ctx, event)
		})

		if err != nil {
			zlog.Errorf("提交发件箱事件处理任务失败: %v, 事件ID: %s", err, event.EventID)
		}
	}
}

// processEvent 处理单个事件
func (p *OutboxProcessor) processEvent(ctx context.Context, event *entity.OutboxEvent) {
	zlog.Infof("开始处理发件箱事件: %s, 类型: %s", event.EventID, event.EventType)

	// 更新状态为处理中
	err := p.outboxRepo.UpdateEventStatus(ctx, event.EventID, entity.OUTBOX_STATUS_PROCESSING, nil, "")
	if err != nil {
		zlog.Errorf("更新事件状态为处理中失败: %v, 事件ID: %s", err, event.EventID)
		return
	}

	// 根据事件类型分发处理
	var processErr error
	switch event.EventType {
	case entity.OUTBOX_EVENT_QUALITY_ASSESSMENT:
		processErr = p.processQualityAssessmentEvent(ctx, event)
	default:
		processErr = fmt.Errorf("未知的事件类型: %s", event.EventType)
	}

	// 更新处理结果
	now := time.Now()
	if processErr != nil {
		// 处理失败
		zlog.Errorf("处理发件箱事件失败: %v, 事件ID: %s", processErr, event.EventID)

		// 增加重试次数
		p.outboxRepo.IncrementRetryCount(ctx, event.EventID)

		// 如果重试次数超过限制，标记为失败
		if event.RetryCount >= 3 {
			p.outboxRepo.UpdateEventStatus(ctx, event.EventID, entity.OUTBOX_STATUS_FAILED, &now, processErr.Error())
		} else {
			// 重新标记为待处理，等待下次重试
			p.outboxRepo.UpdateEventStatus(ctx, event.EventID, entity.OUTBOX_STATUS_PENDING, nil, processErr.Error())
		}
	} else {
		// 处理成功
		zlog.Infof("发件箱事件处理成功: %s", event.EventID)
		p.outboxRepo.UpdateEventStatus(ctx, event.EventID, entity.OUTBOX_STATUS_COMPLETED, &now, "")
	}
}

// processQualityAssessmentEvent 处理质量评估事件
func (p *OutboxProcessor) processQualityAssessmentEvent(ctx context.Context, event *entity.OutboxEvent) error {
	// 解析载荷
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("序列化事件载荷失败: %w", err)
	}

	var task entity.QualityAssessmentTask
	if err := json.Unmarshal(payloadBytes, &task); err != nil {
		return fmt.Errorf("反序列化质量评估任务失败: %w", err)
	}

	// 将任务加入质量评估队列
	if p.qualityQueue == nil {
		return fmt.Errorf("质量评估队列未初始化")
	}

	// 添加指数退避延迟，避免重试时的惊群效应
	if event.RetryCount > 0 {
		delay := p.calculateBackoffWithJitter(event.RetryCount)
		zlog.Infof("重试事件，延迟 %v: %s", delay, event.EventID)
		time.Sleep(delay)
	}

	if err := p.qualityQueue.EnqueueTask(&task); err != nil {
		return fmt.Errorf("将质量评估任务加入队列失败: %w", err)
	}

	return nil
}

// calculateBackoffWithJitter 计算带抖动的指数退避延迟
func (p *OutboxProcessor) calculateBackoffWithJitter(attempt int) time.Duration {
	// 基础延迟: 1秒
	baseDelay := 1 * time.Second

	// 指数退避: 1s, 2s, 4s, 8s...
	exponentialDelay := baseDelay * time.Duration(1<<uint(attempt))

	// 添加抖动 (±25%)
	jitter := time.Duration(rand.Float64() * 0.5 * float64(exponentialDelay))
	if rand.Float64() < 0.5 {
		jitter = -jitter
	}

	finalDelay := exponentialDelay + jitter

	// 最大延迟限制
	maxDelay := 30 * time.Second
	if finalDelay > maxDelay {
		finalDelay = maxDelay
	}

	return finalDelay
}
