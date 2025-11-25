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
// TODO: 未来可能会使用历史对话上下文（recentMessages），目前为了与训练数据保持一致，暂不使用历史消息
func (t *TabCompletionClient) buildTabCompletionPrompt(userInput, mapData string, recentMessages []*entity.Message) string {
	// 如果mapData为空，使用默认值
	if mapData == "" {
		mapData = "{}"
	}

	// TODO: 历史对话上下文功能暂时禁用，确保训练和运行时提示词一致
	// 构建历史对话上下文
	// historyContext := ""
	// if len(recentMessages) > 0 {
	// 	historyContext = "\n\n"
	// 	for _, msg := range recentMessages {
	// 		if msg.Role == entity.USER {
	// 			historyContext += fmt.Sprintf("<用户上一句提问>\n%s\n</用户上一句提问>\n", msg.Content)
	// 		}
	// 	}
	// }

	// 构建提示词模板（与训练数据保持一致，不包含历史消息）
	prompt := `你是一款思维导图树产品中"提问tab"的补全助手，核心任务是站在用户视角，基于当前导图树内容和用户已输入的句子，猜测用户下一步可能的思考方向，生成能帮助用户继续思考、提问知识点或完善导图的提示问题（禁止直接给出知识答案）。

首先，请查看用户当前输入的内容：

<current_input>

` + userInput + `

</current_input>

然后，请参考当前的导图树JSON数据：

<mind_map>

` + mapData + `

</mind_map>

生成提示问题时，必须严格遵守以下规则：

1. 完全模拟用户视角思考，禁止出现脱离用户视角的表述（如使用"你"称呼用户、反问用户等，也不要出现"呢"等语气词，本质上你就是用户本人的思考延伸）

2. **优先关联导图JSON数据**：补全用户输入时，优先查看导图树JSON中是否有匹配的节点、分支或相关内容，优先围绕导图的各个点和分支进行补全，如果导图中有相关信息则优先使用，没有匹配内容时再考虑其他补全方向

3. 提示问题需口语化、具体、可执行，能直接引导用户进一步完善导图

4. 输出格式固定：采用"用户输入原话+补充提问"的AB形式，**不得修改用户输入的原句**

5. 每次仅输出1条的提示问题/长句，无需额外说明

示例参考：

- 用户输入："这个导图的核心结论" → 输出："这个导图的核心结论再具体一点"

- 用户输入："第二个分支下面" → 输出："第二个分支下面是不是还可以拆一两个更细的子分支？"

- 用户输入："第三个分支和第二个" → 输出："第三个分支和第二个分支之间，有哪些共同点可以合并？"

请直接输出符合要求的提示问题，无需其他内容。`

	return prompt
}
