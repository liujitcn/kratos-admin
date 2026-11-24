package core

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coredata "github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/resource/biz"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestSyncAPIPreservesMenuPermissions 验证 API 同步后的角色策略沿用菜单现有授权。
func TestSyncAPIPreservesMenuPermissions(t *testing.T) {
	client, provider, adapter := newAPISyncTestData(t)
	ctx := t.Context()
	menus := []*models.BaseMenu{
		{ID: 90990000, API: `["/base.v1.AiMessageService/SendAiMessage","/base.v1.AiMessageService/ListAiMessageEvent","/custom/read"]`, Meta: "{}", Status: 1},
		{ID: 90991000, API: `["/base.v1.AiMessageService/SendAiMessage","/bi.admin.v1.BiQueryRecordService/GetBiQuery"]`, Meta: "{}", Status: 1},
		{ID: 99101000, API: `["/base.v1.AiMessageService/SendAiMessage","/base.v1.AiMessageService/ListAiMessageEvent"]`, Meta: "{}", Status: 1},
		{ID: 777, API: `["/base.v1.AiMessageService/SendAiMessage"]`, Meta: "{}", Status: 1},
	}
	err := client.DB.Create(menus).Error
	if err != nil {
		t.Fatal(err)
	}
	roles := []*models.BaseRole{
		{ID: 1, TenantID: 1, Code: "chat", Menus: "[90990000]", Status: 1},
		{ID: 2, TenantID: 1, Code: "bi", Menus: "[90991000]", Status: 1},
		{ID: 3, TenantID: 2, Code: "chat", Menus: "[99101000]", Status: 1},
		{ID: 4, TenantID: 2, Code: "denied", Menus: "[]", Status: 1},
		{ID: 5, TenantID: 1, Code: "custom", Menus: "[777]", Status: 1},
	}
	if err = client.DB.Create(roles).Error; err != nil {
		t.Fatal(err)
	}
	if err = client.DB.Create([]*models.BaseTenant{{ID: 1, Code: "tenant-1"}, {ID: 2, Code: "tenant-2"}}).Error; err != nil {
		t.Fatal(err)
	}
	var permissionStore *PermissionStoreAdapter
	permissionStore, err = NewPermissionStoreAdapter(map[string]*kitgorm.Client{kitgorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	var policyCase *biz.CasbinRuleCase
	policyCase, err = biz.NewCasbinRuleCase(permissionStore, biz.NewBaseAPICase(adapter), nil)
	if err != nil {
		t.Fatal(err)
	}
	items := []*coredata.APIRecord{
		{Operation: basev1.OperationAiMessageServiceSendAiMessage, Method: "POST"},
		{Operation: basev1.OperationAiMessageServiceListAiMessageEvent, Method: "GET"},
		{Operation: adminv1.OperationBiQueryRecordServiceGetBiQuery, Method: "GET"},
		{Operation: adminv1.OperationBiQueryServiceAskBiQuery, Method: "POST"},
		{Operation: adminv1.OperationBiQueryServiceRefineBiQuery, Method: "POST"},
		{Operation: "/custom/read", Method: "GET"},
	}
	for range 2 {
		err = provider.Transaction(ctx, func(txCtx context.Context) error {
			err = adapter.ReplaceAll(txCtx, items)
			if err != nil {
				return err
			}
			return policyCase.RebuildPolicyData(txCtx)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	query := provider.Query(ctx).BaseMenu
	for _, menu := range menus {
		var stored *models.BaseMenu
		stored, err = query.WithContext(ctx).Where(query.ID.Eq(menu.ID)).First()
		if err != nil {
			t.Fatal(err)
		}
		if stored.API != menu.API {
			t.Fatalf("API 同步改写了菜单 %d 的权限: %s", menu.ID, stored.API)
		}
	}
	var storedRoles []*models.BaseRole
	storedRoles, err = provider.Query(ctx).BaseRole.WithContext(ctx).Order(provider.Query(ctx).BaseRole.ID).Find()
	if err != nil || len(storedRoles) != len(roles) {
		t.Fatalf("角色查询失败或数量改变: %v", err)
	}
	for index, role := range storedRoles {
		if role.Menus != roles[index].Menus || role.TenantID != roles[index].TenantID || role.Code != roles[index].Code || role.Status != roles[index].Status {
			t.Fatal("角色原授权被改写")
		}
	}
	var policies []coredata.PolicyRecord
	policies, err = permissionStore.ListPolicies(ctx)
	if err != nil {
		t.Fatal(err)
	}
	granted := []string{}
	for _, policy := range policies {
		if policy.V2 == basev1.OperationAiMessageServiceListAiMessageEvent {
			if policy.V3 != "GET" {
				t.Fatal("事件权限不是只读 GET")
			}
			granted = append(granted, policy.V0+":"+policy.V1)
		}
	}
	slices.Sort(granted)
	if !slices.Equal(granted, []string{"tenant-1:chat", "tenant-2:chat"}) {
		t.Fatalf("事件权限越权或缺失: %v", granted)
	}
}

// newAPISyncTestData 创建独立权限仓储，避免测试触碰真实业务库。
func newAPISyncTestData(t *testing.T) (*kitgorm.Client, *data.Data, *APIStoreAdapter) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	registered := []any{new(models.BaseAPI), new(models.BaseMenu), new(models.BaseRole), new(models.BaseTenant), new(models.CasbinRule)}
	for _, model := range registered {
		statement := &gorm.Statement{DB: db}
		if err = statement.Parse(model); err != nil {
			t.Fatal(err)
		}
		statement.Schema.FieldsByName["ID"].DataType = schema.Int
	}
	if err = db.AutoMigrate(registered...); err != nil {
		t.Fatal(err)
	}
	client := &kitgorm.Client{DB: db}
	clients := map[string]*kitgorm.Client{kitgorm.DefaultClientName: client}
	var provider *data.Data
	provider, err = data.NewData(clients)
	if err != nil {
		t.Fatal(err)
	}
	var adapter *APIStoreAdapter
	adapter, err = NewAPIStoreAdapter(clients, nil)
	if err != nil {
		t.Fatal(err)
	}
	return client, provider, adapter
}
