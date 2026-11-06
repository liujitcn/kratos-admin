package biz

import (
	"context"
	"encoding/json"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
)

// TestAgentToolPolicy 验证写接口保守审批且执行入口不对 Agent 开放。
func TestAgentToolPolicy(t *testing.T) {
	for _, operation := range []string{"/system.admin.v1.BaseScriptService/TestRunBaseScript", "/system.admin.v1.CodeGenService/StartCodeGenTask", "/system.admin.v1.BaseJobService/ExecuteBaseJob", "/system.admin.v1.WorkflowInstanceService/StartWorkflowInstance"} {
		if !agentToolBlocked(&models.BaseAPI{Method: "POST", Operation: operation}) {
			t.Fatalf("执行工具未封闭: %s", operation)
		}
	}
	if !agentToolBlocked(&models.BaseAPI{ToolName: "base_v1_ai_message_service_decide_ai_tool_approval", Method: "POST"}) {
		t.Fatal("模型能够审批自己的操作")
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", ""} {
		if agentToolReadOnly(&models.BaseAPI{Method: method}) {
			t.Fatalf("写接口被自动放行: %s", method)
		}
	}
}

// TestPrepareAgentToolArguments 验证动态工具入口使用真实 Proto 校验并补全和隔离租户。
func TestPrepareAgentToolArguments(t *testing.T) {
	payload := &data.UserTokenPayload{TenantId: 7, TenantCode: "tenant7", UserId: 3, RoleCode: "role"}
	ctx := engine.ContextWithAuthClaims(context.Background(), payload.MakeAuthClaims())
	operation := "/system.admin.v1.BaseUserService/PageBaseUser"
	value, err := prepareAgentToolArguments(ctx, operation, `{"page_num":1,"page_size":10}`)
	if err != nil {
		t.Fatal(err)
	}
	var request adminv1.PageBaseUserRequest
	if err = json.Unmarshal([]byte(value), &request); err != nil {
		t.Fatal(err)
	}
	if request.GetTenantId() != 7 {
		t.Fatalf("缺省租户未补全: %s", value)
	}
	for _, args := range []string{`{"tenant_id":8,"page_num":1,"page_size":10}`, `{"page_num":1,"page_size":101}`, `{"unknown":1}`} {
		_, err = prepareAgentToolArguments(ctx, operation, args)
		if err == nil {
			t.Fatalf("无效参数通过校验: %s", args)
		}
	}
}
