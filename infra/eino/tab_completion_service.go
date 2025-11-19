package eino

import (
	"context"
	"fmt"
	"forge/biz/entity"
	"forge/infra/configs"
	"forge/pkg/log/zlog"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type TabCompletionClient struct {
	ApiKey    string
	ModelName string
	ArkClient *arkruntime.Client
}

func NewTabCompletionClient() *TabCompletionClient {
	config := configs.Config().GetAiChatConfig()

	return &TabCompletionClient{
		ApiKey:    config.TabApiKey,
		ModelName: config.TabModelName,
		ArkClient: arkruntime.NewClientWithApiKey(config.TabApiKey),
	}
}

// TabComplete 处理Tab补全请求
func (t *TabCompletionClient) TabComplete(ctx context.Context, userInput, mapData string, recentMessages []*entity.Message) (string, error) {
	// 构建Tab补全提示词
	systemPrompt := t.buildTabCompletionPrompt(userInput, mapData, recentMessages)

	// 构建消息
	messages := []*arkmodel.ChatCompletionMessage{
		{
			Role: "system",
			Content: &arkmodel.ChatCompletionMessageContent{
				StringValue: &systemPrompt,
			},
		},
		{
			Role: "user",
			Content: &arkmodel.ChatCompletionMessageContent{
				StringValue: &userInput,
			},
		},
	}

	// 构建请求
	request := arkmodel.CreateChatCompletionRequest{
		Model:    t.ModelName,
		Messages: messages,
		Thinking: &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled},
	}

	// 调用API
	resp, err := t.ArkClient.CreateChatCompletion(ctx, request)
	if err != nil {
		zlog.CtxErrorf(ctx, "Tab补全调用失败: %v", err)
		return "", fmt.Errorf("Tab补全调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Tab补全API返回结果为空")
	}

	// 提取返回内容
	choice := resp.Choices[0]
	messageContent := choice.Message.Content
	var contentStr string
	if messageContent.StringValue != nil {
		contentStr = *messageContent.StringValue
	} else if len(messageContent.ListValue) > 0 && messageContent.ListValue[0].Text != "" {
		contentStr = messageContent.ListValue[0].Text
	} else {
		return "", fmt.Errorf("Tab补全API返回内容格式不正确")
	}

	return contentStr, nil
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
