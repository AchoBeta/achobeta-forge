package eino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"forge/biz/entity"
	"forge/biz/repo"
	"forge/biz/types"
	"forge/infra/configs"
	"forge/pkg/log/zlog"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type AiChatClient struct {
	ApiKey              string
	ModelName           string
	Agent               compose.Runnable[[]*schema.Message, types.AgentResponse]
	ToolAiClient        *ark.ChatModel
	GenerateMapAiClient *ark.ChatModel
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

	var aiChatClient AiChatClient

	//初始化工具专用模型
	toolModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:   apiKey,
		Model:    modelName,
		Thinking: &model.Thinking{Type: model.ThinkingTypeEnabled},
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

func (a *AiChatClient) SendMessage(ctx context.Context, messages []*entity.Message) (types.AgentResponse, error) {
	input := messagesDo2Input(messages)

	resp, err := a.Agent.Invoke(ctx, input)

	if err != nil {
		zlog.Errorf("模型调用失败%v", err)
		return types.AgentResponse{}, err
	}
	return resp, nil
}

// 传入文本生成导图
func (a *AiChatClient) GenerateMindMap(ctx context.Context, text, userID string) (string, error) {
	message := initGenerateMindMapMessage(text, userID)

	resp, err := a.GenerateMapAiClient.Generate(ctx, message)
	if err != nil {
		zlog.Errorf("模型调用失败%v", err)
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

// generateForSFTTraining 策略1：SFT训练数据策略 - 生成带reasoning_content的高质量数据
func (a *AiChatClient) generateForSFTTraining(ctx context.Context, text, userID string, count int) ([]string, []*entity.Conversation, error) {
	// SFT训练专用系统提示词 - 要求生成带推理过程的高质量导图
	sftSystemPrompt := configs.Config().GetAiChatConfig().GenerateSystemPrompt + `

【SFT训练专用要求】
你需要生成用于SFT（监督微调）训练的高质量数据。请按照以下要求：

1. **深度思考过程**：在生成导图前，请详细说明你的思考过程，包括：
   - 对用户文本的理解和分析
   - 思维导图结构的设计思路  
   - 关键节点和层次关系的规划
   - 为什么选择这样的组织方式

2. **高质量输出**：确保导图具备：
   - 清晰的逻辑结构（2-4层深度）
   - 完整的JSON格式规范
   - 准确的内容表达
   - 合理的信息组织

3. **输出格式**：
   先输出【思考过程】，再输出【导图JSON】
   
请严格按照原有JSON规范输出，确保格式正确。`

	results := make([]string, 0, count)
	conversations := make([]*entity.Conversation, 0, count)

	// 串行生成，确保质量一致性（SFT注重质量而非速度）
	for i := 0; i < count; i++ {
		messages := []*schema.Message{
			{
				Content: sftSystemPrompt,
				Role:    schema.System,
			},
			{
				Content: fmt.Sprintf("userID请填写：%s \n用户文本：%s", userID, text),
				Role:    schema.User,
			},
		}

		// 使用thinking模式生成带推理的内容
		resp, err := a.ToolAiClient.Generate(ctx, messages)
		if err != nil {
			zlog.CtxWarnf(ctx, "SFT生成失败 index:%d, err:%v", i, err)
			continue
		}

		// 创建对话记录
		conversation, err := entity.NewConversation(userID, "test", "BATCH_GENERATION", fmt.Sprintf("SFT训练-%d", i+1))
		if err != nil {
			zlog.CtxWarnf(ctx, "创建对话失败 index:%d, err:%v", i, err)
			continue
		}

		// 添加消息到对话
		conversation.AddMessage(sftSystemPrompt, entity.SYSTEM, "", nil)
		conversation.AddMessage(fmt.Sprintf("userID请填写：%s \n用户文本：%s", userID, text), entity.USER, "", nil)
		conversation.AddMessage(resp.Content, entity.ASSISTANT, "", nil)

		results = append(results, resp.Content)
		conversations = append(conversations, conversation)

		zlog.CtxInfof(ctx, "SFT生成完成 %d/%d", i+1, count)
	}

	if len(results) == 0 {
		return nil, nil, errors.New("SFT策略：所有生成都失败了")
	}

	zlog.CtxInfof(ctx, "SFT策略完成：成功生成 %d 个高质量样本", len(results))
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
		conversation, err := entity.NewConversation(userID, "test", "BATCH_GENERATION", fmt.Sprintf("DPO训练-%s-%d", qualityPrompt.level, i+1))
		if err != nil {
			zlog.CtxWarnf(ctx, "创建对话失败 index:%d, err:%v", i, err)
			continue
		}

		// 添加消息到对话（使用原始提示词保持一致性）
		conversation.AddMessage(basePrompt, entity.SYSTEM, "", nil)
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
