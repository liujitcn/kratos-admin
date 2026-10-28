package biz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	"github.com/liujitcn/kratos-admin/backend/pkg/scripting"
	pkgWorkflow "github.com/liujitcn/kratos-admin/backend/pkg/workflow"
	"github.com/liujitcn/kratos-core/biz"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/database/gorm/driver"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestSelectExclusiveEdgeReturnsConditionError 验证排他网关条件错误不会被当作未命中。
func TestSelectExclusiveEdgeReturnsConditionError(t *testing.T) {
	node := &workflow.Node{
		OutEdges: []workflow.Edge{
			{To: "approved", Condition: "amount >"},
			{To: "rejected", Condition: workflow.ConditionDefault},
		},
	}

	_, _, err := selectExclusiveEdge(node, `{"amount": 100}`)
	if err == nil {
		t.Fatal("期望条件求值失败")
	}
}

// TestWorkflowAnyRoleApproverAutoApproval 验证 super/admin 角色申请自动通过，普通用户进入角色审批待办。
func TestWorkflowAnyRoleApproverAutoApproval(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	err := c.baseUserRepo.UpdateByID(context.Background(), &models.BaseUser{ID: 1, RoleID: 1})
	if err != nil {
		t.Fatal(err)
	}
	err = c.baseUserRepo.UpdateByID(context.Background(), &models.BaseUser{ID: 2, RoleID: 3})
	if err != nil {
		t.Fatal(err)
	}
	config := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"approval","type":"TASK","approvers":[{"type":"ROLE","id":1},{"type":"ROLE","id":3}],"strategy":"ANY"},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"approval"},{"from":"approval","to":"end"}]}`
	var graph *workflow.Graph
	graph, err = workflow.Parse(config)
	if err != nil {
		t.Fatal(err)
	}
	if err = workflow.Validate(graph); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name        string
		applicant   int64
		wantStatus  int32
		wantPending int64
	}{
		{name: "super申请自动通过", applicant: 1, wantStatus: workflowStatusInstanceApproved},
		{name: "admin申请自动通过", applicant: 2, wantStatus: workflowStatusInstanceApproved},
		{name: "普通用户进入审批", applicant: 4, wantStatus: workflowStatusInstancePending, wantPending: 2},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			instance := &models.WorkflowInstance{
				TenantID:        1,
				ApplicantUserID: testCase.applicant,
				FormData:        "{}",
				Variables:       "{}",
				NodeConfig:      config,
				BusinessType:    "ROLE_TEST",
				BusinessID:      testCase.applicant,
				Status:          workflowStatusInstancePending,
			}
			if err = c.Create(context.Background(), instance); err != nil {
				t.Fatal(err)
			}
			if err = c.enterTaskNode(context.Background(), 1, instance.ID, graph, graph.Nodes["approval"], newWalkerState()); err != nil {
				t.Fatal(err)
			}
			var current *models.WorkflowInstance
			current, err = c.FindByID(context.Background(), instance.ID)
			if err != nil {
				t.Fatal(err)
			}
			var pending int64
			pending, err = c.countPendingTasks(context.Background(), 1, instance.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.Status != testCase.wantStatus || pending != testCase.wantPending {
				t.Fatalf("流程状态错误: status=%d pending=%d", current.Status, pending)
			}
		})
	}
}

// TestWorkflowRejectCommitsTerminalStateTogether 验证驳回不会暴露任务已驳回但实例仍待审批的提交状态。
func TestWorkflowRejectCommitsTerminalStateTogether(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	instance, task := createWorkflowTestTask(t, c, 1, 2)
	c.tx = &workflowObservedTransaction{store: store, afterCommit: func() {
		currentTask, err := c.taskRepo.FindByID(context.Background(), task.ID)
		if err != nil {
			t.Fatal(err)
		}
		var currentInstance *models.WorkflowInstance
		currentInstance, err = c.FindByID(context.Background(), instance.ID)
		if err != nil {
			t.Fatal(err)
		}
		if currentTask.Status == workflowStatusTaskRejected && currentInstance.Status == workflowStatusInstancePending {
			t.Error("驳回任务单独提交，其他审批请求可在此时误判流程通过")
		}
	}}
	err := c.handleReject(context.Background(), 2, task, instance, "驳回")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWorkflowForkExpandsAllBranchesBeforeTermination 验证首分支自动通过不会跳过后续审批分支。
func TestWorkflowForkExpandsAllBranchesBeforeTermination(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, _ := createWorkflowTestTask(t, c, 1, 2)
	err := c.cancelPendingTasks(context.Background(), 1, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	graph := &workflow.Graph{Nodes: map[string]*workflow.Node{
		"fork":   {ID: "fork", Type: workflow.NodeTypeParallelFork, OutEdges: []workflow.Edge{{To: "auto"}, {To: "manual"}}},
		"auto":   {ID: "auto", Type: workflow.NodeTypeTask, Approvers: []workflow.Approver{{Type: workflow.ApproverTypeUser, ID: 1}}, OutEdges: []workflow.Edge{{To: "end"}}},
		"manual": {ID: "manual", Type: workflow.NodeTypeTask, Approvers: []workflow.Approver{{Type: workflow.ApproverTypeUser, ID: 2}}, OutEdges: []workflow.Edge{{To: "end"}}},
		"end":    {ID: "end", Type: workflow.NodeTypeEnd},
	}}
	err = c.advanceFromNode(context.Background(), 1, instance.ID, graph, "fork", newWalkerState())
	if err != nil {
		t.Fatal(err)
	}
	instance, err = c.FindByID(context.Background(), instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	var pending int64
	pending, err = c.countPendingTasks(context.Background(), 1, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if instance.Status != workflowStatusInstancePending || pending != 1 {
		t.Fatalf("应等待第二分支审批，实际 status=%d pending=%d", instance.Status, pending)
	}
	var tasks []*models.WorkflowTask
	tasks, err = c.listTasks(context.Background(), 1, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.Status == workflowStatusTaskPending {
			err = c.handleApprove(context.Background(), 2, task, graph, graph.Nodes["manual"], "通过")
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	instance, err = c.FindByID(context.Background(), instance.ID)
	if err != nil || instance.Status != workflowStatusInstanceApproved {
		t.Fatalf("全部分支完成后未通过: instance=%v err=%v", instance, err)
	}
}

// TestWorkflowNestedAutomaticForkCompletes 验证嵌套并行的全部自动分支均执行后才终结一次。
func TestWorkflowNestedAutomaticForkCompletes(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, _ := createWorkflowTestTask(t, c, 1, 2)
	err := c.cancelPendingTasks(context.Background(), 1, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	graph := &workflow.Graph{Nodes: map[string]*workflow.Node{
		"outer": {ID: "outer", Type: workflow.NodeTypeParallelFork, OutEdges: []workflow.Edge{{To: "inner"}, {To: "c"}}},
		"inner": {ID: "inner", Type: workflow.NodeTypeParallelFork, OutEdges: []workflow.Edge{{To: "a"}, {To: "b"}}},
		"end":   {ID: "end", Type: workflow.NodeTypeEnd},
	}}
	for _, id := range []string{"a", "b", "c"} {
		graph.Nodes[id] = &workflow.Node{ID: id, Type: workflow.NodeTypeTask, Approvers: []workflow.Approver{{Type: workflow.ApproverTypeUser, ID: 1}}, OutEdges: []workflow.Edge{{To: "end"}}}
	}
	fired := 0
	c.eventRegistry.Register("TEST", func(context.Context, int64, int64, int64, adminv1.WorkflowInstanceStatus) { fired++ })
	err = c.advanceFromNode(context.Background(), 1, instance.ID, graph, "outer", newWalkerState())
	if err != nil {
		t.Fatal(err)
	}
	var logs []*models.WorkflowLog
	logs, err = c.listLogs(context.Background(), 1, instance.ID)
	if err != nil || len(logs) != 3 || fired != 1 {
		t.Fatalf("嵌套并行分支未全部执行或重复终结: logs=%d fired=%d err=%v", len(logs), fired, err)
	}
}

// TestWorkflowTransferredTaskRejectsStaleActions 验证转办后旧办理人的预读请求不能继续审批或修改任务。
func TestWorkflowTransferredTaskRejectsStaleActions(t *testing.T) {
	for _, action := range []string{"approve", "reject", "forward", "add"} {
		t.Run(action, func(t *testing.T) {
			c, _ := newWorkflowTestCase(t)
			instance, task := createWorkflowTestTask(t, c, 1, 2)
			err := c.handleForward(context.Background(), 2, task, 3)
			if err != nil {
				t.Fatal(err)
			}
			node := &workflow.Node{ID: "task", Type: workflow.NodeTypeTask, OutEdges: []workflow.Edge{{To: "end"}}}
			graph := &workflow.Graph{Nodes: map[string]*workflow.Node{"task": node, "end": {ID: "end", Type: workflow.NodeTypeEnd}}}
			switch action {
			case "approve":
				err = c.handleApprove(context.Background(), 2, task, graph, node, "")
			case "reject":
				err = c.handleReject(context.Background(), 2, task, instance, "")
			case "forward":
				err = c.handleForward(context.Background(), 2, task, 4)
			case "add":
				err = c.handleAddApprover(context.Background(), 2, task, node, 4)
			}
			if err == nil {
				t.Error("旧办理人的请求应返回冲突")
			}
			var current *models.WorkflowTask
			current, err = c.taskRepo.FindByID(context.Background(), task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.AssigneeUserID != 3 || current.Status != workflowStatusTaskPending {
				t.Fatalf("转办结果被旧请求覆盖: %+v", current)
			}
		})
	}
}

// TestWorkflowAddApproverRejectsTerminalInstance 验证终结后到达的加签请求不会产生无法办理的待办。
func TestWorkflowAddApproverRejectsTerminalInstance(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, task := createWorkflowTestTask(t, c, 1, 2)
	err := c.terminateInstance(context.Background(), instance, workflowStatusInstanceRejected)
	if err != nil {
		t.Fatal(err)
	}
	err = c.handleAddApprover(context.Background(), 2, task, &workflow.Node{ID: "task"}, 3)
	if err == nil {
		t.Error("终结后的加签应返回冲突")
	}
	var pending int64
	pending, err = c.countPendingTasks(context.Background(), 1, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("终结实例新增了 %d 个待办", pending)
	}
}

// TestWorkflowTaskAccessUsesAssigneeWithinTenant 验证仅本人数据权限的办理人可访问他人申请且不跨租户。
func TestWorkflowTaskAccessUsesAssigneeWithinTenant(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, task := createWorkflowTestTask(t, c, 1, 2)
	_, _ = createWorkflowTestTask(t, c, 1, 3)
	_, _ = createWorkflowTestTask(t, c, 2, 2)
	taskCase := NewWorkflowTaskCase(c.BaseCase, c.taskRepo, c, c.WorkflowInstanceRepository, c.definitionRepo, c.baseUserRepo)
	ctx := workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser)
	list, err := taskCase.PageMyWorkflowTask(ctx, &adminv1.PageMyWorkflowTaskRequest{ListType: adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_PENDING, PageNum: 1, PageSize: 10, Keyword: "TEST"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].TaskId != task.ID {
		t.Fatalf("办理人待办范围错误: %+v", list)
	}
	for _, deniedCtx := range []context.Context{workflowTestContext(1, 4, databaseGorm.DataScopeSelfUser), workflowTestContext(2, 2, databaseGorm.DataScopeSelfUser)} {
		_, err = taskCase.GetWorkflowTask(deniedCtx, task.ID)
		if err == nil {
			t.Fatal("非办理人或其他租户读取了任务详情")
		}
		err = c.AuditWorkflowTask(deniedCtx, &adminv1.AuditWorkflowTaskRequest{TaskId: task.ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
		if err == nil {
			t.Fatal("非办理人或其他租户办理了任务")
		}
	}
	var detail *adminv1.GetWorkflowTaskResponse
	detail, err = taskCase.GetWorkflowTask(ctx, task.ID)
	if err != nil || detail.GetInstance().GetId() != instance.ID {
		t.Fatalf("无法读取他人申请的任务详情: detail=%v err=%v", detail, err)
	}
	err = c.AuditWorkflowTask(ctx, &adminv1.AuditWorkflowTaskRequest{TaskId: task.ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
	if err != nil {
		t.Fatalf("无法办理他人申请: %v", err)
	}
}

// TestWorkflowTaskUnspecifiedListsAllViews 验证未指定列表类型时返回当前用户的全部审批视角。
func TestWorkflowTaskUnspecifiedListsAllViews(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	_, doneTask := createWorkflowTestTask(t, c, 1, 2)
	submittedInstance, _ := createWorkflowTestTask(t, c, 1, 3)
	_, pendingTask := createWorkflowTestTask(t, c, 1, 4)
	err := c.UpdateByID(context.Background(), &models.WorkflowInstance{ID: submittedInstance.ID, ApplicantUserID: 2})
	if err != nil {
		t.Fatal(err)
	}
	pendingTask.AssigneeUserID = 2
	err = c.taskRepo.UpdateByID(context.Background(), pendingTask)
	if err != nil {
		t.Fatal(err)
	}
	err = c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{
		TaskId: doneTask.ID,
		Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE,
	})
	if err != nil {
		t.Fatal(err)
	}
	taskCase := NewWorkflowTaskCase(c.BaseCase, c.taskRepo, c, c.WorkflowInstanceRepository, c.definitionRepo, c.baseUserRepo)
	list, err := taskCase.PageMyWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.PageMyWorkflowTaskRequest{
		ListType: adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_UNSPECIFIED,
		PageNum:  1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 3 || len(list.Items) != 3 {
		t.Fatalf("全部审批列表结果错误: %+v", list)
	}
	listTypes := make(map[adminv1.WorkflowListType]bool, len(list.Items))
	for _, item := range list.Items {
		listTypes[item.ListType] = true
	}
	for _, listType := range []adminv1.WorkflowListType{
		adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_PENDING,
		adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_DONE,
		adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_SUBMITTED,
	} {
		if !listTypes[listType] {
			t.Fatalf("全部审批列表缺少视角 %v: %+v", listType, list.Items)
		}
	}
}

// TestWorkflowNotificationsUseCommittedContext 验证审批通知在事务提交后仍能查询数据库。
func TestWorkflowNotificationsUseCommittedContext(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, task := createWorkflowTestTask(t, c, 1, 2)
	release := make(chan struct{})
	result := make(chan error, 1)
	notification.SetDefaultPublisher(notification.PublisherFunc(func(ctx context.Context, _ notification.Message) (int64, error) {
		<-release
		current, err := c.FindByID(ctx, instance.ID)
		if err == nil && current.Status != workflowStatusInstanceApproved {
			err = errors.New("通知看到了未提交的审批结果")
		}
		result <- err
		return 1, err
	}))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })
	node := &workflow.Node{ID: "task", OutEdges: []workflow.Edge{{To: "end"}}}
	err := c.handleApprove(context.Background(), 2, task, &workflow.Graph{Nodes: map[string]*workflow.Node{"end": {ID: "end", Type: workflow.NodeTypeEnd}}}, node, "")
	close(release)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		if err != nil {
			t.Fatalf("通知上下文复用了已结束事务: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("未收到审批通知")
	}
}

// TestWorkflowTimeoutReminderRetriesAfterNotificationFailure 验证通知失败时不会提前写入催办标记。
func TestWorkflowTimeoutReminderRetriesAfterNotificationFailure(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	_, task := createWorkflowTestTask(t, c, 1, 2)
	now := time.Now()
	oldCreatedAt := now.Add(-workflowTimeoutReminder - time.Minute)
	taskQuery := c.taskRepo.Query(context.Background()).WorkflowTask
	_, err := taskQuery.WithContext(context.Background()).Where(taskQuery.ID.Eq(task.ID)).UpdateSimple(taskQuery.CreatedAt.Value(oldCreatedAt))
	if err != nil {
		t.Fatal(err)
	}
	var messages []notification.Message
	notification.SetDefaultPublisher(notification.PublisherFunc(func(_ context.Context, message notification.Message) (int64, error) {
		messages = append(messages, message)
		if len(messages) == 1 {
			return 0, errors.New("模拟催办通知失败")
		}
		return 1, nil
	}))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })

	reminded, err := c.scanWorkflowReminder(context.Background(), now)
	if err != nil || reminded != 0 {
		t.Fatalf("通知失败不应标记催办: reminded=%d err=%v", reminded, err)
	}
	current, err := c.taskRepo.FindByID(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.RemindedAt != nil {
		t.Fatal("通知失败后已写入催办标记")
	}

	reminded, err = c.scanWorkflowReminder(context.Background(), now)
	if err != nil || reminded != 1 {
		t.Fatalf("通知成功后未标记催办: reminded=%d err=%v", reminded, err)
	}
	current, err = c.taskRepo.FindByID(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.RemindedAt == nil || len(messages) != 2 || messages[0].IdempotencyKey != messages[1].IdempotencyKey {
		t.Fatalf("催办重试未使用稳定幂等键: task=%+v messages=%+v", current, messages)
	}
}

// TestWorkflowTaskNullableTimesRemainUnset 验证新建待办不会写入非法零日期。
func TestWorkflowTaskNullableTimesRemainUnset(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	_, task := createWorkflowTestTask(t, c, 1, 2)
	if task.RemindedAt != nil || task.EscalatedAt != nil {
		t.Fatalf("新待办的可空时间字段不应有值: task=%+v", task)
	}
	current, err := c.taskRepo.FindByID(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.RemindedAt != nil || current.EscalatedAt != nil {
		t.Fatalf("新待办落库后写入了零日期: task=%+v", current)
	}
}

// TestWorkflowTaskMySQLCreateDoesNotBindZeroDatetime 验证 MySQL 方言下新待办不会绑定零日期。
func TestWorkflowTaskMySQLCreateDoesNotBindZeroDatetime(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "root:invalid@tcp(127.0.0.1:3306)/kratos_admin", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	statement := db.Create(&models.WorkflowTask{TenantID: 1, InstanceID: 1, NodeID: "approval", AssigneeUserID: 2, Status: workflowStatusTaskPending}).Statement
	for _, value := range statement.Vars {
		if timestamp, ok := value.(time.Time); ok && timestamp.IsZero() {
			t.Fatalf("MySQL 新待办绑定了零日期: sql=%s vars=%v", statement.SQL.String(), statement.Vars)
		}
	}
}

// TestWorkflowInstanceBindingUsesTenantAndBusinessIdentity 验证系统动作绑定同时约束租户、实例和业务单据。
func TestWorkflowInstanceBindingUsesTenantAndBusinessIdentity(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, _ := createWorkflowTestTask(t, c, 1, 2)
	bound, err := workflowInstanceBindingExists(c.Query, 1, instance.ID, instance.BusinessID, instance.BusinessType, instance.Status)
	if err != nil || !bound {
		t.Fatalf("合法流程实例未通过绑定校验: bound=%t err=%v", bound, err)
	}
	for _, item := range []struct {
		name         string
		tenantID     int64
		instanceID   int64
		businessID   int64
		businessType string
	}{
		{name: "other tenant", tenantID: 2, instanceID: instance.ID, businessID: instance.BusinessID, businessType: instance.BusinessType},
		{name: "other instance", tenantID: 1, instanceID: instance.ID + 1, businessID: instance.BusinessID, businessType: instance.BusinessType},
		{name: "other business", tenantID: 1, instanceID: instance.ID, businessID: instance.BusinessID + 1, businessType: instance.BusinessType},
		{name: "other type", tenantID: 1, instanceID: instance.ID, businessID: instance.BusinessID, businessType: "OTHER"},
	} {
		t.Run(item.name, func(t *testing.T) {
			bound, err = workflowInstanceBindingExists(c.Query, item.tenantID, item.instanceID, item.businessID, item.businessType, instance.Status)
			if err != nil {
				t.Fatal(err)
			}
			if bound {
				t.Fatal("不匹配的流程绑定被接受")
			}
		})
	}
}

// TestWorkflowBusinessEventWaitsForCommit 验证终结回调只在成功提交后触发，回滚不触发。
func TestWorkflowBusinessEventWaitsForCommit(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[rollback], func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			_, task := createWorkflowTestTask(t, c, 1, 2)
			committed, fired := false, false
			c.tx = &workflowObservedTransaction{store: store, afterCommit: func() { committed = true }, rollback: rollback}
			c.eventRegistry.Register("TEST", func(context.Context, int64, int64, int64, adminv1.WorkflowInstanceStatus) {
				fired = true
				if !committed {
					t.Error("回调在审批事务提交前触发")
				}
			})
			node := &workflow.Node{ID: "task", OutEdges: []workflow.Edge{{To: "end"}}}
			err := c.handleApprove(context.Background(), 2, task, &workflow.Graph{Nodes: map[string]*workflow.Node{"end": {ID: "end", Type: workflow.NodeTypeEnd}}}, node, "")
			if rollback && (err == nil || fired) {
				t.Fatalf("回滚仍触发回调: err=%v fired=%v", err, fired)
			}
			if !rollback && (err != nil || !fired) {
				t.Fatalf("提交后缺少回调: err=%v fired=%v", err, fired)
			}
		})
	}
}

// TestWorkflowSubmitBindsBeforeSynchronousCompletion 验证无人工审批时先保存业务关联再触发终结回调。
func TestWorkflowSubmitBindsBeforeSynchronousCompletion(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	c.eventRegistry.RegisterBindingVerifier("TEST", func(context.Context, int64, int64, int64) (bool, error) { return false, nil })
	ctx := workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser)
	err := c.definitionRepo.Create(context.Background(), &models.WorkflowDefinition{TenantID: 1, Code: "AUTO", Version: 1, Status: workflowStatusDefinitionEnabled, NodeConfig: `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"end"}]}`})
	if err != nil {
		t.Fatal(err)
	}
	var boundID int64
	fired := false
	c.eventRegistry.Register("TEST", func(_ context.Context, _, instanceID, _ int64, status adminv1.WorkflowInstanceStatus) {
		fired = true
		if boundID != instanceID || status != adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_APPROVED {
			t.Errorf("回调早于业务关联: bound=%d instance=%d status=%v", boundID, instanceID, status)
		}
	})
	var instanceID int64
	instanceID, err = pkgWorkflow.Submit(ctx, pkgWorkflow.SubmitParams{Code: "AUTO", Version: 1, BusinessType: "TEST", BusinessID: 1, OnCreated: func(_ context.Context, id int64) error {
		_, err = c.FindByID(context.Background(), id)
		if err != nil {
			return fmt.Errorf("关联回写前实例尚未提交: %w", err)
		}
		boundID = id
		return nil
	}})
	if err != nil || !fired || instanceID != boundID {
		t.Fatalf("同步通过与关联回写失败: id=%d bound=%d fired=%v err=%v", instanceID, boundID, fired, err)
	}
}

// TestWorkflowSubmitRejectsWhenBindingFails 验证宿主关联回写失败时不遗留进行中实例或触发业务回调。
func TestWorkflowSubmitRejectsWhenBindingFails(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	c.eventRegistry.RegisterBindingVerifier("TEST", func(context.Context, int64, int64, int64) (bool, error) { return false, nil })
	err := c.definitionRepo.Create(context.Background(), &models.WorkflowDefinition{TenantID: 1, Code: "AUTO", Version: 1, Status: workflowStatusDefinitionEnabled, NodeConfig: `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"end"}]}`})
	if err != nil {
		t.Fatal(err)
	}
	fired := false
	c.eventRegistry.Register("TEST", func(context.Context, int64, int64, int64, adminv1.WorkflowInstanceStatus) { fired = true })
	_, err = pkgWorkflow.Submit(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), pkgWorkflow.SubmitParams{Code: "AUTO", Version: 1, BusinessType: "TEST", BusinessID: 1, OnCreated: func(context.Context, int64) error { return errors.New("关联回写失败") }})
	if err == nil || fired {
		t.Fatalf("关联失败仍提交或回调: err=%v fired=%v", err, fired)
	}
	var instances []*models.WorkflowInstance
	instances, err = c.List(context.Background())
	if err != nil || len(instances) != 1 || instances[0].Status != workflowStatusInstanceRejected {
		t.Fatalf("关联失败仍有进行中实例: instances=%v err=%v", instances, err)
	}
}

// workflowObservedTransaction 在真实事务提交后观察状态，或注入提交前失败以核验回滚。
type workflowObservedTransaction struct {
	store       data.Transaction
	afterCommit func()
	rollback    bool
}

// Transaction 执行真实数据库事务并暴露可确定的提交边界。
func (tx *workflowObservedTransaction) Transaction(ctx context.Context, fn func(context.Context) error) error {
	var err error
	err = tx.store.Transaction(ctx, func(ctx context.Context) error {
		err = fn(ctx)
		if err != nil {
			return err
		}
		if tx.rollback {
			return errors.New("测试回滚")
		}
		return nil
	})
	if err == nil && tx.afterCommit != nil {
		tx.afterCommit()
	}
	return err
}

// newWorkflowTestCase 使用临时 SQLite 和生产隔离回调构建真实仓储，避免改动开发库。
func newWorkflowTestCase(t *testing.T) (*WorkflowInstanceCase, *data.Data) {
	t.Helper()
	driver.Opens["workflow-test-sqlite"] = sqlite.Open
	modelList := []interface{}{&models.WorkflowInstance{}, &models.WorkflowTask{}, &models.WorkflowDefinition{}, &models.WorkflowLog{}, &models.WorkflowJoin{}, &models.WorkflowDelegation{}, &models.BaseUser{}, &models.WorkflowScriptRun{}, &models.BaseScript{}, &models.BaseScriptLog{}}
	client, cleanup, err := databaseGorm.NewGormClient(&configv1.Data_Database{Driver: "workflow-test-sqlite", Source: filepath.Join(t.TempDir(), "workflow.db")}, databaseGorm.WithMigrateModels(modelList...))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	// 生成模型显式指定 bigint；SQLite 自增主键必须声明为 integer。
	for _, model := range modelList {
		stmt := &gorm.Statement{DB: client.DB}
		err = stmt.Parse(model)
		if err != nil {
			t.Fatal(err)
		}
		stmt.Schema.FieldsByName["ID"].DataType = schema.Int
	}
	err = client.AutoMigrate(modelList...)
	if err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*databaseGorm.Client{databaseGorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	c := NewWorkflowInstanceCase(&biz.BaseCase{}, nil, store, data.NewWorkflowInstanceRepository(store), data.NewWorkflowDefinitionRepository(store), data.NewWorkflowTaskRepository(store), data.NewWorkflowLogRepository(store), data.NewWorkflowJoinRepository(store), data.NewWorkflowDelegationRepository(store), data.NewBaseUserRepository(store), pkgWorkflow.NewEventRegistry(), data.NewWorkflowScriptRunRepository(store), data.NewBaseScriptLogRepository(store), scripting.NewRuntime())
	for userID := int64(1); userID <= 4; userID++ {
		err = c.baseUserRepo.Create(context.Background(), &models.BaseUser{ID: userID, TenantID: 1, UserName: fmt.Sprintf("user%d", userID), UserCode: fmt.Sprintf("U%d", userID), Status: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	return c, store
}

// createWorkflowTestTask 创建申请人一的单节点实例与其他用户的待办任务。
func createWorkflowTestTask(t *testing.T, c *WorkflowInstanceCase, tenantID, assigneeID int64) (*models.WorkflowInstance, *models.WorkflowTask) {
	t.Helper()
	ctx := workflowTestContext(tenantID, 1, databaseGorm.DataScopeAll)
	definition := &models.WorkflowDefinition{TenantID: tenantID, Code: fmt.Sprintf("TEST-%d-%d", tenantID, assigneeID), Version: 1, Status: workflowStatusDefinitionEnabled, NodeConfig: `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"task","type":"TASK","approvers":[{"type":"USER","id":2}]},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"task"},{"from":"task","to":"end"}]}`}
	err := c.definitionRepo.Create(ctx, definition)
	if err != nil {
		t.Fatal(err)
	}
	instance := &models.WorkflowInstance{TenantID: tenantID, DefinitionID: definition.ID, ApplicantUserID: 1, Status: workflowStatusInstancePending, BusinessType: "TEST", BusinessID: 1, NodeConfig: definition.NodeConfig, Variables: "{}"}
	err = c.Create(ctx, instance)
	if err != nil {
		t.Fatal(err)
	}
	task := &models.WorkflowTask{TenantID: tenantID, InstanceID: instance.ID, NodeID: "task", AssigneeUserID: assigneeID, Status: workflowStatusTaskPending}
	err = c.taskRepo.Create(ctx, task)
	if err != nil {
		t.Fatal(err)
	}
	return instance, task
}

// workflowTestContext 构造带租户、用户和数据范围的真实认证上下文。
func workflowTestContext(tenantID, userID int64, scope int32) context.Context {
	identity := &authData.UserTokenPayload{TenantId: tenantID, TenantCode: "TEST", UserId: userID, DataScope: scope}
	return engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
}

// 工作流脚本执行、恢复和终态动作测试。
// TestWorkflowScriptBeforeSubmitRejectsWithoutInstance 验证真实脚本拒绝申请但保留审计。
func TestWorkflowScriptBeforeSubmitRejectsWithoutInstance(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	definition := publishScriptWorkflow(t, c, store, `{"before_submit":101}`, "", `({allowed:false,message:"额度不足"})`)
	_, err := c.SubmitWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.SubmitWorkflowInstanceRequest{Code: definition.Code, Version: 1, FormData: `{}`, BusinessType: "SCRIPT_TEST", BusinessId: 10})
	if err == nil {
		t.Fatal("脚本拒绝仍提交成功")
	}
	var instances []*models.WorkflowInstance
	instances, err = c.List(context.Background())
	if err != nil || len(instances) != 0 {
		t.Fatalf("拒绝留下实例: %v %v", instances, err)
	}
	var logs []*models.BaseScriptLog
	logs, err = data.NewBaseScriptLogRepository(store).List(context.Background())
	if err != nil || len(logs) == 0 {
		t.Fatalf("拒绝缺少审计: %v %v", logs, err)
	}
}

// TestWorkflowScriptVariablesAndPublishedSnapshot 验证变量网关与发布快照不受脚本库变更影响。
func TestWorkflowScriptVariablesAndPublishedSnapshot(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	graph := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"calc","type":"SCRIPT","script_id":101},{"id":"gate","type":"EXCLUSIVE_GATEWAY"},{"id":"task","type":"TASK","approvers":[{"type":"USER","id":2}]},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"calc"},{"from":"calc","to":"gate"},{"from":"gate","to":"task","condition":"variables.total > 1000"},{"from":"gate","to":"end","condition":"default"},{"from":"task","to":"end"}]}`
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `({variables:{total:ctx.form.amount*2}})`)
	scriptRepo := data.NewBaseScriptRepository(store)
	script, err := scriptRepo.FindByID(context.Background(), 101)
	if err != nil {
		t.Fatal(err)
	}
	script.Source = `({variables:{total:0}})`
	script.Version = 2
	err = scriptRepo.UpdateByID(context.Background(), script)
	if err != nil {
		t.Fatal(err)
	}
	id := submitScriptWorkflow(t, c, definition, `{"amount":600}`)
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, []string{"task:2"})
	var instance *models.WorkflowInstance
	instance, err = c.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(instance.Variables, "1200") || !strings.Contains(instance.NodeConfig, "ctx.form.amount*2") {
		t.Fatalf("快照或变量丢失: %+v", instance)
	}
	var detail *adminv1.GetWorkflowInstanceResponse
	detail, err = c.GetWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.GetScriptRuns()) != 2 || strings.Contains(detail.GetInstance().GetNodeConfig(), "ctx.form.amount*2") {
		t.Fatalf("详情运行记录缺失或源码泄漏: %+v", detail)
	}

	definition.NodeConfig = `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"end"}]}`
	err = c.definitionRepo.UpdateByID(context.Background(), definition)
	if err != nil {
		t.Fatal(err)
	}
	tasks := scriptWorkflowTasks(t, c, id)
	err = c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: tasks[0].ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
	if err != nil {
		t.Fatal(err)
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstanceApproved, nil)
}

// TestWorkflowScriptDynamicApproverFailureAndCancellation 验证非法审批人暂停、重试权限与撤回防复活。
func TestWorkflowScriptDynamicApproverFailureAndCancellation(t *testing.T) {
	for _, source := range []string{`({user_ids:[]})`, `({user_ids:[999]})`, `({user_ids:[5]})`, `({user_ids:["2"]})`} {
		t.Run(source, func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			err := c.baseUserRepo.Create(context.Background(), &models.BaseUser{ID: 5, TenantID: 2, UserName: "outsider", UserCode: "U5", Status: 1})
			if err != nil {
				t.Fatal(err)
			}
			graph := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"task","type":"TASK","approver_script_id":101},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"task"},{"from":"task","to":"end"}]}`
			definition := publishScriptWorkflow(t, c, store, `{}`, graph, source)
			id := submitScriptWorkflow(t, c, definition, `{}`)
			assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, nil)
			runs := scriptWorkflowRuns(t, c, id)
			if len(runs) != 1 || runs[0].Status != "FAILED" {
				t.Fatalf("未保留失败: %+v", runs)
			}
			for _, ctx := range []context.Context{workflowTestContext(2, 1, databaseGorm.DataScopeAll), workflowTestContext(1, 4, databaseGorm.DataScopeSelfUser)} {
				err = c.RetryWorkflowScript(ctx, &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
				if err == nil {
					t.Fatal("越权重试成功")
				}
			}
			err = c.WithdrawWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), id)
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(24*time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			assertClassicWorkflowState(t, c, id, workflowStatusInstanceWithdrawn, nil)
			runs = scriptWorkflowRuns(t, c, id)
			if runs[0].Status != "CANCELED" {
				t.Fatalf("撤回未取消失败节点: %+v", runs[0])
			}
		})
	}
}

// TestWorkflowScriptApproverRetryRestoresPausedNode 验证审批人启用后人工重试恢复原暂停节点。
func TestWorkflowScriptApproverRetryRestoresPausedNode(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	err := c.baseUserRepo.UpdateByID(context.Background(), &models.BaseUser{ID: 2, Status: 2})
	if err != nil {
		t.Fatal(err)
	}
	graph := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"task","type":"TASK","approver_script_id":101},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"task"},{"from":"task","to":"end"}]}`
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `({user_ids:[2]})`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, nil)
	runs := scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != "FAILED" {
		t.Fatalf("未暂停非法审批人: %+v", runs)
	}
	err = c.baseUserRepo.UpdateByID(context.Background(), &models.BaseUser{ID: 2, Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, []string{"task:2"})
	runs = scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != "SUCCEEDED" || runs[0].Attempts != 2 {
		t.Fatalf("重试未复用原执行记录: %+v", runs)
	}
}

// TestWorkflowScriptTerminalActionsRetry 验证三类终态、多动作进度及扫描和人工重试幂等边界。
func TestWorkflowScriptTerminalActionsRetry(t *testing.T) {
	for _, terminal := range []string{"approved", "rejected", "withdrawn"} {
		for _, manual := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/manual=%v", terminal, manual), func(t *testing.T) {
				c, store := newWorkflowTestCase(t)
				definition := publishScriptWorkflow(t, c, store, fmt.Sprintf(`{"%s":101}`, terminal), "", `({actions:[{type:"first",payload:{amount:10}},{type:"second",payload:{}}]})`)
				firstCalls, secondCalls := 0, 0
				var keys []string
				c.eventRegistry.RegisterAction("SCRIPT_TEST", "first", func(_ context.Context, input pkgWorkflow.ActionContext) error {
					firstCalls++
					if input.TenantID != 1 || input.BusinessID != 10 || input.InstanceID == 0 || input.IdempotencyKey == "" {
						t.Errorf("动作上下文错误: %+v", input)
					}
					return nil
				})
				c.eventRegistry.RegisterAction("SCRIPT_TEST", "second", func(_ context.Context, input pkgWorkflow.ActionContext) error {
					secondCalls++
					keys = append(keys, input.IdempotencyKey)
					if secondCalls == 1 {
						return errors.New("首次失败")
					}
					return nil
				})
				id := submitScriptWorkflow(t, c, definition, `{}`)
				var err error
				status := workflowStatusInstanceApproved
				if terminal == "withdrawn" {
					status = workflowStatusInstanceWithdrawn
					err = c.WithdrawWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), id)
				} else {
					tasks := scriptWorkflowTasks(t, c, id)
					action := adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE
					if terminal == "rejected" {
						status = workflowStatusInstanceRejected
						action = adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_REJECT
					}
					err = c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: tasks[0].ID, Action: action})
				}
				if err != nil {
					t.Fatal(err)
				}
				assertClassicWorkflowState(t, c, id, status, nil)
				runs := scriptWorkflowRuns(t, c, id)
				if len(runs) != 1 || runs[0].Status != "FAILED" || runs[0].EffectsCompleted != 1 {
					t.Fatalf("终态失败进度错误: %+v", runs)
				}
				if manual {
					err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
				} else {
					_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(time.Hour))
				}
				if err != nil {
					t.Fatal(err)
				}
				runs = scriptWorkflowRuns(t, c, id)
				if runs[0].Status != "SUCCEEDED" || firstCalls != 1 || secondCalls != 2 || keys[0] != keys[1] {
					t.Fatalf("恢复重复副作用或键漂移: run=%+v first=%d second=%d keys=%v", runs[0], firstCalls, secondCalls, keys)
				}
				err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
				if err != nil {
					t.Logf("已成功记录拒绝再次重试: %v", err)
				}
				if firstCalls != 1 || secondCalls != 2 {
					t.Fatal("重复重试执行已成功动作")
				}
			})
		}
	}
}

// TestWorkflowScriptNodeFailureNeverAutoApproves 验证计算节点异常不会因无待办而通过。
func TestWorkflowScriptNodeFailureNeverAutoApproves(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	graph := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"calc","type":"SCRIPT","script_id":101},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"calc"},{"from":"calc","to":"end"}]}`
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `throw new Error("计算不可用")`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, nil)
	runs := scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != "FAILED" || runs[0].Stage != "NODE" {
		t.Fatalf("计算失败未暂停: %+v", runs)
	}
}

// TestWorkflowScriptAutomaticAttemptsAreBounded 验证自动重试指数退避且五次后停止，人工重试仍可执行。
func TestWorkflowScriptAutomaticAttemptsAreBounded(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	definition := publishScriptWorkflow(t, c, store, `{"approved":101}`, "", `({actions:[{type:"always.fail",payload:{}}]})`)
	calls := 0
	c.eventRegistry.RegisterAction("SCRIPT_TEST", "always.fail", func(context.Context, pkgWorkflow.ActionContext) error { calls++; return errors.New("持续故障") })
	id := submitScriptWorkflow(t, c, definition, `{}`)
	tasks := scriptWorkflowTasks(t, c, id)
	err := c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: tasks[0].ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
	if err != nil {
		t.Fatal(err)
	}
	var previousDelay time.Duration
	for i := 0; i < 8; i++ {
		runs := scriptWorkflowRuns(t, c, id)
		before := runs[0].Attempts
		if before < 5 {
			delay := runs[0].NextRetryAt.Sub(runs[0].UpdatedAt)
			if delay <= previousDelay {
				t.Fatalf("退避未递增: %v <= %v", delay, previousDelay)
			}
			previousDelay = delay
		}
		_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(48*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
	}
	runs := scriptWorkflowRuns(t, c, id)
	if calls != 5 || runs[0].Attempts != 5 {
		t.Fatalf("自动尝试未限五次: calls=%d run=%+v", calls, runs[0])
	}
	err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
	if calls != 6 {
		t.Fatalf("人工重试被次数限制: calls=%d err=%v", calls, err)
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstanceApproved, nil)
}

// TestWorkflowScriptTerminalRollbackHasNoJob 验证审批回滚不遗留终态执行任务。
func TestWorkflowScriptTerminalRollbackHasNoJob(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	definition := publishScriptWorkflow(t, c, store, `{"approved":101}`, "", `({actions:[]})`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	c.tx = &workflowObservedTransaction{store: store, rollback: true}
	tasks := scriptWorkflowTasks(t, c, id)
	err := c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: tasks[0].ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
	if err == nil {
		t.Fatal("事务回滚未返回错误")
	}
	if runs := scriptWorkflowRuns(t, c, id); len(runs) != 0 {
		t.Fatalf("回滚留下终态任务: %+v", runs)
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, []string{"task:2"})
}

// publishScriptWorkflow 使用真实定义发布入口捕获共享脚本快照。
func publishScriptWorkflow(t *testing.T, c *WorkflowInstanceCase, store *data.Data, hooks, graph, source string) *models.WorkflowDefinition {
	t.Helper()
	scriptRepo := data.NewBaseScriptRepository(store)
	err := scriptRepo.Create(context.Background(), &models.BaseScript{ID: 101, Name: "script-101", Language: "JAVASCRIPT", Source: source, Version: 1, Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	if graph == "" {
		graph = `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"task","type":"TASK","approvers":[{"type":"USER","id":2}]},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"task"},{"from":"task","to":"end"}]}`
	}
	var config map[string]any
	err = json.Unmarshal([]byte(graph), &config)
	if err != nil {
		t.Fatal(err)
	}
	var hookConfig map[string]any
	err = json.Unmarshal([]byte(hooks), &hookConfig)
	if err != nil {
		t.Fatal(err)
	}
	config["hooks"] = hookConfig
	var raw []byte
	raw, err = json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	definitionCase := NewWorkflowDefinitionCase(c.BaseCase, store, c.definitionRepo, c.WorkflowInstanceRepository, scriptRepo, NewBaseScriptCase(c.BaseCase, store, scriptRepo, NewBaseScriptLogCase(c.BaseCase, c.scriptLogRepo), c.runtime))
	ctx := workflowTestContext(1, 1, databaseGorm.DataScopeAll)
	err = definitionCase.CreateWorkflowDefinition(ctx, &adminv1.WorkflowDefinitionForm{Name: "脚本集成测试", Code: "SCRIPT_TEST", Version: 1, NodeConfig: string(raw), FormSchema: "[]"})
	if err != nil {
		t.Fatal(err)
	}
	var definitions []*models.WorkflowDefinition
	definitions, err = c.definitionRepo.List(ctx)
	if err != nil || len(definitions) != 1 {
		t.Fatalf("创建定义失败: %v %v", definitions, err)
	}
	err = definitionCase.SetWorkflowDefinitionStatus(ctx, &adminv1.SetWorkflowDefinitionStatusRequest{Id: definitions[0].ID, Status: adminv1.WorkflowDefinitionStatus_WORKFLOW_DEFINITION_STATUS_ENABLED})
	if err != nil {
		t.Fatal(err)
	}
	var definition *models.WorkflowDefinition
	definition, err = c.definitionRepo.FindByID(ctx, definitions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

// submitScriptWorkflow 经公开提交入口创建流程。
func submitScriptWorkflow(t *testing.T, c *WorkflowInstanceCase, definition *models.WorkflowDefinition, form string) int64 {
	t.Helper()
	id, err := c.SubmitWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.SubmitWorkflowInstanceRequest{Code: definition.Code, Version: definition.Version, FormData: form, BusinessType: "SCRIPT_TEST", BusinessId: 10})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// scriptWorkflowRuns 读取实例持久化执行记录。
func scriptWorkflowRuns(t *testing.T, c *WorkflowInstanceCase, id int64) []*models.WorkflowScriptRun {
	t.Helper()
	query := c.scriptRunRepo.Query(context.Background()).WorkflowScriptRun
	runs, err := c.scriptRunRepo.List(context.Background(), repository.Where(query.Stage.Neq("START")))
	if err != nil {
		t.Fatal(err)
	}
	var result []*models.WorkflowScriptRun
	for _, run := range runs {
		if run.InstanceID == id {
			result = append(result, run)
		}
	}
	return result
}

// scriptWorkflowTasks 返回实例真实任务快照。
func scriptWorkflowTasks(t *testing.T, c *WorkflowInstanceCase, id int64) []*models.WorkflowTask {
	t.Helper()
	tasks, err := c.listTasks(context.Background(), 1, id)
	if err != nil {
		t.Fatal(err)
	}
	return tasks
}

// TestWorkflowScriptBindingFailureNeverDispatchesActions 验证宿主绑定失败不触发未关联单据的终态动作。
func TestWorkflowScriptBindingFailureNeverDispatchesActions(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	c.eventRegistry.RegisterBindingVerifier("SCRIPT_TEST", func(context.Context, int64, int64, int64) (bool, error) { return false, nil })
	definition := publishScriptWorkflow(t, c, store, `{"rejected":101}`, "", `({actions:[{type:"unexpected",payload:{}}]})`)
	calls := 0
	c.eventRegistry.RegisterAction("SCRIPT_TEST", "unexpected", func(context.Context, pkgWorkflow.ActionContext) error { calls++; return nil })
	_, err := pkgWorkflow.Submit(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), pkgWorkflow.SubmitParams{Code: definition.Code, Version: 1, FormData: `{}`, BusinessType: "SCRIPT_TEST", BusinessID: 10, OnCreated: func(context.Context, int64) error { return errors.New("宿主绑定失败") }})
	if err == nil {
		t.Fatal("绑定失败未返回错误")
	}
	if calls != 0 {
		t.Fatalf("单据关联失败仍执行 %d 次业务动作", calls)
	}
	var runs []*models.WorkflowScriptRun
	runs, err = c.scriptRunRepo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range runs {
		if run.Stage == "REJECTED" {
			t.Fatalf("绑定失败遗留可被扫描执行的业务动作: %+v", run)
		}
	}
}

// TestWorkflowScriptInputUsesGraphNodeIdentity 验证脚本接收实际图节点类型而非执行阶段。
func TestWorkflowScriptInputUsesGraphNodeIdentity(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	graph := `{"version":2,"nodes":[{"id":"begin","type":"START"},{"id":"calc","type":"SCRIPT","script_id":101},{"id":"finish","type":"END"}],"edges":[{"from":"begin","to":"calc"},{"from":"calc","to":"finish"}]}`
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `({variables:{node_id:ctx.node.id,node_type:ctx.node.type}})`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	instance, err := c.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var variables map[string]any
	err = json.Unmarshal([]byte(instance.Variables), &variables)
	if err != nil {
		t.Fatal(err)
	}
	if variables["node_id"] != "calc" || variables["node_type"] != "SCRIPT" {
		t.Fatalf("输入图节点身份错误: %v", variables)
	}
}

// TestWorkflowScriptDedicatedEndpointsEnforceSavedSandboxScripts 验证专用选择和试跑只接受启用的受限语言且契约无源码入口。
func TestWorkflowScriptDedicatedEndpointsEnforceSavedSandboxScripts(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	scriptRepo := data.NewBaseScriptRepository(store)
	for index, language := range []string{"JAVASCRIPT", "LUA", "WASM", "PYTHON", "GO", "CEL", "EXPR", "STARLARK", "TCL"} {
		err := scriptRepo.Create(context.Background(), &models.BaseScript{ID: int64(index + 1), Name: language, Language: language, Source: `({saved:true})`, Version: 1, Status: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	err := scriptRepo.Create(context.Background(), &models.BaseScript{ID: 20, Name: "disabled-js", Language: "JAVASCRIPT", Source: `({saved:true})`, Version: 1, Status: 2})
	if err != nil {
		t.Fatal(err)
	}
	definitionCase := NewWorkflowDefinitionCase(c.BaseCase, store, c.definitionRepo, c.WorkflowInstanceRepository, scriptRepo, NewBaseScriptCase(c.BaseCase, store, scriptRepo, NewBaseScriptLogCase(c.BaseCase, c.scriptLogRepo), c.runtime))
	ctx := workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser)
	var page *adminv1.PageBaseScriptResponse
	page, err = definitionCase.PageWorkflowScript(ctx, &adminv1.PageBaseScriptRequest{PageNum: 1, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.BaseScripts) != 1 {
		t.Fatalf("专用选择泄露禁用或非受限语言: %+v", page)
	}
	for _, item := range page.BaseScripts {
		if item.Id != 1 {
			t.Fatalf("专用列表包含不安全或禁用脚本: %+v", item)
		}
	}
	for _, id := range []int64{0, 2, 3, 4, 5, 6, 7, 8, 9, 20} {
		_, err = definitionCase.TestWorkflowScript(ctx, &adminv1.TestWorkflowScriptRequest{ScriptId: id, ContextJson: `{}`})
		if err == nil {
			t.Fatalf("专用试跑允许脚本 %d", id)
		}
	}
	var response *adminv1.TestRunBaseScriptResponse
	response, err = definitionCase.TestWorkflowScript(ctx, &adminv1.TestWorkflowScriptRequest{ScriptId: 1, ContextJson: `{}`})
	if err != nil || !response.GetSuccess() || !strings.Contains(response.GetOutputJson(), `"saved":true`) {
		t.Fatalf("保存脚本试跑失败: response=%v err=%v", response, err)
	}
	fields := (&adminv1.TestWorkflowScriptRequest{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != 2 || fields.ByName("source") != nil || fields.ByName("language") != nil {
		t.Fatal("专用试跑契约存在任意源码或语言入口")
	}
}

// TestWorkflowScriptBuiltinNotificationRetry 验证内置站内信失败恢复、稳定消息幂等键及已成功动作不重复发布。
func TestWorkflowScriptBuiltinNotificationRetry(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	var messages []notification.Message
	notification.SetDefaultPublisher(notification.PublisherFunc(func(_ context.Context, message notification.Message) (int64, error) {
		if message.Title != "脚本通知" {
			return 1, nil
		}
		messages = append(messages, message)
		if len(messages) == 1 {
			return 0, errors.New("模拟站内信事务失败")
		}
		return 100, nil
	}))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })
	definition := publishScriptWorkflow(t, c, store, `{"approved":101}`, "", `({actions:[{type:"workflow.notify",payload:{title:"脚本通知",content:"报销已通过",user_ids:[2,2,3]}}]})`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	tasks := scriptWorkflowTasks(t, c, id)
	err := c.AuditWorkflowTask(workflowTestContext(1, 2, databaseGorm.DataScopeSelfUser), &adminv1.AuditWorkflowTaskRequest{TaskId: tasks[0].ID, Action: adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE})
	if err != nil {
		t.Fatal(err)
	}
	runs := scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != scriptFailed || len(messages) != 1 {
		t.Fatalf("通知失败未保留任务: runs=%+v calls=%d", runs, len(messages))
	}
	err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatalf("重试发布次数错误: %d", len(messages))
	}
	first, second := messages[0], messages[1]
	expectedKey := fmt.Sprintf("workflow:%x", sha256.Sum256([]byte(fmt.Sprintf("workflow:1:%d:APPROVED:end:action:0", id))))
	if first.IdempotencyKey != expectedKey {
		t.Fatalf("消息幂等摘要不匹配: got=%s want=%s", first.IdempotencyKey, expectedKey)
	}
	if first.IdempotencyKey != second.IdempotencyKey || len(first.IdempotencyKey) != 73 || !strings.HasPrefix(first.IdempotencyKey, "workflow:") || first.TenantID != 1 || first.CategoryCode != "WORKFLOW" || first.Source != "workflow" || len(first.Audiences) != 2 {
		t.Fatalf("消息契约或幂等键错误: %+v %+v", first, second)
	}
	err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 {
		t.Fatal("成功后重复发布脚本通知")
	}
}

// TestWorkflowScriptBuiltinNotificationRejectsIllegalAudience 验证非法受众在消息发布之前被拒绝。
func TestWorkflowScriptBuiltinNotificationRejectsIllegalAudience(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	err := c.baseUserRepo.Create(context.Background(), &models.BaseUser{ID: 5, TenantID: 2, UserName: "outsider", UserCode: "U5", Status: 1})
	if err != nil {
		t.Fatal(err)
	}
	err = c.baseUserRepo.UpdateByID(context.Background(), &models.BaseUser{ID: 4, Status: 2})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	notification.SetDefaultPublisher(notification.PublisherFunc(func(context.Context, notification.Message) (int64, error) { calls++; return 1, nil }))
	t.Cleanup(func() { notification.SetDefaultPublisher(nil) })
	for _, users := range [][]int64{nil, {0}, {999}, {4}, {5}, {2, 5}} {
		err = c.sendScriptNotification(context.Background(), pkgWorkflow.ActionContext{TenantID: 1, IdempotencyKey: "stable", Payload: map[string]any{"title": "测试", "content": "正文", "user_ids": users}})
		if err == nil {
			t.Fatalf("非法受众被接受: %v", users)
		}
	}
	if calls != 0 {
		t.Fatalf("非法受众仍发布 %d 条消息", calls)
	}
}

// TestWorkflowScriptExpiredFifthLeaseRequiresManualRetry 验证第五次执行崩溃后扫描释放为失败以允许人工恢复。
func TestWorkflowScriptExpiredFifthLeaseRequiresManualRetry(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	instance, _ := createWorkflowTestTask(t, c, 1, 2)
	err := c.UpdateByID(context.Background(), &models.WorkflowInstance{ID: instance.ID, Status: workflowStatusInstanceApproved})
	if err != nil {
		t.Fatal(err)
	}
	run := &models.WorkflowScriptRun{TenantID: 1, InstanceID: instance.ID, Stage: "APPROVED", NodeID: "end", ExecutionKey: "APPROVED:end", Status: scriptRunning, Attempts: 5, LockedUntil: time.Now().Add(-time.Minute), NextRetryAt: time.Now().Add(-time.Minute)}
	err = c.scriptRunRepo.Create(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ScanWorkflowScripts(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	run, err = c.scriptRunRepo.FindByID(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != scriptFailed || run.Attempts != 5 || run.LastError == "" {
		t.Fatalf("耗尽租约仍不可人工重试: %+v", run)
	}
}

// TestWorkflowScriptStaleLeaseCannotOverwriteNewAttempt 验证旧执行token不能覆盖新执行进度或成功状态。
func TestWorkflowScriptStaleLeaseCannotOverwriteNewAttempt(t *testing.T) {
	c, _ := newWorkflowTestCase(t)
	run := &models.WorkflowScriptRun{TenantID: 1, InstanceID: 1, Stage: "APPROVED", NodeID: "end", ExecutionKey: "APPROVED:end", Status: scriptRunning, Attempts: 2, OutputJSON: `{"actions":[]}`}
	err := c.scriptRunRepo.Create(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	stale := *run
	stale.Attempts = 1
	stale.Status = scriptSucceeded
	stale.EffectsCompleted = 9
	err = c.saveClaimedScriptRun(context.Background(), &stale)
	if err == nil {
		t.Fatal("旧尝试覆盖了新租约")
	}
	run.Status = scriptSucceeded
	run.EffectsCompleted = 2
	err = c.saveClaimedScriptRun(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	stale.Attempts = 2
	stale.Status = scriptFailed
	err = c.saveClaimedScriptRun(context.Background(), &stale)
	if err == nil {
		t.Fatal("旧执行覆盖已成功状态")
	}
	var current *models.WorkflowScriptRun
	current, err = c.scriptRunRepo.FindByID(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != scriptSucceeded || current.EffectsCompleted != 2 || current.Attempts != 2 {
		t.Fatalf("新执行结果被覆盖: %+v", current)
	}
}

// TestWorkflowScriptCombinedInputBudgetRejectsBeforePersist 验证表单加变量超出完整输入预算时当前计算失败且保留人工恢复入口。
func TestWorkflowScriptCombinedInputBudgetRejectsBeforePersist(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	graph := `{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"calc","type":"SCRIPT","script_id":101},{"id":"task","type":"TASK","approvers":[{"type":"USER","id":2}]},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"calc"},{"from":"calc","to":"task"},{"from":"task","to":"end"}]}`
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `({variables:{large:"x".repeat(180*1024)}})`)
	form, err := json.Marshal(map[string]any{"reason": strings.Repeat("f", 40*1024)})
	if err != nil {
		t.Fatal(err)
	}
	id := submitScriptWorkflow(t, c, definition, string(form))
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, nil)
	runs := scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != scriptFailed || runs[0].Attempts != 1 {
		t.Fatalf("完整输入超限未在当前节点失败: %+v", runs)
	}
	var instance *models.WorkflowInstance
	instance, err = c.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(instance.Variables, "large") {
		t.Fatalf("不可消费变量仍被持久化，长度=%d", len(instance.Variables))
	}
	var detail *adminv1.GetWorkflowInstanceResponse
	detail, err = c.GetWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), id)
	if err != nil {
		t.Fatalf("失败记录不可人工访问: %v", err)
	}
	foundFailure := false
	for _, item := range detail.GetScriptRuns() {
		if item.Stage == "NODE" && item.Status == scriptFailed {
			foundFailure = true
		}
	}
	if !foundFailure {
		t.Fatal("详情未显示当前节点失败记录")
	}
	// 发布快照不允许临时改脚本；通过缩小申请表单模拟管理员修复后恢复同一执行记录。
	instance.FormData = `{}`
	err = c.UpdateByID(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}
	// 原执行输入不可变，超限失败保持可重试，而不能悄悄改用已损坏的变量快照。
	err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: id, Id: runs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	runs = scriptWorkflowRuns(t, c, id)
	if runs[0].Attempts != 2 || runs[0].Status != scriptSucceeded {
		t.Fatalf("修复后失败记录无法恢复: %+v", runs[0])
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, []string{"task:2"})
}

// TestWorkflowScriptEscapedNodeBudgetRejectsBeforePersist 验证长转义节点和执行键开销计入完整输入预算。
func TestWorkflowScriptEscapedNodeBudgetRejectsBeforePersist(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	nodeID := strings.Repeat("<", 100)
	graph := fmt.Sprintf(`{"version":2,"nodes":[{"id":"start","type":"START"},{"id":"%s","type":"SCRIPT","script_id":101},{"id":"task","type":"TASK","approvers":[{"type":"USER","id":2}]},{"id":"end","type":"END"}],"edges":[{"from":"start","to":"%s"},{"from":"%s","to":"task"},{"from":"task","to":"end"}]}`, nodeID, nodeID, nodeID)
	definition := publishScriptWorkflow(t, c, store, `{}`, graph, `({variables:{large:"x".repeat(199*1024)}})`)
	id := submitScriptWorkflow(t, c, definition, `{}`)
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, nil)
	runs := scriptWorkflowRuns(t, c, id)
	if len(runs) != 1 || runs[0].Status != scriptFailed || runs[0].NodeID != nodeID {
		t.Fatalf("转义输入超限未在当前节点失败: records=%d", len(runs))
	}
	instance, err := c.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(instance.Variables, "large") {
		t.Fatalf("边界变量已落库，长度=%d", len(instance.Variables))
	}
}

// 工作流启动关联确认与恢复测试。
// TestWorkflowStartScannerRestoresConfirmedBindings 验证进程在启动前退出时扫描恢复首批待办且重复扫描不重复创建。
func TestWorkflowStartScannerRestoresConfirmedBindings(t *testing.T) {
	for _, state := range [][2]bool{{false, false}, {true, true}, {true, false}} {
		required, confirmed := state[0], state[1]
		t.Run(fmt.Sprintf("required=%v/confirmed=%v", required, confirmed), func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			definition := publishScriptWorkflow(t, c, store, `{}`, "", `({variables:{}})`)
			c.eventRegistry.RegisterBindingVerifier("SCRIPT_TEST", func(context.Context, int64, int64, int64) (bool, error) { return true, nil })
			instance, run := createWorkflowStartRecovery(t, c, definition, required, confirmed, scriptPending)
			_, err := c.ScanWorkflowScripts(context.Background(), time.Now())
			if err != nil {
				t.Fatal(err)
			}
			assertClassicWorkflowState(t, c, instance.ID, workflowStatusInstancePending, []string{"task:2"})
			run, err = c.scriptRunRepo.FindByID(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if run.Status != scriptSucceeded {
				t.Fatalf("启动记录未完成: %+v", run)
			}
			_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			tasks := scriptWorkflowTasks(t, c, instance.ID)
			if len(tasks) != 1 {
				t.Fatalf("重复扫描生成重复待办: %+v", tasks)
			}
		})
	}
}

// TestWorkflowStartScannerWaitsForUnconfirmedBinding 验证绑定未确认且宿主验证失败时保持待审批并保留人工重试。
func TestWorkflowStartScannerWaitsForUnconfirmedBinding(t *testing.T) {
	for _, verificationErr := range []error{nil, errors.New("宿主暂时不可查询")} {
		t.Run(fmt.Sprintf("error=%v", verificationErr), func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			definition := publishScriptWorkflow(t, c, store, `{"rejected":101}`, "", `({actions:[{type:"unexpected",payload:{}}]})`)
			calls := 0
			c.eventRegistry.RegisterAction("SCRIPT_TEST", "unexpected", func(context.Context, pkgWorkflow.ActionContext) error { calls++; return nil })
			verified := false
			c.eventRegistry.RegisterBindingVerifier("SCRIPT_TEST", func(context.Context, int64, int64, int64) (bool, error) { return verified, verificationErr })
			instance, run := createWorkflowStartRecovery(t, c, definition, true, false, scriptRunning)
			_, err := c.ScanWorkflowScripts(context.Background(), time.Now())
			if err != nil {
				t.Fatal(err)
			}
			assertClassicWorkflowState(t, c, instance.ID, workflowStatusInstancePending, nil)
			run, err = c.scriptRunRepo.FindByID(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if run.Status != scriptFailed {
				t.Fatalf("绑定未确认缺少失败重试记录: %+v", run)
			}
			if calls != 0 || len(scriptWorkflowRuns(t, c, instance.ID)) != 0 {
				t.Fatalf("未确认绑定仍调度业务动作: calls=%d", calls)
			}
			if verificationErr == nil {
				verified = true
				err = c.RetryWorkflowScript(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), &adminv1.RetryWorkflowScriptRequest{InstanceId: instance.ID, Id: run.ID})
				if err != nil {
					t.Fatal(err)
				}
				assertClassicWorkflowState(t, c, instance.ID, workflowStatusInstancePending, []string{"task:2"})
			}
		})
	}
}

// TestWorkflowStartTerminalInstanceCancelsStartup 验证撤回或终结实例会取消启动任务且扫描不能复活。
func TestWorkflowStartTerminalInstanceCancelsStartup(t *testing.T) {
	for _, withdraw := range []bool{false, true} {
		t.Run(fmt.Sprintf("withdraw=%v", withdraw), func(t *testing.T) {
			c, store := newWorkflowTestCase(t)
			definition := publishScriptWorkflow(t, c, store, `{}`, "", `({variables:{}})`)
			instance, run := createWorkflowStartRecovery(t, c, definition, false, false, scriptPending)
			var err error
			if withdraw {
				err = c.WithdrawWorkflowInstance(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), instance.ID)
			} else {
				err = c.terminateInstance(context.Background(), instance, workflowStatusInstanceRejected)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.ScanWorkflowScripts(context.Background(), time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			run, err = c.scriptRunRepo.FindByID(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if run.Status != scriptCanceled {
				t.Fatalf("终结未取消启动: %+v", run)
			}
			tasks := scriptWorkflowTasks(t, c, instance.ID)
			if len(tasks) != 0 {
				t.Fatalf("扫描复活终结实例待办: %+v", tasks)
			}
		})
	}
}

// createWorkflowStartRecovery 构造已持久化但未进入首节点的崩溃现场。
func createWorkflowStartRecovery(t *testing.T, c *WorkflowInstanceCase, definition *models.WorkflowDefinition, required, confirmed bool, status string) (*models.WorkflowInstance, *models.WorkflowScriptRun) {
	t.Helper()
	instance := &models.WorkflowInstance{TenantID: 1, DefinitionID: definition.ID, ApplicantUserID: 1, FormData: `{}`, Variables: `{}`, NodeConfig: definition.NodeConfig, BusinessType: "SCRIPT_TEST", BusinessID: 10, Status: workflowStatusInstancePending, SubmitAt: time.Now()}
	err := c.Create(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}
	run := &models.WorkflowScriptRun{TenantID: 1, InstanceID: instance.ID, NodeID: "start", Stage: "START", ScriptID: 0, ScriptName: "流程启动", ExecutionKey: "START", Status: status, InputJSON: fmt.Sprintf(`{"binding_required":%t,"binding_confirmed":%t}`, required, confirmed), NextRetryAt: time.Now().Add(-time.Minute), LockedUntil: time.Now().Add(-time.Minute)}
	err = c.scriptRunRepo.Create(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	return instance, run
}

// TestWorkflowStartRequiresBindingVerifier 验证有宿主绑定回调却未注册验证器时提交前拒绝。
func TestWorkflowStartRequiresBindingVerifier(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	definition := publishScriptWorkflow(t, c, store, `{}`, "", `({variables:{}})`)
	called := false
	_, err := pkgWorkflow.Submit(workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser), pkgWorkflow.SubmitParams{Code: definition.Code, Version: 1, FormData: `{}`, BusinessType: "SCRIPT_TEST", BusinessID: 10, OnCreated: func(context.Context, int64) error { called = true; return nil }})
	if err == nil || called {
		t.Fatalf("缺少验证器仍进入业务绑定: err=%v called=%v", err, called)
	}
	var instances []*models.WorkflowInstance
	instances, err = c.List(context.Background())
	if err != nil || len(instances) != 0 {
		t.Fatalf("未验证提交留下实例: %v %v", instances, err)
	}
}

// TestWorkflowStartWithdrawWaitsForBindingConfirmation 验证宿主绑定回调未结束时不能撤回，确认推进后可以撤回。
func TestWorkflowStartWithdrawWaitsForBindingConfirmation(t *testing.T) {
	c, store := newWorkflowTestCase(t)
	definition := publishScriptWorkflow(t, c, store, `{}`, "", `({variables:{}})`)
	c.eventRegistry.RegisterBindingVerifier("SCRIPT_TEST", func(context.Context, int64, int64, int64) (bool, error) { return true, nil })
	ctx := workflowTestContext(1, 1, databaseGorm.DataScopeSelfUser)
	checked := false
	id, err := pkgWorkflow.Submit(ctx, pkgWorkflow.SubmitParams{Code: definition.Code, Version: 1, FormData: `{}`, BusinessType: "SCRIPT_TEST", BusinessID: 10, OnCreated: func(bindCtx context.Context, instanceID int64) error {
		err := c.WithdrawWorkflowInstance(bindCtx, instanceID)
		if err == nil {
			t.Error("宿主绑定尚未确认却允许撤回")
		}
		var current *models.WorkflowInstance
		current, err = c.FindByID(bindCtx, instanceID)
		if err != nil {
			return err
		}
		if current.Status != workflowStatusInstancePending {
			t.Errorf("绑定过程中提前终结: status=%d", current.Status)
		}
		checked = true
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !checked {
		t.Fatal("未执行绑定回调中的撤回边界检查")
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstancePending, []string{"task:2"})
	err = c.WithdrawWorkflowInstance(ctx, id)
	if err != nil {
		t.Fatalf("绑定确认后无法撤回: %v", err)
	}
	assertClassicWorkflowState(t, c, id, workflowStatusInstanceWithdrawn, nil)
}
