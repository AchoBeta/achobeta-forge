package generationservice

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"forge/biz/entity"
	"forge/biz/repo"
	"forge/pkg/log/zlog"
)

// TODO: 阶段2 - Few-Shot自动扩充功能
// Few-Shot生成器：基于高质量种子数据自动生成更多训练样本
// 关键：AI生成的样本 loss_weight=0.1-0.5（质量不如人工标注的1.0）

// FewShotGenerator Few-Shot生成器
type FewShotGenerator struct {
	aiChatRepo  repo.AiChatRepo
	seedManager *SeedManager
	// TODO: 对接 infra/eino 的 AI 客户端
	// einoClient *eino.AiChatClient
}

// NewFewShotGenerator 创建Few-Shot生成器
func NewFewShotGenerator(aiChatRepo repo.AiChatRepo, seedManager *SeedManager) *FewShotGenerator {
	return &FewShotGenerator{
		aiChatRepo:  aiChatRepo,
		seedManager: seedManager,
	}
}

// GenerateFewShotSamples 基于种子数据生成新样本
// inputTexts: 待生成思维导图的文本列表
// shotCount: Few-Shot示例数量（3-5个）
// 返回：成功生成的样本列表
func (f *FewShotGenerator) GenerateFewShotSamples(ctx context.Context,
	inputTexts []string,
	shotCount int) ([]*entity.GenerationResult, error) {

	// 1. 获取高质量种子数据（评分>0.8）
	seeds, err := f.seedManager.GetHighQualitySeeds(ctx, "", 0.8, 20)
	if err != nil {
		return nil, fmt.Errorf("获取种子数据失败: %w", err)
	}

	if len(seeds) < 3 {
		return nil, fmt.Errorf("种子数据不足（至少需要3条，当前%d条）", len(seeds))
	}

	// 2. 选择多样化的示例（3-5个）
	exampleSeeds := f.seedManager.SelectDiverseSeeds(seeds, shotCount)

	// 3. 构建Few-Shot prompt
	fewShotPrompt, err := f.buildFewShotPrompt(ctx, exampleSeeds)
	if err != nil {
		return nil, err
	}

	// 4. 批量生成
	var results []*entity.GenerationResult

	for _, inputText := range inputTexts {
		fullPrompt := fewShotPrompt + fmt.Sprintf("\n\n现在请为以下文本生成思维导图JSON：\n%s", inputText)

		// TODO: 调用AI模型生成
		generatedJSON, err := f.callAIModel(ctx, fullPrompt)
		if err != nil {
			zlog.CtxWarnf(ctx, "AI生成失败: %v", err)
			continue
		}

		// 格式校验（只接受格式正确的）
		metrics, err := ValidateMindMapQuality(generatedJSON)
		if err != nil || metrics.FormatScore == 0 {
			zlog.CtxWarnf(ctx, "生成的JSON格式错误，跳过")
			continue
		}

		result := &entity.GenerationResult{
			MapJSON: generatedJSON,
			// TODO: 设置其他字段
		}
		results = append(results, result)
	}

	return results, nil
}

// buildFewShotPrompt 构建Few-Shot提示词
// 格式：标准System Prompt + 高质量示例
func (f *FewShotGenerator) buildFewShotPrompt(ctx context.Context, seeds []*entity.GenerationResult) (string, error) {
	var examples []string

	for i, seed := range seeds {
		// 获取种子数据对应的对话，提取用户输入
		conversation, err := f.aiChatRepo.GetConversation(ctx, seed.ConversationID, "")
		if err != nil {
			continue
		}

		var userInput string
		for _, msg := range conversation.Messages {
			if msg.Role == entity.USER {
				userInput = msg.Content
				break
			}
		}

		example := fmt.Sprintf("【示例%d】\n输入文本：\n%s\n\n输出JSON：\n%s", i+1, userInput, seed.MapJSON)
		examples = append(examples, example)
	}

	prompt := SFTStandardSystemPrompt + "\n\n以下是高质量示例：\n\n" + strings.Join(examples, "\n\n---\n\n")

	return prompt, nil
}

// callAIModel 调用AI模型生成
// TODO: 对接 infra/eino 的 AI 客户端
func (f *FewShotGenerator) callAIModel(ctx context.Context, prompt string) (string, error) {
	// TODO: 实现AI调用逻辑
	// 1. 构建messages
	// 2. 调用 einoClient.ToolAiClient.Generate
	// 3. 返回生成的JSON
	return "", fmt.Errorf("未实现：需要对接eino AI客户端")
}

// BuildFewShotSFTRecord 构建Few-Shot生成的SFT记录
// 关键：AI生成样本的loss_weight随机0.1-0.5（质量不如人工标注的1.0）
func BuildFewShotSFTRecord(inputText, outputJSON string) (*SFTRecord, error) {
	messages := []SFTMessage{
		{
			Role:    "system",
			Content: SFTStandardSystemPrompt,
		},
		{
			Role:    "user",
			Content: inputText,
		},
		{
			Role:    "assistant",
			Content: outputJSON,
			// AI生成样本：loss_weight随机0.1-0.5
			// 这样在导出时可以通过minLossWeight筛选
			// minLossWeight=1.0 只导出人工标注
			// minLossWeight=0.5 导出人工+部分AI生成
			// minLossWeight=0.0 导出全部
			LossWeight: floatPtr(0.1 + rand.Float64()*0.4),
		},
	}

	return &SFTRecord{
		Messages: messages,
		Thinking: "disabled",
	}, nil
}

func floatPtr(f float64) *float64 {
	return &f
}
