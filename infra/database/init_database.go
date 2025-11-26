package database

import (
	"fmt"
	"forge/infra/configs"
	"gorm.io/gorm"
)

var db *gorm.DB

func MustInitDatabase(config configs.IConfig) {
	switch config.GetDBConfig().Driver {
	case "mysql":
		err := initMysql(config)
		if err != nil {
			panic(err)
		}
	default:
		panic(fmt.Sprintf("no require driver:%s", config.GetDBConfig().Driver))
	}
}
func ForgeDB() *gorm.DB {
	return db
}
