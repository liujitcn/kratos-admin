package admin

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// TestBuildBiCardGuideBlocksPreservesChartType 验证助手建卡引导保留问数建议图表类型。
func TestBuildBiCardGuideBlocksPreservesChartType(t *testing.T) {
	blocks := buildBiCardGuideBlocks(agent.ToolUsage{Output: `{"title":"指标概览","chart_type":"metric_grid","query":{"source_code":"test"},"result":{"query_record_id":7,"rows":[{"values":["1"]}]}}`})
	if len(blocks) != 1 || blocks[0]["chart_type"] != "metric_grid" || blocks[0]["dialog"] != "bi_card_guide" {
		t.Fatalf("建卡引导未保留图表类型: %v", blocks)
	}
}
