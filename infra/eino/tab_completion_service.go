package eino

import (
	"context"
	"fmt"
	"forge/biz/entity"
	"forge/infra/configs"
	"forge/pkg/log/zlog"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type TabCompletionClient struct {
	ApiKey    string
	ModelName string
	ChatModel *ark.ChatModel
}

func NewTabCompletionClient() *TabCompletionClient {
	config := configs.Config().GetAiChatConfig()
	
	// 使用 Eino 框架创建模型客户端
	ctx := context.Background()
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   config.TabApiKey,
		Model:    config.TabModelName,
		Thinking: &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled},
	})
	if err != nil {
		zlog.Errorf("创建Tab补全模型失败: %v", err)
		// 如果创建失败，返回 nil 模型，在调用时会处理
	}

	return &TabCompletionClient{
		ApiKey:    config.TabApiKey,
		ModelName: config.TabModelName,
		ChatModel: chatModel,
	}
}

// TabComplete 处理Tab补全请求 - 使用 Eino 框架确保被 CozeLoop 追踪
func (t *TabCompletionClient) TabComplete(ctx context.Context, userInput, mapData string, recentMessages []*entity.Message) (string, error) {
	if t.ChatModel == nil {
		return "", fmt.Errorf("Tab补全模型未初始化")
	}

	// 构建消息
	systemPrompt := t.buildTabCompletionPrompt(userInput, mapData, recentMessages)
	messages := []*schema.Message{
		{
			Content: systemPrompt,
			Role:    schema.System,
		},
		{
			Content: userInput,
			Role:    schema.User,
		},
	}

	// 使用 Eino 框架调用，确保被 CozeLoop 回调追踪
	resp, err := t.ChatModel.Generate(ctx, messages)
	if err != nil {
		zlog.CtxErrorf(ctx, "Tab补全Eino调用失败: %v", err)
		return "", fmt.Errorf("Tab补全Eino调用失败: %w", err)
	}

	return resp.Content, nil
}

// buildTabCompletionPrompt 构建Tab补全提示词
func (t *TabCompletionClient) buildTabCompletionPrompt(userInput, mapData string, recentMessages []*entity.Message) string {
	// 构建历史对话上下文
	historyContext := ""
	if len(recentMessages) > 0 {
		historyContext = "\n【用户最近提问】\n"
		for _, msg := range recentMessages {
			if msg.Role == entity.USER {
				historyContext += fmt.Sprintf("- %s\n", msg.Content)
			}
		}
	}

	prompt := fmt.Sprintf(`你是一个专门服务于一款思维导图树产品中的提问补全助手，猜测用户下一步输入，帮助用户继续思考和完善导图，而不是直接给出知识答案。

【用户当前输入】
%s

【导图上下文】
%s
%s
请根据以上信息，补全用户当前输入，帮助用户继续思考。提示问题必须完整保留用户已输入的内容，并且以口语化的方式提出，具体且可执行。`, userInput, mapData, historyContext)

	return prompt
}

// TabCompleteWithEino 使用eino架构的Tab补全（备用方案）
func (t *TabCompletionClient) TabCompleteWithEino(ctx context.Context, userInput, mapData string, recentMessages []*entity.Message) (string, error) {
	// 创建eino模型客户端
	tabModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   t.ApiKey,
		Model:    t.ModelName,
		Thinking: &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled},
	})
	if err != nil {
		return "", fmt.Errorf("创建Tab补全模型失败: %w", err)
	}

	// 构建消息
	systemPrompt := t.buildTabCompletionPrompt(userInput, mapData, recentMessages)
	messages := []*schema.Message{
		{
			Content: systemPrompt,
			Role:    schema.System,
		},
		{
			Content: userInput,
			Role:    schema.User,
		},
	}

	// 调用模型
	resp, err := tabModel.Generate(ctx, messages)
	if err != nil {
		zlog.CtxErrorf(ctx, "Tab补全eino调用失败: %v", err)
		return "", fmt.Errorf("Tab补全eino调用失败: %w", err)
	}

	return resp.Content, nil
}
