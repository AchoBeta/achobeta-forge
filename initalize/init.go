package initalize

import (
	_ "embed"
	"fmt"
	"forge/biz/aichatservice"
	"forge/biz/cosservice"
	"forge/biz/mindmapservice"
	"forge/biz/userservice"
	"forge/infra/cache"
	"forge/infra/configs"
	"forge/infra/cos"
	"forge/infra/coze"
	"forge/infra/database"
	"forge/infra/eino"
	"forge/infra/notification"
	"forge/infra/storage"
	"forge/interface/handler"
	"forge/interface/router"
	"forge/pkg/log"
	"github.com/unidoc/unioffice/v2/common/license"
	pdfLicense "github.com/unidoc/unipdf/v4/common/license"
	// "forge/pkg/loop"
	"forge/util"
)

func Init() {
	// load env
	path := initPath()
	introduce()
	log.InitLog(path, configs.Config())
	configs.MustInit(path)
	log.InitLog(path, configs.Config())
	database.MustInitDatabase(configs.Config())
	cache.MustInitCache(configs.Config())
	// TODO: cozeloop配置好后启用
	// loop.MustInitLoop()
	coze.InitCozeService()
	notification.InitCodeService(configs.Config().GetSMTPConfig(), configs.Config().GetSMSConfig())

	storage.InitUserStorage()
	storage.InitMindMapStorage()
	storage.InitAiChatStorage()

	// snowflake - 从配置文件读取节点ID
	snowflakeConfig := configs.Config().GetSnowflakeConfig()
	if err := util.InitSnowflake(snowflakeConfig.NodeID); err != nil {
		// 初始化失败，直接 panic 提示原因
		panic(fmt.Sprintf("init snowflake failed: %v", err))
	}

	// 从配置文件读取JWT配置并创建JWTUtil
	jwtConfig := configs.Config().GetJWTConfig()
	jwtUtil := util.NewJWTUtil(jwtConfig.SecretKey, jwtConfig.ExpireHours)

	us := userservice.NewUserServiceImpl(storage.GetUserPersistence(), coze.GetCozeService(), jwtUtil, notification.GetCodeService())

	// 依赖注入：创建COS服务实例
	cosConfig := configs.Config().GetCOSConfig()
	cosService := cos.NewCOSService(cosConfig)

	mms := mindmapservice.NewMindMapServiceImpl(storage.GetMindMapPersistence())
	cs := cosservice.NewCOSServiceImpl(cosService, cosConfig)

	// 依赖注入: 创建ai服务实例
	aiConfig := configs.Config().GetAiChatConfig()
	acs := aichatservice.NewAiChatService(storage.GetAiChatPersistence(), eino.NewAiChatClient(aiConfig.ApiKey, aiConfig.ModelName))
	handler.MustInitHandler(us, mms, cs, acs)

	//从配置文件中读取解析文件apikey
	uniOfficeConfig := configs.Config().GetUniOfficeConfig()
	license.SetMeteredKey(uniOfficeConfig.MeteredKey)
	pdfLicense.SetMeteredKey(uniOfficeConfig.MeteredKey)

	// 初始化JWT鉴权中间件
	router.InitJWTAuth(us)

}
func initPath() string {
	return util.GetRootPath("")
}

// dont like? see https://patorjk.com/software/taag/#p=display&f=Merlin1&t=PLUTO
//
//go:embed .logo
var logo string

func introduce() {
	fmt.Println(logo)
}
