package logmiddleware

import (
	"encoding/json"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestBuildEventConsentUsesValidJSON 验证协议确认审计事件的 JSON 字段始终可入库。
func TestBuildEventConsentUsesValidJSON(t *testing.T) {
	middleware := &Middleware{}
	event, ok, err := middleware.buildEvent(adminTask{
		EventID: "event-1",
		Kind:    "consent",
		Request: request{AgreementVersion: "version", AgreementSite: "2", UserID: 1, UserName: "super"},
	})
	if err != nil {
		t.Fatalf("构造协议确认审计事件失败: %v", err)
	}
	if !ok {
		t.Fatal("协议确认审计事件未生成")
	}

	var item models.BaseOperationLog
	err = json.Unmarshal(event.Payload, &item)
	if err != nil {
		t.Fatalf("解析协议确认审计载荷失败: %v", err)
	}
	for field, value := range map[string]string{
		"changed_fields": item.ChangedFields,
		"before_data":    item.BeforeData,
		"after_data":     item.AfterData,
	} {
		if !json.Valid([]byte(value)) {
			t.Fatalf("%s 不是合法 JSON: %q", field, value)
		}
	}
	if item.ChangedFields != "[]" {
		t.Fatalf("changed_fields = %q，期望 []", item.ChangedFields)
	}
	if item.BeforeData != "{}" {
		t.Fatalf("before_data = %q，期望 {}", item.BeforeData)
	}
}

// TestNormalizeAdminLogJSONFillsEmptyValues 验证历史审计载荷的空 JSON 字段可以安全重放。
func TestNormalizeAdminLogJSONFillsEmptyValues(t *testing.T) {
	operation := &models.BaseOperationLog{}
	permission := &models.BasePermissionLog{}
	dataAccess := &models.BaseDataAccessLog{}

	normalizeAdminLogJSON(operation)
	normalizeAdminLogJSON(permission)
	normalizeAdminLogJSON(dataAccess)

	if operation.ChangedFields != "[]" || operation.BeforeData != "{}" || operation.AfterData != "{}" {
		t.Fatalf("操作日志 JSON 默认值错误: %+v", operation)
	}
	if permission.OldValue != "{}" || permission.NewValue != "{}" {
		t.Fatalf("权限日志 JSON 默认值错误: %+v", permission)
	}
	if dataAccess.FieldScope != "[]" {
		t.Fatalf("数据访问日志 JSON 默认值错误: %q", dataAccess.FieldScope)
	}
}
