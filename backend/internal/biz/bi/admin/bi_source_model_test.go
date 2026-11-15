package admin

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// TestModelRevisions 验证指标删除重建的身份单调且画布移动不改变结构身份。
func TestModelRevisions(t *testing.T) {
	old := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 2, Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "orders"}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "sales", Name: "销售额", Expression: "SUM(amount)", Type: bi.FieldTypeNumber, Revision: 5}}}
	candidate := old
	candidate.Tables = []dto.ConfiguredSourceTable{{Key: "orders", Name: "orders", X: 600, Y: 300}}
	candidate = reviseSourceModel(old, candidate, 10)
	if candidate.ModelRevision != 2 || candidate.Measures[0].Revision != 5 {
		t.Fatal("画布坐标使公共指标失效")
	}
	removed := old
	removed.Measures = nil
	candidate = reviseSourceModel(removed, old, 12)
	if candidate.Measures[0].Revision != 12 {
		t.Fatal("删除重建指标复用了旧修订")
	}
	candidate.Measures = []dto.ConfiguredSourceMeasure{{Code: "sales", Name: "净销售额", Expression: "SUM(amount)", Type: bi.FieldTypeNumber, Unit: "元"}}
	candidate = reviseSourceModel(old, candidate, 13)
	if candidate.Measures[0].Revision != 13 {
		t.Fatal("名称单位变化未修改指标修订")
	}
}
