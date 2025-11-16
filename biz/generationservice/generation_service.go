package generationservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/biz/entity"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/pkg/log/zlog"
	"forge/util"
)

type GenerationService struct {
	generationRepo repo.IGenerationRepo
	aiChatRepo     repo.AiChatRepo
	mindMapRepo    repo.IMindMapRepo
}

func NewGenerationService(generationRepo repo.IGenerationRepo, aiChatRepo repo.AiChatRepo, mindMapRepo repo.IMindMapRepo) types.IGenerationService {
	return &GenerationService{
		generationRepo: generationRepo,
		aiChatRepo:     aiChatRepo,
		mindMapRepo:    mindMapRepo,
	}
}

// GetBatchWithResults 获取批次及其结果
func (g *GenerationService) GetBatchWithResults(ctx context.Context, batchID string) (*entity.GenerationBatch, []*entity.GenerationResult, error) {
	// 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("无法获取用户信息")
	}

	// 获取批次信息
	batch, err := g.generationRepo.GetGenerationBatch(ctx, batchID, user.UserID)
	if err != nil {
		return nil, nil, err
	}

	// 获取结果列表
	results, err := g.generationRepo.GetGenerationResultsByBatchID(ctx, batchID)
	if err != nil {
		return nil, nil, err
	}

	return batch, results, nil
}

// ListUserBatches 获取用户批次列表
func (g *GenerationService) ListUserBatches(ctx context.Context, userID string, page, pageSize int) ([]*entity.GenerationBatch, int64, error) {
	return g.generationRepo.ListUserGenerationBatches(ctx, userID, page, pageSize)
}

// LabelResultWithSave 标记结果并可能保存导图
func (g *GenerationService) LabelResultWithSave(ctx context.Context, resultID string, label int) (*entity.MindMap, error) {
	// 验证标签值
	if label != -1 && label != 0 && label != 1 {
		return nil, fmt.Errorf("标签值必须是-1、0或1")
	}

	// 更新标签
	if err := g.generationRepo.UpdateGenerationResultLabel(ctx, resultID, label); err != nil {
		return nil, err
	}

	// 如果标记为正值(1)，保存导图到正式系统
	if label == 1 {
		mindMap, err := g.SaveSelectedMindMap(ctx, resultID)
		if err != nil {
			zlog.CtxWarnf(ctx, "保存选中导图失败: %v", err)
			// 返回错误，因为用户期望保存成功
			return nil, fmt.Errorf("保存选中导图失败: %w", err)
		}
		return mindMap, nil
	}

	return nil, nil
}

// LabelResult 标记结果（兼容旧接口）
func (g *GenerationService) LabelResult(ctx context.Context, resultID string, label int) error {
	_, err := g.LabelResultWithSave(ctx, resultID, label)
	return err
}

// SFTRecord SFT训练记录结构
type SFTRecord struct {
	Messages []SFTMessage `json:"messages"`
	Thinking string       `json:"thinking"`
}

type SFTMessage struct {
	Role             string   `json:"role"`
	Content          string   `json:"content"`
	LossWeight       *float64 `json:"loss_weight,omitempty"`
	ReasoningContent *string  `json:"reasoning_content,omitempty"`
}

// buildSFTRecord 构建SFT记录
// 当前业务场景：单轮对话（一个user输入 + 一个assistant输出）
// TODO: 多轮对话场景需要按火山要求拆分样本（每轮最后的assistant带reasoning_content）
func (g *GenerationService) buildSFTRecord(conversation *entity.Conversation, label int, dataSource string) (*SFTRecord, error) {
	if len(conversation.Messages) < 2 {
		return nil, fmt.Errorf("对话消息不足")
	}

	var messages []SFTMessage

	// 处理每条消息
	for i, message := range conversation.Messages {
		sftMessage := SFTMessage{
			Role:    strings.ToLower(message.Role),
			Content: message.Content,
		}

		// system和user消息直接使用（因为生成时已经用标准prompt）
		// 不需要替换，保持生成时的prompt一致性

		// 处理最后一条assistant消息
		if sftMessage.Role == "assistant" && i == len(conversation.Messages)-1 {
			// 提取纯JSON
			jsonContent, err := extractMindMapJSONFromAssistant(message.Content)
			if err != nil {
				return nil, fmt.Errorf("提取JSON失败: %w", err)
			}
			sftMessage.Content = jsonContent

			// 根据数据来源设置loss_weight
			// 人工标注(BATCH_GENERATION)：loss_weight=1.0（高质量）
			// AI生成(FEWSHOT_GENERATION)：loss_weight=0.1-0.5（中等质量）
			var lossWeight float64
			if dataSource == "FEWSHOT_GENERATION" {
				// AI生成样本：随机0.1-0.5
				lossWeight = 0.1 + (float64(len(jsonContent)%40) / 100.0) // 伪随机，基于内容长度
			} else {
				// 人工标注样本：固定1.0
				lossWeight = 1.0
			}
			sftMessage.LossWeight = &lossWeight
		}

		messages = append(messages, sftMessage)
	}

	record := &SFTRecord{
		Messages: messages,
		Thinking: "disabled", // 符合火山要求
	}

	return record, nil
}

// TODO：用于多轮的，但是目前生成导图是1-1
// extractReasoningContent 从assistant消息中提取思考过程
func (g *GenerationService) extractReasoningContent(content string) string {
	// SFT生成的内容格式：先输出【思考过程】，再输出【导图JSON】
	// 需要提取【思考过程】部分作为reasoning_content

	// 查找思考过程的标记
	thinkingStart := strings.Index(content, "【思考过程】")
	if thinkingStart == -1 {
		// 尝试其他可能的标记格式
		thinkingStart = strings.Index(content, "思考过程")
		if thinkingStart == -1 {
			return ""
		}
	}

	// 查找JSON开始的位置（通常以{开始）
	jsonStart := strings.Index(content[thinkingStart:], "{")
	if jsonStart == -1 {
		// 如果没找到JSON，返回从思考过程开始的所有内容
		return strings.TrimSpace(content[thinkingStart:])
	}

	// 提取思考过程部分（从标记开始到JSON之前）
	reasoningContent := strings.TrimSpace(content[thinkingStart : thinkingStart+jsonStart])

	// 清理内容，移除标记符号
	reasoningContent = strings.ReplaceAll(reasoningContent, "【思考过程】", "")
	reasoningContent = strings.ReplaceAll(reasoningContent, "【导图JSON】", "")
	reasoningContent = strings.TrimSpace(reasoningContent)

	return reasoningContent
}

// extractMindMapJSONFromAssistant 从assistant消息中提取并校验导图JSON
// 要求：
// 1. 从消息文本中提取第一个完整JSON对象
// 2. JSON中必须包含root字段
// 3. root字段必须能反序列化为entity.MindMapData（确保与后端结构体一致）
func extractMindMapJSONFromAssistant(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("assistant内容为空")
	}

	// 优先尝试整体作为JSON
	jsonStr := content
	if !isValidMindMapJSON(jsonStr) {
		// 从文本中提取第一个完整JSON对象
		jsonStr = extractFirstJSONObjectFromText(content)
		if jsonStr == "" {
			return "", fmt.Errorf("未在assistant内容中找到JSON对象")
		}
		if !isValidMindMapJSON(jsonStr) {
			return "", fmt.Errorf("提取到的JSON不符合导图结构要求")
		}
	}

	return jsonStr, nil
}

// isValidMindMapJSON 校验JSON是否符合 title/layout/root + MindMapData 结构
func isValidMindMapJSON(jsonStr string) bool {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return false
	}

	var aiResponse map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &aiResponse); err != nil {
		return false
	}

	rootData, exists := aiResponse["root"]
	if !exists {
		return false
	}

	rootBytes, err := json.Marshal(rootData)
	if err != nil {
		return false
	}

	var mindMapData entity.MindMapData
	if err := json.Unmarshal(rootBytes, &mindMapData); err != nil {
		return false
	}

	return true
}

// extractFirstJSONObjectFromText 从文本中提取第一个完整的JSON对象
func extractFirstJSONObjectFromText(text string) string {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	braceCount := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		char := text[i]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '"' {
			inString = !inString
			continue
		}
		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					return text[start : i+1]
				}
			}
		}
	}

	if braceCount > 0 {
		return text[start:]
	}
	return ""
}

// ExportDPOData 导出DPO数据
func (g *GenerationService) ExportDPOData(ctx context.Context, startDate, endDate, userID string) (string, error) {
	// 获取已标记的结果（正负样本）
	labeledResults, err := g.generationRepo.GetLabeledResults(ctx, userID, startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("获取已标记结果失败: %w", err)
	}

	if len(labeledResults) == 0 {
		return "", nil
	}

	// 按批次ID分组
	batchGroups := make(map[string][]*entity.GenerationResult)
	for _, result := range labeledResults {
		batchGroups[result.BatchID] = append(batchGroups[result.BatchID], result)
	}

	var dpoRecords []string

	// 为每个批次生成DPO对比对
	for batchID, results := range batchGroups {
		// 分离正负样本（只处理DPO策略生成的数据）
		positiveResults := g.selectPositiveSamples(ctx, results, batchID)
		var negativeResults []*entity.GenerationResult

		for _, result := range results {
			// 只选择DPO策略生成的负样本 (label = -1 且 strategy = 2)
			if result.Label == -1 && result.Strategy != nil && *result.Strategy == 2 {
				negativeResults = append(negativeResults, result)
			}
		}

		// 检查是否有足够的正负样本进行配对
		if len(positiveResults) == 0 {
			zlog.CtxWarnf(ctx, "批次 %s：没有正样本，跳过DPO配对", batchID)
			continue
		}
		if len(negativeResults) == 0 {
			zlog.CtxWarnf(ctx, "批次 %s：没有负样本，跳过DPO配对", batchID)
			continue
		}

		// 智能配对策略：优先配对质量差异明显的样本
		pairs := g.generateOptimalDPOPairs(ctx, positiveResults, negativeResults, batchID)

		for _, pair := range pairs {
			dpoRecord, err := g.buildDPORecord(ctx, pair.positive, pair.negative, userID)
			if err != nil {
				zlog.CtxWarnf(ctx, "构建DPO记录失败 batchID:%s, positive:%s, negative:%s, err:%v",
					batchID, pair.positive.ResultID, pair.negative.ResultID, err)
				continue
			}
			dpoRecords = append(dpoRecords, dpoRecord)
		}

		zlog.CtxInfof(ctx, "批次 %s：生成了 %d 个DPO配对（正样本:%d, 负样本:%d）",
			batchID, len(pairs), len(positiveResults), len(negativeResults))
	}

	return strings.Join(dpoRecords, "\n"), nil
}

// selectPositiveSamples 选择DPO正样本（按策略过滤）
func (g *GenerationService) selectPositiveSamples(ctx context.Context, results []*entity.GenerationResult, batchID string) []*entity.GenerationResult {
	var positiveResults []*entity.GenerationResult

	// 收集DPO策略生成的正样本 (label = 1 且 strategy = 2)
	for _, result := range results {
		if result.Label == 1 && result.Strategy != nil && *result.Strategy == 2 {
			positiveResults = append(positiveResults, result)
		}
	}

	zlog.CtxInfof(ctx, "批次 %s：选择DPO策略正样本 %d 个", batchID, len(positiveResults))
	return positiveResults
}

// selectSFTSamples 选择SFT正样本（按策略过滤）
func (g *GenerationService) selectSFTSamples(ctx context.Context, results []*entity.GenerationResult) []*entity.GenerationResult {
	var positiveResults []*entity.GenerationResult

	// SFT只使用正样本 (label = 1) 且必须是SFT策略生成的数据 (strategy = 1)
	for _, result := range results {
		if result.Label == 1 && result.Strategy != nil && *result.Strategy == 1 {
			positiveResults = append(positiveResults, result)
		}
	}

	zlog.CtxInfof(ctx, "SFT导出：选择SFT策略正样本 %d 个", len(positiveResults))
	return positiveResults
}

// loadBatchesForResults 加载批次信息（用于去重）
func (g *GenerationService) loadBatchesForResults(ctx context.Context, results []*entity.GenerationResult) map[string]*entity.GenerationBatch {
	batches := make(map[string]*entity.GenerationBatch)
	for _, result := range results {
		if _, exists := batches[result.BatchID]; !exists {
			batch, err := g.generationRepo.GetGenerationBatch(ctx, result.BatchID, "")
			if err == nil {
				batches[result.BatchID] = batch
			}
		}
	}
	return batches
}

// DPOPair DPO配对结构
type DPOPair struct {
	positive *entity.GenerationResult
	negative *entity.GenerationResult
}

// generateOptimalDPOPairs 生成最优DPO配对
func (g *GenerationService) generateOptimalDPOPairs(ctx context.Context, positiveResults, negativeResults []*entity.GenerationResult, batchID string) []DPOPair {
	var pairs []DPOPair

	// 配对策略：
	// 1. 每个正样本最多配对3个负样本，避免数据爆炸
	// 2. 优先选择时间相近的样本（同一批次内生成时间相近的样本更具可比性）
	// 3. 如果有策略信息，优先配对不同策略生成的样本

	maxPairsPerPositive := 3
	if len(negativeResults) < 3 {
		maxPairsPerPositive = len(negativeResults) // 如果负样本不足3个，则全部配对
	}

	for _, positive := range positiveResults {
		// 为当前正样本选择最佳的负样本
		selectedNegatives := g.selectBestNegativesForPositive(positive, negativeResults, maxPairsPerPositive)

		for _, negative := range selectedNegatives {
			pairs = append(pairs, DPOPair{
				positive: positive,
				negative: negative,
			})
		}
	}

	zlog.CtxDebugf(ctx, "批次 %s：智能配对完成，正样本 %d 个，负样本 %d 个，生成配对 %d 个",
		batchID, len(positiveResults), len(negativeResults), len(pairs))

	return pairs
}

// selectBestNegativesForPositive 为正样本选择最佳负样本
func (g *GenerationService) selectBestNegativesForPositive(positive *entity.GenerationResult, negativeResults []*entity.GenerationResult, maxCount int) []*entity.GenerationResult {
	if len(negativeResults) <= maxCount {
		return negativeResults // 如果负样本数量不超过限制，全部返回
	}

	// 计算每个负样本与正样本的匹配度分数
	type scoredNegative struct {
		result *entity.GenerationResult
		score  float64
	}

	var scored []scoredNegative

	for _, negative := range negativeResults {
		score := g.calculatePairScore(positive, negative)
		scored = append(scored, scoredNegative{
			result: negative,
			score:  score,
		})
	}

	// 按分数排序，选择最佳的几个
	// 这里使用简单的选择排序，因为数量不大
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 取前maxCount个
	var selected []*entity.GenerationResult
	for i := 0; i < maxCount && i < len(scored); i++ {
		selected = append(selected, scored[i].result)
	}

	return selected
}

// calculatePairScore 计算正负样本配对的匹配度分数
func (g *GenerationService) calculatePairScore(positive, negative *entity.GenerationResult) float64 {
	score := 0.0

	// 1. 时间相近性（同一批次内时间越近越好）
	timeDiff := positive.CreatedAt.Sub(negative.CreatedAt)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	// 时间差越小分数越高，最大1分
	timeScore := 1.0 - float64(timeDiff.Minutes())/60.0 // 假设1小时内的时间差为满分
	if timeScore < 0 {
		timeScore = 0
	}
	score += timeScore

	// 2. 策略差异性（如果有策略信息，不同策略的配对更有价值）
	if positive.Strategy != nil && negative.Strategy != nil {
		if *positive.Strategy != *negative.Strategy {
			score += 1.0 // 不同策略加1分
		}
	}

	// 3. 错误信息差异（有错误的负样本与无错误的正样本配对更有价值）
	if positive.ErrorMessage == nil && negative.ErrorMessage != nil {
		score += 0.5 // 错误差异加0.5分
	}

	return score
}

// buildDPORecord 构建DPO记录
func (g *GenerationService) buildDPORecord(ctx context.Context, positive, negative *entity.GenerationResult, userID string) (string, error) {
	// 获取正样本对话
	positiveConversation, err := g.aiChatRepo.GetConversation(ctx, positive.ConversationID, userID)
	if err != nil {
		return "", fmt.Errorf("获取正样本对话失败: %w", err)
	}

	// 获取负样本对话用于校验
	negativeConversation, err := g.aiChatRepo.GetConversation(ctx, negative.ConversationID, userID)
	if err != nil {
		return "", fmt.Errorf("获取负样本对话失败: %w", err)
	}

	// 校验正负样本的输入一致性（system和user消息应该相同）
	if err := g.validateConversationConsistency(positiveConversation, negativeConversation); err != nil {
		return "", fmt.Errorf("正负样本对话不一致: %w", err)
	}

	// 构建DPO格式记录
	dpoRecord := &DPORecord{
		Messages: make([]DPOMessage, 0, len(positiveConversation.Messages)-1), // 不包含最后一条assistant消息
	}

	// 添加system和user消息
	for _, message := range positiveConversation.Messages {
		if message.Role == entity.ASSISTANT {
			// 跳过assistant消息，最后会特殊处理
			continue
		}
		dpoMessage := DPOMessage{
			Role:    strings.ToLower(message.Role),
			Content: message.Content,
		}
		dpoRecord.Messages = append(dpoRecord.Messages, dpoMessage)
	}

	// 添加最后的assistant消息，包含chosen和rejected
	finalMessage := DPOMessage{
		Role:     "assistant",
		Chosen:   positive.MapJSON,
		Rejected: negative.MapJSON,
	}
	dpoRecord.Messages = append(dpoRecord.Messages, finalMessage)

	// 转换为JSON字符串
	jsonBytes, err := json.Marshal(dpoRecord)
	if err != nil {
		return "", fmt.Errorf("序列化DPO记录失败: %w", err)
	}

	return string(jsonBytes), nil
}

// validateConversationConsistency 校验正负样本对话的输入一致性
func (g *GenerationService) validateConversationConsistency(positive, negative *entity.Conversation) error {
	// 检查消息数量（至少要有system和user消息）
	if len(positive.Messages) < 2 || len(negative.Messages) < 2 {
		return fmt.Errorf("对话消息数量不足")
	}

	// 校验非assistant消息的一致性
	positiveInputs := make([]*entity.Message, 0)
	negativeInputs := make([]*entity.Message, 0)

	for _, msg := range positive.Messages {
		if msg.Role != entity.ASSISTANT {
			positiveInputs = append(positiveInputs, msg)
		}
	}

	for _, msg := range negative.Messages {
		if msg.Role != entity.ASSISTANT {
			negativeInputs = append(negativeInputs, msg)
		}
	}

	// 检查输入消息数量是否一致
	if len(positiveInputs) != len(negativeInputs) {
		return fmt.Errorf("正负样本输入消息数量不一致: positive=%d, negative=%d", len(positiveInputs), len(negativeInputs))
	}

	// 校验每条输入消息的内容是否一致
	for i, posMsg := range positiveInputs {
		negMsg := negativeInputs[i]
		if posMsg.Role != negMsg.Role {
			return fmt.Errorf("第%d条消息角色不一致: positive=%s, negative=%s", i+1, posMsg.Role, negMsg.Role)
		}
		// 对于DPO训练，我们允许system消息有所不同（因为可能使用不同质量的提示词）
		// 但user消息必须完全一致
		if posMsg.Role == entity.USER && posMsg.Content != negMsg.Content {
			return fmt.Errorf("第%d条用户消息内容不一致", i+1)
		}
	}

	return nil
}

// DPORecord DPO训练记录结构
type DPORecord struct {
	Messages []DPOMessage `json:"messages"`
}

type DPOMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content,omitempty"`
	Chosen   string `json:"chosen,omitempty"`
	Rejected string `json:"rejected,omitempty"`
}

// ExportSFTDataToFile 导出SFT数据到文件（支持loss_weight筛选）
// 返回：jsonlData, filename, error
func (g *GenerationService) ExportSFTDataToFile(ctx context.Context, startDate, endDate, userID string, minLossWeight float64) (string, string, error) {
	zlog.CtxInfof(ctx, "开始导出SFT数据到文件: userID=%s, minLossWeight=%.2f", userID, minLossWeight)

	// Step 1: 获取已标记数据
	results, err := g.generationRepo.GetLabeledResults(ctx, userID, startDate, endDate)
	if err != nil {
		return "", "", fmt.Errorf("获取标记数据失败: %w", err)
	}
	zlog.CtxInfof(ctx, "Step1: 原始标记数据 %d 条", len(results))

	if len(results) == 0 {
		return "", "", fmt.Errorf("没有标记数据")
	}

	// Step 2: 策略过滤（只要SFT策略的正样本）
	sftResults := g.selectSFTSamples(ctx, results)
	zlog.CtxInfof(ctx, "Step2: 策略过滤 %d 条 (strategy=1, label=1)", len(sftResults))

	// Step 3: 异常过滤
	cleaned := FilterAnomalies(sftResults)
	zlog.CtxInfof(ctx, "Step3: 异常过滤 %d 条", len(cleaned))

	// Step 4: 去重
	batches := g.loadBatchesForResults(ctx, cleaned)
	deduped := DeduplicateResults(cleaned, batches)
	zlog.CtxInfof(ctx, "Step4: 去重 %d 条", len(deduped))

	// Step 5: 格式校验（格式错误直接丢弃）
	var qualityPassed []*entity.GenerationResult
	for _, result := range deduped {
		metrics, err := ValidateMindMapQuality(result.MapJSON)
		if err != nil || metrics.FormatScore == 0 {
			zlog.CtxWarnf(ctx, "格式错误 resultID=%s: %v", result.ResultID, err)
			continue
		}
		qualityPassed = append(qualityPassed, result)
	}
	zlog.CtxInfof(ctx, "Step5: 格式校验通过 %d 条", len(qualityPassed))

	// TODO: 质量报告功能待实现（统计格式通过率、平均内容分、深度分布等）

	// Step 6: 构建JSONL（支持loss_weight筛选）
	var jsonlLines []string
	filteredCount := 0

	for _, result := range qualityPassed {
		conversation, err := g.aiChatRepo.GetConversation(ctx, result.ConversationID, userID)
		if err != nil {
			zlog.CtxWarnf(ctx, "获取对话失败 conversationID=%s", result.ConversationID)
			continue
		}

		// 根据对话MapID判断数据来源
		// BATCH_GENERATION = 人工标注样本
		// FEWSHOT_GENERATION = AI生成样本
		dataSource := "BATCH_GENERATION" // 默认人工标注
		if conversation.MapID == "FEWSHOT_GENERATION" {
			dataSource = "FEWSHOT_GENERATION"
		}

		record, err := g.buildSFTRecord(conversation, result.Label, dataSource)
		if err != nil {
			zlog.CtxWarnf(ctx, "构建SFT记录失败: %v", err)
			continue
		}

		// 关键优化：支持loss_weight筛选
		// minLossWeight=1.0 -> 只导出人工标注
		// minLossWeight=0.5 -> 导出人工+部分AI生成
		// minLossWeight=0.0 -> 导出全部
		assistantMsg := record.Messages[len(record.Messages)-1]
		if assistantMsg.LossWeight != nil && *assistantMsg.LossWeight < minLossWeight {
			filteredCount++
			zlog.CtxDebugf(ctx, "筛选低权重样本: loss_weight=%.2f < %.2f", *assistantMsg.LossWeight, minLossWeight)
			continue
		}

		jsonBytes, err := json.Marshal(record)
		if err != nil {
			continue
		}

		jsonlLines = append(jsonlLines, string(jsonBytes))
	}

	zlog.CtxInfof(ctx, "导出完成: 总样本%d, 筛选掉%d, 最终%d条", len(qualityPassed), filteredCount, len(jsonlLines))

	jsonlData := strings.Join(jsonlLines, "\n")
	if jsonlData == "" {
		return "", "", fmt.Errorf("没有可导出的数据")
	}

	// 生成文件名（包含筛选信息）
	timestamp := time.Now().Format("20060102_150405")
	var filename string
	if minLossWeight >= 1.0 {
		filename = fmt.Sprintf("SFT_Human_Only_%s_%s.jsonl", userID, timestamp)
	} else if minLossWeight >= 0.5 {
		filename = fmt.Sprintf("SFT_Mixed_%.1f_%s_%s.jsonl", minLossWeight, userID, timestamp)
	} else {
		filename = fmt.Sprintf("SFT_All_%.1f_%s_%s.jsonl", minLossWeight, userID, timestamp)
	}

	// 返回JSONL数据和文件名
	return jsonlData, filename, nil
}

// SaveSelectedMindMap 保存选中的导图到正式系统
func (g *GenerationService) SaveSelectedMindMap(ctx context.Context, resultID string) (*entity.MindMap, error) {
	// 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("无法获取用户信息")
	}

	// 获取生成结果
	result, err := g.generationRepo.GetGenerationResult(ctx, resultID)
	if err != nil {
		return nil, fmt.Errorf("获取生成结果失败: %w", err)
	}

	// 检查结果是否属于当前用户（通过批次验证）
	batch, err := g.generationRepo.GetGenerationBatch(ctx, result.BatchID, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("权限验证失败: %w", err)
	}

	// 生成MapID
	mapID, err := util.GenerateStringID()
	if err != nil {
		return nil, fmt.Errorf("生成导图ID失败: %w", err)
	}

	// 解析AI返回的JSON到临时结构
	var aiResponse map[string]interface{}
	if err := json.Unmarshal([]byte(result.MapJSON), &aiResponse); err != nil {
		zlog.CtxErrorf(ctx, "JSON解析失败，完整JSON内容: %s", result.MapJSON)
		return nil, fmt.Errorf("解析导图JSON失败: %w", err)
	}

	// 提取title和layout
	title := fmt.Sprintf("批量生成-%s", batch.BatchID[:8])
	layout := "mindMap"
	desc := "From batch generation"

	if aiTitle, ok := aiResponse["title"].(string); ok && aiTitle != "" {
		title = aiTitle
	}
	if aiLayout, ok := aiResponse["layout"].(string); ok && aiLayout != "" {
		layout = aiLayout
	}

	// 提取root数据并转换为MindMapData
	rootData, exists := aiResponse["root"]
	if !exists {
		return nil, fmt.Errorf("JSON中缺少root字段")
	}

	rootBytes, err := json.Marshal(rootData)
	if err != nil {
		return nil, fmt.Errorf("序列化root数据失败: %w", err)
	}
	var mindMapData entity.MindMapData
	if err := json.Unmarshal(rootBytes, &mindMapData); err != nil {
		return nil, fmt.Errorf("解析root数据失败: %w", err)
	}

	// 创建MindMap实体（模仿原有CreateMindMap实现）
	mindMap := &entity.MindMap{
		MapID:  mapID,
		UserID: user.UserID,
		Title:  title,
		Desc:   desc,
		Layout: layout,
		Data:   mindMapData, // 直接使用解析好的结构化数据
	}

	// 保存到正式导图系统
	if err := g.mindMapRepo.CreateMindMap(ctx, mindMap); err != nil {
		return nil, fmt.Errorf("保存导图失败: %w", err)
	}

	zlog.CtxInfof(ctx, "成功保存选中导图: mapID=%s, resultID=%s", mindMap.MapID, resultID)

	return mindMap, nil
}

// SaveGenerationBatch 保存批次和结果（事务操作）
func (g *GenerationService) SaveGenerationBatch(ctx context.Context, batch *entity.GenerationBatch, results []*entity.GenerationResult, conversations []*entity.Conversation) error {
	return g.generationRepo.SaveGenerationBatch(ctx, batch, results, conversations)
}
