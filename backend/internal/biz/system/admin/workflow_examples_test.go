package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// TestWorkflowClassicExamples 使用经典流程配置，通过真实业务入口验证每一步待办、终态和业务回调。
func TestWorkflowClassicExamples(t *testing.T) {
	// workflowExampleStep 描述一次办理动作及其提交后的完整待办集合。
	type workflowExampleStep struct {
		node    string
		user    int64
		action  adminv1.WorkflowAuditAction
		target  int64
		pending []string
	}
	approve := adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE
	reject := adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_REJECT
	forward := adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_FORWARD
	add := adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_ADD_APPROVER
	tests := []struct {
		name     string
		file     string
		form     string
		initial  []string
		steps    []workflowExampleStep
		status   int32
		canceled int
		withdraw bool
	}{
		{name: "请假_逐级通过", file: "leave-serial", initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: approve, pending: []string{"hr:3"}}, {node: "hr", user: 3, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "请假_主管驳回不产生人事待办", file: "leave-serial", initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: reject}}, status: workflowStatusInstanceRejected},
		{name: "请假_人事驳回", file: "leave-serial", initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: approve, pending: []string{"hr:3"}}, {node: "hr", user: 3, action: reject}}, status: workflowStatusInstanceRejected},
		{name: "请假_申请人撤回", file: "leave-serial", initial: []string{"supervisor:2"}, status: workflowStatusInstanceWithdrawn, canceled: 1, withdraw: true},
		{name: "请假_转办后继续逐级审批", file: "leave-serial", initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: forward, target: 4, pending: []string{"supervisor:4"}}, {node: "supervisor", user: 4, action: approve, pending: []string{"hr:3"}}, {node: "hr", user: 3, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "请假_加签完成后才进入人事", file: "leave-serial", initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: add, target: 4, pending: []string{"supervisor:2", "supervisor:4"}}, {node: "supervisor", user: 2, action: approve, pending: []string{"supervisor:4"}}, {node: "supervisor", user: 4, action: approve, pending: []string{"hr:3"}}, {node: "hr", user: 3, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "报销_999元走默认分支", file: "expense-exclusive", form: `{"category":"办公","amount":999,"reason":"测试报销"}`, initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "报销_1000元边界走默认分支", file: "expense-exclusive", form: `{"category":"办公","amount":1000,"reason":"测试报销"}`, initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "报销_1001元增加财务审批", file: "expense-exclusive", form: `{"category":"办公","amount":1001,"reason":"测试报销"}`, initial: []string{"supervisor:2"}, steps: []workflowExampleStep{{node: "supervisor", user: 2, action: approve, pending: []string{"finance:3"}}, {node: "finance", user: 3, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "合同_会签等待全员通过", file: "contract-all", initial: []string{"joint_review:2", "joint_review:3"}, steps: []workflowExampleStep{{node: "joint_review", user: 2, action: approve, pending: []string{"joint_review:3"}}, {node: "joint_review", user: 3, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "合同_任一驳回取消其他待办", file: "contract-all", initial: []string{"joint_review:2", "joint_review:3"}, steps: []workflowExampleStep{{node: "joint_review", user: 3, action: reject}}, status: workflowStatusInstanceRejected, canceled: 1},
		{name: "值班_第一人通过取消另一待办", file: "duty-any", initial: []string{"duty_review:2", "duty_review:3"}, steps: []workflowExampleStep{{node: "duty_review", user: 2, action: approve}}, status: workflowStatusInstanceApproved, canceled: 1},
		{name: "值班_第二人通过同样有效", file: "duty-any", initial: []string{"duty_review:2", "duty_review:3"}, steps: []workflowExampleStep{{node: "duty_review", user: 3, action: approve}}, status: workflowStatusInstanceApproved, canceled: 1},
		{name: "值班_先驳回则整单驳回", file: "duty-any", initial: []string{"duty_review:2", "duty_review:3"}, steps: []workflowExampleStep{{node: "duty_review", user: 2, action: reject}}, status: workflowStatusInstanceRejected, canceled: 1},
		{name: "采购_财务先完成仍等待法务汇聚", file: "purchase-parallel", initial: []string{"finance:2", "legal:3"}, steps: []workflowExampleStep{{node: "finance", user: 2, action: approve, pending: []string{"legal:3"}}, {node: "legal", user: 3, action: approve, pending: []string{"manager:4"}}, {node: "manager", user: 4, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "采购_法务先完成仍等待财务汇聚", file: "purchase-parallel", initial: []string{"finance:2", "legal:3"}, steps: []workflowExampleStep{{node: "legal", user: 3, action: approve, pending: []string{"finance:2"}}, {node: "finance", user: 2, action: approve, pending: []string{"manager:4"}}, {node: "manager", user: 4, action: approve}}, status: workflowStatusInstanceApproved},
		{name: "采购_分支驳回取消另一个分支", file: "purchase-parallel", initial: []string{"finance:2", "legal:3"}, steps: []workflowExampleStep{{node: "legal", user: 3, action: reject}}, status: workflowStatusInstanceRejected, canceled: 1},
		{name: "采购_汇聚等待中撤回清理待办", file: "purchase-parallel", initial: []string{"finance:2", "legal:3"}, steps: []workflowExampleStep{{node: "finance", user: 2, action: approve, pending: []string{"legal:3"}}}, status: workflowStatusInstanceWithdrawn, canceled: 1, withdraw: true},
	}
	notification.SetDefaultPublisher(notification.PublisherFunc(func(context.Context, notification.Message) (int64, error) { return 1, nil }))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			definition := createClassicWorkflowDefinition(t, c, store, tt.file)
			ctx := workflowTestContext(1, 1, gorm.DataScopeSelfUser)
			formData := tt.form
			if formData == "" {
				formData = classicWorkflowFormData(t, tt.file)
			}
			var events []adminv1.WorkflowInstanceStatus
			c.eventRegistry.Register("CLASSIC_TEST", func(_ context.Context, _, _, _ int64, status adminv1.WorkflowInstanceStatus) {
				events = append(events, status)
			})
			instanceID, err := c.SubmitWorkflowInstance(ctx, &adminv1.SubmitWorkflowInstanceRequest{Code: definition.Code, Version: definition.Version, FormData: formData, BusinessType: "CLASSIC_TEST", BusinessId: 1})
			if err != nil {
				t.Fatal(err)
			}
			assertClassicWorkflowState(t, c, instanceID, workflowStatusInstancePending, tt.initial)
			for i, step := range tt.steps {
				var tasks []*models.WorkflowTask
				tasks, err = c.listTasks(context.Background(), 1, instanceID)
				if err != nil {
					t.Fatal(err)
				}
				var taskID int64
				for _, task := range tasks {
					if task.NodeID == step.node && task.AssigneeUserID == step.user && task.Status == workflowStatusTaskPending {
						taskID = task.ID
					}
				}
				if taskID == 0 {
					t.Fatalf("第 %d 步找不到待办 %s:%d", i+1, step.node, step.user)
				}
				err = c.AuditWorkflowTask(workflowTestContext(1, step.user, gorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: taskID, Action: step.action, ForwardTo: step.target, AdditionalApprover: step.target, Comment: tt.name})
				if err != nil {
					t.Fatalf("第 %d 步审批失败: %v", i+1, err)
				}
				wantStatus := workflowStatusInstancePending
				if len(step.pending) == 0 {
					wantStatus = tt.status
				}
				assertClassicWorkflowState(t, c, instanceID, wantStatus, step.pending)
				if wantStatus == workflowStatusInstancePending && len(events) != 0 {
					t.Fatal("审批未结束时提前触发了终态回调")
				}
			}
			if tt.withdraw {
				err = c.WithdrawWorkflowInstance(ctx, instanceID)
				if err != nil {
					t.Fatal(err)
				}
			}
			assertClassicWorkflowState(t, c, instanceID, tt.status, nil)
			var tasks []*models.WorkflowTask
			tasks, err = c.listTasks(context.Background(), 1, instanceID)
			if err != nil {
				t.Fatal(err)
			}
			canceled := 0
			for _, task := range tasks {
				if task.Status == workflowStatusTaskCanceled {
					canceled++
				}
				if task.Status == workflowStatusTaskCanceled || task.Status == workflowStatusTaskApproved || task.Status == workflowStatusTaskRejected {
					err = c.AuditWorkflowTask(workflowTestContext(1, task.AssigneeUserID, gorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: task.ID, Action: approve})
					if err == nil {
						t.Fatalf("终结后仍可办理任务 %d", task.ID)
					}
				}
			}
			if canceled != tt.canceled {
				t.Fatalf("取消任务数量不符: got=%d want=%d", canceled, tt.canceled)
			}
			if len(events) != 1 || int32(events[0]) != tt.status {
				t.Fatalf("终态回调应恰好一次且状态正确: %v", events)
			}
		})
	}
}

// createClassicWorkflowDefinition 读取可复用示例，经定义创建和启用入口完成配置校验与持久化。
func createClassicWorkflowDefinition(t *testing.T, c *WorkflowInstanceCase, store *data.Data, name string) *adminv1.WorkflowDefinitionForm {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("../../../../testdata/workflow", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var example struct {
		Name       string          `json:"name"`
		Code       string          `json:"code"`
		Version    int32           `json:"version"`
		Remark     string          `json:"remark"`
		FormSchema json.RawMessage `json:"form_schema"`
		NodeConfig json.RawMessage `json:"node_config"`
	}
	err = json.Unmarshal(content, &example)
	if err != nil {
		t.Fatal(err)
	}
	form := &adminv1.WorkflowDefinitionForm{Name: example.Name, Code: example.Code, Version: example.Version, Remark: example.Remark, FormSchema: string(example.FormSchema), NodeConfig: string(example.NodeConfig)}
	definitionCase := NewWorkflowDefinitionCase(c.BaseCase, store, c.definitionRepo, c.WorkflowInstanceRepository, data.NewBaseScriptRepository(store), NewBaseScriptCase(c.BaseCase, store, data.NewBaseScriptRepository(store), NewBaseScriptLogCase(c.BaseCase, c.scriptLogRepo), c.runtime))
	ctx := workflowTestContext(1, 1, gorm.DataScopeAll)
	err = definitionCase.CreateWorkflowDefinition(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	var page *adminv1.PageWorkflowDefinitionResponse
	page, err = definitionCase.PageWorkflowDefinition(ctx, &adminv1.PageWorkflowDefinitionRequest{Code: example.Code, PageNum: 1, PageSize: 10})
	if err != nil || len(page.GetDefinitions()) != 1 {
		t.Fatalf("无法读取创建的流程定义: page=%v err=%v", page, err)
	}
	form.Id = page.Definitions[0].Id
	if int32(page.Definitions[0].Status) != workflowStatusDefinitionDraft {
		t.Fatal("新流程定义应为草稿")
	}
	err = definitionCase.SetWorkflowDefinitionStatus(ctx, &adminv1.SetWorkflowDefinitionStatusRequest{Id: form.Id, Status: adminv1.WorkflowDefinitionStatus_WORKFLOW_DEFINITION_STATUS_ENABLED})
	if err != nil {
		t.Fatal(err)
	}
	var applyForm *adminv1.GetWorkflowApplyFormResponse
	applyForm, err = definitionCase.GetWorkflowApplyForm(ctx, &adminv1.GetWorkflowApplyFormRequest{Code: form.Code, Version: form.Version})
	if err != nil || applyForm.GetFormSchema() != form.FormSchema {
		t.Fatalf("申请表单读取不一致: form=%v err=%v", applyForm, err)
	}
	return form
}

// classicWorkflowFormData 提供与五组申请表单一致的完整测试数据。
func classicWorkflowFormData(t *testing.T, name string) string {
	t.Helper()
	forms := map[string]string{
		"leave-serial":      `{"leave_type":"年假","start_date":"2026-10-26","days":2,"reason":"测试请假"}`,
		"contract-all":      `{"contract_name":"测试采购合同","amount":50000,"reason":"测试合同会签"}`,
		"duty-any":          `{"duty_date":"2026-10-26","shift":"白班","reason":"测试调班"}`,
		"purchase-parallel": `{"item_name":"测试办公设备","quantity":2,"amount":12000,"reason":"测试采购"}`,
	}
	form, ok := forms[name]
	if !ok {
		t.Fatalf("缺少测试表单: %s", name)
	}
	return form
}

// assertClassicWorkflowState 比对实例状态与全部待办，防止漏展开分支或提前推进。
func assertClassicWorkflowState(t *testing.T, c *WorkflowInstanceCase, instanceID int64, status int32, pending []string) {
	t.Helper()
	instance, err := c.FindByID(context.Background(), instanceID)
	if err != nil {
		t.Fatal(err)
	}
	var tasks []*models.WorkflowTask
	tasks, err = c.listTasks(context.Background(), 1, instanceID)
	if err != nil {
		t.Fatal(err)
	}
	var actual []string
	for _, task := range tasks {
		if task.Status == workflowStatusTaskPending {
			actual = append(actual, fmt.Sprintf("%s:%d", task.NodeID, task.AssigneeUserID))
		}
	}
	want := slices.Clone(pending)
	slices.Sort(actual)
	slices.Sort(want)
	if instance.Status != status || !slices.Equal(actual, want) {
		t.Fatalf("流转不符: status=%d want=%d, pending=%v want=%v", instance.Status, status, actual, want)
	}
	if status != workflowStatusInstancePending && (instance.EndAt == nil || instance.EndAt.IsZero()) {
		t.Fatal("终结实例未记录结束时间")
	}
}
