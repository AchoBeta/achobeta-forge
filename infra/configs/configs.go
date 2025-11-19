package configs

import (
	"flag"
	"forge/constant"
	"forge/pkg/log/zlog"
	"time"

	"github.com/spf13/viper"
)

type IConfig interface {
	GetRedisConfig() RedisConfig
	GetDBConfig() DBConfig
	GetAppConfig() ApplicationConfig
	GetLoggerConfig() LoggerConfig
	GetJWTConfig() JWTConfig
	GetSnowflakeConfig() SnowflakeConfig
	GetSMTPConfig() SMTPConfig // SMTP服务 发送邮件
	GetCOSConfig() COSConfig
	GetAiChatConfig() AiChatConfig
	GetSMSConfig() SMSConfig
	GetUniOfficeConfig() UniOfficeConfig
	GetOAuthConfig() OAuthConfig // OAuth 第三方登录配置
}

var (
	conf = new(config)
)

func Config() IConfig {
	return conf
}
func MustInit(path string) {
	mustInit(path)
}

func (c *config) GetRedisConfig() RedisConfig {
	return c.RedisConfig

}

func (c *config) GetDBConfig() DBConfig {
	return c.DBConfig
}

func (c *config) GetAppConfig() ApplicationConfig {
	return c.AppConfig
}

func (c *config) GetLoggerConfig() LoggerConfig {
	return c.LogConfig
}

// jwt配置读取
func (c *config) GetJWTConfig() JWTConfig {
	return c.JWTConfig
}

// snowflake配置读取
func (c *config) GetSnowflakeConfig() SnowflakeConfig {
	return c.SnowflakeConfig
}

// smtp配置读取
func (c *config) GetSMTPConfig() SMTPConfig {
	return c.SMTPConfig
}

// cos配置读取
func (c *config) GetCOSConfig() COSConfig {
	return c.COSConfig
}

// ai模型配置读取
func (c *config) GetAiChatConfig() AiChatConfig { return c.AiChatConfig }

// sms配置读取
func (c *config) GetSMSConfig() SMSConfig { return c.SMSConfig }

// oauth配置读取
func (c *config) GetOAuthConfig() OAuthConfig { return c.OAuthConfig }

func mustInit(path string) *config {
	// 初始化时间为东八区的时间
	var cstZone = time.FixedZone("CST", 8*3600) // 东八
	time.Local = cstZone

	// 默认配置文件路径
	var configPath string
	flag.StringVar(&configPath, "c", path+constant.DEFAULT_CONFIG_FILE_PATH, "配置文件绝对路径或相对路径")
	flag.Parse()
	zlog.Infof("配置文件路径为 %s", configPath)
	// 初始化配置文件
	viper.SetConfigFile(configPath)
	viper.WatchConfig()
	// 观察配置文件变动
	//viper.OnConfigChange(func(in fsnotify.Event) {
	//	zlog.Warnf("配置文件发生变化")
	//	if err := viper.Unmarshal(&configs.Conf); err != nil {
	//		zlog.Errorf("无法反序列化配置文件 %v", err)
	//	}
	//	zlog.Debugf("%+v", configs.Conf)
	//
	//	Eve()
	//	Init()
	//})
	// 将配置文件读入 viper
	if err := viper.ReadInConfig(); err != nil {
		zlog.Panicf("无法读取配置文件 err: %v", err)
	}
	_config := config{}
	// 解析到变量中
	if err := viper.Unmarshal(&_config); err != nil {
		zlog.Panicf("无法解析配置文件 err: %v", err)
	}
	zlog.Debugf("配置文件为 ： %+v", _config)
	conf = &_config
	return conf

}

func (c *config) GetUniOfficeConfig() UniOfficeConfig { return c.UniOfficeConfig }

type config struct {
	AppConfig       ApplicationConfig `mapstructure:"app"`
	LogConfig       LoggerConfig      `mapstructure:"log"`
	DBConfig        DBConfig          `mapstructure:"database"`
	RedisConfig     RedisConfig       `mapstructure:"redis"`
	JWTConfig       JWTConfig         `mapstructure:"jwt"`
	SnowflakeConfig SnowflakeConfig   `mapstructure:"snowflake"`
	SMTPConfig      SMTPConfig        `mapstructure:"smtp"`
	COSConfig       COSConfig         `mapstructure:"cos"`
	AiChatConfig    AiChatConfig      `mapstructure:"ai_client"`
	SMSConfig       SMSConfig         `mapstructure:"sms"`
	UniOfficeConfig UniOfficeConfig   `mapstructure:"unioffice"`
	OAuthConfig     OAuthConfig       `mapstructure:"oauth"`
}

type ApplicationConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	Env                string `mapstructure:"env"`
	LogfilePath        string `mapstructure:"logfilePath"`
	YourFrontendDomain string `mapstructure:"your_frontend_domain"`
}
type LoggerConfig struct {
	Level    int8   `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Director string `mapstructure:"director"`
	ShowLine bool   `mapstructure:"show-line"`
}

type DBConfig struct {
	Driver      string `mapstructure:"driver"`
	AutoMigrate bool   `mapstructure:"migrate"`
	Dsn         string `mapstructure:"dsn"`
}
type RedisConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	host string `mapstructure:"host"`
	port int    `mapstructure:"port"`
}

type JWTConfig struct {
	SecretKey   string `mapstructure:"secret_key"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type SnowflakeConfig struct {
	NodeID int64 `mapstructure:"node_id"`
}

// SMTP配置
type SMTPConfig struct {
	SmtpHost    string `mapstructure:"smtp_host"`
	SmtpPort    int    `mapstructure:"smtp_port"`
	SmtpUser    string `mapstructure:"smtp_user"`
	SmtpPass    string `mapstructure:"smtp_pass"`
	EncodedName string `mapstructure:"encoded_name"`
}

type COSConfig struct {
	SecretID    string `mapstructure:"secret_id"`
	SecretKey   string `mapstructure:"secret_key"`
	Region      string `mapstructure:"region"`
	Bucket      string `mapstructure:"bucket"`
	AppID       string `mapstructure:"app_id"`
	BaseURL     string `mapstructure:"base_url"`
	STSDuration int64  `mapstructure:"sts_duration"`
}

type AiChatConfig struct {
	ApiKey               string `mapstructure:"api_key"`
	ModelName            string `mapstructure:"model_name"`
	SystemPrompt         string `mapstructure:"system_prompt"`
	UpdateSystemPrompt   string `mapstructure:"update_system_prompt"`
	GenerateSystemPrompt string `mapstructure:"generate_system_prompt"`
}

type SMSConfig struct {
	Key      string `mapstructure:"key"`
	Endpoint string `mapstructure:"endpoint"`
}

type UniOfficeConfig struct {
	MeteredKey string `mapstructure:"metered_key"`
}

// OAuthConfig OAuth 第三方登录配置
type OAuthConfig struct {
	GitHubClientID     string `mapstructure:"github_client_id"`
	GitHubClientSecret string `mapstructure:"github_client_secret"`
	GitHubCallbackURL  string `mapstructure:"github_callback_url"`
	WechatAppID        string `mapstructure:"wechat_app_id"`
	WechatAppSecret    string `mapstructure:"wechat_app_secret"`
	WechatCallbackURL  string `mapstructure:"wechat_callback_url"`
	SessionSecret      string `mapstructure:"session_secret"`
}
