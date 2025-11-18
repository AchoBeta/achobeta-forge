package router

import (
	"fmt"
	"forge/biz/types"
	"forge/infra/configs"
	"forge/interface/middleware"
	"forge/pkg/log/zlog"
	"forge/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

var (
	jwtAuthMiddleware gin.HandlerFunc
)

// InitJWTAuth 初始化JWT鉴权中间件
func InitJWTAuth(userService types.IUserService) {
	jwtConfig := configs.Config().GetJWTConfig()

	// 如果secret_key为空，使用默认值（仅开发环境）
	secretKey := jwtConfig.SecretKey
	if secretKey == "" {
		secretKey = "default-secret-key-change-in-production"
		zlog.Warnf("JWT secret_key is empty, using default key. Please set it in config.yaml")
	}

	jwtUtil := util.NewJWTUtil(secretKey, jwtConfig.ExpireHours)
	jwtAuthMiddleware = middleware.JWTAuth(jwtUtil, userService)
}

func RunServer() {
	r := register()
	run(r)
}

func register() (router *gin.Engine) {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.RouterGroup = *r.Group("/api/biz/v1", middleware.AddTracer())

	// 用户服务：不需要JWT的路由（登录、注册、发送验证码、重置密码）
	userGroup := r.Group("user")
	loadUserService(userGroup)

	// OAuth 第三方登录路由（不需要JWT）
	authGroup := r.Group("auth")
	loadOAuthService(authGroup)

	// 用户服务：需要JWT鉴权的路由（更新头像, 查看个人主页，更新联系方式）
	userAuthGroup := r.Group("user", jwtAuthMiddleware)
	loadUserAuthService(userAuthGroup)

	// mindmap路由组需要JWT鉴权
	mindMapGroup := r.Group("mindmap", jwtAuthMiddleware)
	loadMindMapService(mindMapGroup)
	loadGenerationService(mindMapGroup) // 添加生成相关路由

	// cos路由组需要JWT鉴权
	cosGroup := r.Group("cos", jwtAuthMiddleware)
	loadCOSService(cosGroup)

	//ai路由组需要jwt鉴权
	aiChat := r.Group("aichat", jwtAuthMiddleware)
	loadAiChat(aiChat)

	return r
}

func run(router *gin.Engine) {
	prot := cast.ToString(configs.Config().GetAppConfig().Port)
	host := configs.Config().GetAppConfig().Host

	zlog.Infof("server run success")
	router.Run(fmt.Sprintf("%s:%s", host, prot))
	zlog.Infof("close run success")
}

const (
	POST   = "POST"
	GET    = "GET"
	PUT    = "PUT"
	DELETE = "DELETE"
)

func loadUserService(r *gin.RouterGroup) {
	r.Handle(POST, "login", Login())

	// 注册接口 user/api/biz/v1/register
	// [POST] /api/biz/v1/user/register
	r.Handle(POST, "register", Register())

	// 发送验证码接口
	// [POST] /api/biz/v1/user/send_code
	r.Handle(POST, "send_code", SendCode())

	// 重置密码接口
	// [POST] /api/biz/v1/user/reset_password
	r.Handle(POST, "reset_password", ResetPassword())
}

func loadUserAuthService(r *gin.RouterGroup) {
	// 个人主页接口
	// [GET] /api/biz/v1/user/home
	r.Handle(GET, "home", GetHome())

	// 发送验证码接口（换绑场景，需要JWT认证）
	// [POST] /api/biz/v1/user/send_code_for_change
	r.Handle(POST, "send_code_for_change", SendCode())

	// 更新联系方式接口（绑定/换绑） 手机号/邮箱
	// [POST] /api/biz/v1/user/account
	r.Handle(POST, "account", UpdateAccount())

	// 解绑联系方式接口（手机号/邮箱）
	// [DELETE] /api/biz/v1/user/contact
	r.Handle(DELETE, "contact", UnbindAccount())

	// 更新头像接口（改为POST，因为要上传文件）
	// [POST] /api/biz/v1/user/avatar
	r.Handle(POST, "avatar", UpdateAvatar())

	// 更新用户名接口
	// [POST] /api/biz/v1/user/username
	r.Handle(POST, "username", UpdateUserName())
}

func loadMindMapService(r *gin.RouterGroup) {
	// 创建思维导图
	// [POST] /api/biz/v1/mindmap
	r.Handle(POST, "", CreateMindMap())

	// 获取思维导图详情
	// [GET] /api/biz/v1/mindmap/:id
	r.Handle(GET, ":id", GetMindMap())

	// 获取思维导图列表
	// [GET] /api/biz/v1/mindmap/list
	r.Handle(GET, "list", ListMindMaps())

	// 更新思维导图
	// [PUT] /api/biz/v1/mindmap/:id
	r.Handle(PUT, ":id", UpdateMindMap())

	// 批量删除思维导图
	// [POST] /api/biz/v1/mindmap/batch_delete
	r.Handle(POST, "batch_delete", BatchDeleteMindMap())
}

func loadCOSService(r *gin.RouterGroup) {
	// 获取OSS临时凭证
	// [POST] /api/biz/v1/cos/sts/credentials
	r.Handle(POST, "sts/credentials", GetOSSCredentials())
}

func loadAiChat(r *gin.RouterGroup) {
	// 基础ai对话
	// [POST] /api/biz/v1/aichat/send_message
	r.Handle(POST, "send_message", SendMessage())

	//新增会话
	// [POST] /api/biz/v1/aichat/save_conversation
	r.Handle(POST, "save_conversation", SaveNewConversation())

	//获取该导图的所有会话
	// [GET] /api/biz/v1/aichat/get_conversation_list?map_id=
	r.Handle(GET, "get_conversation_list", GetConversationList())

	//删除会话
	// [POST] /api/biz/v1/aichat/del_conversation
	r.Handle(POST, "del_conversation", DelConversation())

	//获取某个会话的详细信息
	// [GET] /api/biz/v1/aichat/get_conversation?conversation_id=
	r.Handle(GET, "get_conversation", GetConversation())

	//更新某个会话的标题
	// [POST] /api/biz/v1/aichat/update_conversation_title
	r.Handle(POST, "update_conversation_title", UpdateConversationTitle())

	//生成导图
	// [POST] /api/biz/v1/aichat/generate_mind_map
	// 表单名称 file
	r.Handle(POST, "generate_mind_map", GenerateMindMap())
}
