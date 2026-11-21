package biz

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/plugin"
)

// TestAgentWorkflowToolSchema 验证审批任务参数使用整数枚举契约。
func TestAgentWorkflowToolSchema(t *testing.T) {
	registered, err := adminv1.NewWorkflowTaskServicePageMyWorkflowTaskAgentTool(nil)
	if err != nil {
		t.Fatal(err)
	}
	var info *schema.ToolInfo
	info, err = registered.Info(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var value *jsonschema.Schema
	value, err = info.ToJSONSchema()
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = agentToolSchema("/system.admin.v1.WorkflowTaskService/PageMyWorkflowTask", raw)
	if err != nil {
		t.Fatal(err)
	}
	var exposed tool.BaseTool
	exposed, err = agentToolWithSchema(t.Context(), registered, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	info, err = exposed.Info(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	value, err = info.ToJSONSchema()
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		value string
		valid bool
	}{
		{`1`, true}, {`"WORKFLOW_LIST_TYPE_PENDING"`, true},
		{`"1"`, false}, {`"pending"`, false}, {`999`, false},
	} {
		err = plugin.ValidateArguments(string(raw), `{"list_type":`+test.value+`}`)
		if (err == nil) != test.valid {
			t.Errorf("枚举 %s 的工具契约校验结果不符: %v", test.value, err)
		}
	}
}

// TestPrepareAgentToolArguments 验证枚举名与数值均可执行，非法值和重复字段仍被拒绝。
func TestPrepareAgentToolArguments(t *testing.T) {
	for _, value := range []string{`1`, `"WORKFLOW_LIST_TYPE_PENDING"`} {
		prepared, err := prepareAgentToolArguments(t.Context(), "/system.admin.v1.WorkflowTaskService/PageMyWorkflowTask", fmt.Sprintf(`{"list_type":%s,"page_num":1,"page_size":10}`, value))
		if err != nil || !strings.Contains(prepared, `"list_type":1`) {
			t.Fatalf("合法枚举未转换为执行参数: %s, %v", prepared, err)
		}
	}
	for _, arguments := range []string{
		`{"list_type":999,"page_num":1,"page_size":10}`,
		`{"list_type":1,"listType":2,"page_num":1,"page_size":10}`,
		`{"list_type":"1","page_num":1,"page_size":10}`,
	} {
		_, err := prepareAgentToolArguments(t.Context(), "/system.admin.v1.WorkflowTaskService/PageMyWorkflowTask", arguments)
		if err == nil {
			t.Fatalf("非法参数被放行: %s", arguments)
		}
	}
}
