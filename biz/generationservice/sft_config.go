package generationservice

// SFTStandardSystemPrompt 标准System Prompt
// 用于SFT训练数据生成（使用结构化输出，无需格式要求）
const SFTStandardSystemPrompt = `你是思维导图生成专家。将用户文本转换为思维导图。

【要求】
- 树深度2-7层
- 节点文本≤20字
- 内容准确、逻辑清晰
- 层次关系合理

直接开始转换，无需说明格式。`

// GetMindMapJSONSchema 返回思维导图的 JSON Schema 定义
// 用于结构化输出，确保模型输出符合预期的 JSON 结构
func GetMindMapJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "思维导图标题",
			},
			"desc": map[string]interface{}{
				"type":        "string",
				"description": "思维导图描述",
			},
			"layout": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"mindMap"},
				"description": "布局类型，固定为mindMap",
			},
			"root": map[string]interface{}{
				"$ref": "#/$defs/MindMapNode",
			},
		},
		"required":             []string{"title", "desc", "layout", "root"},
		"additionalProperties": false,
		"$defs": map[string]interface{}{
			"MindMapNode": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"text": map[string]interface{}{
								"type":        "string",
								"description": "节点文本，不超过20字",
							},
						},
						"required":             []string{"text"},
						"additionalProperties": false,
					},
					"children": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"$ref": "#/$defs/MindMapNode",
						},
					},
				},
				"required":             []string{"data", "children"},
				"additionalProperties": false,
			},
		},
	}
}
