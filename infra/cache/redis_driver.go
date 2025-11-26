package cache

import (
	"context"
	"fmt"
	"time"

	"forge/infra/configs"
	"forge/pkg/log/zlog"

	"github.com/go-redis/redis/v8"
)

const (
	redisAddr = "%s:%d"
)

func initRedis(config configs.IConfig) error {
	redisConfig := config.GetRedisConfig()
	if !config.GetRedisConfig().Enable {
		zlog.Warnf("不使用Redis模式")
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Network:            "",
		Addr:               fmt.Sprintf(redisAddr, redisConfig.Host, redisConfig.Port),
		Dialer:             nil,
		OnConnect:          nil,
		Username:           "",
		Password:           redisConfig.Password,
		DB:                 redisConfig.DB,
		MaxRetries:         0,
		MinRetryBackoff:    0,
		MaxRetryBackoff:    0,
		DialTimeout:        0,
		ReadTimeout:        0,
		WriteTimeout:       0,
		PoolFIFO:           false,
		PoolSize:           1000,
		MinIdleConns:       1,
		MaxConnAge:         0,
		PoolTimeout:        0,
		IdleTimeout:        0,
		IdleCheckFrequency: 0,
		TLSConfig:          nil,
		Limiter:            nil,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		zlog.Errorf("redis无法链接 %v", err)
		return err
	}
	redisClient = client
	return nil
}

// SetRedis 设置键值对，带过期时间
func SetRedis(ctx context.Context, key string, value string, expiration time.Duration) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// GetRedis 获取键对应的值
func GetRedis(ctx context.Context, key string) (string, error) {
	if redisClient == nil {
		return "", fmt.Errorf("redis client not initialized")
	}
	result, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // 键不存在
	}
	return result, err
}

// DelRedis 删除键
func DelRedis(ctx context.Context, key string) error {
	if redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return redisClient.Del(ctx, key).Err()
}

// GetRedisClient 获取 Redis 客户端实例
func GetRedisClient() *redis.Client {
	return redisClient
}
