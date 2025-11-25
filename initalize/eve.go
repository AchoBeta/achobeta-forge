package initalize

import (
	"context"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
	"runtime"
)

func Eve() {
	// 关闭 CozeLoop 客户端
	loop.Close(context.Background())
	
	//zlog.Warnf("开始释放资源！")
	//errRedis := global.Rdb.Close()
	//if errRedis != nil {
	//	zlog.Errorf("Redis关闭失败 ：%v", errRedis.Error())
	//}
	//
	//sqlDB, _ := global.DB.DB()
	//errDB := sqlDB.Close()
	//if errDB != nil {
	//	zlog.Errorf("数据库关闭失败 ：%v", errDB.Error())
	//}
	runtime.GC()
	//if errDB == nil && errRedis == nil {
	zlog.Warnf("资源释放成功！")
	//}
}
