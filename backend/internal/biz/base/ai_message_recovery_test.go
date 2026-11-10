package biz

import (
	"os"
	"strconv"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAiMessageRecoveryIntegration 验证过期生成可解除阻塞且旧运行不能覆盖新一轮结果。
func TestAiMessageRecoveryIntegration(t *testing.T) {
	store := newAiApprovalTestData(t)
	for _, scenario := range []struct {
		name         string
		runStatus    int32
		active       bool
		freshMessage bool
	}{{"orphan", -1, false, false}, {"expired_run", aiRunRunning, false, false}, {"unfinished_save", aiRunCompleted, false, false}, {"active_orphan", -1, true, true}, {"active_run", aiRunRunning, true, false}, {"direct_delete", aiRunRunning, false, false}, {"direct_lookup", -1, false, false}} {
		t.Run(scenario.name, func(t *testing.T) {
			var err error
			messageRepo := data.NewAiMessageRepository(store)
			approvalRepo := data.NewAiToolApprovalRepository(store)
			sessionCase := NewAiSessionCase(&biz.BaseCase{}, store, data.NewAiSessionRepository(store), messageRepo, approvalRepo, nil)
			c := &AiMessageCase{BaseCase: &biz.BaseCase{}, tx: store, aiMessageRepo: messageRepo, aiApprovalRepo: approvalRepo, aiSessionCase: sessionCase}
			now := time.Now().Truncate(time.Millisecond)
			startedAt := now.Add(-11 * time.Minute)
			if scenario.freshMessage {
				startedAt = now
			}
			session := &models.AiSession{TenantID: 7, UserID: 3, Terminal: ai.TerminalAdmin, CreatedAt: startedAt, UpdatedAt: startedAt}
			if err = sessionCase.Create(t.Context(), session); err != nil {
				t.Fatal(err)
			}
			item := &models.AiMessage{TenantID: 7, UserID: 3, SessionID: session.ID, InputContent: "{}", OutputContent: "{}", Attachments: "[]", Tools: "[]", Token: "{}", Status: int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING), CreatedAt: startedAt, UpdatedAt: startedAt}
			if err = messageRepo.Create(t.Context(), item); err != nil {
				t.Fatal(err)
			}
			payload := &authdata.UserTokenPayload{TenantId: 7, UserId: 3}
			ctx := engine.ContextWithAuthClaims(t.Context(), payload.MakeAuthClaims())
			var run *models.AiToolApproval
			if scenario.runStatus >= 0 {
				expiresAt := now.Add(-time.Minute)
				if scenario.active {
					expiresAt = now.Add(time.Minute)
				}
				run = &models.AiToolApproval{ApprovalID: uuid.NewString(), TenantID: 7, UserID: 3, SessionID: session.ID, MessageID: item.ID, RuntimeState: "{}", PendingCalls: "[]", CheckpointData: []byte("stale checkpoint"), Status: scenario.runStatus, ExpiresAt: expiresAt, CreatedAt: startedAt, UpdatedAt: startedAt}
				if err = approvalRepo.Create(ctx, run); err != nil {
					t.Fatal(err)
				}
				if scenario.name == "expired_run" && c.saveAiRuntime(ctx, run, nil, aiRunCompleted) == nil {
					t.Fatal("过期运行在清理前仍能保存或续期")
				}
			}
			sessionID := strconv.FormatInt(session.ID, 10)
			messageID := strconv.FormatInt(item.ID, 10)
			outsider := &authdata.UserTokenPayload{TenantId: 7, UserId: 4}
			_, err = c.ListAiMessage(engine.ContextWithAuthClaims(ctx, outsider.MakeAuthClaims()), &basev1.ListAiMessageRequest{SessionId: sessionID})
			if err == nil {
				t.Fatal("非属主可以访问并恢复会话")
			}
			if scenario.name == "direct_delete" {
				if err = c.DeleteAiMessage(ctx, &basev1.DeleteAiMessageRequest{SessionId: sessionID, MessageId: messageID}); err != nil {
					t.Fatalf("未刷新也应能删除过期生成: %v", err)
				}
				return
			}
			var current *models.AiMessage
			if scenario.name == "direct_lookup" {
				current, _, err = c.findCurrentUserMessage(ctx, sessionID, messageID)
			} else {
				_, err = c.ListAiMessage(ctx, &basev1.ListAiMessageRequest{SessionId: sessionID})
				if err == nil {
					current, err = messageRepo.FindByID(ctx, item.ID)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario.active {
				if current.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_GENERATING) {
					t.Fatal("仍有效的生成被提前终止")
				}
				if err = c.ensureAiSessionIdle(ctx, session.TenantID, session.ID); err == nil {
					t.Fatal("仍有效的生成未阻止新一轮请求")
				}
				return
			}
			if current.Status != int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED) {
				t.Fatalf("过期生成仍阻塞会话: status=%d", current.Status)
			}
			if err = c.ensureAiSessionIdle(ctx, session.TenantID, session.ID); err != nil {
				t.Fatal(err)
			}
			if run != nil && scenario.runStatus == aiRunRunning {
				var saved *models.AiToolApproval
				saved, err = approvalRepo.FindByID(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if saved.Status != aiRunExpired || len(saved.CheckpointData) != 0 {
					t.Fatalf("过期运行仍保留可恢复的快照: status=%d bytes=%d", saved.Status, len(saved.CheckpointData))
				}
				if err = c.saveAiRuntime(ctx, run, nil, aiRunCompleted); err == nil {
					t.Fatal("过期运行仍可保存状态")
				}
			}
			if err = c.finishAiMessage(ctx, session, item, &ai.Response{Content: "late result"}, time.Now(), 0, 0, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err == nil {
				t.Fatal("过期请求覆盖了恢复后的消息")
			}
			if err = c.markAiMessageGenerating(ctx, session, current, ai.InputContentPayload{}, time.Now()); err != nil {
				t.Fatal(err)
			}
			if err = c.finishAiMessage(ctx, session, item, &ai.Response{Content: "late result"}, time.Now(), 0, 0, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err == nil {
				t.Fatal("过期请求覆盖了新一轮生成")
			}
			if err = c.finishAiMessage(ctx, session, current, &ai.Response{Content: "new result"}, time.Now(), 0, 0, int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS)); err != nil {
				t.Fatalf("当前轮结果未正常保存: %v", err)
			}
		})
	}
}

// newAiApprovalTestData 在独立测试库内按生成模型准备审批与消息表。
func newAiApprovalTestData(t *testing.T) *data.Data {
	t.Helper()
	dsn := os.Getenv("APPROVAL_TEST_DSN")
	if dsn == "" {
		t.Skip("设置 APPROVAL_TEST_DSN 指向独立测试库以执行数据库集成测试")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&models.AiSession{}, &models.AiMessage{}, &models.AiToolApproval{}); err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	return store
}
