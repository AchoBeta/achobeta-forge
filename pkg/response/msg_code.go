package response

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type JsonMsgResponse struct {
	Ctx *gin.Context
}

type JsonMsgResult struct {
	Code    int
	Message string
	Data    interface{}
}
type nilStruct struct{}

const SUCCESS_CODE = 200
const SUCCESS_MSG = "成功"
const ERROR_MSG = "错误"

func NewResponse(c *gin.Context) *JsonMsgResponse {
	return &JsonMsgResponse{Ctx: c}
}

// injectLogID 将 response 注入 logid
func (r *JsonMsgResponse) injectLogID(res JsonMsgResult) {
	// 从 Gin context 获取 logid
	logID := ""
	if value, exists := r.Ctx.Get("log_id"); exists {
		if id, ok := value.(string); ok {
			logID = id
		}
	}
	
	// 如果没有 logid，直接返回原 response
	if logID == "" {
		r.Ctx.JSON(http.StatusOK, res)
		return
	}

	// 将 response 序列化为 map
	resBytes, err := json.Marshal(res)
	if err != nil {
		// 序列化失败，直接返回原 response
		r.Ctx.JSON(http.StatusOK, res)
		return
	}

	var resMap map[string]interface{}
	if err := json.Unmarshal(resBytes, &resMap); err != nil {
		// 反序列化失败，直接返回原 response
		r.Ctx.JSON(http.StatusOK, res)
		return
	}

	// 注入 log_id
	resMap["log_id"] = logID

	// 返回注入后的 response
	r.Ctx.JSON(http.StatusOK, resMap)
}

func (r *JsonMsgResponse) Success(data interface{}) {
	res := JsonMsgResult{}
	res.Code = SUCCESS_CODE
	res.Message = SUCCESS_MSG
	res.Data = data
	r.injectLogID(res)
}

func (r *JsonMsgResponse) Error(mc MsgCode) {
	r.error(mc.Code, mc.Msg)
}

func (r *JsonMsgResponse) error(code int, message string) {
	if message == "" {
		message = ERROR_MSG
	}
	res := JsonMsgResult{}
	res.Code = code
	res.Message = message
	res.Data = nilStruct{}
	r.injectLogID(res)
}

// AllInOne 统一处理成功和错误响应，注入 logid
func (r *JsonMsgResponse) AllInOne(data interface{}, err error, errorCode int, errorMsg string) {
	if err != nil {
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		r.error(errorCode, errorMsg)
	} else {
		r.Success(data)
	}
}
