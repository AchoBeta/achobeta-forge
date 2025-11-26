package cache

import (
	"forge/infra/configs"
	"forge/pkg/log/zlog"

	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
)

func MustInitCache(config configs.IConfig) {
	err := initRedis(config)
	if err != nil {
		zlog.Errorf("初始化链接redis失败:%v", err.Error())
		panic(err)
	}
}
