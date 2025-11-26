package constant

// 所有常量文件读取位置
const (
	DEFAULT_CONFIG_FILE_PATH = "/conf/config.yaml"
	// LOGID 已删除，使用 trace 模块的 trace_id 替代
)

// Redis Key 常量
const (
	// REDIS_VERIFICATION_CODE_KEY 验证码 Redis key
	REDIS_VERIFICATION_CODE_KEY = "verification_code:%s"
)
