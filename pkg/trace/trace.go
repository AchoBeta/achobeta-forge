package trace

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type traceKey string

const (
	traceIDKey     traceKey = "trace_id"
	requestTimeKey traceKey = "request_time"
)

// SetTraceID 设置 trace_id 到 Request Context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID 从 Request Context 获取 trace_id
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}

// GenerateTraceID 生成新的 trace_id
func GenerateTraceID() string {
	return uuid.New().String()
}

// SetRequestTime 设置请求开始时间
func SetRequestTime(ctx context.Context, requestTime time.Time) context.Context {
	return context.WithValue(ctx, requestTimeKey, requestTime)
}

// GetRequestTime 获取请求开始时间
func GetRequestTime(ctx context.Context) (time.Time, bool) {
	requestTime, ok := ctx.Value(requestTimeKey).(time.Time)
	return requestTime, ok
}

// GetTraceIDFromGin 从 Gin Context 的 Request Context 读取 trace_id
func GetTraceIDFromGin(gCtx *gin.Context) (string, bool) {
	return GetTraceID(gCtx.Request.Context())
}

// GetRequestTimeFromGin 从 Gin Context 的 Request Context 读取 request_time
func GetRequestTimeFromGin(gCtx *gin.Context) (time.Time, bool) {
	return GetRequestTime(gCtx.Request.Context())
}
