package core

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/resource/openapi/dto"
)

// TestSchemaContainsTenantMessage 验证响应字段树只识别包含 tenant_id 的 Proto 消息。
func TestSchemaContainsTenantMessage(t *testing.T) {
	_ = adminv1.BaseRedactOutputPolicy{}
	tests := []struct {
		name   string
		schema *dto.OpenAPISchema
		want   bool
	}{
		{name: "nil", schema: nil, want: false},
		{name: "普通消息", schema: &dto.OpenAPISchema{Ref: "#/components/schemas/system.admin.v1.BaseApiDoc"}, want: false},
		{name: "租户消息", schema: &dto.OpenAPISchema{Ref: "#/components/schemas/system.admin.v1.BaseRedactOutputPolicy"}, want: true},
		{name: "嵌套租户消息", schema: &dto.OpenAPISchema{Children: []*dto.OpenAPISchema{{Ref: "#/components/schemas/system.admin.v1.BaseRedactOutputPolicy"}}}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := schemaContainsTenantMessage(test.schema); got != test.want {
				t.Fatalf("schemaContainsTenantMessage() = %v, want %v", got, test.want)
			}
		})
	}
}
