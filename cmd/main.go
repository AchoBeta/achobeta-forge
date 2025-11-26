package main

import (
	"forge/initalize"
	"forge/interface/router"
	"forge/pkg/log/zlog"
)

func main() {
	initalize.Init()
	// 释放资源 todo优雅退出
	defer initalize.Eve()
	router.RunServer()
	zlog.Infof("server is done")
}
