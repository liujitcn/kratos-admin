package biz

import (
	"fmt"
	"strconv"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"gorm.io/gorm"
)

// TestAiShortcutResumeAndRetry 验证继续和重试都使用已授权的动作和服务端保存的结果。
func TestAiShortcutResumeAndRetry(t *testing.T) {
	c, ctx, session, source := newAiEventTestCase(t, &models.AiShortcut{}, &models.BaseUser{})
	store := c.tx.(*data.Data)
	query := store.Query(ctx).AiMessage
	db := query.WithContext(ctx).UnderlyingDB() //nolint:forbidigo // 仅为 SQLite 测试表注册 BIGINT 主键自增回调。
	err := db.Callback().Create().Before("gorm:create").Register("test_shortcut_message_id", func(tx *gorm.DB) {
		if tx.Statement.Schema.Table == models.TableNameAiMessage {
			tx.Statement.SetColumn("ID", source.ID+1)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	sessionQuery := store.Query(ctx).AiSession
	if _, err = sessionQuery.WithContext(ctx).Where(sessionQuery.ID.Eq(session.ID)).UpdateSimple(sessionQuery.Terminal.Value(ai.TerminalAdmin)); err != nil {
		t.Fatal(err)
	}
	c.baseUserCase = NewBaseUserCase(c.BaseCase, data.NewBaseUserRepository(store))
	if err = c.baseUserCase.Create(ctx, &models.BaseUser{ID: session.UserID, TenantID: session.TenantID, UserName: "tester"}); err != nil {
		t.Fatal(err)
	}
	c.aiRuntime = &ai.Runtime{Runtime: agent.NewRuntime(agent.RuntimeConfig{})}
	repo := data.NewAiShortcutRepository(store)
	c.aiToolCase = NewAiToolCase(c.BaseCase, c.aiRuntime, repo, nil)
	const flow = "ai-shortcut/7/1"
	shortcut := &models.AiShortcut{
		ID: 7, TenantID: session.TenantID, Terminal: ai.TerminalAdmin, Status: int32(commonv1.Status_STATUS_ENABLE), Revision: 1,
		NodeConfig: `{
			"nodes":[{"id":"start","type":"START","step":"start"},{"id":"confirm","type":"CONFIRM","step":"confirm"},
				{"id":"branch","type":"BRANCH","step":"branch"},{"id":"matched","type":"CONFIRM","step":"matched"},{"id":"end","type":"END","step":"end"}],
			"edges":[{"from":"start","to":"confirm"},{"from":"confirm","to":"branch"},
				{"from":"branch","to":"matched","condition":"allowed == true"},{"from":"branch","to":"end","condition":"default"},{"from":"matched","to":"end"}]
		}`,
	}
	if err = repo.Create(ctx, shortcut); err != nil {
		t.Fatal(err)
	}
	reply := &ai.Response{
		Flow: flow, Step: "confirm", FlowResult: new(`{"allowed":true}`),
		BlocksJSON: fmt.Sprintf(`[{"action":{"flow":%q,"step":"confirm","type":"shortcut_continue"}}]`, flow),
	}
	const success = basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS
	if err = c.finishAiMessage(ctx, session, source, reply, 0, 0, int32(success)); err != nil {
		t.Fatal(err)
	}
	sessionID := strconv.FormatInt(session.ID, 10)
	sourceID := strconv.FormatInt(source.ID, 10)
	var response *basev1.SendAiMessageResponse
	response, err = c.SendAiMessage(ctx, &basev1.SendAiMessageRequest{
		SessionId: sessionID, Content: "继续",
		Action: &basev1.AiAction{
			Flow: flow, Step: "confirm", Type: "shortcut_continue",
			SourceMessageId: sourceID, FlowVersion: source.ID, ActionId: sourceID + ":0",
			PayloadJson: `{"allowed":false}`,
		},
	})
	if err != nil || len(response.GetMessages()) != 1 || response.Messages[0].GetOutputContent().GetStep() != "matched" {
		t.Fatalf("继续未使用服务端保存的结果: %v %v", response, err)
	}
	message := response.Messages[0]
	response, err = c.RegenerateAiMessage(ctx, &basev1.RegenerateAiMessageRequest{SessionId: sessionID, MessageId: message.GetId()})
	if err != nil || len(response.GetMessages()) != 1 || response.Messages[0].GetOutputContent().GetStep() != "matched" {
		t.Fatalf("重新生成没有恢复同一流程步骤: %v %v", response, err)
	}
	const failed = basev1.AiMessageStatus_AI_MESSAGE_STATUS_FAILED
	target := query.WithContext(ctx).Where(query.ID.Eq(source.ID + 1))
	if _, err = target.UpdateSimple(query.Status.Value(int32(failed))); err != nil {
		t.Fatal(err)
	}
	retry := &basev1.RetryAiUserMessageRequest{SessionId: sessionID, MessageId: message.GetId()}
	response, err = c.RetryAiUserMessage(ctx, retry)
	if err != nil || len(response.GetMessages()) != 1 || response.Messages[0].GetOutputContent().GetStep() != "matched" {
		t.Fatalf("重试没有恢复同一流程步骤: %v %v", response, err)
	}
	if _, err = target.UpdateSimple(
		query.Status.Value(int32(failed)),
		query.InputContent.Value(ai.MarshalInputContent("继续", nil, "")),
		query.OutputContent.Value(ai.MarshalEmptyOutputContent()),
		query.Tools.Value(ai.MarshalTools([]ai.ToolUsage{{Name: "read", CallID: ai.ShortcutToolCallPrefix + "read"}})),
	); err != nil {
		t.Fatal(err)
	}
	if _, err = c.RetryAiUserMessage(ctx, retry); err == nil {
		t.Fatal("缺少原始动作的旧流程被当作普通消息执行")
	}
}
