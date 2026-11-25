package biz

import (
	"strconv"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// TestAiShortcutTenantRevisionAndVisibility 验证建表、入口持久化、启停、过期修订和按租户展示。
func TestAiShortcutTenantRevisionAndVisibility(t *testing.T) {
	_, provider := newSecurityData(t, new(models.AiShortcut))
	repo := data.NewAiShortcutRepository(provider)
	baseCase := &biz.BaseCase{Context: bootstrap.NewContext(t.Context(), nil)}
	toolCase := basebiz.NewAiToolCase(baseCase, &ai.Runtime{}, repo, nil)
	c := NewAiShortcutCase(baseCase, repo, data.NewTransaction(provider), toolCase)
	actor := &authdata.UserTokenPayload{TenantId: 1, UserId: 3, TenantCode: "0000", RoleCode: "mgr"}
	ctx := engine.ContextWithAuthClaims(t.Context(), actor.MakeAuthClaims())
	other := &authdata.UserTokenPayload{TenantId: 2, UserId: 4, TenantCode: "other", RoleCode: "mgr"}
	otherCtx := engine.ContextWithAuthClaims(t.Context(), other.MakeAuthClaims())
	form := &adminv1.AiShortcutForm{
		ShortcutKey: "test_prompt", Title: "原始入口", Prompt: "列出我的待办",
		SourceType: adminv1.AiShortcutSourceType_AI_SHORTCUT_SOURCE_TYPE_CUSTOM,
		Terminal:   basev1.Terminal_TERMINAL_ADMIN, Status: commonv1.Status_STATUS_DISABLE,
	}
	err := c.CreateAiShortcut(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	listReq := &basev1.ListAiShortcutRequest{Terminal: basev1.Terminal_TERMINAL_ADMIN}
	var list *basev1.ListAiShortcutResponse
	list, err = toolCase.ListAiShortcut(ctx, listReq)
	if err != nil || len(list.GetShortcuts()) != 0 {
		t.Fatalf("停用入口可见: %v %v", list, err)
	}
	var page *adminv1.PageAiShortcutResponse
	page, err = c.PageAiShortcut(ctx, &adminv1.PageAiShortcutRequest{PageNum: 1, PageSize: 10})
	if err != nil || page.GetTotal() != 1 || page.GetShortcuts()[0].GetId() <= 0 {
		t.Fatalf("数据库未保存创建结果: %v %v", page, err)
	}
	entry := page.Shortcuts[0]
	if entry.GetStatus() != commonv1.Status_STATUS_DISABLE || entry.GetSourceType() != form.SourceType {
		t.Fatalf("入口 DTO 映射错误: %v", entry)
	}
	if _, err = c.GetAiShortcut(otherCtx, entry.GetId()); err == nil {
		t.Fatal("其他租户可以查看入口")
	}
	if err = c.SetAiShortcutStatus(ctx, &adminv1.SetAiShortcutStatusRequest{Id: entry.Id, Revision: entry.Revision, Status: commonv1.Status_STATUS_ENABLE}); err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateAiShortcut(ctx, entry); err == nil {
		t.Fatal("过期修订覆盖了启用状态")
	}
	entry, err = c.GetAiShortcut(ctx, entry.GetId())
	if err != nil {
		t.Fatal(err)
	}
	entry.Title = "更新后的入口"
	entry.Status = commonv1.Status_STATUS_DISABLE
	if err = c.UpdateAiShortcut(ctx, entry); err != nil {
		t.Fatal(err)
	}
	list, err = toolCase.ListAiShortcut(ctx, listReq)
	if err != nil || len(list.GetShortcuts()) != 1 || list.GetShortcuts()[0].GetTitle() != entry.GetTitle() {
		t.Fatalf("编辑未保存标题或改写了启用状态: %v %v", list, err)
	}
	list, err = toolCase.ListAiShortcut(otherCtx, listReq)
	if err != nil || len(list.GetShortcuts()) != 0 {
		t.Fatalf("对话入口泄露到其他租户: %v %v", list, err)
	}
	id := strconv.FormatInt(entry.Id, 10)
	if err = c.DeleteAiShortcut(ctx, id+",9000"); err == nil {
		t.Fatal("包含不存在记录的批量删除未被拒绝")
	}
	if _, err = c.GetAiShortcut(ctx, entry.Id); err != nil {
		t.Fatalf("批量删除发生部分写入: %v", err)
	}
	if err = c.DeleteAiShortcut(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = c.GetAiShortcut(ctx, entry.Id); err == nil {
		t.Fatal("已删除入口仍可查看")
	}
}
