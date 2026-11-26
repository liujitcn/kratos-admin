package admin

import (
	"encoding/json"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/errorsx"
)

const (
	biAskToolName     = "bi_admin_v1_bi_query_service_ask_bi_query"
	biRefineToolName  = "bi_admin_v1_bi_query_service_refine_bi_query"
	biChatPreviewRows = 6
)

// init 按问数及调整查询的真实执行结果注册 BI 展示规则。
func init() {
	for _, name := range []string{biAskToolName, biRefineToolName} {
		ai.RegisterResponseBlockRule(ai.ResponseBlockRule{
			ToolName:      name,
			IncludeErrors: true,
			Build:         buildBiCardGuideBlocks,
		})
	}
}

// buildBiCardGuideBlocks 将真实问数记录、经校验的澄清或权限结果转成对话展示块。
func buildBiCardGuideBlocks(tool agent.ToolUsage) []map[string]any {
	if tool.Status == "error" {
		return buildBiAskFailureBlocks(tool)
	}
	var output adminv1.AskBiQueryResponse
	if err := json.Unmarshal([]byte(tool.Output), &output); err != nil || output.GetQuery() == nil || output.GetResult().GetQueryRecordId() <= 0 {
		return nil
	}
	result := output.GetResult()
	if len(result.GetRows()) == 0 {
		return []map[string]any{{
			"type":     "bi_query_empty",
			"question": biAskQuestion(tool.Input),
			"query":    output.GetQuery(),
			"columns":  result.GetColumns(),
		}}
	}
	return []map[string]any{{
		"type":            "bi_query_result",
		"query_record_id": result.GetQueryRecordId(),
		"query":           output.GetQuery(),
		"default_name":    output.GetTitle(),
		"chart_type":      output.GetChartType(),
		"columns":         result.GetColumns(),
		"rows":            result.GetRows()[:min(len(result.GetRows()), biChatPreviewRows)],
		"row_count":       len(result.GetRows()),
		"truncated":       result.GetTruncated(),
	}}
}

// buildBiAskFailureBlocks 只接受服务端产生的权限原因和可信澄清选项。
func buildBiAskFailureBlocks(tool agent.ToolUsage) []map[string]any {
	var failure struct {
		Reason  string `json:"reason"`
		Details struct {
			Clarification *dto.AiMetricClarification `json:"bi_metric_clarification"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(tool.Output), &failure); err != nil {
		return nil
	}
	if failure.Reason == errorsx.ReasonPermissionDenied {
		return []map[string]any{{"type": "bi_permission_denied", "code": failure.Reason}}
	}
	if clarification := failure.Details.Clarification; clarification != nil {
		return []map[string]any{{
			"type":        "bi_query_clarification",
			"question":    biAskQuestion(tool.Input),
			"source_code": clarification.SourceCode,
			"source_name": clarification.SourceName,
			"options":     clarification.Options,
		}}
	}
	return nil
}

// biAskQuestion 从已校验的工具参数提取原问题供用户编辑。
func biAskQuestion(input string) string {
	var request adminv1.AskBiQueryRequest
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return ""
	}
	return request.GetQuestion()
}
