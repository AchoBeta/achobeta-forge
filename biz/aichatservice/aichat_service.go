package aichatservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/pkg/log/zlog"
	"forge/util"
	"strings"
	"time"
)

var (
	CONVERSATION_ID_NOT_NULL    = errors.New("会话ID不能为空")
	USER_ID_NOT_NULL            = errors.New("用户ID不能为空")
	MAP_ID_NOT_NULL             = errors.New("导图ID不能为空")
	CONVERSATION_TITLE_NOT_NULL = errors.New("会话标题不能为空")
	CONVERSATION_NOT_EXIST      = errors.New("该会话不存在")
	AI_CHAT_PERMISSION_DENIED   = errors.New("会话权限不足")
	MIND_MAP_NOT_EXIST          = errors.New("该导图不存在")
	AI_CHAT_MESSAGE_MAX         = errors.New("会话长度已达上限，请开启新的会话")
)

type AiChatService struct {
	aiChatRepo repo.AiChatRepo
	einoServer repo.EinoServer
}

func NewAiChatService(aiChatRepo repo.AiChatRepo, einoServer repo.EinoServer) *AiChatService {
	return &AiChatService{aiChatRepo: aiChatRepo, einoServer: einoServer}
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

	//长度限制o
	if len(conversation.Messages) > 100 {
		return types.AgentResponse{}, AI_CHAT_MESSAGE_MAX
	}

	//将数据写入ctx
	ctx = entity.WithConversation(ctx, conversation)

	//更新导图数据
	conversation.UpdateMapData(req.MapData)
	//更新导图提示词
	conversation.ProcessSystemPrompt()

	//添加用户聊天记录
	conversation.AddMessage(req.Message, entity.USER, "", nil)

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
