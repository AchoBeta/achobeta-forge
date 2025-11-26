package eino

import (
	"fmt"
	"forge/biz/entity"
	"forge/infra/configs"
	"github.com/cloudwego/eino/schema"
)

func messagesDo2Input(Messages []*entity.Message) []*schema.Message {

	res := make([]*schema.Message, 0)

	for _, msg := range Messages {
		res = append(res, &schema.Message{
			Content:    msg.Content,
			ToolCalls:  msg.ToolCalls,
			ToolCallID: msg.ToolCallID,
			Role:       schema.RoleType(msg.Role),
		})
	}

	return res
}

func initGenerateMindMapMessage(text, userID string) []*schema.Message {
	res := make([]*schema.Message, 0)
	res = append(res, &schema.Message{
		Content: configs.Config().GetAiChatConfig().GenerateSystemPrompt,
		Role:    schema.System,
	})
	res = append(res, &schema.Message{
		Content: fmt.Sprintf("mapId请填写\"xxx\",userId请填写：%s \n用户文本：%s", userID, text),
		Role:    schema.User,
	})
	return res
}

func initToolUpdateMindMap(mapData, requirement string) []*schema.Message {
	res := make([]*schema.Message, 0)
	res = append(res, &schema.Message{
		Content: fmt.Sprintf(configs.Config().GetAiChatConfig().UpdateSystemPrompt, mapData, requirement),
		Role:    schema.System,
	})
	return res
}

type UpdateMindMapParams struct {
	Requirement string `json:"requirement" jsonschema:"description=更改导图的要求"`
}
