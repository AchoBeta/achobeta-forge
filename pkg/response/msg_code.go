package response

import (
	"encoding/json"
	"net/http"
	"time"

	"forge/pkg/trace"

	"github.com/gin-gonic/gin"
)

type JsonMsgResponse struct {
	Ctx *gin.Context
}

type JsonMsgResult struct {
	Code    int
	Message string
	Data    interface{}
	// TODO: Code 和 Data.Success 字段功能重复，但为了兼容性暂时保留
	// 建议：未来版本移除顶层 Message 字段，通过 Code 判断成功/失败
}

// Meta 元数据字段
type Meta struct {
	TraceID     string `json:"traceId"`
	RequestTime string `json:"requestTime"`
}

type nilStruct struct{}

const SUCCESS_CODE = 200
const SUCCESS_MSG = "成功"
const ERROR_MSG = "错误"

func NewResponse(c *gin.Context) *JsonMsgResponse {
	return &JsonMsgResponse{Ctx: c}
}

// injectMeta 将元数据注入到响应体
func (r *JsonMsgResponse) injectMeta(res JsonMsgResult) {
	r.injectMetaWithStatus(res, http.StatusOK)
}

// injectMetaWithStatus 将元数据注入到响应体并设置HTTP状态码
func (r *JsonMsgResponse) injectMetaWithStatus(res JsonMsgResult, httpStatus int) {
	// 从 Request Context 获取追踪标识（通过 trace 模块）
	traceID, _ := trace.GetTraceIDFromGin(r.Ctx)
	requestTime, _ := trace.GetRequestTimeFromGin(r.Ctx)

	// 构建元数据
	meta := Meta{
		TraceID:     traceID,
		RequestTime: requestTime.Format(time.RFC3339),
	}

	// 将 response 序列化为 map
	resBytes, err := json.Marshal(res)
	if err != nil {
		r.Ctx.JSON(httpStatus, res)
		return
	}

	var resMap map[string]interface{}
	if err := json.Unmarshal(resBytes, &resMap); err != nil {
		r.Ctx.JSON(httpStatus, res)
		return
	}

	// 注入 meta 字段
	resMap["meta"] = meta

	// 返回注入后的 response
	r.Ctx.JSON(httpStatus, resMap)
}

func (r *JsonMsgResponse) Success(data interface{}) {
	res := JsonMsgResult{}
	res.Code = SUCCESS_CODE
	res.Message = SUCCESS_MSG
	res.Data = data
	r.injectMeta(res)
}

func (r *JsonMsgResponse) Error(mc MsgCode) {
	r.error(mc.Code, mc.Msg)
}

// ErrorWithStatus 返回错误响应并设置HTTP状态码
func (r *JsonMsgResponse) ErrorWithStatus(mc MsgCode, httpStatus int) {
	r.errorWithStatus(mc.Code, mc.Msg, httpStatus)
}

func (r *JsonMsgResponse) error(code int, message string) {
	r.errorWithStatus(code, message, http.StatusOK)
}

func (r *JsonMsgResponse) errorWithStatus(code int, message string, httpStatus int) {
	if message == "" {
		message = ERROR_MSG
	}
	res := JsonMsgResult{}
	res.Code = code
	res.Message = message
	res.Data = nilStruct{}
	r.injectMetaWithStatus(res, httpStatus)
}

// AllInOne 统一处理成功和错误响应，注入元数据
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
