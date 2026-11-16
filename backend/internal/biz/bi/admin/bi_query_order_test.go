package admin

import (
	"encoding/json"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestConfiguredSQLPreservesDatabaseOrder 防止返回前再次按数值排序破坏数据库字符串排序和 Top-N 顺序。
func TestConfiguredSQLPreservesDatabaseOrder(t *testing.T) {
	cards, _ := newBiWorkflowTestCases(t)
	db := cards.GormClients[gorm.DefaultClientName].DB
	err := db.AutoMigrate(&modelTestOrder{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&[]modelTestOrder{{ID: 1, TenantID: 1, Shop: "2"}, {ID: 2, TenantID: 1, Shop: "10"}, {ID: 3, TenantID: 1, Shop: "9"}}).Error; err != nil {
		t.Fatal(err)
	}
	model := dto.ConfiguredSourceSchema{BaseTable: "orders", ModelRevision: 1, Tables: []dto.ConfiguredSourceTable{{Key: "orders", Name: "model_test_orders", TenantColumn: "tenant_id", Fields: map[string]bi.FieldType{"shop": bi.FieldTypeString}}}, Dimensions: []dto.ConfiguredSourceDimension{{Code: "shop", Name: "门店", Type: bi.FieldTypeString, Table: "orders", Column: "shop"}}, Measures: []dto.ConfiguredSourceMeasure{{Code: "count", Name: "笔数", Type: bi.FieldTypeNumber, Expression: "COUNT(*)", Revision: 1}}}
	var payload []byte
	payload, err = json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	identity := &data.UserTokenPayload{TenantId: 1, TenantCode: gorm.DefaultTenantCode, UserId: 10, DataScope: gorm.DataScopeAll}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	if err = cards.biQueryCase.biSourceCase.sourceRepo.Create(ctx, &models.BiSource{TenantID: 1, Code: "ordered", Name: "排序", SourceName: gorm.DefaultClientName, Status: 1, Version: 1, MaxRows: 100, SchemaJSON: string(payload)}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		direction string
		want      []string
	}{
		{direction: "asc", want: []string{"10", "2"}},
		{direction: "desc", want: []string{"9", "2"}},
	} {
		t.Run(test.direction, func(t *testing.T) {
			var result bi.QueryResult
			result, err = cards.biQueryCase.execute(ctx, bi.Scope{TenantID: 1, UserID: 10}, bi.QuerySpec{SourceCode: "ordered", Dimensions: []string{"shop"}, Measures: []bi.MeasureSpec{{Code: "count"}}, Orders: []bi.OrderSpec{{Field: "shop", Direction: test.direction}}, Limit: 2}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Rows) != 2 || !result.Truncated {
				t.Fatal("Top-N 未正确裁剪或缺少截断标记")
			}
			for index, expected := range test.want {
				if result.Rows[index][0] != expected {
					t.Errorf("第%d行=%s，数据库字符串排序应为%s", index+1, result.Rows[index][0], expected)
				}
			}
		})
	}
}
