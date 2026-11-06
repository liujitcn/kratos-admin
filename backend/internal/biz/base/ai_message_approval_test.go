package biz

import (
	"context"
	"crypto/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	agentmodel "github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	appdata "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	modelconfig "github.com/liujitcn/kratos-kit/ai/model"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// approvalBusinessModel 产生一次业务写调用，然后结束对话。
type approvalBusinessModel struct{ count atomic.Int32 }

// Generate 返回固定工具调用或最终文本。
func (m *approvalBusinessModel) Generate(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.AgenticMessage, error) {
	if m.count.Add(1) == 1 {
		return &schema.AgenticMessage{Role: schema.AgenticRoleTypeAssistant, ContentBlocks: []*schema.ContentBlock{schema.NewContentBlock(&schema.FunctionToolCall{CallID: "write1", Name: "write", Arguments: "{}"})}}, nil
	}
	return message.AIText("done"), nil
}

// Stream 将固定模型输出包装为流。
func (m *approvalBusinessModel) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	reply, err := m.Generate(ctx, input, opts...)
	return schema.StreamReaderFromArray([]*schema.AgenticMessage{reply}), err
}

// approvalBusinessGate 模拟已经授权但每次写入仍须确认的业务接口。
type approvalBusinessGate struct{}

// ToolConfigs 在集成测试中启用固定工具。
func (approvalBusinessGate) ToolConfigs(_ context.Context, _ string, names []string) (map[string]agent.ToolConfig, error) {
	configs := make(map[string]agent.ToolConfig)
	for _, name := range names {
		configs[name] = agent.ToolConfig{Enabled: true}
	}
	return configs, nil
}

// PrepareToolCall 固定要求人工确认。
func (approvalBusinessGate) PrepareToolCall(_ context.Context, _ string, _ string, arguments string) (string, bool, error) {
	return arguments, true, nil
}

// TestAiToolApprovalIntegration 在独立 MySQL 测试库验证属主隔离、过期、删除清理和并发确认防重放。
func TestAiToolApprovalIntegration(t *testing.T) {
	store := newAiApprovalTestData(t)
	for _, scenario := range []string{"approve", "reject", "expired", "delete", "delete_session", "deleted_run", "historical", "lock_expired", "large_attachment", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			var effects atomic.Int32
			value, err := agent.InferTool("write", "业务写入", func(context.Context, struct{}) (string, error) {
				effects.Add(1)
				return "ok", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			messageRepo := data.NewAiMessageRepository(store)
			approvalRepo := data.NewAiToolApprovalRepository(store)
			sessionCase := NewAiSessionCase(&biz.BaseCase{}, store, data.NewAiSessionRepository(store), messageRepo, approvalRepo, nil)
			var checkpoints *appdata.AiCheckPointStore
			checkpoints, err = appdata.NewAiCheckPointStore(approvalRepo)
			if err != nil {
				t.Fatal(err)
			}
			client := agentmodel.NewAssistantClient(&modelconfig.ModelConfig{APIType: modelconfig.APITypeResponses})
			client.AgenticModel = &approvalBusinessModel{}
			runtime := &ai.Runtime{Runtime: agent.NewRuntime(agent.RuntimeConfig{Client: client, Checker: approvalBusinessGate{}, AdminTools: []agent.Tool{value}, CheckPointStore: checkpoints})}
			c := &AiMessageCase{BaseCase: &biz.BaseCase{}, tx: store, aiMessageRepo: messageRepo, aiApprovalRepo: approvalRepo, aiSessionCase: sessionCase, aiRuntime: runtime}
			now := time.Now().Truncate(time.Millisecond)
			session := &models.AiSession{TenantID: 7, UserID: 3, Terminal: ai.TerminalAdmin, CreatedAt: now, UpdatedAt: now}
			if err = sessionCase.Create(context.Background(), session); err != nil {
				t.Fatal(err)
			}
			item := &models.AiMessage{TenantID: 7, UserID: 3, SessionID: session.ID, InputContent: "{}", OutputContent: "{}", Attachments: "[]", Tools: "[]", Token: "{}", Status: int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING), CreatedAt: now, UpdatedAt: now}
			if err = messageRepo.Create(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			payload := &authdata.UserTokenPayload{TenantId: 7, TenantCode: "tenant7", UserId: 3, RoleCode: "role"}
			ctx := engine.ContextWithAuthClaims(context.Background(), payload.MakeAuthClaims())
			var reply *ai.Response
			input := ai.RuntimeInput{Terminal: "admin", Content: "write"}
			if scenario == "large_attachment" {
				input.Attachments = []ai.Attachment{{Name: "report.pdf", MIMEType: "application/pdf", Bytes: make([]byte, 20*1024*1024)}}
				if _, err = rand.Read(input.Attachments[0].Bytes); err != nil {
					t.Fatal(err)
				}
			}
			reply, err = c.runAiRuntime(ctx, session, item.ID, input, nil)
			if err != nil {
				t.Fatal(err)
			}
			if reply.Approval == nil || effects.Load() != 0 {
				t.Fatalf("首次调用未暂停: reply=%+v effects=%d", reply, effects.Load())
			}
			saveCtx := ctx
			if scenario == "cancelled" {
				var cancel context.CancelFunc
				saveCtx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if err = c.finishAiMessage(saveCtx, session, item, reply, now, 0, 0, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
				t.Fatal(err)
			}
			req := &basev1.DecideAiToolApprovalRequest{SessionId: strconv.FormatInt(session.ID, 10), MessageId: strconv.FormatInt(item.ID, 10), ApprovalId: reply.Approval.ID, Approved: scenario != "reject"}
			for _, outsider := range []*authdata.UserTokenPayload{{TenantId: 7, UserId: 4}, {TenantId: 8, UserId: 3}} {
				_, err = c.DecideAiToolApproval(engine.ContextWithAuthClaims(ctx, outsider.MakeAuthClaims()), req)
				if err == nil || effects.Load() != 0 {
					t.Fatal("非属主能够操作审批")
				}
			}
			if scenario == "delete_session" || scenario == "deleted_run" {
				deleteReq := &basev1.DeleteAiSessionRequest{Id: req.SessionId}
				query := approvalRepo.Query(ctx).AiToolApproval
				var run *models.AiToolApproval
				run, err = query.WithContext(ctx).Where(query.ApprovalID.Eq(req.ApprovalId)).First()
				if err != nil {
					t.Fatal(err)
				}
				wantStatuses := map[int64]int32{run.ID: aiRunRejected}
				for _, fixture := range []struct{ status, want int32 }{{aiRunRunning, aiRunRejected}, {aiRunCompleted, aiRunCompleted}, {aiRunExpired, aiRunExpired}} {
					extra := *run
					extra.ID = 0
					extra.ApprovalID = uuid.NewString()
					extra.Status = fixture.status
					if err = approvalRepo.Create(ctx, &extra); err != nil {
						t.Fatal(err)
					}
					wantStatuses[extra.ID] = fixture.want
				}
				otherSession := *session
				otherSession.ID = 0
				if err = sessionCase.Create(ctx, &otherSession); err != nil {
					t.Fatal(err)
				}
				otherMessage := *item
				otherMessage.ID = 0
				otherMessage.SessionID = otherSession.ID
				if err = messageRepo.Create(ctx, &otherMessage); err != nil {
					t.Fatal(err)
				}
				otherRun := *run
				otherRun.ID = 0
				otherRun.ApprovalID = uuid.NewString()
				otherRun.SessionID = otherSession.ID
				otherRun.MessageID = otherMessage.ID
				if err = approvalRepo.Create(ctx, &otherRun); err != nil {
					t.Fatal(err)
				}
				for _, outsider := range []*authdata.UserTokenPayload{{TenantId: 7, UserId: 4}, {TenantId: 8, UserId: 3}} {
					_, err = sessionCase.DeleteAiSession(engine.ContextWithAuthClaims(ctx, outsider.MakeAuthClaims()), deleteReq)
					if err == nil {
						t.Fatal("非属主能够删除会话")
					}
				}
				var saved *models.AiToolApproval
				saved, err = approvalRepo.FindByID(ctx, run.ID)
				if err != nil || saved.Status != aiRunPending || len(saved.CheckpointData) == 0 {
					t.Fatalf("非属主删除改变了审批状态或快照: %v", err)
				}
				_, err = sessionCase.DeleteAiSession(ctx, deleteReq)
				if err != nil {
					t.Fatal(err)
				}
				_, err = c.DecideAiToolApproval(ctx, req)
				if err == nil || effects.Load() != 0 {
					t.Fatal("已删除会话仍能恢复执行")
				}
				if scenario == "deleted_run" {
					_, err = c.runAiRuntime(ctx, session, item.ID, input, nil)
					if err == nil {
						t.Fatal("已删除会话仍能创建运行记录")
					}
					var count int64
					count, err = query.WithContext(ctx).Where(query.SessionID.Eq(session.ID)).Count()
					if err != nil || count != int64(len(wantStatuses)) {
						t.Fatalf("删除后创建了孤儿运行: count=%d err=%v", count, err)
					}
				}
				for id, want := range wantStatuses {
					saved, err = approvalRepo.FindByID(ctx, id)
					if err != nil {
						t.Fatal(err)
					}
					if saved.Status != want || len(saved.CheckpointData) != 0 {
						t.Fatalf("删除会话未清理运行快照: status=%d want=%d bytes=%d", saved.Status, want, len(saved.CheckpointData))
					}
					if err = checkpoints.Set(ctx, strconv.FormatInt(id, 10), []byte("late checkpoint")); err == nil {
						t.Fatal("删除后迟到的快照仍能写入")
					}
					if err = c.saveAiRuntime(ctx, saved, &ai.Response{Pending: reply.Pending}, aiRunCompleted); err == nil {
						t.Fatal("删除后迟到的运行仍能重新申请审批")
					}
				}
				saved, err = approvalRepo.FindByID(ctx, otherRun.ID)
				if err != nil || saved.Status != aiRunPending || len(saved.CheckpointData) == 0 {
					t.Fatalf("删除会话影响了其他会话的审批状态或快照: %v", err)
				}
				return
			}
			if scenario == "historical" {
				req.Approved = false
				_, err = c.DecideAiToolApproval(ctx, req)
				if err != nil {
					t.Fatal(err)
				}
				later := *item
				later.ID = 0
				later.Status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
				later.CreatedAt = time.Now().Add(time.Second)
				if err = messageRepo.Create(ctx, &later); err != nil {
					t.Fatal(err)
				}
				item.Status = int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)
				err = c.markAiMessageGenerating(ctx, session, item, ai.InputContentPayload{}, time.Now())
				if err == nil {
					t.Fatal("历史消息不应重新生成，以免产生无法处理的旧审批")
				}
				return
			}
			if scenario == "lock_expired" {
				query := approvalRepo.Query(ctx).AiToolApproval
				_, err = query.WithContext(ctx).Where(query.ApprovalID.Eq(req.ApprovalId)).UpdateSimple(query.ExpiresAt.Value(time.Now().Add(500 * time.Millisecond)))
				if err != nil {
					t.Fatal(err)
				}
				locked := make(chan struct{})
				lockDone := make(chan error, 1)
				go func() {
					lockDone <- store.Transaction(ctx, func(txCtx context.Context) error {
						lockErr := sessionCase.lockAiSession(txCtx, session)
						close(locked)
						if lockErr != nil {
							return lockErr
						}
						time.Sleep(800 * time.Millisecond)
						return nil
					})
				}()
				<-locked
				_, err = c.DecideAiToolApproval(ctx, req)
				if err != nil {
					t.Fatal(err)
				}
				if err = <-lockDone; err != nil {
					t.Fatal(err)
				}
				if effects.Load() != 0 {
					t.Fatal("锁等待跨过确认期限后仍执行了写操作")
				}
				return
			}
			if scenario == "delete" {
				if err = c.DeleteAiMessage(ctx, &basev1.DeleteAiMessageRequest{SessionId: req.SessionId, MessageId: req.MessageId}); err != nil {
					t.Fatal(err)
				}
				_, err = c.DecideAiToolApproval(ctx, req)
				if err == nil || effects.Load() != 0 {
					t.Fatal("已删除消息仍能恢复执行")
				}
				return
			}
			if scenario == "expired" {
				query := approvalRepo.Query(ctx).AiToolApproval
				_, err = query.WithContext(ctx).Where(query.ApprovalID.Eq(req.ApprovalId)).UpdateSimple(query.ExpiresAt.Value(now.Add(-time.Second)))
				if err != nil {
					t.Fatal(err)
				}
			}
			var accepted atomic.Int32
			var group sync.WaitGroup
			for i := 0; i < 4; i++ {
				group.Go(func() {
					_, callErr := c.DecideAiToolApproval(ctx, req)
					if callErr == nil {
						accepted.Add(1)
					}
				})
			}
			group.Wait()
			want := int32(0)
			if scenario == "approve" || scenario == "large_attachment" || scenario == "cancelled" {
				want = 1
			}
			if accepted.Load() != 1 || effects.Load() != want {
				t.Fatalf("审批重复或错误执行: accepted=%d effects=%d want=%d", accepted.Load(), effects.Load(), want)
			}
			_, err = c.DecideAiToolApproval(ctx, req)
			if err == nil || effects.Load() != want {
				t.Fatal("旧确认编号被重复执行")
			}
		})
	}
}
