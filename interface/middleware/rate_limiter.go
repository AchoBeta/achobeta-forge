package middleware

import (
	"context"
	"fmt"
	"forge/infra/cache"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"forge/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"net/http"
	"time"
)

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	Limit  int           // 时间窗口内允许的最大请求数
	Window time.Duration // 时间窗口
}

// RateLimiter 基于 Redis 的分布式全局限流中间件
// 使用滑动窗口算法实现
func RateLimiter() gin.HandlerFunc {
	// 从配置中读取限流参数
	rateLimitConfig := configs.Config().GetRateLimitConfig()

	if !rateLimitConfig.Enable {
		zlog.Infof("限流功能未启用")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limit := rateLimitConfig.Limit
	windowSeconds := rateLimitConfig.WindowSeconds

	if limit <= 0 || windowSeconds <= 0 {
		zlog.Warnf("限流配置无效，跳过限流")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	window := time.Duration(windowSeconds) * time.Second

	zlog.Infof("限流中间件已启用: %d 请求/%d 秒", limit, windowSeconds)

	return func(c *gin.Context) {
		redisClient := cache.GetRedisClient()
		if redisClient == nil {
			zlog.Warnf("Redis 客户端未初始化，跳过限流")
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// 使用全局限流 key
		key := "rate_limit:global"

		// 执行限流检查
		allowed, err := checkRateLimit(ctx, redisClient, key, limit, window)
		if err != nil {
			zlog.Errorf("限流检查失败: %v", err)
			// 发生错误时允许请求通过，避免影响服务
			c.Next()
			return
		}

		if !allowed {
			zlog.Warnf("请求被限流: %s %s", c.Request.Method, c.Request.URL.Path)
			resp := response.NewResponse(c)
			resp.ErrorWithStatus(response.TOO_MANY_REQUESTS, http.StatusTooManyRequests)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiterByIP 基于 IP 的限流中间件
func RateLimiterByIP() gin.HandlerFunc {
	rateLimitConfig := configs.Config().GetRateLimitConfig()

	if !rateLimitConfig.Enable {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limit := rateLimitConfig.Limit
	windowSeconds := rateLimitConfig.WindowSeconds
	window := time.Duration(windowSeconds) * time.Second

	return func(c *gin.Context) {
		redisClient := cache.GetRedisClient()
		if redisClient == nil {
			c.Next()
			zlog.Warnf("Redis 客户端未初始化，跳过 IP 限流")
			return
		}

		ctx := c.Request.Context()
		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:ip:%s", ip)

		allowed, err := checkRateLimit(ctx, redisClient, key, limit, window)
		if err != nil {
			zlog.Errorf("限流检查失败: %v", err)
			c.Next()
			return
		}

		if !allowed {
			zlog.Warnf("IP %s 请求被限流: %s %s", ip, c.Request.Method, c.Request.URL.Path)
			resp := response.NewResponse(c)
			resp.ErrorWithStatus(response.TOO_MANY_REQUESTS, http.StatusTooManyRequests)
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRateLimit 使用滑动窗口算法检查限流
// 基于 Redis Sorted Set 实现滑动窗口
func checkRateLimit(ctx context.Context, client *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	pipe := client.Pipeline()

	// 1. 移除窗口之外的旧记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// 2. 统计当前窗口内的请求数
	pipe.ZCard(ctx, key)

	// 3. 添加当前请求到窗口
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d-%s", now, uuid.NewString()),
	})

	// 4. 设置 key 过期时间（窗口时间的 2 倍，确保数据能被清理）
	pipe.Expire(ctx, key, window*2)

	// 执行 pipeline
	cmds, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("执行 Redis pipeline 失败: %w", err)
	}

	// 获取窗口内的请求数（在添加当前请求之前的数量）
	if len(cmds) < 2 {
		return false, fmt.Errorf("Redis pipeline 返回结果不足")
	}

	countCmd, ok := cmds[1].(*redis.IntCmd)
	if !ok {
		return false, fmt.Errorf("无法解析请求计数")
	}

	count := countCmd.Val()

	// 如果当前窗口内的请求数（不包括本次）小于限制，则允许请求
	return count < int64(limit), nil
}
