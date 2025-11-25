package response

type MsgCode struct {
	Code int
	Msg  string
	Err  error
}

func (c *MsgCode) WithErr(err error) *MsgCode {
	c.Err = err
	return c
}

var (
	/* 成功 */
	SUCCESS = MsgCode{Code: 200, Msg: "成功"}

	/* 默认失败 */
	COMMON_FAIL = MsgCode{Code: -4396, Msg: "失败"}

	/* 请求错误 <0 */
	TOKEN_IS_EXPIRED = MsgCode{Code: -2, Msg: "token已过期"}

	/* 内部错误 600 ~ 999 */
	INTERNAL_ERROR             = MsgCode{Code: 601, Msg: "内部错误, check log"}
	INTERNAL_FILE_UPLOAD_ERROR = MsgCode{Code: 602, Msg: "文件上传失败"}
	/* 参数错误：1000 ~ 1999 */
	PARAM_NOT_VALID    = MsgCode{Code: 1001, Msg: "参数无效"}
	PARAM_IS_BLANK     = MsgCode{Code: 1002, Msg: "参数为空"}
	PARAM_TYPE_ERROR   = MsgCode{Code: 1003, Msg: "参数类型错误"}
	PARAM_NOT_COMPLETE = MsgCode{Code: 1004, Msg: "参数缺失"}
	INVALID_PARAMS     = MsgCode{Code: 1005, Msg: "请求体无效"}

	PARAM_FILE_SIZE_TOO_BIG = MsgCode{Code: 1010, Msg: "文件过大"}

	/* 用户错误 2000 ~ 2999 */
	USER_NOT_LOGIN             = MsgCode{Code: 2001, Msg: "用户未登录"}
	USER_PASSWORD_DIFFERENT    = MsgCode{Code: 2002, Msg: "用户两次密码输入不一致"}
	USER_ACCOUNT_NOT_EXIST     = MsgCode{Code: 2003, Msg: "账号不存在"}
	USER_CREDENTIALS_ERROR     = MsgCode{Code: 2004, Msg: "密码错误"}
	USER_ACCOUNT_ALREADY_EXIST = MsgCode{Code: 2008, Msg: "账号已存在"}
	ACCOUNT_ALREADY_IN_USE     = MsgCode{Code: 2009, Msg: "该账号已被使用"}
	// EMAIL_ALREADY_IN_USE 已废弃，请使用 ACCOUNT_ALREADY_IN_USE（保持向后兼容）
	EMAIL_ALREADY_IN_USE    = ACCOUNT_ALREADY_IN_USE
	PASSWORD_REQUIRED       = MsgCode{Code: 2010, Msg: "密码必填"}
	ACCOUNT_LAST_CONTACT    = MsgCode{Code: 2011, Msg: "无法解绑唯一联系方式"}
	CAPTCHA_ERROR           = MsgCode{Code: 2100, Msg: "验证码错误"}
	INSUFFICENT_PERMISSIONS = MsgCode{Code: 2200, Msg: "权限不足"}

	/* 思维导图错误 3000 ~ 3999 */
	MINDMAP_NOT_FOUND         = MsgCode{Code: 3001, Msg: "思维导图不存在"}
	MINDMAP_ALREADY_EXISTS    = MsgCode{Code: 3002, Msg: "思维导图已存在"}
	MINDMAP_PERMISSION_DENIED = MsgCode{Code: 3003, Msg: "思维导图权限不足"}

	/* COS错误 4000 ~ 4999 */
	COS_INVALID_RESOURCE_PATH  = MsgCode{Code: 4001, Msg: "无效的资源路径"}
	COS_INVALID_DURATION       = MsgCode{Code: 4002, Msg: "无效的有效期"}
	COS_GET_CREDENTIALS_FAILED = MsgCode{Code: 4003, Msg: "获取COS凭证失败"}
	COS_PERMISSION_DENIED      = MsgCode{Code: 4004, Msg: "COS权限不足"}

	/* ai对话错误 5000~5999 */

	INVALID_CONTENT_TYPE        = MsgCode{Code: 5000, Msg: "只接受 application/json 或 multipart/form-data"}
	AI_CHAT_MESSAGE_MAX         = MsgCode{Code: 5001, Msg: "会话长度已达上限，请开启新的会话"}
	CONVERSATION_ID_NOT_NULL    = MsgCode{Code: 5200, Msg: "会话ID不能为空"}
	USER_ID_NOT_NULL            = MsgCode{Code: 5201, Msg: "用户ID不能为空"}
	MAP_ID_NOT_NULL             = MsgCode{Code: 5202, Msg: "导图ID不能为空"}
	CONVERSATION_TITLE_NOT_NULL = MsgCode{Code: 5203, Msg: "会话标题不能为空"}
	CONVERSATION_NOT_EXIST      = MsgCode{Code: 5204, Msg: "该会话不存在"}
	AI_CHAT_PERMISSION_DENIED   = MsgCode{Code: 5205, Msg: "会话权限不足"}
	MIND_MAP_NOT_EXIST          = MsgCode{Code: 5206, Msg: "该导图不存在"}

	/* 限流错误 */
	TOO_MANY_REQUESTS = MsgCode{Code: 429, Msg: "请求过于频繁，请稍后再试"}
)
