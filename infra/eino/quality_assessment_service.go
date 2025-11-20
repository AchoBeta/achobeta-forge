package eino

import (
	"context"
	"fmt"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"strconv"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type QualityAssessmentClient struct {
	ApiKey    string
	ModelName string
	ChatModel *ark.ChatModel
}

func NewQualityAssessmentClient() *QualityAssessmentClient {
	config := configs.Config().GetAiChatConfig()
	
	// 使用 Eino 框架创建模型客户端
	ctx := context.Background()
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   config.QualityApiKey,
		Model:    config.QualityModelName,
		Thinking: &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled},
	})
	if err != nil {
		zlog.Errorf("创建质量评估模型失败: %v", err)
		// 如果创建失败，返回 nil 模型，在调用时会处理
	}

	return &QualityAssessmentClient{
		ApiKey:    config.QualityApiKey,
		ModelName: config.QualityModelName,
		ChatModel: chatModel,
	}
}

// AssessQuality 评估用户输入的质量 - 使用 Eino 框架确保被 CozeLoop 追踪
func (q *QualityAssessmentClient) AssessQuality(ctx context.Context, userInput, mapData string) (int, error) {
	if q.ChatModel == nil {
		return 0, fmt.Errorf("质量评估模型未初始化")
	}

	// 构建消息
	systemPrompt := q.buildQualityAssessmentPrompt(userInput, mapData)
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
	resp, err := q.ChatModel.Generate(ctx, messages)
	if err != nil {
		zlog.CtxErrorf(ctx, "质量评估Eino调用失败: %v", err)
		return 0, fmt.Errorf("质量评估Eino调用失败: %w", err)
	}

	// 解析质量评分
	return q.parseQualityScore(resp.Content)
}

// buildQualityAssessmentPrompt 构建质量评估提示词
func (q *QualityAssessmentClient) buildQualityAssessmentPrompt(userInput, mapData string) string {
	prompt := fmt.Sprintf(`你是一个专业的思维导图问答质量评估助手。你的任务是判断用户输入的问题是否有质量。高质量的问题应该与导图相关，具体且有思考价值，而不是闲聊、脏话或与导图无关的内容。

【用户输入】
%s

【导图上下文】
%s

请根据以上信息，判断用户输入的问题是否有质量。如果有质量，输出 1；否则，输出 0。注意你只能输出1或者0。`, userInput, mapData)

	return prompt
}

// parseQualityScore 解析质量评分结果
func (q *QualityAssessmentClient) parseQualityScore(response string) (int, error) {
	// 清理响应内容
	response = strings.TrimSpace(response)

	// 尝试直接解析数字
	if score, err := strconv.Atoi(response); err == nil {
		if score == 0 || score == 1 {
			return score, nil
		}
		// 如果是其他数字，转换为0或1
		if score > 0 {
			return 1, nil
		}
		return 0, nil
	}

	// 如果不是纯数字，查找包含的数字
	if strings.Contains(response, "1") {
		return 1, nil
	}
	if strings.Contains(response, "0") {
		return 0, nil
	}

	// 基于关键词判断
	response = strings.ToLower(response)
	if strings.Contains(response, "有质量") || strings.Contains(response, "高质量") ||
		strings.Contains(response, "quality") || strings.Contains(response, "good") {
		return 1, nil
	}

	// 默认返回低质量
	zlog.Warnf("无法解析质量评估结果: %s，默认返回0", response)
	return 0, nil
}

// AssessQualityWithEino 使用eino架构的质量评估（备用方案）
func (q *QualityAssessmentClient) AssessQualityWithEino(ctx context.Context, userInput, mapData string) (int, error) {
	// 创建eino模型客户端
	qualityModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   q.ApiKey,
		Model:    q.ModelName,
		Thinking: &arkmodel.Thinking{Type: arkmodel.ThinkingTypeDisabled},
	})
	if err != nil {
		return 0, fmt.Errorf("创建质量评估模型失败: %w", err)
	}

	// 构建消息
	systemPrompt := q.buildQualityAssessmentPrompt(userInput, mapData)
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
	resp, err := qualityModel.Generate(ctx, messages)
	if err != nil {
		zlog.CtxErrorf(ctx, "质量评估eino调用失败: %v", err)
		return 0, fmt.Errorf("质量评估eino调用失败: %w", err)
	}

	// 解析质量评分
	return q.parseQualityScore(resp.Content)
}
