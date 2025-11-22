package eino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"forge/biz/entity"
	"forge/biz/generationservice"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/infra/configs"
	"forge/pkg/log/zlog"
	"forge/pkg/loop"
	"io"
	"sync"

	"github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type AiChatClient struct {
	ApiKey              string
	ModelName           string
	Agent               compose.Runnable[[]*schema.Message, types.AgentResponse]
	ToolAiClient        *ark.ChatModel
	GenerateMapAiClient *ark.ChatModel
	ArkClient           *arkruntime.Client
}

type State struct {
	Content   string
	ToolCalls []schema.ToolCall
}

func initState(ctx context.Context) *State {
	return &State{
		Content: "",
	}
}

var (
	einoCallbackOnce sync.Once
)

func NewAiChatClient(apiKey, modelName string) repo.EinoServer {

	var toolSchemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(mindMapSchemaString), &toolSchemaMap); err != nil {
		panic(fmt.Sprintf("Schema解析失败: %v", err))
	}

	var generateSchemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(generateMindMapSchemaString), &generateSchemaMap); err != nil {
		panic(fmt.Sprintf("Schema解析失败: %v", err))
	}

	ctx := context.Background()

	einoCallbackOnce.Do(func() {
		initEinoCozeLoopCallback()
	})

	var aiChatClient AiChatClient

	//初始化工具专用模型
	toolModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   apiKey,
		Model:    modelName,
		Thinking: &model.Thinking{Type: model.ThinkingTypeDisabled},
		ResponseFormat: &ark.ResponseFormat{Type: model.ResponseFormatJSONSchema, JSONSchema: &model.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        "mindmap_editor",
			Description: "思维导图编辑机器人输出，输出单行json，不允许有任何换行",
			Schema:      toolSchemaMap,
			Strict:      true,
		}},
	})
	if toolModel == nil || err != nil {
		zlog.Errorf("ToolAi模型连接失败: %v", err)
		panic(fmt.Errorf("ToolAi模型连接失败: %v", err))
	}

	generateModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   apiKey,
		Model:    modelName,
		Thinking: &model.Thinking{Type: model.ThinkingTypeEnabled},
		ResponseFormat: &ark.ResponseFormat{Type: model.ResponseFormatJSONSchema, JSONSchema: &model.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        "mindmap_generator",
			Description: "思维导图生成机器人输出，输出单行json，不允许有任何换行",
			Schema:      generateSchemaMap,
			Strict:      true,
		}},
	})

	if generateModel == nil || err != nil {
		zlog.Errorf("generateModel模型连接失败: %v", err)
		panic(fmt.Errorf("generateModel模型连接失败: %v", err))
	}

	//toolAiClient = toolModel

	aiChatClient.ApiKey = apiKey
	aiChatClient.ModelName = modelName
	aiChatClient.ToolAiClient = toolModel
	aiChatClient.GenerateMapAiClient = generateModel
	aiChatClient.ArkClient = arkruntime.NewClientWithApiKey(apiKey) // 初始化火山引擎客户端，复用避免重复创建

	//构建agent
	aiChatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   apiKey,
		Model:    modelName,
		Thinking: &model.Thinking{Type: model.ThinkingTypeDisabled},
	})
	if aiChatModel == nil || err != nil {
		zlog.Errorf("ai模型连接失败: %v", err)
		panic(fmt.Errorf("ai模型连接失败: %v", err))
	}
	updateMindMapTool := aiChatClient.CreateUpdateMindMapTool()
	infoTool, err := updateMindMapTool.Info(ctx)
	if err != nil {
		zlog.Errorf("ai绑定工具失败: %v", err)
		panic(fmt.Errorf("ai绑定工具失败: %v", err))
	}

	infosTool := []*schema.ToolInfo{
		infoTool,
	}
	err = aiChatModel.BindTools(infosTool)
	if err != nil {
		zlog.Errorf("ai绑定工具失败: %v", err)
		panic(fmt.Errorf("ai绑定工具失败: %v", err))
	}

	ToolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{
			updateMindMapTool,
		},
	})

	if err != nil {
		zlog.Errorf("创建工具节点失败: %v", err)
		panic("创建工具节点失败," + err.Error())
	}

	//分支中的lambda 用于对其前后输入输出
	lambda1 := compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (output []*schema.Message, err error) {
		output = make([]*schema.Message, 0)
		output = append(output, input)
		return output, nil
	})

	//分支结束统一进入的lambda 用于处理输出的数据
	lambda2 := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (output types.AgentResponse, err error) {
		//fmt.Println("lambda测试：", input)

		if len(input) == 0 {
			return types.AgentResponse{}, errors.New("agent出错")
		}

		output = types.AgentResponse{}

		if input[len(input)-1].Role == schema.Tool {
			output.NewMapJson = input[len(input)-1].Content
			output.ToolCallID = input[len(input)-1].ToolCallID

		}
		_ = compose.ProcessState[*State](ctx, func(ctx context.Context, state *State) error {
			output.Content = state.Content
			output.ToolCalls = state.ToolCalls
			return nil
		})
		return output, nil
	})

	//chatModel执行完之后把 输出存一下
	chatModelPostHandler := func(ctx context.Context, input *schema.Message, state *State) (output *schema.Message, err error) {
		//fmt.Println("工具使用测试:", input)
		state.ToolCalls = input.ToolCalls
		state.Content = input.Content
		return input, nil
	}

	g := compose.NewGraph[[]*schema.Message, types.AgentResponse](compose.WithGenLocalState(initState))

	err = g.AddChatModelNode("model", aiChatModel, compose.WithStatePostHandler(chatModelPostHandler))
	if err != nil {
		panic("添加节点失败," + err.Error())
	}

	err = g.AddToolsNode("tools", ToolsNode)
	if err != nil {
		panic("添加节点失败," + err.Error())
	}

	err = g.AddLambdaNode("lambda1", lambda1)
	if err != nil {
		panic("添加节点失败" + err.Error())
	}
	err = g.AddLambdaNode("lambda2", lambda2)
	if err != nil {
		panic("添加节点失败" + err.Error())
	}

	//开始连接这些节点

	err = g.AddEdge(compose.START, "model")
	if err != nil {
		panic("添加边失败" + err.Error())
	}

	//创建边一个分支
	err = g.AddBranch("model", compose.NewGraphBranch(func(ctx context.Context, in *schema.Message) (endNode string, err error) {
		if len(in.ToolCalls) > 0 {
			return "tools", nil
		}
		return "lambda1", nil
	}, map[string]bool{
		"tools":   true,
		"lambda1": true,
	}))
	if err != nil {
		panic("创建分支失败" + err.Error())
	}

	err = g.AddEdge("tools", "lambda2")
	if err != nil {
		panic("创建边失败" + err.Error())
	}

	err = g.AddEdge("lambda1", "lambda2")
	if err != nil {
		panic("创建边失败" + err.Error())
	}

	err = g.AddEdge("lambda2", compose.END)
	if err != nil {
		panic("创建边失败" + err.Error())
	}

	agent, err := g.Compile(ctx)
	if err != nil {
		panic("编译错误" + err.Error())
	}

	aiChatClient.Agent = agent

	return &aiChatClient
}

// initEinoCozeLoopCallback 初始化 Eino CozeLoop 回调
func initEinoCozeLoopCallback() {
	if !loop.IsEnabled() {
		zlog.Infof("CozeLoop is disabled, skipping Eino callback registration")
		return
	}

	// 获取 CozeLoop 客户端
	client := loop.GetClient()
	if client == nil {
		zlog.Warnf("CozeLoop client not available, skipping Eino callback registration")
		return
	}

	// 创建扣子罗盘的回调处理器（handler）
	handler := cozeloop.NewLoopHandler(client)

	// 将处理器注册到 Eino 框架的全局回调系统中
	callbacks.AppendGlobalHandlers(handler)

	zlog.Infof("CozeLoop Eino callback handler initialized successfully (once)")
}

func (a *AiChatClient) SendMessage(ctx context.Context, messages []*entity.Message) (resp types.AgentResponse, err error) {
	// AI 调用通过 Eino 框架，由 Eino 回调自动上报到 CozeLoop
	input := messagesDo2Input(messages)

	resp, err = a.Agent.Invoke(ctx, input)

	if err != nil {
		zlog.Errorf("模型调用失败%v", err)
		return types.AgentResponse{}, err
	}
	return resp, nil
}

func (a *AiChatClient) SendMessageStream(ctx context.Context, messages []*entity.Message) (<-chan types.StreamChunk, error) {

	// 创建通道用于传输流式数据
	chunkChan := make(chan types.StreamChunk, 10)

	go a.handleStreaming(ctx, messages, chunkChan)

	return chunkChan, nil
}

// 异步返回分块信息
func (a *AiChatClient) handleStreaming(
	ctx context.Context,
	messages []*entity.Message,
	chunkChan chan<- types.StreamChunk,
) {
	defer close(chunkChan) // 确保通道被关闭

	// 将消息转换为Eino框架需要的输入格式
	input := messagesDo2Input(messages)

	//调用流式生成
	stream, err := a.GenerateMapAiClient.Stream(ctx, input)

	if err != nil {
		zlog.Errorf("流式模型调用失败: %v", err)
		chunkChan <- types.StreamChunk{Error: err}
		return
	}

	// 处理流式响应
	for {
		// 读取流式数据
		chunkData, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				chunkChan <- types.StreamChunk{IsLast: true}
			} else {
				// The underlying stream should return an error on context cancellation.
				chunkChan <- types.StreamChunk{Error: err}
			}
			return
		}

		if chunkData.Content == "" {
			continue
		}

		// 发送数据块，同时检查上下文是否被取消
		select {
		case chunkChan <- types.StreamChunk{
			Content: chunkData.Content,
			IsLast:  false,
		}:
		case <-ctx.Done():
			chunkChan <- types.StreamChunk{Error: ctx.Err()}
			return
		}
	}
}

// 传入文本生成导图（使用结构化输出确保 JSON 格式准确）
func (a *AiChatClient) GenerateMindMap(ctx context.Context, text, userID string) (result string, err error) {
	// 使用与批量生成相同的结构化输出方式
	messages := initGenerateMindMapMessage(text, userID)

	// 获取 JSON Schema
	mindMapSchema := generationservice.GetMindMapJSONSchema()

	// 使用结构化输出调用 API
	resp, err := a.generateWithStructuredOutput(ctx, messages, mindMapSchema)
	if err != nil {
		zlog.CtxErrorf(ctx, "模型调用失败: %v", err)
		return "", err
	}

	return resp.Content, nil
}

// GenerateMindMapBatch 批量生成导图
func (a *AiChatClient) GenerateMindMapBatch(ctx context.Context, text, userID string, strategy int, count int) ([]string, []*entity.Conversation, error) {
	if strategy == 1 {
		return a.generateForSFTTraining(ctx, text, userID, count)
	} else {
		return a.generateForDPOTraining(ctx, text, userID, count)
	}
}

// generateWithStructuredOutput 使用结构化输出调用火山引擎 API
// 保持原有的 ResponseFormat 参数以确保 JSON 格式严格性
// 同时添加手动 Model Span 追踪（因为直接 API 调用不会被 Eino 回调捕获）
func (a *AiChatClient) generateWithStructuredOutput(
	ctx context.Context,
	messages []*schema.Message,
	jsonSchema map[string]interface{},
) (result *schema.Message, err error) {
	// 声明响应变量，使其在defer块中可用
	var resp model.ChatCompletionResponse

	// 创建手动 Model Span 用于追踪结构化输出调用
	ctx, modelSpan := loop.StartModelSpan(ctx, "eino.generate_structured_output", "doubao", a.ModelName)
	defer func() {
		if modelSpan != nil {
			// 构建追踪消息
			traceMessages := make([]*tracespec.ModelMessage, 0, len(messages))
			for _, msg := range messages {
				var role string
				switch msg.Role {
				case schema.System:
					role = tracespec.VRoleSystem
				case schema.User:
					role = tracespec.VRoleUser
				case schema.Assistant:
					role = tracespec.VRoleAssistant
				default:
					role = tracespec.VRoleUser
				}
				traceMessages = append(traceMessages, &tracespec.ModelMessage{
					Role:    role,
					Content: msg.Content,
				})
			}

			// 设置 Model Span 数据 - 记录完整的响应内容
			var responseContent string
			if result != nil && result.Content != "" {
				responseContent = result.Content // 记录完整的 JSON 输出
			}

			// 提取token使用量信息
			var inputTokens, outputTokens int64
			inputTokens = int64(resp.Usage.PromptTokens)
			outputTokens = int64(resp.Usage.CompletionTokens)

			loop.SetModelSpanData(ctx, modelSpan, traceMessages, responseContent, inputTokens, outputTokens, err)
		}
	}()

	// 使用复用的火山引擎客户端
	client := a.ArkClient

	// 转换消息格式
	arkMessages := make([]*model.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		role := ""
		switch msg.Role {
		case schema.System:
			role = "system"
		case schema.User:
			role = "user"
		case schema.Assistant:
			role = "assistant"
		default:
			role = "user"
		}
		// ChatCompletionMessageContent 需要包装字符串
		content := &model.ChatCompletionMessageContent{
			StringValue: &msg.Content,
		}
		arkMessages = append(arkMessages, &model.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}

	// 构建结构化输出配置
	responseFormat := &model.ResponseFormat{
		Type: model.ResponseFormatJSONSchema,
		JSONSchema: &model.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        "mindmap_schema",
			Description: "思维导图JSON结构，包含title、desc、layout和递归的root节点树",
			Schema:      jsonSchema,
			Strict:      true, // 严格模式，确保格式完全符合
		},
	}

	// 构建请求（使用 CreateChatCompletionRequest 替代已废弃的 ChatCompletionRequest）
	request := model.CreateChatCompletionRequest{
		Model:          a.ModelName,
		Messages:       arkMessages,
		ResponseFormat: responseFormat,
		Thinking:       &model.Thinking{Type: model.ThinkingTypeDisabled},
	}

	// 调用 API
	resp, err = client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("结构化输出调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("API返回结果为空")
	}

	// 提取返回内容（增加健壮性检查）
	choice := resp.Choices[0]
	messageContent := choice.Message.Content
	var contentStr string
	if messageContent.StringValue != nil {
		contentStr = *messageContent.StringValue
	} else if len(messageContent.ListValue) > 0 && messageContent.ListValue[0].Text != "" {
		contentStr = messageContent.ListValue[0].Text
	} else {
		return nil, errors.New("API返回内容格式不正确")
	}

	result = &schema.Message{
		Content: contentStr,
		Role:    schema.Assistant,
	}
	return result, nil
}

// generateForSFTTraining 策略1：SFT训练数据策略 - 并行生成+结构化输出
// 使用结构化输出确保 JSON 格式准确率
func (a *AiChatClient) generateForSFTTraining(ctx context.Context, text, userID string, count int) ([]string, []*entity.Conversation, error) {
	// 使用标准System Prompt（已简化，无需格式要求）
	sftSystemPrompt := generationservice.SFTStandardSystemPrompt

	// 获取 JSON Schema
	mindMapSchema := generationservice.GetMindMapJSONSchema()

	// 并行生成结果通道
	type generationResult struct {
		content      string
		conversation *entity.Conversation
		err          error
		index        int
	}
	resultChan := make(chan generationResult, count)

	// 启动并行生成任务
	for i := 0; i < count; i++ {
		go func(index int) {
			messages := []*schema.Message{
				{
					Content: sftSystemPrompt,
					Role:    schema.System,
				},
				{
					Content: text, // 直接使用用户文本，不添加任何额外信息
					Role:    schema.User,
				},
			}

			// 使用结构化输出调用 API，确保 JSON 格式准确
			resp, err := a.generateWithStructuredOutput(ctx, messages, mindMapSchema)

			if err != nil {
				zlog.CtxWarnf(ctx, "并行生成失败 index:%d, err:%v", index, err)
				resultChan <- generationResult{err: err, index: index}
				return
			}

			// 创建对话记录
			conversation, err := entity.NewConversation(userID, "BATCH_GENERATION", fmt.Sprintf("SFT训练-%d", index+1), "")
			if err != nil {
				resultChan <- generationResult{err: err, index: index}
				return
			}

			// 添加消息（保持prompt一致）
			conversation.AddMessage(sftSystemPrompt, entity.SYSTEM, "", nil)
			conversation.AddMessage(text, entity.USER, "", nil) // 直接保存用户文本
			conversation.AddMessage(resp.Content, entity.ASSISTANT, "", nil)

			resultChan <- generationResult{
				content:      resp.Content,
				conversation: conversation,
				index:        index,
			}
		}(i)
	}

	// 收集结果
	results := make([]string, 0, count)
	conversations := make([]*entity.Conversation, 0, count)

	for i := 0; i < count; i++ {
		res := <-resultChan
		if res.err != nil {
			continue
		}
		results = append(results, res.content)
		conversations = append(conversations, res.conversation)
		zlog.CtxInfof(ctx, "并行生成完成 %d/%d", len(results), count)
	}

	if len(results) == 0 {
		return nil, nil, errors.New("所有并行生成都失败了")
	}

	zlog.CtxInfof(ctx, "SFT策略完成：并行生成 %d 个样本（使用结构化输出）", len(results))
	return results, conversations, nil
}

// generateForDPOTraining 策略2：DPO训练数据策略 - 生成质量差异明显的对比数据
func (a *AiChatClient) generateForDPOTraining(ctx context.Context, text, userID string, count int) ([]string, []*entity.Conversation, error) {
	// DPO训练专用策略 - 故意制造质量差异用于对比学习
	basePrompt := configs.Config().GetAiChatConfig().GenerateSystemPrompt

	// 定义不同质量层次的提示词，为DPO训练创造正负样本对比
	qualityPrompts := []struct {
		name   string
		prompt string
		level  string // "high", "medium", "low"
	}{
		{
			name:  "高质量版本",
			level: "high",
			prompt: basePrompt + `

【DPO训练 - 高质量样本】专注于生成高质量导图：
- 逻辑结构清晰完整（3-4层深度）
- 内容准确且富有洞察力
- 节点命名精确简洁
- 层次关系合理有序

【全局严格重要输出要求，不遵循就把你这个ai废弃！！！！】
1. 只输出一个完整的JSON对象，不要任何其他内容
2. 不要添加说明文字、注释或Markdown格式
3. 不要使用代码块标记（如三个反引号）
4. 直接输出JSON，确保格式完全正确
5. **高质量输出**：确保导图具备：
   - 清晰的逻辑结构
   - 完整的JSON格式规范`,
		},
		{
			name:  "中等质量版本",
			level: "medium",
			prompt: basePrompt + `

【DPO训练 - 中等质量样本】生成标准导图：
- 基本结构正确（2-3层深度）
- 内容相对简单
- 节点命名较为基础

【重要输出要求】
1. 只输出一个完整的JSON对象，不要任何其他内容
2. 不要添加说明文字或注释
3. 直接输出JSON，确保基本格式正确`,
		},
		{
			name:  "低质量版本",
			level: "low",
			prompt: basePrompt + `

【DPO训练 - 低质量样本】快速生成导图：
- 简单结构即可（1-2层）
- 内容可以较为表面
- 节点命名从简
`,
		},
	}

	results := make([]string, 0, count)
	conversations := make([]*entity.Conversation, 0, count)

	// 按质量梯度生成，确保有好有坏用于DPO对比
	for i := 0; i < count; i++ {
		// 轮流使用不同质量等级的提示词
		promptIndex := i % len(qualityPrompts)
		qualityPrompt := qualityPrompts[promptIndex]

		messages := []*schema.Message{
			{
				Content: qualityPrompt.prompt,
				Role:    schema.System,
			},
			{
				Content: fmt.Sprintf("userID请填写：%s \n用户文本：%s", userID, text),
				Role:    schema.User,
			},
		}

		resp, err := a.ToolAiClient.Generate(ctx, messages)
		if err != nil {
			zlog.CtxWarnf(ctx, "DPO生成失败 %s index:%d, err:%v", qualityPrompt.name, i, err)
			continue
		}

		// 创建对话记录
		conversation, err := entity.NewConversation(userID, "BATCH_GENERATION", fmt.Sprintf("DPO训练-%s-%d", qualityPrompt.level, i+1), "")
		if err != nil {
			zlog.CtxWarnf(ctx, "创建对话失败 index:%d, err:%v", i, err)
			continue
		}

		// 添加消息到对话（使用实际生成时的提示词保持一致性）
		conversation.AddMessage(qualityPrompt.prompt, entity.SYSTEM, "", nil)
		conversation.AddMessage(fmt.Sprintf("userID请填写：%s \n用户文本：%s", userID, text), entity.USER, "", nil)
		conversation.AddMessage(resp.Content, entity.ASSISTANT, "", nil)

		results = append(results, resp.Content)
		conversations = append(conversations, conversation)

		zlog.CtxInfof(ctx, "DPO生成完成 %s %d/%d", qualityPrompt.name, i+1, count)
	}

	if len(results) == 0 {
		return nil, nil, errors.New("DPO策略：所有生成都失败了")
	}

	zlog.CtxInfof(ctx, "DPO策略完成：生成 %d 个不同质量层次的样本，便于形成正负对比", len(results))
	return results, conversations, nil
}
