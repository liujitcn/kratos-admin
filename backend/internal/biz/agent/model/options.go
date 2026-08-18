package model

import (
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	componentsModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go/v3/responses"
)

// WithTools 构造携带工具定义的模型调用选项。
func WithTools(toolInfos []*schema.ToolInfo) Option {
	return componentsModel.WithTools(toolInfos)
}

// ResponsesServerToolOptions 构造 Responses 内置服务端工具选项。
func ResponsesServerToolOptions() []Option {
	return []Option{
		agenticopenai.WithResponsesServerTools([]*agenticopenai.ResponsesServerToolConfig{
			{
				// 目前只开放联网搜索，其他 Responses 服务端工具后续按业务需求逐个接入。
				WebSearch: &responses.WebSearchToolParam{
					Type: responses.WebSearchToolTypeWebSearch,
				},
			},
		}),
	}
}
