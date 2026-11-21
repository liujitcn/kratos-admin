package biz

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	admindata "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestAPIConfigurationInvalidatesSnapshot 验证API工具开关提交后刷新缓存、递增插件修订，旧缓存不绕过执行检查。
func TestAPIConfigurationInvalidatesSnapshot(t *testing.T) {
	client, provider := newSecurityData(t, new(models.BaseAPI), new(models.AiPlugin), new(models.AiPluginInstallation))
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	const name = "system_admin_v1_base_dict_service_page_base_dict"
	for _, row := range []any{&models.BaseAPI{ID: 1, ToolName: name, AgentStatus: 1, ToolPrompts: "[]"}, &models.AiPlugin{ID: 1, Source: "API"}, &models.AiPluginInstallation{ID: 1, PluginID: 1, TenantID: 1, Revision: 1}} {
		if err = client.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := data.NewBaseAPIRepository(provider)
	if _, err = admindata.LoadBaseAPICache(t.Context(), repo, store); err != nil {
		t.Fatal(err)
	}
	identity := &authdata.UserTokenPayload{UserId: 1, TenantId: 1, TenantCode: "0000", RoleCode: "super"}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	baseCase := &biz.BaseCase{Context: bootstrap.NewContext(t.Context(), nil), Cache: store}
	c := &BaseAPICase{BaseCase: baseCase, BaseAPIRepository: repo, tx: data.NewTransaction(provider), aiPluginRepo: data.NewAiPluginRepository(provider), aiPluginInstallationRepo: data.NewAiPluginInstallationRepository(provider)}
	if err = c.SetBaseAPIAgentStatus(ctx, &adminv1.SetBaseApiAgentStatusRequest{Id: 1, AgentStatus: 0}); err != nil {
		t.Fatal(err)
	}
	var snapshot map[string][]*models.BaseAPI
	snapshot, err = admindata.LoadBaseAPICache(ctx, repo, store)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot[name][0].AgentStatus != 0 {
		t.Fatal("开关已提交但仍使用旧缓存")
	}
	query := provider.Query(ctx)
	var installation *models.AiPluginInstallation
	installation, err = query.AiPluginInstallation.WithContext(ctx).Where(query.AiPluginInstallation.ID.Eq(1)).First()
	if err != nil {
		t.Fatal(err)
	}
	if installation.Revision != 2 {
		t.Fatalf("插件修订=%d，期望2", installation.Revision)
	}
	// 模拟外部修改尚未触发失效：缓存允许工具，实际调用仍须读取数据库拒绝。
	if _, err = query.BaseAPI.WithContext(ctx).Where(query.BaseAPI.ID.Eq(1)).Update(query.BaseAPI.AgentStatus, 1); err != nil {
		t.Fatal(err)
	}
	if err = admindata.InvalidateBaseAPICache(store); err != nil {
		t.Fatal(err)
	}
	if _, err = admindata.LoadBaseAPICache(ctx, repo, store); err != nil {
		t.Fatal(err)
	}
	if _, err = query.BaseAPI.WithContext(ctx).Where(query.BaseAPI.ID.Eq(1)).Update(query.BaseAPI.AgentStatus, 0); err != nil {
		t.Fatal(err)
	}
	var gate *basebiz.McpCase
	gate, err = basebiz.NewMcpCase(baseCase, repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = gate.PrepareToolCall(ctx, "admin", name, "{}"); err == nil {
		t.Fatal("旧缓存绕过了最新工具禁用状态")
	}
	// 非平台管理员的更新失败不能改变快照或安装修订。
	identity.RoleCode = "user"
	ctx = engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	if err = c.SetBaseAPIAgentStatus(ctx, &adminv1.SetBaseApiAgentStatusRequest{Id: 1, AgentStatus: 1}); err == nil {
		t.Fatal("非平台管理员修改成功")
	}
}
