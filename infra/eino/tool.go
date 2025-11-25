package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"forge/biz/entity"
	"forge/pkg/log/zlog"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

func (a *AiChatClient) UpdateMindMap(ctx context.Context, params *UpdateMindMapParams) (string, error) {
	conversation, ok := entity.GetConversation(ctx)
	if !ok {
		return "", fmt.Errorf("未能从上下文中获取到导图数据")
	}
	//fmt.Println(conversation.MapData)
	message := initToolUpdateMindMap(conversation.MapData, params.Requirement)

	resp, err := a.ToolAiClient.Generate(ctx, message)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (a *AiChatClient) CreateUpdateMindMapTool() tool.InvokableTool {
	updateMindMapTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "update_mind_map",
			Desc: "用于修改导图,需要修改导图时调用该工具,返回完整新导图JSON",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"requirement": {
						Type:     schema.String,
						Desc:     "需要工具修改导图的需求，例如「把 root.children[0].data.text 改成『新产品』」",
						Required: true,
					},
				},
			),
		}, a.UpdateMindMap)
	return updateMindMapTool
}

// WebSearch 网络搜索工具实现
func (a *AiChatClient) WebSearch(ctx context.Context, params *WebSearchParams) (string, error) {
	if params.Query == "" {
		return "", fmt.Errorf("搜索关键词不能为空")
	}

	zlog.CtxInfof(ctx, "开始网络搜索: query=%s, maxResults=%d", params.Query, params.MaxResults)

	// 执行搜索
	results, err := a.SearchService.Search(ctx, params.Query, params.MaxResults)
	if err != nil {
		zlog.CtxErrorf(ctx, "网络搜索失败: %v", err)
		return "", fmt.Errorf("网络搜索失败: %w", err)
	}

	// 格式化搜索结果为可读文本
	if len(results) == 0 {
		return "未找到相关搜索结果", nil
	}

	// 构建搜索结果摘要
	resultText := fmt.Sprintf("搜索「%s」找到 %d 条结果：\n\n", params.Query, len(results))
	for i, result := range results {
		resultText += fmt.Sprintf("%d. **%s**\n", i+1, result.Title)
		resultText += fmt.Sprintf("   链接: %s\n", result.Link)
		if result.Snippet != "" {
			resultText += fmt.Sprintf("   摘要: %s\n", result.Snippet)
		}
		resultText += "\n"
	}

	zlog.CtxInfof(ctx, "网络搜索完成，返回 %d 条结果", len(results))
	return resultText, nil
}

// CreateWebSearchTool 创建网络搜索工具
func (a *AiChatClient) CreateWebSearchTool() tool.InvokableTool {
	webSearchTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "web_search",
			Desc: "用于在互联网上搜索信息。当用户询问实时信息、最新新闻、需要查找网络资料时调用此工具。返回搜索结果的标题、链接和摘要。",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"query": {
						Type:     schema.String,
						Desc:     "搜索关键词，例如「人工智能最新进展」、「Go语言教程」等",
						Required: true,
					},
					"max_results": {
						Type:     schema.Integer,
						Desc:     "返回的最大结果数量，默认5条，最多10条",
						Required: false,
					},
				},
			),
		}, a.WebSearch)
	return webSearchTool
}

// WebSearchParams 网络搜索参数
type WebSearchParams struct {
	Query      string `json:"query" jsonschema:"description=搜索关键词"`
	MaxResults int    `json:"max_results" jsonschema:"description=返回的最大结果数量"`
}

// MarshalJSON 自定义JSON序列化，处理默认值
func (p *WebSearchParams) UnmarshalJSON(data []byte) error {
	type Alias WebSearchParams
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	// 设置默认值
	if p.MaxResults == 0 {
		p.MaxResults = 5
	}
	return nil
}
