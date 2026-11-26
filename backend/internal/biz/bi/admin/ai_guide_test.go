package admin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestBuildBiCardGuideBlocksRendersQueryPreview 验证展示实际列、保留建卡来源并限制预览行数。
func TestBuildBiCardGuideBlocksRendersQueryPreview(t *testing.T) {
	blocks := buildBiCardGuideBlocks(agent.ToolUsage{Status: "success", Output: `{"title":"指标概览","chart_type":"metric_grid","query":{"source_code":"test"},"result":{"query_record_id":7,"columns":[{"code":"total","name":"数量","unit":"件"}],"rows":[{"values":["9007199254740999"]},{"values":["2"]},{"values":["3"]},{"values":["4"]},{"values":["5"]},{"values":["6"]},{"values":["7"]},{"values":["8"]}],"truncated":true}}`})
	if len(blocks) != 1 || blocks[0]["type"] != "bi_query_result" || blocks[0]["chart_type"] != "metric_grid" || blocks[0]["query_record_id"] != int64(7) {
		t.Fatalf("查询预览未保留真实建卡来源: %v", blocks)
	}
	raw, err := json.Marshal(blocks[0])
	if err != nil {
		t.Fatal(err)
	}
	var block struct {
		Columns []struct {
			Name string `json:"name"`
			Unit string `json:"unit"`
		} `json:"columns"`
		Rows []struct {
			Values []string `json:"values"`
		} `json:"rows"`
		RowCount  int  `json:"row_count"`
		Truncated bool `json:"truncated"`
	}
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatal(err)
	}
	if len(block.Columns) != 1 || block.Columns[0].Name != "数量" || block.Columns[0].Unit != "件" || len(block.Rows) != 6 || block.Rows[0].Values[0] != "9007199254740999" || block.RowCount != 8 || !block.Truncated {
		t.Fatalf("查询预览未保留列、值或裁剪状态: %+v", block)
	}
}

// TestAskBiQueryEmitsOnlyCatalogClarification 验证嵌套模型只能提出当前目录中可查询的指标口径。
func TestAskBiQueryEmitsOnlyCatalogClarification(t *testing.T) {
	for _, test := range []struct {
		name    string
		answer  string
		allowed bool
	}{
		{name: "当前项目可用指标", answer: `{"clarification":{"source_code":"orders","measure_codes":["paid","net"]}}`, allowed: true},
		{name: "模型虚构指标", answer: `{"clarification":{"source_code":"orders","measure_codes":["paid","invented"]}}`},
		{name: "其他项目数据源", answer: `{"clarification":{"source_code":"another_project","measure_codes":["paid","net"]}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cards, _ := newBiWorkflowTestCases(t, test.answer)
			queries := cards.biQueryCase
			err := queries.biSourceCase.registry.Register(biClarificationTestSource{})
			if err != nil {
				t.Fatal(err)
			}
			var client *model.AssistantClient
			client, err = queries.modelStore.ResolveAssistantClient(1, "test")
			if err != nil {
				t.Fatal(err)
			}
			ctx := agent.ContextWithAssistantClient(biCardTestContext(1, 10, 42), client)
			handler := middleware.NewToolMetricsHandler(nil, nil)
			var wrapped func(context.Context, string, ...tool.Option) (string, error)
			wrapped, err = handler.WrapInvokableToolCall(ctx, func(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
				_, askErr := queries.AskBiQuery(ctx, &adminv1.AskBiQueryRequest{Question: "统计销售额"})
				return "", askErr
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var output string
			output, err = wrapped(ctx, "{}")
			if err != nil {
				t.Fatal(err)
			}
			blocks := buildBiCardGuideBlocks(agent.ToolUsage{Status: "error", Input: `{"question":"统计销售额"}`, Output: output})
			if !test.allowed {
				if len(blocks) != 0 || strings.Contains(output, "invented") || strings.Contains(output, "another_project") {
					t.Fatalf("不可用口径不得进入对话: %s %v", output, blocks)
				}
				return
			}
			if len(blocks) != 1 || blocks[0]["type"] != "bi_query_clarification" || blocks[0]["question"] != "统计销售额" {
				t.Fatalf("缺少可交互的澄清选项: %s %v", output, blocks)
			}
			if !strings.Contains(output, "实付销售额") || !strings.Contains(output, "净销售额") || !strings.Contains(output, "订单数据") {
				t.Fatalf("澄清应使用目录的原始名称: %s", output)
			}
		})
	}
}

// biClarificationTestSource 提供可由当前查询范围读取的指标目录。
type biClarificationTestSource struct{ biQueryRecordTestSource }

// Code 返回澄清测试使用的数据源编码。
func (biClarificationTestSource) Code() string { return "orders" }

// Descriptor 返回两项已定义的销售额指标。
func (biClarificationTestSource) Descriptor(context.Context, bi.Scope) (bi.SourceDescriptor, error) {
	return bi.SourceDescriptor{Code: "orders", Name: "订单数据", Measures: []bi.MeasureDescriptor{
		{Code: "paid", Name: "实付销售额", Type: bi.FieldTypeNumber, Unit: "元"},
		{Code: "net", Name: "净销售额", Type: bi.FieldTypeNumber, Unit: "元"},
	}}, nil
}

// TestBuildBiCardGuideBlocksHandlesEmptyRows 验证空结果保留真实查询范围并省略建卡动作。
func TestBuildBiCardGuideBlocksHandlesEmptyRows(t *testing.T) {
	blocks := buildBiCardGuideBlocks(agent.ToolUsage{Status: "success", Input: `{"question":"统计今天的订单"}`, Output: `{"query":{"source_code":"test","time_range":{"field":"date","start":"2026-11-25","end":"2026-11-26","end_exclusive":true}},"result":{"query_record_id":7}}`})
	if len(blocks) != 1 || blocks[0]["type"] != "bi_query_empty" || blocks[0]["question"] != "统计今天的订单" || blocks[0]["query_record_id"] != nil || blocks[0]["query"] == nil {
		t.Fatalf("空结果没有保留查询范围或错误地展示建卡来源: %v", blocks)
	}
}

// TestBuildBiCardGuideBlocksUsesTypedPermission 验证权限提示依赖服务端原因，不从文案推断。
func TestBuildBiCardGuideBlocksUsesTypedPermission(t *testing.T) {
	blocks := buildBiCardGuideBlocks(agent.ToolUsage{Status: "error", Output: `{"error":"internal summary","reason":"PERMISSION_DENIED"}`})
	if len(blocks) != 1 || blocks[0]["type"] != "bi_permission_denied" || blocks[0]["code"] != "PERMISSION_DENIED" {
		t.Fatalf("权限原因未生成结构化提示: %v", blocks)
	}
	blocks = buildBiCardGuideBlocks(agent.ToolUsage{Status: "error", Output: `{"error":"PERMISSION_DENIED"}`})
	if len(blocks) != 0 {
		t.Fatalf("文案不应生成权限提示: %v", blocks)
	}
}
