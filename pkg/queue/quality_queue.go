package queue

import (
	"context"
	"errors"
	"fmt"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/infra/eino"
	"forge/pkg/log/zlog"
	"math/rand"
	"time"

	"github.com/panjf2000/ants/v2"
)

// QualityAssessmentTask 定义质量评估任务
type QualityAssessmentTask entity.QualityAssessmentTask

// QualityQueue 质量评估队列
type QualityQueue struct {
	taskChan      chan *QualityAssessmentTask
	workerPool    *ants.Pool
	stopChan      chan struct{}
	qualityClient *eino.QualityAssessmentClient
	aiChatRepo    repo.AiChatRepo
}

var globalQualityQueue *QualityQueue

// InitQualityQueue 初始化全局质量评估队列
func InitQualityQueue(aiChatRepo repo.AiChatRepo) error {
	poolSize := 3       // 协程池大小
	queueCapacity := 50 // 队列容量

	pool, err := ants.NewPool(poolSize)
	if err != nil {
		return fmt.Errorf("创建协程池失败: %w", err)
	}

	globalQualityQueue = &QualityQueue{
		taskChan:      make(chan *QualityAssessmentTask, queueCapacity),
		workerPool:    pool,
		stopChan:      make(chan struct{}),
		qualityClient: eino.NewQualityAssessmentClient(),
		aiChatRepo:    aiChatRepo,
	}

	// 启动队列消费者
	go globalQualityQueue.start()

	zlog.Infof("质量评估队列初始化成功，协程池大小: %d, 队列容量: %d", poolSize, queueCapacity)
	return nil
}

// GetQualityQueue 获取全局质量评估队列
func GetQualityQueue() *QualityQueue {
	return globalQualityQueue
}

// start 启动队列消费者
func (q *QualityQueue) start() {
	zlog.Infof("质量评估队列启动")
	for {
		select {
		case <-q.stopChan:
			zlog.Infof("质量评估队列停止")
			return
		case task := <-q.taskChan:
			// 提交任务到协程池
			err := q.workerPool.Submit(func() {
				q.processTask(task)
			})
			if err != nil {
				zlog.Errorf("提交质量评估任务到协程池失败: %v, 任务内容: %+v", err, task)
			}
		}
	}
}

// processTask 处理质量评估任务
func (q *QualityQueue) processTask(task *QualityAssessmentTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	zlog.CtxInfof(ctx, "开始处理质量评估任务: 会话ID=%s, 消息ID=%s", task.ConversationID, task.MessageID)

	// 调用质量评估模型
	qualityScore, err := q.qualityClient.AssessQuality(ctx, task.MessageContent, task.MapData)
	if err != nil {
		zlog.CtxErrorf(ctx, "质量评估失败: %v, 任务内容: %+v", err, task)
		return
	}

	// 转换评分：1表示高质量，-1表示低质量
	finalScore := qualityScore
	if qualityScore == 0 {
		finalScore = -1 // 低质量用-1表示
	}

	// 带重试的更新数据库中的质量评分 - 使用指数退避+抖动
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = q.aiChatRepo.UpdateMessageQuality(ctx, task.ConversationID, task.MessageID, finalScore)
		if err == nil {
			break // 成功，退出重试循环
		}

		if i < maxRetries-1 {
			// 指数退避 + 随机抖动，避免惊群效应
			waitTime := q.calculateBackoffWithJitter(i)
			zlog.CtxWarnf(ctx, "更新消息质量评分失败，%v后重试 (第%d次): %v", waitTime, i+1, err)
			time.Sleep(waitTime)
		}
	}

	if err != nil {
		zlog.CtxErrorf(ctx, "更新消息质量评分最终失败: %v, 会话ID=%s, 消息ID=%s, 评分=%d",
			err, task.ConversationID, task.MessageID, finalScore)
		return
	}

	zlog.CtxInfof(ctx, "质量评估任务完成: 会话ID=%s, 消息ID=%s, 评分=%d",
		task.ConversationID, task.MessageID, finalScore)
}

// EnqueueTask 将任务加入队列
func (q *QualityQueue) EnqueueTask(task *entity.QualityAssessmentTask) error {
	if q == nil {
		return errors.New("质量评估队列未初始化")
	}

	queueTask := (*QualityAssessmentTask)(task)

	select {
	case q.taskChan <- queueTask:
		zlog.Infof("质量评估任务已加入队列: 会话ID=%s, 消息ID=%s", task.ConversationID, task.MessageID)
		return nil
	default:
		return errors.New("质量评估队列已满，任务提交失败")
	}
}

// Stop 停止队列消费者并关闭协程池
func (q *QualityQueue) Stop() {
	if q == nil {
		return
	}

	close(q.stopChan)
	q.workerPool.Release()
	zlog.Infof("质量评估队列和协程池已关闭")
}

// calculateBackoffWithJitter 计算带抖动的指数退避延迟
func (q *QualityQueue) calculateBackoffWithJitter(attempt int) time.Duration {
	// 基础延迟: 500ms
	baseDelay := 500 * time.Millisecond

	// 指数退避: 500ms, 1s, 2s
	exponentialDelay := baseDelay * time.Duration(1<<uint(attempt))

	// 添加抖动 (±25%)
	jitter := time.Duration(rand.Float64() * 0.5 * float64(exponentialDelay))
	if rand.Float64() < 0.5 {
		jitter = -jitter
	}

	finalDelay := exponentialDelay + jitter

	// 最大延迟限制
	maxDelay := 10 * time.Second
	if finalDelay > maxDelay {
		finalDelay = maxDelay
	}

	// 最小延迟限制
	minDelay := 100 * time.Millisecond
	if finalDelay < minDelay {
		finalDelay = minDelay
	}

	return finalDelay
}
