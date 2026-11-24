package biz

import (
	"context"
	"database/sql"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/callback"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm/clause"
)

// TestAiEventReadOnly 验证补读不会偷偷过期消息、恢复模型或执行工具。
func TestAiEventReadOnly(t *testing.T) {
	c, ctx, session, message := newAiEventTestCase(t)
	query := c.aiMessageRepo.Query(ctx).AiMessage
	old := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	_, err := query.WithContext(ctx).Where(query.ID.Eq(message.ID)).UpdateColumnSimple(query.UpdatedAt.Value(old), query.CreatedAt.Value(old))
	if err != nil {
		t.Fatal(err)
	}
	var response *basev1.ListAiMessageEventResponse
	response, err = c.ListAiMessageEvent(ctx, &basev1.ListAiMessageEventRequest{SessionId: strconv.FormatInt(session.ID, 10), MessageId: strconv.FormatInt(message.ID, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if response.Messages[0].Status != basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING {
		t.Fatal("只读事件接口改变了消息状态")
	}
}

// TestAiEventIsolationAndCancellation 验证事件隔离、内部缓存隐藏、断流持久终态与旧运行拒写。
func TestAiEventIsolationAndCancellation(t *testing.T) {
	c, ctx, session, message := newAiEventTestCase(t)
	parent, cancel := context.WithCancel(ctx)
	runCtx, _, err := c.startAiMessageEvents(parent, session, message, nil)
	if err != nil {
		t.Fatal(err)
	}
	callback.EmitActivity(runCtx, callback.Activity{ID: "call-1", Phase: "tool", Status: "running", ToolName: "read"})
	execution := runCtx.Value(aiExecutionContextKey{}).(*dto.AiExecution)
	if err = c.recordAiMessageEvent(runCtx, execution, "context_summary", map[string]any{"secret": "internal"}); err != nil {
		t.Fatal(err)
	}
	cancel()
	callback.EmitActivity(runCtx, callback.Activity{ID: "call-1", Phase: "tool", Status: "failed", ToolName: "read"})
	if err = c.finishAiMessage(runCtx, session, message, &ai.Response{Content: "请求已取消，已发出的外部操作结果未知，不能自动重试"}, 0, 1, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED)); err != nil {
		t.Fatal(err)
	}
	req := &basev1.ListAiMessageEventRequest{SessionId: strconv.FormatInt(session.ID, 10), MessageId: strconv.FormatInt(message.ID, 10)}
	var page *basev1.ListAiMessageEventResponse
	page, err = c.ListAiMessageEvent(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 3 || page.Events[2].Event != "error" || page.RunId != execution.RunID {
		t.Fatalf("持久事件不完整或泄露内部事件: %+v", page.Events)
	}
	for i, event := range page.Events {
		if i > 0 && event.Id <= page.Events[i-1].Id {
			t.Fatal("事件顺序不稳定")
		}
	}
	req.AfterId = page.NextCursor
	var next *basev1.ListAiMessageEventResponse
	next, err = c.ListAiMessageEvent(ctx, req)
	if err != nil || len(next.Events) != 0 {
		t.Fatalf("游标重复读取: %v %+v", err, next)
	}
	for _, owner := range []*authdata.UserTokenPayload{{TenantId: 7, UserId: 4}, {TenantId: 8, UserId: 3}} {
		_, err = c.ListAiMessageEvent(engine.ContextWithAuthClaims(ctx, owner.MakeAuthClaims()), req)
		if err == nil {
			t.Fatal("跨用户或跨租户读取成功")
		}
	}
	if err = c.recordAiMessageEvent(ctx, execution, "activity", dto.AiStreamPayload{}); err == nil {
		t.Fatal("已结束运行仍能追加事件")
	}
	if err = c.markAiMessageGenerating(ctx, session, message, ai.InputContentPayload{}); err != nil {
		t.Fatal(err)
	}
	var newCtx context.Context
	newCtx, _, err = c.startAiMessageEvents(ctx, session, message, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer newCtx.Value(aiExecutionContextKey{}).(*dto.AiExecution).Cancel(nil)
	callback.EmitActivity(newCtx, callback.Activity{ID: "new", Phase: "reasoning", Status: "running"})
	if err = c.recordAiMessageEvent(ctx, execution, "activity", dto.AiStreamPayload{}); err == nil {
		t.Fatal("旧执行可以污染新运行")
	}
	page, err = c.ListAiMessageEvent(ctx, req)
	if err != nil || len(page.Events) != 1 || page.RunId == execution.RunID {
		t.Fatalf("补读混入旧运行: %v %+v", err, page)
	}
}

// TestAiEventTerminalRequiresCommit 验证保存失败时不能伪造终态，让客户端继续只读核实结果。
func TestAiEventTerminalRequiresCommit(t *testing.T) {
	c, ctx, session, message := newAiEventTestCase(t)
	sent := 0
	runCtx, emitter, err := c.startAiMessageEvents(ctx, session, message, dto.AiStreamEmitterFunc(func(dto.AiStreamEvent, dto.AiStreamPayload) error {
		sent++
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer runCtx.Value(aiExecutionContextKey{}).(*dto.AiExecution).Cancel(nil)
	if err = emitter.EmitAiStream(dto.AiStreamEventError, dto.AiStreamPayload{}); err == nil || sent != 0 {
		t.Fatal("尚未持久化的错误被当作权威终态发送")
	}
	if err = c.finishAiMessage(runCtx, session, message, &ai.Response{Content: "done"}, 0, 1, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
		t.Fatal(err)
	}
	if err = emitter.EmitAiStream(dto.AiStreamEventFinish, dto.AiStreamPayload{}); err != nil || sent != 1 {
		t.Fatalf("已持久终态未正常发送: %v", err)
	}
}

// TestAiEventPaginationAndDeletion 验证分页无遗漏、非法游标拒绝以及删除同步清理所有内部事件。
func TestAiEventPaginationAndDeletion(t *testing.T) {
	for _, deleteSession := range []bool{false, true} {
		t.Run(strconv.FormatBool(deleteSession), func(t *testing.T) {
			c, ctx, session, message := newAiEventTestCase(t)
			runCtx, _, err := c.startAiMessageEvents(ctx, session, message, nil)
			if err != nil {
				t.Fatal(err)
			}
			for i := range 102 {
				callback.EmitActivity(runCtx, callback.Activity{ID: strconv.Itoa(i), Phase: "tool", Status: "completed"})
			}
			if err = c.finishAiMessage(runCtx, session, message, &ai.Response{Content: "done"}, 0, 1, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
				t.Fatal(err)
			}
			req := &basev1.ListAiMessageEventRequest{SessionId: "101", MessageId: "201"}
			var page *basev1.ListAiMessageEventResponse
			page, err = c.ListAiMessageEvent(ctx, req)
			if err != nil || len(page.Events) != 100 || !page.HasMore {
				t.Fatalf("首屏分页错误: %v %+v", err, page)
			}
			req.AfterId = page.NextCursor
			page, err = c.ListAiMessageEvent(ctx, req)
			if err != nil || len(page.Events) != 3 || page.HasMore || page.Events[2].Event != "finish" {
				t.Fatalf("尾屏遗漏终态: %v %+v", err, page)
			}
			for _, cursor := range []string{"-1", "invalid", "9223372036854775808"} {
				req.AfterId = cursor
				if _, err = c.ListAiMessageEvent(ctx, req); err == nil {
					t.Fatalf("非法游标被接受: %s", cursor)
				}
			}
			req.AfterId = ""
			req.MessageId = "999"
			if _, err = c.ListAiMessageEvent(ctx, req); err == nil {
				t.Fatal("不存在的消息被接受")
			}
			if deleteSession {
				_, err = c.aiSessionCase.DeleteAiSession(ctx, &basev1.DeleteAiSessionRequest{Id: "101"})
			} else {
				err = c.DeleteAiMessage(ctx, &basev1.DeleteAiMessageRequest{SessionId: "101", MessageId: "201"})
			}
			if err != nil {
				t.Fatal(err)
			}
			query := c.aiMessageRepo.Query(ctx).AiMessageEvent
			var count int64
			count, err = query.WithContext(ctx).Count()
			if err != nil || count != 0 {
				t.Fatalf("删除后遗留事件: count=%d err=%v", count, err)
			}
		})
	}
}

// TestAiMessageToolsRemainBlocked 验证补读、审批和会话入口不会成为Agent或MCP递归调用工具。
func TestAiMessageToolsRemainBlocked(t *testing.T) {
	for _, name := range []string{"base_v1_ai_message_service_list_ai_message_event", "base_v1_ai_message_service_decide_ai_tool_approval", "base_v1_ai_session_service_list_ai_message"} {
		if !agentToolBlocked(&models.BaseAPI{ToolName: name, Method: "GET"}) {
			t.Fatalf("会话工具未封闭: %s", name)
		}
	}
}

// TestAiApprovalDecisionIsSingleUse 验证同一确认编号处理后不能再次领取，拒绝不执行冻结操作。
func TestAiApprovalDecisionIsSingleUse(t *testing.T) {
	c, ctx, session, message := newAiEventTestCase(t)
	c.aiRuntime = &ai.Runtime{Runtime: agent.NewRuntime(agent.RuntimeConfig{})}
	query := c.aiMessageRepo.Query(ctx).AiMessage
	output := ai.MarshalReplyContent(&ai.Response{Approval: &ai.Approval{ID: "single-use"}})
	_, err := query.WithContext(ctx).Where(query.ID.Eq(message.ID)).UpdateColumnSimple(query.Status.Value(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_WAITING_APPROVAL)), query.OutputContent.Value(output))
	if err != nil {
		t.Fatal(err)
	}
	run := &models.AiToolApproval{ID: 301, ApprovalID: "single-use", TenantID: session.TenantID, UserID: session.UserID, SessionID: session.ID, MessageID: message.ID, RuntimeState: `{"terminal":"admin"}`, PendingCalls: `[{"name":"write","interrupt_id":"call-1"}]`, CheckpointData: []byte("frozen"), Status: aiRunPending, ExpiresAt: time.Now().Add(time.Minute)}
	if err = c.aiApprovalRepo.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	req := &basev1.DecideAiToolApprovalRequest{SessionId: "101", MessageId: "201", ApprovalId: "single-use", Approved: false}
	if _, err = c.DecideAiToolApproval(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err = c.DecideAiToolApproval(ctx, req); err == nil {
		t.Fatal("同一审批被重复领取")
	}
	var saved *models.AiToolApproval
	saved, err = c.aiApprovalRepo.FindByID(ctx, run.ID)
	if err != nil || saved.Status != aiRunRejected || len(saved.CheckpointData) != 0 {
		t.Fatalf("审批拒绝后仍可恢复: %v %+v", err, saved)
	}
}

// TestAiFileTenantBoundary 验证文件仍按租户共享且跨租户不可访问，不擅自改成仅创建者可见。
func TestAiFileTenantBoundary(t *testing.T) {
	c, ctx, _, _ := newAiEventTestCase(t)
	file := &models.AiFile{ID: 301, TenantID: 7, CreatedBy: 99, Status: aiFileStatusActive}
	err := c.aiFileCase.Create(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.aiFileCase.ownedAiFile(ctx, file.ID, aiFileStatusActive); err != nil {
		t.Fatal("同租户文件共享语义被改变", err)
	}
	other := engine.ContextWithAuthClaims(ctx, (&authdata.UserTokenPayload{TenantId: 8, UserId: 3}).MakeAuthClaims())
	if _, err = c.aiFileCase.ownedAiFile(other, file.ID, aiFileStatusActive); err == nil {
		t.Fatal("跨租户文件读取成功")
	}
}

// newAiEventTestCase 用独立内存数据库构造消息链路，不读取或改动开发库记录。
func newAiEventTestCase(t *testing.T) (*AiMessageCase, context.Context, *models.AiSession, *models.AiMessage) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.AiSession{}, &models.AiMessage{}, &models.AiToolApproval{}, &models.AiMessageEvent{}, &models.AiFile{}); err != nil {
		t.Fatal(err)
	}
	installTestTenantScope(t, db)
	// SQLite 的 BIGINT 主键不等同 INTEGER ROWID，用测试回调模拟MySQL自增，不改生成模型。
	var eventID atomic.Int64
	if err = db.Callback().Create().Before("gorm:create").Register("test_event_id", func(tx *gorm.DB) {
		if tx.Statement.Schema.Table == models.TableNameAiMessageEvent {
			tx.Statement.SetColumn("ID", eventID.Add(1))
		}
	}); err != nil {
		t.Fatal(err)
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	var store *data.Data
	store, err = data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	messageRepo := data.NewAiMessageRepository(store)
	approvalRepo := data.NewAiToolApprovalRepository(store)
	sessionCase := NewAiSessionCase(&biz.BaseCase{}, data.NewTransaction(store), data.NewAiSessionRepository(store), messageRepo, approvalRepo, nil, nil, nil)
	c := &AiMessageCase{BaseCase: &biz.BaseCase{}, tx: store, aiMessageRepo: messageRepo, aiApprovalRepo: approvalRepo, aiSessionCase: sessionCase, aiFileCase: NewAiFileCase(&biz.BaseCase{}, data.NewAiFileRepository(store))}
	claims := (&authdata.UserTokenPayload{TenantId: 7, UserId: 3}).MakeAuthClaims()
	ctx := engine.ContextWithAuthClaims(t.Context(), claims)
	now := time.Now().Truncate(time.Millisecond)
	session := &models.AiSession{ID: 101, TenantID: 7, UserID: 3, Title: "test", Plugins: "[]", CreatedAt: now, UpdatedAt: now}
	if err = sessionCase.Create(ctx, session); err != nil {
		t.Fatal(err)
	}
	message := &models.AiMessage{ID: 201, TenantID: 7, UserID: 3, SessionID: session.ID, InputContent: "{}", OutputContent: "{}", Attachments: "[]", Tools: "[]", Token: "{}", Status: 1, CreatedAt: now, UpdatedAt: now}
	if err = messageRepo.Create(ctx, message); err != nil {
		t.Fatal(err)
	}
	return c, ctx, session, message
}

// installTestTenantScope 给测试库安装与 kratos-kit 数据隔离回调同语义的租户条件注入；
// 测试库不经 kit 客户端引导，缺失时查询/更新/删除不再自动按登录租户收敛。
func installTestTenantScope(t *testing.T, db *gorm.DB) {
	t.Helper()
	scope := func(tx *gorm.DB) {
		authInfo, err := auth.FromContext(tx.Statement.Context)
		if err != nil || authInfo == nil || authInfo.TenantCode == databasegorm.DefaultTenantCode || authInfo.TenantId <= 0 {
			return
		}
		if tx.Statement.Schema == nil {
			return
		}
		if _, hasTenant := tx.Statement.Schema.FieldsByDBName["tenant_id"]; !hasTenant {
			return
		}
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"}, Value: authInfo.TenantId}}})
	}
	if err := db.Callback().Query().Before("gorm:query").Register("test_tenant_scope", scope); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Row().Before("gorm:row").Register("test_tenant_scope", scope); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test_tenant_scope", scope); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("test_tenant_scope", scope); err != nil {
		t.Fatal(err)
	}
}
