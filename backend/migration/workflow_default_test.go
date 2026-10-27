package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
)

// TestWorkflowDefaultSQL 验证双数据库初始化脚本中的默认审批单都能被工作流引擎解析。
func TestWorkflowDefaultSQL(t *testing.T) {
	pattern := regexp.MustCompile(`VALUES \(\d+, 1, '[^']+', '([^']+)', 1, '([^']+)', '([^']+)'`)
	expected := map[string][]string{
		"BASE_TENANT_CREATE":         {"name", "contact_name", "contact_phone", "remark"},
		"BASE_TENANT_PROJECT_CREATE": {"tenant_id", "code", "name", "sort", "remark"},
	}
	for _, dataSource := range []string{"mysql", "postgres"} {
		t.Run(dataSource, func(t *testing.T) {
			path := filepath.Join("assets", "v0.0.1", dataSource, "workflow_default.up.sql")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("读取初始化脚本失败: %v", err)
			}
			matches := pattern.FindAllStringSubmatch(string(raw), -1)
			definitions := make(map[string][]string, len(matches))
			for _, match := range matches {
				code, nodeConfig, formSchema := match[1], match[2], match[3]
				fields, ok := expected[code]
				if !ok {
					t.Fatalf("发现未预期的默认流程: %s", code)
				}
				if _, exists := definitions[code]; exists {
					t.Fatalf("默认流程编码重复: %s", code)
				}
				definitions[code] = fields
				graph, err := workflow.Parse(nodeConfig)
				if err != nil {
					t.Fatalf("%s 节点图解析失败: %v", code, err)
				}
				if err = workflow.Validate(graph); err != nil {
					t.Fatalf("%s 节点图校验失败: %v", code, err)
				}
				approvalNode := graph.Nodes["admin_approval"]
				if len(approvalNode.Approvers) != 2 {
					t.Fatalf("%s 默认审批角色数量错误: got %d, want 2", code, len(approvalNode.Approvers))
				}
				for index, wantID := range []int64{1, 3} {
					approval := approvalNode.Approvers[index]
					if approval.Type != workflow.ApproverTypeRole || approval.ID != wantID {
						t.Fatalf("%s 默认审批角色错误: got %#v, want ROLE:%d", code, approval, wantID)
					}
				}
				if !approvalNode.IsAnyStrategy() {
					t.Fatalf("%s 默认审批策略不是或签: got %s", code, approvalNode.Strategy)
				}
				var formFields []struct {
					Key string `json:"key"`
				}
				if err = json.Unmarshal([]byte(formSchema), &formFields); err != nil {
					t.Fatalf("%s 表单定义解析失败: %v", code, err)
				}
				if len(formFields) != len(fields) {
					t.Fatalf("%s 表单字段数量错误: got %d, want %d", code, len(formFields), len(fields))
				}
				for index, field := range fields {
					if formFields[index].Key != field {
						t.Fatalf("%s 表单字段 %d 错误: got %s, want %s", code, index, formFields[index].Key, field)
					}
				}
			}
			if len(definitions) != len(expected) {
				t.Fatalf("默认审批单数量错误: got %d, want %d", len(definitions), len(expected))
			}
		})
	}
}
