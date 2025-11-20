package middleware

import (
	"bytes"
	"fmt"
	"forge/constant"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
	"io"

	cozeloop "github.com/coze-dev/cozeloop-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// responseBodyWriter 用于捕获响应体
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// AddTracer
//
//	@Description: add traced in logger and CozeLoop tracing with full request/response body
//	@return app.HandlerFunc
func AddTracer() gin.HandlerFunc {
	return func(gCtx *gin.Context) {
		// Trace ID 存在于 HTTP Header "X-Request-ID" 中 nginx会注入，如果不注入自己注入一个
		logID := gCtx.Request.Header.Get("X-Request-ID")
		if logID == "" {
			logID = uuid.New().String()
			gCtx.Request.Header.Set("X-Request-ID", logID)
		}

		// 增加Logid
		ctx := gCtx.Request.Context()
		ctx = zlog.WithLogKey(ctx, zap.String(constant.LOGID, logID))

		// 读取请求体
		var requestBody string
		if gCtx.Request.Body != nil {
			bodyBytes, err := io.ReadAll(gCtx.Request.Body)
			if err == nil {
				requestBody = string(bodyBytes)
				// 重新设置请求体，以便后续处理器可以读取
				gCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 创建响应体捕获器
		responseBuffer := &bytes.Buffer{}
		responseWriter := &responseBodyWriter{
			ResponseWriter: gCtx.Writer,
			body:           responseBuffer,
		}
		gCtx.Writer = responseWriter

		// 集成 CozeLoop 链路追踪 - 创建 Root Span
		var span cozeloop.Span
		if loop.IsEnabled() {
			// 构建 span 名称：HTTP方法 + 路径
			spanName := gCtx.Request.Method + " " + gCtx.FullPath()
			if spanName == " " { // 如果路径为空，使用原始路径
				spanName = gCtx.Request.Method + " " + gCtx.Request.URL.Path
			}

			// 创建 Root Span
			ctx, span = loop.StartRootSpan(ctx, spanName)

			// 记录完整的请求信息
			if span != nil {
				// 设置请求输入
				span.SetInput(ctx, requestBody)

				// 设置标签和元数据
				tags := map[string]interface{}{
					"http.method":       gCtx.Request.Method,
					"http.path":         gCtx.Request.URL.Path,
					"http.query":        gCtx.Request.URL.RawQuery,
					"http.user_agent":   gCtx.Request.UserAgent(),
					"http.remote_addr":  gCtx.ClientIP(),
					"http.content_type": gCtx.Request.Header.Get("Content-Type"),
					"request_id":        logID,
				}
				span.SetTags(ctx, tags)
			}
		}

		gCtx.Request = gCtx.Request.WithContext(ctx)

		// 执行后续处理器
		gCtx.Next()

		// 请求完成后记录完整的响应信息
		if loop.IsEnabled() && span != nil {
			statusCode := gCtx.Writer.Status()
			responseBody := responseBuffer.String()

			// 设置响应输出
			span.SetOutput(ctx, responseBody)

			// 设置响应相关标签
			responseTags := map[string]interface{}{
				"http.status_code":   statusCode,
				"http.response_size": gCtx.Writer.Size(),
				"http.response_type": gCtx.Writer.Header().Get("Content-Type"),
			}
			span.SetTags(ctx, responseTags)

			// 设置状态和错误
			if statusCode >= 400 {
				err := fmt.Errorf("HTTP %d", statusCode)
				span.SetError(ctx, err)
				span.SetStatusCode(ctx, 1)
			} else {
				span.SetStatusCode(ctx, 0)
			}

			// 完成 span
			span.Finish(ctx)
		}
	}
}
