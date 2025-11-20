package loop

import (
	"context"
	"forge/biz/entity"
	"forge/constant"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"os"
	"sync"

	cozeloop "github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

var (
	client    cozeloop.Client
	isEnabled bool
	initOnce  sync.Once
)

// InitCozeLoop 初始化 CozeLoop 客户端（全局单例）
func InitCozeLoop() {
	initOnce.Do(func() {
		// 获取 CozeLoop 配置
		config := configs.Config().GetCozeLoopConfig()

		// 如果未启用，跳过初始化
		if !config.Enable {
			zlog.Infof("CozeLoop is disabled, skipping initialization")
			isEnabled = false
			return
		}

		// 设置环境变量
		os.Setenv("COZELOOP_WORKSPACE_ID", config.WorkspaceID)
		os.Setenv("COZELOOP_API_TOKEN", config.APIToken)

		// 创建 CozeLoop 客户端，打开 prompt trace 上报开关
		_client, err := cozeloop.NewClient(cozeloop.WithPromptTrace(config.PromptTrace))
		if err != nil {
			zlog.Errorf("Failed to initialize CozeLoop client: %v", err)
			isEnabled = false
			return
		}

		client = _client
		isEnabled = true
		zlog.Infof("CozeLoop client initialized successfully with prompt trace: %v", config.PromptTrace)
	})
}

// IsEnabled 检查 CozeLoop 是否已启用
func IsEnabled() bool {
	return isEnabled
}

// GetClient 获取 CozeLoop 客户端
func GetClient() cozeloop.Client {
	return client
}

// StartRootSpan 创建 Root Span（用于 HTTP 请求级别）
func StartRootSpan(ctx context.Context, spanName string) (context.Context, cozeloop.Span) {
	if !isEnabled || client == nil {
		return ctx, nil
	}

	// 使用 Low-Level API 创建 Root Span
	ctx, span := cozeloop.StartSpan(ctx, spanName, "root")

	// 设置业务信息和 Baggage
	if span != nil {
		user, ok := entity.GetUser(ctx)
		if ok {
			span.SetUserIDBaggage(ctx, user.UserID)
		}
		logid, ok := zlog.GetLogId(ctx)
		if ok {
			span.SetTags(ctx, map[string]interface{}{
				"log_id": logid,
			})
		}
	}

	return ctx, span
}

// StartCustomSpan 创建自定义 Span（用于业务逻辑）
func StartCustomSpan(ctx context.Context, spanName string, spanType string) (context.Context, cozeloop.Span) {
	if !isEnabled || client == nil {
		return ctx, nil
	}

	// 使用 Low-Level API 创建自定义 Span
	ctx, span := cozeloop.StartSpan(ctx, spanName, spanType)

	// 设置业务信息
	if span != nil {
		user, ok := entity.GetUser(ctx)
		if ok {
			span.SetUserIDBaggage(ctx, user.UserID)
		}
		logid, ok := zlog.GetLogId(ctx)
		if ok {
			span.SetTags(ctx, map[string]interface{}{
				"log_id": logid,
			})
		}
	}

	return ctx, span
}

// StartModelSpan 创建 Model Span（用于 AI 模型调用）
func StartModelSpan(ctx context.Context, spanName string, modelProvider string, modelName string) (context.Context, cozeloop.Span) {
	if !isEnabled || client == nil {
		return ctx, nil
	}

	// 使用标准的 Model Span 类型
	ctx, span := client.StartSpan(ctx, spanName, tracespec.VModelSpanType)

	if span != nil {
		// 设置模型信息
		span.SetModelProvider(ctx, modelProvider)
		span.SetModelName(ctx, modelName)

		// 设置业务信息
		user, ok := entity.GetUser(ctx)
		if ok {
			span.SetUserIDBaggage(ctx, user.UserID)
		}
		logid, ok := zlog.GetLogId(ctx)
		if ok {
			span.SetTags(ctx, map[string]interface{}{
				"log_id": logid,
			})
		}
	}

	return ctx, span
}

// SetSpanAllInOne 一次性设置 Span 的输入、输出和状态（用于普通业务逻辑）
func SetSpanAllInOne(ctx context.Context, sp cozeloop.Span, input, output any, err error) {
	if sp == nil {
		return
	}
	sp.SetInput(ctx, input)
	sp.SetOutput(ctx, output)
	if err != nil {
		sp.SetError(ctx, err)
		sp.SetStatusCode(ctx, 1)
	} else {
		sp.SetStatusCode(ctx, 0)
	}
	sp.Finish(ctx)
}

// SetModelSpanData 设置 Model Span 的数据（使用标准格式）
func SetModelSpanData(ctx context.Context, sp cozeloop.Span,
	messages []*tracespec.ModelMessage,
	response string,
	inputTokens, outputTokens int64,
	err error) {
	if sp == nil {
		return
	}

	// 设置模型输入（使用标准格式）
	sp.SetInput(ctx, tracespec.ModelInput{
		Messages: messages,
	})

	// 设置模型输出（使用标准格式）
	if response != "" {
		sp.SetOutput(ctx, tracespec.ModelOutput{
			Choices: []*tracespec.ModelChoice{
				{
					Message: &tracespec.ModelMessage{
						Role:    tracespec.VRoleAssistant,
						Content: response,
					},
				},
			},
		})
	}

	// 设置 Token 数量
	if inputTokens > 0 {
		sp.SetInputTokens(ctx, int(inputTokens))
	}
	if outputTokens > 0 {
		sp.SetOutputTokens(ctx, int(outputTokens))
	}

	// 设置状态
	if err != nil {
		sp.SetError(ctx, err)
		sp.SetStatusCode(ctx, 1)
	} else {
		sp.SetStatusCode(ctx, 0)
	}

	sp.Finish(ctx)
}

// SetModelSpanWithPrompt 设置带 Prompt 信息的 Model Span 数据
func SetModelSpanWithPrompt(ctx context.Context, sp cozeloop.Span,
	messages []*tracespec.ModelMessage,
	response string,
	inputTokens, outputTokens int64,
	promptKey string,
	promptVersion string,
	modelCallOptions *tracespec.ModelCallOption,
	startTimeFirstResp int64,
	err error) {
	if sp == nil {
		return
	}

	// 设置模型输入（使用标准格式）
	sp.SetInput(ctx, tracespec.ModelInput{
		Messages: messages,
	})

	// 设置模型输出（使用标准格式）
	if response != "" {
		sp.SetOutput(ctx, tracespec.ModelOutput{
			Choices: []*tracespec.ModelChoice{
				{
					Message: &tracespec.ModelMessage{
						Role:    tracespec.VRoleAssistant,
						Content: response,
					},
				},
			},
		})
	}

	// 关联 PromptKey 和 Version（如果有）
	if promptKey != "" {
		sp.SetTags(ctx, map[string]interface{}{
			"prompt_key":     promptKey,
			"prompt_version": promptVersion,
		})
	}

	// 设置模型调用参数
	if modelCallOptions != nil {
		sp.SetModelCallOptions(ctx, *modelCallOptions)
	}

	// 设置 Token 数量
	if inputTokens > 0 {
		sp.SetInputTokens(ctx, int(inputTokens))
	}
	if outputTokens > 0 {
		sp.SetOutputTokens(ctx, int(outputTokens))
	}

	// 如果使用流式返回，记录首Token返回的微秒时间戳
	if startTimeFirstResp > 0 {
		sp.SetStartTimeFirstResp(ctx, startTimeFirstResp)
	}

	// 设置状态
	if err != nil {
		sp.SetError(ctx, err)
		sp.SetStatusCode(ctx, 1)
	} else {
		sp.SetStatusCode(ctx, 0)
	}

	sp.Finish(ctx)
}

// GetNewSpan 向后兼容函数
func GetNewSpan(ctx context.Context, spanName string, spanType constant.LoopSpanType, opts ...cozeloop.StartSpanOption) (sCtx context.Context, sp cozeloop.Span) {
	switch spanType {
	case constant.LoopSpanType_Root:
		return StartRootSpan(ctx, spanName)
	case constant.LoopSpanType_Function, constant.LoopSpanType_Handle, constant.LoopSpanType_StepCall:
		return StartCustomSpan(ctx, spanName, spanType.String())
	default:
		return StartCustomSpan(ctx, spanName, spanType.String())
	}
}

// 辅助函数

func SetSpanInput(ctx context.Context, sp cozeloop.Span, input any) {
	if sp == nil {
		return
	}
	sp.SetInput(ctx, input)
}

func SetSpanOutput(ctx context.Context, sp cozeloop.Span, output any) {
	if sp == nil {
		return
	}
	sp.SetOutput(ctx, output)
}

func SetSpanFinish(ctx context.Context, sp cozeloop.Span) {
	if sp == nil {
		return
	}
	sp.Finish(ctx)
}

func SetSpanInputWithTags(ctx context.Context, sp cozeloop.Span, input any, tags map[string]interface{}) {
	if sp == nil {
		return
	}
	sp.SetInput(ctx, input)
	sp.SetTags(ctx, tags)
}

// Close 关闭 CozeLoop 客户端
func Close(ctx context.Context) {
	if client != nil {
		client.Close(ctx)
		zlog.Infof("CozeLoop client closed")
	}
}
