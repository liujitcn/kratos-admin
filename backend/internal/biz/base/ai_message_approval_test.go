package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	agentmessage "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data"
	gendata "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
)

// TestAiAutomationRecoveryPreservesApprovalOutcome 验证后台到期和租约丢失均保留准确的审批结果。
func TestAiAutomationRecoveryPreservesApprovalOutcome(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		runStatus      string
		approvalStatus int32
		messageStatus  basev1.AiMessageStatus
		wantRun        string
		wantState      string
		wantExecution  string
	}{
		{"pending approval expires", "waiting_approval", aiRunPending, basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL, "expired", "expired", "not_run"},
		{"approved tool loses lease", "running", aiRunRunning, basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING, "unknown", "approved", "unknown"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			c, ctx, session, message := newAiEventTestCase(t, &models.AiAutomation{}, &models.AiAutomationRun{})
			automation := NewAiAutomationCase(c.BaseCase, c.tx, gendata.NewAiAutomationRepository(c.aiMessageRepo), gendata.NewAiAutomationRunRepository(c.aiMessageRepo), nil, c.aiSessionCase, c, nil, nil)
			now := time.Now().Truncate(time.Millisecond)
			plan := &models.AiAutomation{ID: 401, TenantID: session.TenantID, OwnerUserID: session.UserID, ActiveRunID: 402, Revision: 1, Status: "active", KnowledgeBaseIds: "[]", Plugins: "[]", Schedule: "{}"}
			err := automation.Create(ctx, plan)
			if err != nil {
				t.Fatal(err)
			}
			run := &models.AiAutomationRun{ID: plan.ActiveRunID, AutomationID: plan.ID, TenantID: session.TenantID, OwnerUserID: session.UserID, SessionID: session.ID, Status: scenario.runStatus, Configuration: "{}", LeaseVersion: 1, LeaseUntil: now.Add(-time.Minute).UnixMilli()}
			output := &ai.Response{Approval: &ai.Approval{ID: "current", ExpiresAt: now}}
			if scenario.approvalStatus == aiRunRunning {
				run.ToolStarted = 1
				output.Approval = nil
				output.BlocksJSON = `[{"type":"approval_outcome","approval_id":"current","state":"approved","execution":"unknown"}]`
			}
			if err = automation.runRepo.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			sessionQuery := c.aiSessionCase.Query(ctx).AiSession
			_, err = sessionQuery.WithContext(ctx).Where(sessionQuery.ID.Eq(session.ID)).UpdateColumnSimple(sessionQuery.AutomationID.Value(plan.ID), sessionQuery.AutomationRunID.Value(run.ID))
			if err != nil {
				t.Fatal(err)
			}
			messageQuery := c.aiMessageRepo.Query(ctx).AiMessage
			_, err = messageQuery.WithContext(ctx).Where(messageQuery.ID.Eq(message.ID)).UpdateColumnSimple(messageQuery.Status.Value(int32(scenario.messageStatus)), messageQuery.OutputContent.Value(ai.MarshalReplyContent(output)))
			if err != nil {
				t.Fatal(err)
			}
			approval := &models.AiToolApproval{ID: 301, ApprovalID: "current", TenantID: session.TenantID, UserID: session.UserID, SessionID: session.ID, MessageID: message.ID, RuntimeState: "{}", PendingCalls: "[]", CheckpointData: []byte("checkpoint"), Status: scenario.approvalStatus, ExpiresAt: now}
			if err = c.aiApprovalRepo.Create(ctx, approval); err != nil {
				t.Fatal(err)
			}
			if err = automation.recoverAutomationRuns(ctx, now); err != nil {
				t.Fatal(err)
			}
			run, err = automation.runRepo.FindByID(ctx, run.ID)
			if err != nil || run.Status != scenario.wantRun || run.LeaseVersion != 2 {
				t.Fatalf("自动化执行未终止: %v %+v", err, run)
			}
			plan, err = automation.FindByID(ctx, plan.ID)
			if err != nil || plan.ActiveRunID != 0 {
				t.Fatalf("计划仍然被占用: %v %+v", err, plan)
			}
			approval, err = c.aiApprovalRepo.FindByID(ctx, approval.ID)
			if err != nil || approval.Status != aiRunExpired || len(approval.CheckpointData) != 0 {
				t.Fatalf("审批检查点仍可恢复: %v %+v", err, approval)
			}
			message, err = c.aiMessageRepo.FindByID(ctx, message.ID)
			if err != nil {
				t.Fatal(err)
			}
			item := c.ToDTO(message)
			assertAiApprovalOutcome(t, item, approval.ApprovalID, scenario.wantState, scenario.wantExecution, false)
			if item.GetOutputContent().GetApproval() != nil {
				t.Fatal("已终止的执行仍显示待审批")
			}
		})
	}
}

// TestAiApprovalExpiryReleasesOwnedConversation 验证只有原会话持有人能对过期审批落库，并可领取新一轮生成。
func TestAiApprovalExpiryReleasesOwnedConversation(t *testing.T) {
	for _, previouslyApproved := range []bool{false, true} {
		t.Run(strconv.FormatBool(previouslyApproved), func(t *testing.T) {
			c, ctx, session, message := newAiEventTestCase(t)
			now := time.Now().Add(-time.Minute)
			blocks := ""
			if previouslyApproved {
				blocks = `[{"type":"approval_outcome","approval_id":"earlier","state":"approved","execution":"succeeded"}]`
			}
			output := ai.MarshalReplyContent(&ai.Response{
				Approval:   &ai.Approval{ID: "expired", ExpiresAt: now},
				BlocksJSON: blocks,
			})
			query := c.aiMessageRepo.Query(ctx).AiMessage
			_, err := query.WithContext(ctx).Where(query.ID.Eq(message.ID)).UpdateColumnSimple(
				query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL)),
				query.OutputContent.Value(output),
			)
			if err != nil {
				t.Fatal(err)
			}
			run := &models.AiToolApproval{ID: 301, ApprovalID: "expired", TenantID: session.TenantID, UserID: session.UserID, SessionID: session.ID, MessageID: message.ID, RuntimeState: "{}", PendingCalls: "[]", CheckpointData: []byte("expired checkpoint"), Status: aiRunPending, ExpiresAt: now}
			if err = c.aiApprovalRepo.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			req := &basev1.ListAiMessageRequest{SessionId: "101"}
			other := engine.ContextWithAuthClaims(ctx, (&authdata.UserTokenPayload{TenantId: session.TenantID, UserId: session.UserID + 1}).MakeAuthClaims())
			if _, err = c.ListAiMessage(other, req); err == nil {
				t.Fatal("非属主能读取会话")
			}
			var saved *models.AiToolApproval
			saved, err = c.aiApprovalRepo.FindByID(ctx, run.ID)
			if err != nil || saved.Status != aiRunPending {
				t.Fatalf("非属主更改了审批: %v %+v", err, saved)
			}
			var list *basev1.ListAiMessageResponse
			list, err = c.ListAiMessage(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			item := list.GetMessages()[0]
			assertAiApprovalOutcome(t, item, "expired", "expired", "not_run", !previouslyApproved)
			if item.GetOutputContent().GetApproval() != nil || item.GetStatus() != basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS {
				t.Fatalf("过期审批仍在等待: %+v", item)
			}
			saved, err = c.aiApprovalRepo.FindByID(ctx, run.ID)
			if err != nil || saved.Status != aiRunExpired || len(saved.CheckpointData) != 0 {
				t.Fatalf("过期检查点可恢复: %v %+v", err, saved)
			}
			// 旧编号即使再次点击确认，也只能得到原过期结果。
			var decided *basev1.SendAiMessageResponse
			decided, err = c.DecideAiToolApproval(ctx, &basev1.DecideAiToolApprovalRequest{SessionId: "101", MessageId: "201", ApprovalId: "expired", Approved: true})
			if err != nil {
				t.Fatal(err)
			}
			assertAiApprovalOutcome(t, decided.GetMessages()[0], "expired", "expired", "not_run", !previouslyApproved)
			if err = c.ensureAiSessionIdle(ctx, session.TenantID, session.ID); err != nil {
				t.Fatal(err)
			}
			if previouslyApproved {
				if _, err = c.RegenerateAiMessage(ctx, &basev1.RegenerateAiMessageRequest{SessionId: "101", MessageId: "201"}); err == nil {
					t.Fatal("可以重放含已批准操作的过期轮次")
				}
			}
		})
	}
}

// TestAiApprovalOutcomeUsesConfirmedToolResult 验证审批的执行状态取自恢复工具的实际处理结果。
func TestAiApprovalOutcomeUsesConfirmedToolResult(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		want  string
		cause error
	}{
		{name: "success", want: "succeeded"},
		{name: "business failure", want: "failed", cause: errorsx.InvalidArgument("操作业务校验失败")},
		{name: "timeout", want: "unknown", cause: context.DeadlineExceeded},
		{name: "remote transport failure", want: "unknown", cause: errorsx.Internal("远端调用失败").WithCause(middleware.ErrToolOutcomeUnknown)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			c, ctx, session, message := newAiEventTestCase(t)
			store, err := data.NewAiCheckPointStore(c.aiApprovalRepo)
			if err != nil {
				t.Fatal(err)
			}
			executions := 0
			value := utils.NewTool(&schema.ToolInfo{Name: "write_counter", Desc: "变更计数"}, func(context.Context, map[string]any) (string, error) {
				executions++
				return `{"count":42}`, scenario.cause
			})
			c.aiRuntime = &ai.Runtime{Runtime: agent.NewRuntime(agent.RuntimeConfig{
				Client:          &agent.AssistantClient{AgenticModel: &approvalTestModel{}, ContextWindow: 32768, MaxOutputTokens: 1024, DisableStreaming: true},
				AdminTools:      []agent.Tool{value},
				Checker:         approvalTestChecker{},
				CheckPointStore: store,
			})}
			run := &models.AiToolApproval{ID: 301, ApprovalID: "seed", TenantID: session.TenantID, UserID: session.UserID, SessionID: session.ID, MessageID: message.ID, RuntimeState: "{}", PendingCalls: "[]", CheckpointData: []byte{}, Status: aiRunRunning, ExpiresAt: time.Now().Add(time.Minute)}
			if err = c.aiApprovalRepo.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			var reply *ai.Response
			reply, err = c.aiRuntime.Run(ctx, ai.RuntimeInput{CheckPointID: "301", Terminal: "admin", Content: "更新计数"})
			if err != nil || reply.Pending == nil || executions != 0 {
				t.Fatalf("工具未进入审批: %v %+v", err, reply)
			}
			if err = c.saveAiRuntime(ctx, run, reply, aiRunCompleted); err != nil {
				t.Fatal(err)
			}
			if err = c.finishAiMessage(ctx, session, message, reply, 0, 0, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
				t.Fatal(err)
			}
			req := &basev1.DecideAiToolApprovalRequest{SessionId: "101", MessageId: "201", ApprovalId: reply.Approval.ID, Approved: true}
			var response *basev1.SendAiMessageResponse
			response, err = c.DecideAiToolApproval(ctx, req)
			if err != nil || executions != 1 {
				t.Fatalf("审批执行次数=%d，err=%v", executions, err)
			}
			assertAiApprovalOutcome(t, response.GetMessages()[0], req.GetApprovalId(), "approved", scenario.want, false)
			if scenario.want == "unknown" {
				assertAiReplyErrorCode(t, response.GetMessages()[0].GetOutputContent().GetBlocksJson(), "TOOL_OUTCOME_UNKNOWN")
			}
			if _, err = c.DecideAiToolApproval(ctx, req); err == nil || executions != 1 {
				t.Fatal("旧确认重复执行")
			}
			var refreshed *basev1.ListAiMessageResponse
			refreshed, err = c.ListAiMessage(ctx, &basev1.ListAiMessageRequest{SessionId: "101"})
			if err != nil {
				t.Fatal(err)
			}
			assertAiApprovalOutcome(t, refreshed.GetMessages()[0], req.GetApprovalId(), "approved", scenario.want, false)
		})
	}
}

// TestAiReplyFailureUsesAuthoritativeError 验证失败状态依据错误类型生成，模型报错文本不参与分类。
func TestAiReplyFailureUsesAuthoritativeError(t *testing.T) {
	c := &AiMessageCase{}
	for _, scenario := range []struct {
		name  string
		cause error
		tools []ai.ToolUsage
		want  string
	}{
		{name: "timeout", cause: fmt.Errorf("provider: %w", context.DeadlineExceeded), want: "REPLY_TIMEOUT"},
		{name: "permission", cause: fmt.Errorf("business: %w", errorsx.PermissionDenied("无权访问")), want: "PERMISSION_DENIED"},
		{name: "untyped", cause: errors.New("MODEL_TIMEOUT"), want: ""},
		{name: "model timeout after known tool failure", cause: context.DeadlineExceeded, tools: []ai.ToolUsage{{Type: "function", Status: "error"}}, want: "REPLY_TIMEOUT"},
		{name: "tool result unknown", cause: fmt.Errorf("tool transport: %w", middleware.ErrToolOutcomeUnknown), want: "TOOL_OUTCOME_UNKNOWN"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			reply := c.buildAiFailedReply(t.Context(), &ai.Response{Tools: scenario.tools}, scenario.cause, "", "")
			assertAiReplyErrorCode(t, reply.BlocksJSON, scenario.want)
		})
	}
}

// TestAiReplyUnknownOutcomeBlocksRegeneration 验证外部结果未确认时不能重生成覆盖原消息。
func TestAiReplyUnknownOutcomeBlocksRegeneration(t *testing.T) {
	c, ctx, _, message := newAiEventTestCase(t)
	blocks := `[{"type":"reply_error","code":"TOOL_OUTCOME_UNKNOWN"}]`
	query := c.aiMessageRepo.Query(ctx).AiMessage
	_, err := query.WithContext(ctx).Where(query.ID.Eq(message.ID)).UpdateColumnSimple(
		query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)),
		query.OutputContent.Value(ai.MarshalReplyContent(&ai.Response{BlocksJSON: blocks})),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.RegenerateAiMessage(ctx, &basev1.RegenerateAiMessageRequest{SessionId: "101", MessageId: "201"})
	if !errors.Is(err, errorsx.Conflict("")) {
		t.Fatalf("结果未知时未阻止重生成: %v", err)
	}
}

// assertAiReplyErrorCode 核对回复块展示的服务端错误类型。
func assertAiReplyErrorCode(t *testing.T, raw, want string) {
	t.Helper()
	var blocks []struct{ Type, Code string }
	if raw != "" {
		err := json.Unmarshal([]byte(raw), &blocks)
		if err != nil {
			t.Fatal(err)
		}
	}
	got := ""
	for _, block := range blocks {
		if block.Type == "reply_error" {
			got = block.Code
		}
	}
	if got != want {
		t.Fatalf("错误类型=%q，期望%q", got, want)
	}
}

// assertAiApprovalOutcome 验证发送响应和刷新响应共同使用的审批块格式。
func assertAiApprovalOutcome(t *testing.T, item *basev1.AiMessage, id, state, execution string, retryable bool) {
	t.Helper()
	var blocks []struct {
		Type      string
		ID        string `json:"approval_id"`
		State     string
		Execution string
		Retryable bool
	}
	raw := item.GetOutputContent().GetBlocksJson()
	err := json.Unmarshal([]byte(raw), &blocks)
	if err != nil {
		t.Fatalf("审批块无法解析: %s | %v", raw, err)
	}
	for _, block := range blocks {
		if block.Type == "approval_outcome" && block.ID == id {
			if block.State != state || block.Execution != execution || block.Retryable != retryable {
				t.Fatalf("审批块=%+v，期望 %s/%s/%t", block, state, execution, retryable)
			}
			return
		}
	}
	t.Fatalf("缺少审批块 %q: %s", id, raw)
}

type approvalTestModel struct{ calls int }

// Generate 通过现有动态发现选择真实测试工具，并在恢复后返回正文。
func (m *approvalTestModel) Generate(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.AgenticMessage, error) {
	m.calls++
	switch m.calls {
	case 1:
		return &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.FunctionToolCall{CallID: "search-1", Name: "tool_search", Arguments: `{"query":"select:write_counter"}`}),
		}}, nil
	case 2:
		return &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.FunctionToolCall{CallID: "write-1", Name: "write_counter", Arguments: "{}"}),
		}}, nil
	default:
		return agentmessage.AIText("操作回复"), nil
	}
}

// Stream 复用确定性的测试模型输出。
func (m *approvalTestModel) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	value, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{value}), nil
}

type approvalTestChecker struct{}

// ToolConfigs 只开放用于复现审批的写入工具。
func (approvalTestChecker) ToolConfigs(context.Context, string, []string) (map[string]agent.ToolConfig, error) {
	return map[string]agent.ToolConfig{"write_counter": {Enabled: true}}, nil
}

// PrepareToolCall 在工具实际执行前要求确认。
func (approvalTestChecker) PrepareToolCall(_ context.Context, _ string, _ string, arguments string) (string, bool, error) {
	return arguments, true, nil
}
