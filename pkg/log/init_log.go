package log

import (
	"forge/infra/configs"
	"forge/pkg/log/zlog"
)

func InitLog(path string, config configs.IConfig) {
	logger := GetZap(path, config)
	zlog.InitLogger(logger)
}
