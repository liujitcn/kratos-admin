package admin

import (
	"encoding/json"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// biAskToolName 是智能问数工具在 base_api 注册的工具名。
const biAskToolName = "bi_admin_v1_bi_query_service_ask_bi_query"

// aiAskOutput 对应智能问数工具出参中建卡引导需要的字段。
type aiAskOutput struct {
	Title  string         `json:"title"`
	Query  map[string]any `json:"query"`
	Result struct {
		QueryRecordID int64             `json:"query_record_id"`
		Rows          []json.RawMessage `json:"rows"`
	} `json:"result"`
}

// init 注册 BI 模块的响应引导块规则：智能问数执行成功后引导用户一键生成数据卡片。
func init() {
	ai.RegisterResponseBlockRule(ai.ResponseBlockRule{
		ToolName: biAskToolName,
		Build:    buildBiCardGuideBlocks,
	})
}

// buildBiCardGuideBlocks 将成功的智能问数结果转换为“生成数据卡片”引导块；
// 记录缺失或结果为空时不产出引导块，避免保存一张没有数据的空卡片。
func buildBiCardGuideBlocks(tool agent.ToolUsage) []map[string]any {
	if tool.Output == "" {
		return nil
	}
	var output aiAskOutput
	// 记录缺失或结果为空时都不注入引导块。
	if json.Unmarshal([]byte(tool.Output), &output) != nil || output.Result.QueryRecordID <= 0 || len(output.Result.Rows) == 0 {
		return nil
	}
	return []map[string]any{{
		"type":            "bi_card_guide",
		"title":           "生成数据卡片",
		"desc":            "智能问数已完成，可将本次结果保存为数据卡片。",
		"query_record_id": output.Result.QueryRecordID,
		"query":           output.Query,
		"default_name":    output.Title,
	}}
}
