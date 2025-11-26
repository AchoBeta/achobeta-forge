package database

import (
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDataBases 初始化
func initMysql(config configs.IConfig) error {
	dsn := config.GetDBConfig().Dsn

	// 确保DSN包含charset=utf8mb4参数
	if !strings.Contains(dsn, "charset=") {
		if strings.Contains(dsn, "?") {
			dsn += "&charset=utf8mb4"
		} else {
			dsn += "?charset=utf8mb4"
		}
	} else if !strings.Contains(dsn, "utf8mb4") {
		// 如果已有charset但不是utf8mb4，替换为utf8mb4
		dsn = strings.ReplaceAll(dsn, "charset=utf8", "charset=utf8mb4")
		dsn = strings.ReplaceAll(dsn, "charset=latin1", "charset=utf8mb4")
	}

	_db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		zlog.Panicf("MySQL无法连接数据库！: %v", err)
		return err
	}
	zlog.Infof("MySQL连接数据库成功！")
	db = _db

	return nil
}
