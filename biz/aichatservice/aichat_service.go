package aichatservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/infra/eino"
	"forge/pkg/log/zlog"
	"forge/pkg/queue"
	"forge/util"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	CONVERSATION_ID_NOT_NULL    = errors.New("会话ID不能为空")
	USER_ID_NOT_NULL            = errors.New("用户ID不能为空")
	MAP_ID_NOT_NULL             = errors.New("导图ID不能为空")
	CONVERSATION_TITLE_NOT_NULL = errors.New("会话标题不能为空")
	CONVERSATION_NOT_EXIST      = errors.New("该会话不存在")
	AI_CHAT_PERMISSION_DENIED   = errors.New("会话权限不足")
	MIND_MAP_NOT_EXIST          = errors.New("该导图不存在")
)

type AiChatService struct {
	aiChatRepo          repo.AiChatRepo
	einoServer          repo.EinoServer
	tabCompletionClient *eino.TabCompletionClient
	qualityClient       *eino.QualityAssessmentClient
}

func NewAiChatService(aiChatRepo repo.AiChatRepo, einoServer repo.EinoServer) *AiChatService {
	return &AiChatService{
		aiChatRepo:          aiChatRepo,
		einoServer:          einoServer,
		tabCompletionClient: eino.NewTabCompletionClient(),
		qualityClient:       eino.NewQualityAssessmentClient(),
	}
}

func (a *AiChatService) ProcessUserMessage(ctx context.Context, req *types.ProcessUserMessageParams) (types.AgentResponse, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return types.AgentResponse{}, AI_CHAT_PERMISSION_DENIED
	}

	conversation, err := a.aiChatRepo.GetConversation(ctx, req.ConversationID, user.UserID)
	if err != nil {
		return types.AgentResponse{}, err
	}
	//将数据写入ctx
	ctx = entity.WithConversation(ctx, conversation)

	//更新导图数据
	conversation.UpdateMapData(req.MapData)
	//更新导图提示词
	conversation.ProcessSystemPrompt()

	//添加用户聊天记录
	userMessage := conversation.AddMessage(req.Message, entity.USER, "", nil)

	//调用ai 返回ai消息
	aiMsg, err := a.einoServer.SendMessage(ctx, conversation.Messages)
	if err != nil {
		return types.AgentResponse{}, err
	}

	//添加ai消息
	conversation.AddMessage(aiMsg.Content, entity.ASSISTANT, "", aiMsg.ToolCalls)
	if aiMsg.NewMapJson != "" {
		conversation.AddMessage(aiMsg.NewMapJson, entity.TOOL, aiMsg.ToolCallID, nil)
	}

	//更新会话聊天记录
	err = a.aiChatRepo.UpdateConversationMessage(ctx, conversation)
	if err != nil {
		return types.AgentResponse{}, err
	}

	// 只对真实用户对话进行质量评估，排除SFT训练数据
	// 重要：在保存对话后再入队，确保消息已存在于数据库中
	if conversation.IsRealUserConversation() {
		// 将用户消息加入质量评估队列（异步处理，不影响聊天响应）
		go func() {
			// 稍微延迟确保数据库事务完全提交
			time.Sleep(100 * time.Millisecond)

			task := &entity.QualityAssessmentTask{
				MessageID:      userMessage.ID,
				MessageContent: userMessage.Content,
				ConversationID: req.ConversationID,
				MapData:        req.MapData,
			}

			qualityQueue := queue.GetQualityQueue()
			if qualityQueue != nil {
				if err := qualityQueue.EnqueueTask(task); err != nil {
					zlog.Warnf("将用户消息加入质量评估队列失败: %v", err)
				}
			}
		}()
	}

	return aiMsg, nil
}

func (a *AiChatService) SaveNewConversation(ctx context.Context, req *types.SaveNewConversationParams) (string, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return "", AI_CHAT_PERMISSION_DENIED
	}

	conversation, err := entity.NewConversation(user.UserID, req.MapID, req.Title, req.MapData)
	if err != nil {
		return "", err
	}
	//初始化系统提示词
	conversation.ProcessSystemPrompt()

	err = a.aiChatRepo.SaveConversation(ctx, conversation)
	if err != nil {
		return "", err
	}
	return conversation.ConversationID, nil
}

func (a *AiChatService) GetConversationList(ctx context.Context, req *types.GetConversationListParams) ([]*entity.Conversation, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return nil, AI_CHAT_PERMISSION_DENIED
	}

	conversationList, err := a.aiChatRepo.GetMapAllConversation(ctx, req.MapID, user.UserID)
	if err != nil {
		return nil, err
	}

	return conversationList, nil
}

func (a *AiChatService) DelConversation(ctx context.Context, req *types.DelConversationParams) error {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return AI_CHAT_PERMISSION_DENIED
	}

	err := a.aiChatRepo.DeleteConversation(ctx, req.ConversationID, user.UserID)
	if err != nil {
		return err
	}

	return nil
}

func (a *AiChatService) GetConversation(ctx context.Context, req *types.GetConversationParams) (*entity.Conversation, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return nil, AI_CHAT_PERMISSION_DENIED
	}

	conversation, err := a.aiChatRepo.GetConversation(ctx, req.ConversationID, user.UserID)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

func (a *AiChatService) UpdateConversationTitle(ctx context.Context, req *types.UpdateConversationTitleParams) error {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return AI_CHAT_PERMISSION_DENIED
	}

	conversation, err := a.aiChatRepo.GetConversation(ctx, req.ConversationID, user.UserID)
	if err != nil {
		return err
	}

	conversation.UpdateTitle(req.Title)

	err = a.aiChatRepo.UpdateConversationTitle(ctx, conversation)
	if err != nil {
		return err
	}
	return nil
}

func (a *AiChatService) GenerateMindMap(ctx context.Context, req *types.GenerateMindMapParams) (string, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return "", AI_CHAT_PERMISSION_DENIED
	}

	if req.File == nil {
		resp, err := a.einoServer.GenerateMindMap(ctx, req.Text, user.UserID)
		if err != nil {
			return "", err
		}
		return resp, nil
	} else {
		text, err := util.ParseFile(ctx, req.File)
		if err != nil {
			return "", err
		}

		resp, err := a.einoServer.GenerateMindMap(ctx, text, user.UserID)

		if err != nil {
			return "", err
		}
		return resp, nil
	}
}

// ProcessTabCompletion 处理Tab补全请求
func (a *AiChatService) ProcessTabCompletion(ctx context.Context, req *types.TabCompletionParams) (string, error) {
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return "", AI_CHAT_PERMISSION_DENIED
	}

	// 获取对话信息以获取最近的消息历史
	conversation, err := a.aiChatRepo.GetConversation(ctx, req.ConversationID, user.UserID)
	if err != nil {
		return "", err
	}

	// 只提取最近的一条用户消息作为历史上下文
	var recentMessages []*entity.Message
	for i := len(conversation.Messages) - 1; i >= 0; i-- {
		if conversation.Messages[i].Role == entity.USER {
			recentMessages = []*entity.Message{conversation.Messages[i]}
			break
		}
	}

	// 调用Tab补全客户端
	completedText, err := a.tabCompletionClient.TabComplete(ctx, req.UserInput, req.MapData, recentMessages)
	if err != nil {
		zlog.CtxErrorf(ctx, "Tab补全失败: %v", err)
		return "", err
	}

	return completedText, nil
}

// ExportQualityConversations 导出高质量对话数据为JSONL格式
// 复用现有SFT导出的架构模式
// 注意：只导出真实用户的高质量对话，不包含SFT训练数据
func (a *AiChatService) ExportQualityConversations(ctx context.Context, req *types.ExportQualityDataParams) (string, int, error) {
	// 获取高质量对话数据（已在存储层过滤SFT数据，只获取真实用户对话）
	conversations, err := a.aiChatRepo.GetQualityConversations(ctx, req.StartDate, req.EndDate, req.Limit)
	if err != nil {
		return "", 0, fmt.Errorf("获取质量对话数据失败: %w", err)
	}

	zlog.CtxInfof(ctx, "准备导出Tab补全训练数据，共 %d 个真实用户对话", len(conversations))

	// 导出为JSONL格式
	return a.buildTabCompletionJSONL(conversations)
}

// buildTabCompletionJSONL 构建Tab补全训练的JSONL数据
// 遵循现有SFT导出的架构模式
func (a *AiChatService) buildTabCompletionJSONL(conversations []*entity.Conversation) (string, int, error) {
	var jsonlLines []string
	totalRecords := 0

	for _, conversation := range conversations {
		// 双重安全检查：确保不是SFT训练数据
		if !conversation.IsRealUserConversation() {
			zlog.Warnf("跳过SFT训练数据，会话ID: %s, MapID: %s", conversation.ConversationID, conversation.MapID)
			continue
		}

		// 提取导图数据
		mapData := conversation.MapData
		if mapData == "" {
			mapData = "{}" // 默认空导图
		}

		// 处理每条高质量的用户消息
		for _, message := range conversation.Messages {
			// 只处理高质量的用户消息
			if message.Role != entity.USER || message.QualityScore != 1 {
				continue
			}

			// 生成多个截断变体
			variants := a.generateTruncationVariants(message.Content)

			for _, truncatedContent := range variants {
				// 构建JSONL记录
				record := a.buildTabCompletionRecord(truncatedContent, message.Content, mapData)

				// 序列化为JSON
				jsonBytes, err := json.Marshal(record)
				if err != nil {
					continue // 跳过序列化失败的记录
				}

				jsonlLines = append(jsonlLines, string(jsonBytes))
				totalRecords++
			}
		}
	}

	// 合并所有JSONL行
	result := strings.Join(jsonlLines, "\n")
	return result, totalRecords, nil
}

// generateTruncationVariants 生成截断变体
func (a *AiChatService) generateTruncationVariants(content string) []string {
	// 如果内容太短，不进行截断
	if utf8.RuneCountInString(content) < 6 {
		return []string{content}
	}

	var variants []string
	runes := []rune(content)
	totalLength := len(runes)

	// 生成3个不同的截断变体
	truncationRatios := []float64{0.33, 0.5, 0.67} // 1/3, 1/2, 2/3

	for _, ratio := range truncationRatios {
		truncateLength := int(float64(totalLength) * ratio)

		// 确保截断长度合理
		if truncateLength < 2 {
			truncateLength = 2
		}
		if truncateLength >= totalLength-1 {
			truncateLength = totalLength - 1
		}

		// 进行智能截断（避免在词语中间截断）
		truncated := a.smartTruncate(runes, truncateLength)
		if truncated != "" && truncated != content {
			variants = append(variants, truncated)
		}
	}

	// 如果没有生成任何变体，返回原内容
	if len(variants) == 0 {
		variants = append(variants, content)
	}

	return variants
}

// smartTruncate 智能截断，避免在词语中间截断
func (a *AiChatService) smartTruncate(runes []rune, targetLength int) string {
	if targetLength >= len(runes) {
		return string(runes)
	}

	// 从目标位置向前查找合适的截断点
	for i := targetLength; i >= targetLength-5 && i > 0; i-- {
		char := runes[i-1]

		// 在标点符号、空格或中文字符后截断
		if a.isGoodTruncationPoint(char) {
			return strings.TrimSpace(string(runes[:i]))
		}
	}

	// 如果找不到好的截断点，直接截断
	return strings.TrimSpace(string(runes[:targetLength]))
}

// isGoodTruncationPoint 判断是否是好的截断点
func (a *AiChatService) isGoodTruncationPoint(r rune) bool {
	// 标点符号
	if r == '，' || r == '。' || r == '？' || r == '！' || r == '；' || r == '：' {
		return true
	}
	if r == ',' || r == '.' || r == '?' || r == '!' || r == ';' || r == ':' {
		return true
	}

	// 空格
	if r == ' ' || r == '\t' || r == '\n' {
		return true
	}

	// 中文字符（大部分中文字符都可以作为截断点）
	if r >= 0x4e00 && r <= 0x9fff {
		return true
	}

	return false
}

// buildTabCompletionRecord 构建Tab补全JSONL记录
func (a *AiChatService) buildTabCompletionRecord(userInput, fullContent, mapData string) entity.JSONLRecord {
	// 构建系统提示词
	systemPrompt := a.buildTabCompletionSystemPrompt(mapData)

	return entity.JSONLRecord{
		Messages: []entity.JSONLMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userInput,
			},
			{
				Role:    "assistant",
				Content: fullContent,
			},
		},
	}
}

// buildTabCompletionSystemPrompt 构建Tab补全系统提示词
func (a *AiChatService) buildTabCompletionSystemPrompt(mapData string) string {
	basePrompt := `你是一个专门服务于一款思维导图树产品中的提问补全助手，猜测用户下一步输入，帮助用户继续思考和完善导图，而不是直接给出知识答案。

请根据以下信息，补全用户当前输入，帮助用户继续思考。提示问题必须完整保留用户已输入的内容，并且以口语化的方式提出，具体且可执行。`

	if mapData != "" && mapData != "{}" {
		return fmt.Sprintf("%s\n\n【导图上下文】\n%s", basePrompt, mapData)
	}

	return basePrompt
}

// TriggerQualityAssessment 手动触发质量评估（暂时移除，使用实时队列）
func (a *AiChatService) TriggerQualityAssessment(ctx context.Context, date string) (int, int, int, error) {
	// TODO: 如果需要批量处理历史数据，可以在这里实现
	return 0, 0, 0, fmt.Errorf("手动触发功能暂时不可用，系统使用实时队列处理")
}

// GenerateMindMapPro 批量生成思维导图（Pro版本，用于数据收集）
func (a *AiChatService) GenerateMindMapPro(ctx context.Context, req *types.GenerateMindMapProParams) (*entity.GenerationBatch, []*entity.GenerationResult, []*entity.Conversation, error) {
	// 1. 获取用户信息
	user, ok := entity.GetUser(ctx)
	if !ok {
		zlog.CtxErrorf(ctx, "未能从上下文中获取用户信息")
		return nil, nil, nil, AI_CHAT_PERMISSION_DENIED
	}

	// 2. 处理输入文本
	inputText := req.Text
	if req.File != nil {
		text, err := util.ParseFile(ctx, req.File)
		if err != nil {
			return nil, nil, nil, err
		}
		inputText = text
	}

	// 3. 创建批次记录
	batchID, err := util.GenerateStringID()
	if err != nil {
		return nil, nil, nil, err
	}

	batch := &entity.GenerationBatch{
		BatchID:            batchID,
		UserID:             user.UserID,
		InputText:          inputText,
		GenerationCount:    req.Count,
		GenerationStrategy: req.Strategy,
	}

	if err := batch.Validate(); err != nil {
		return nil, nil, nil, err
	}

	// 4. 调用AI层批量生成
	results, conversations, err := a.einoServer.GenerateMindMapBatch(ctx, inputText, user.UserID, req.Strategy, req.Count)
	if err != nil {
		return nil, nil, nil, err
	}

	// 5. 构建结果实体
	strategy := req.Strategy // 优化：移出循环，避免重复赋值
	generationResults := make([]*entity.GenerationResult, 0, len(results))
	for i, result := range results {
		now := time.Now() // 优化：每次迭代时记录时间，保证时间一致性

		resultID, err := util.GenerateStringID()
		if err != nil {
			zlog.CtxWarnf(ctx, "生成结果ID失败: %v", err)
			continue
		}

		var conversationID string
		if i < len(conversations) {
			conversationID = conversations[i].ConversationID
		}

		// 验证JSON格式并处理
		var generationResult *entity.GenerationResult

		// 根据策略提取JSON并验证格式
		extractedJSON := extractJSONFromResult(result, strategy)

		var mindMapData map[string]interface{}
		if err := json.Unmarshal([]byte(extractedJSON), &mindMapData); err != nil {
			// JSON反序列化失败 - 自动标记为负样本，用于DPO训练
			displayJSON := result
			if len(result) > 200 {
				displayJSON = result[:200] + "..."
			}
			zlog.CtxWarnf(ctx, "AI生成JSON反序列化失败，自动标记为负样本: %v, JSON: %s", err, displayJSON)

			errorMessage := fmt.Sprintf("JSON反序列化失败: %v", err)
			generationResult = &entity.GenerationResult{
				ResultID:       resultID,
				BatchID:        batchID,
				ConversationID: conversationID,
				MapJSON:        extractedJSON, // 保存提取的JSON（即使格式错误）
				Label:          -1,            // 自动标记为负样本
				LabeledAt:      &now,          // 设置标记时间
				CreatedAt:      now,           // 优化：使用循环开始时的时间
				Strategy:       &strategy,
				ErrorMessage:   &errorMessage, // 记录具体错误信息
			}
		} else {
			// JSON反序列化成功 - 默认未标记，等待用户手动标记
			zlog.CtxDebugf(ctx, "AI生成JSON格式正确，等待用户标记")
			generationResult = &entity.GenerationResult{
				ResultID:       resultID,
				BatchID:        batchID,
				ConversationID: conversationID,
				MapJSON:        extractedJSON, // 保存提取的有效JSON
				Label:          0,             // 默认未标记，等待用户手动标记
				CreatedAt:      now,           // 优化：使用循环开始时的时间
				Strategy:       &strategy,
			}
		}
		generationResults = append(generationResults, generationResult)
	}

	// 6. 返回批次、结果和对话数据，由Handler层负责保存
	return batch, generationResults, conversations, nil
}

// extractJSONFromResult 根据策略从AI生成结果中提取JSON
func extractJSONFromResult(result string, strategy int) string {
	if strategy == 1 {
		// 策略1（SFT）：从【思考过程】...【导图JSON】格式中提取JSON
		return extractJSONFromSFTResult(result)
	} else {
		// 策略2（DPO）：智能提取JSON，容错处理额外文字
		return extractJSONFromDPOResult(result)
	}
}

// extractJSONFromSFTResult 从SFT格式结果中提取JSON
func extractJSONFromSFTResult(result string) string {
	// 查找【导图JSON】标记
	jsonStartMarkers := []string{"【导图JSON】", "【导图JSON】\n", "[导图JSON]"}

	for _, marker := range jsonStartMarkers {
		if idx := strings.Index(result, marker); idx != -1 {
			// 找到标记，提取后面的内容
			jsonStart := idx + len(marker)
			jsonContent := result[jsonStart:]

			// 去掉前后的空白字符
			jsonContent = strings.TrimSpace(jsonContent)

			// 如果找到JSON内容，返回第一个完整的JSON对象
			if jsonContent != "" {
				return extractFirstJSONObject(jsonContent)
			}
		}
	}

	// 如果没找到标记，尝试直接提取JSON（可能AI没按格式输出）
	return extractFirstJSONObject(result)
}

// extractFirstJSONObject 从文本中提取第一个完整的JSON对象
func extractFirstJSONObject(text string) string {
	text = strings.TrimSpace(text)

	// 查找第一个 '{'
	start := strings.Index(text, "{")
	if start == -1 {
		return text // 没有找到JSON，返回原文本
	}

	// 从 '{' 开始，查找匹配的 '}'
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
					// 找到完整的JSON对象
					return text[start : i+1]
				}
			}
		}
	}

	// 没有找到完整的JSON，返回从第一个 '{' 开始的内容
	return text[start:]
}

// extractJSONFromDPOResult 从DPO格式结果中提取JSON（智能容错）
func extractJSONFromDPOResult(result string) string {
	// 首先尝试直接解析（如果AI按要求只输出了JSON）
	result = strings.TrimSpace(result)
	var testData map[string]interface{}
	if json.Unmarshal([]byte(result), &testData) == nil {
		// 直接是有效JSON，返回
		return result
	}

	// 如果不是纯JSON，进行智能提取
	// 1. 去除常见的Markdown代码块标记
	if strings.HasPrefix(result, "```json") && strings.HasSuffix(result, "```") {
		content := result[7 : len(result)-3] // 去掉 ```json 和 ```
		return strings.TrimSpace(content)
	}

	if strings.HasPrefix(result, "```") && strings.HasSuffix(result, "```") {
		firstNewline := strings.Index(result, "\n")
		if firstNewline > 0 {
			content := result[firstNewline+1 : len(result)-3]
			return strings.TrimSpace(content)
		}
	}

	// 2. 查找常见的说明文字后的JSON
	commonPrefixes := []string{
		"以下是生成的思维导图：",
		"思维导图JSON如下：",
		"导图JSON：",
		"生成的导图：",
		"JSON格式：",
		"导图数据：",
	}

	for _, prefix := range commonPrefixes {
		if idx := strings.Index(result, prefix); idx != -1 {
			jsonStart := idx + len(prefix)
			remaining := strings.TrimSpace(result[jsonStart:])
			if remaining != "" {
				return extractFirstJSONObject(remaining)
			}
		}
	}

	// 3. 直接提取第一个完整的JSON对象
	return extractFirstJSONObject(result)
}
