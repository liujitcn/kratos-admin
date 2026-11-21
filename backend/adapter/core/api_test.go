package core

import (
	"context"
	"encoding/json"
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

// TestSyncAiEventPermissions 验证启动同步幂等补齐三端入口，且保留原授权和租户角色隔离。
func TestSyncAiEventPermissions(t *testing.T) {
	client, provider, adapter := newAPISyncTestData(t)
	ctx := t.Context()
	menus := []*models.BaseMenu{
		{ID: 90990000, API: `["/base.v1.AiMessageService/SendAiMessage","/custom/read"]`, Meta: "{}", Status: 1},
		{ID: 90991000, API: `["/base.v1.AiMessageService/SendAiMessage"]`, Meta: "{}", Status: 1},
		{ID: 99101000, API: `["/base.v1.AiMessageService/SendAiMessage"]`, Meta: "{}", Status: 1},
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
	var actual []*models.BaseMenu
	actual, err = provider.Query(ctx).BaseMenu.WithContext(ctx).Order(provider.Query(ctx).BaseMenu.ID).Find()
	if err != nil {
		t.Fatal(err)
	}
	for _, menu := range actual {
		var operations []string
		if err = json.Unmarshal([]byte(menu.API), &operations); err != nil {
			t.Fatal(err)
		}
		want := 0
		if menu.ID != 777 {
			want = 1
		}
		count := 0
		for _, operation := range operations {
			if operation == basev1.OperationAiMessageServiceListAiMessageEvent {
				count++
			}
		}
		if count != want || !slices.Contains(operations, basev1.OperationAiMessageServiceSendAiMessage) {
			t.Fatalf("菜单 %d 权限错误: %s", menu.ID, menu.API)
		}
		if menu.ID == 90990000 && !slices.Contains(operations, "/custom/read") {
			t.Fatal("覆盖了原有自定义授权")
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
	if !slices.Equal(granted, []string{"tenant-1:bi", "tenant-1:chat", "tenant-2:chat"}) {
		t.Fatalf("事件权限越权或缺失: %v", granted)
	}
}

// TestSyncBiQueryRefinementPermissions 验证只为具有原有问数能力的 BI 入口幂等补齐调整权限。
func TestSyncBiQueryRefinementPermissions(t *testing.T) {
	client, provider, adapter := newAPISyncTestData(t)
	ctx := t.Context()
	menus := []*models.BaseMenu{
		{ID: 90991000, API: `["/base.v1.AiMessageService/SendAiMessage","/bi.admin.v1.BiQueryRecordService/GetBiQuery","/custom/read"]`, Meta: "{}", Status: 1},
		{ID: 91020000, API: `["/bi.admin.v1.BiQueryService/AskBiQuery","/bi.admin.v1.BiQueryRecordService/GetBiQuery"]`, Meta: "{}", Status: 1},
		{ID: 777, API: `["/bi.admin.v1.BiQueryService/AskBiQuery","/bi.admin.v1.BiQueryRecordService/GetBiQuery"]`, Meta: "{}", Status: 1},
	}
	err := client.DB.Create(menus).Error
	if err != nil {
		t.Fatal(err)
	}
	items := []*coredata.APIRecord{{Operation: adminv1.OperationBiQueryServiceRefineBiQuery, Method: "POST"}}
	for range 2 {
		if err = adapter.ReplaceAll(ctx, items); err != nil {
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
		var want, actual []string
		if err = json.Unmarshal([]byte(menu.API), &want); err != nil {
			t.Fatal(err)
		}
		if menu.ID != 777 {
			want = append(want, adminv1.OperationBiQueryServiceRefineBiQuery)
		}
		if err = json.Unmarshal([]byte(stored.API), &actual); err != nil || !slices.Equal(actual, want) {
			t.Fatalf("菜单 %d 权限错误: %s; %v", menu.ID, stored.API, err)
		}
	}
	// 已撤销原入口的菜单不会因启动同步恢复问数能力。
	const readOnly = `["/bi.admin.v1.BiQueryRecordService/GetBiQuery"]`
	if _, err = query.WithContext(ctx).Where(query.ID.Eq(91020000)).UpdateSimple(query.API.Value(readOnly)); err != nil {
		t.Fatal(err)
	}
	if err = adapter.ReplaceAll(ctx, items); err != nil {
		t.Fatal(err)
	}
	var stored *models.BaseMenu
	stored, err = query.WithContext(ctx).Where(query.ID.Eq(91020000)).First()
	if err != nil || stored.API != readOnly {
		t.Fatalf("已撤销问数权限的菜单发生变化: %v, %v", stored, err)
	}
}

// TestSyncAiEventPermissionBoundaries 验证不恢复已撤销或禁用的入口，错误数据使整个资源同步回滚。
func TestSyncAiEventPermissionBoundaries(t *testing.T) {
	client, provider, adapter := newAPISyncTestData(t)
	ctx := t.Context()
	menus := []*models.BaseMenu{
		{ID: 90990000, API: "[]", Meta: "{}", Status: 1},
		{ID: 90991000, API: `["/base.v1.AiMessageService/SendAiMessage"]`, Meta: "{}", Status: 2},
		{ID: 99101000, API: `["/base.v1.AiMessageService/SendAiMessage"]`, Meta: "{}", Status: 1, DeletedAt: 1},
	}
	err := client.DB.Create(menus).Error
	if err != nil {
		t.Fatal(err)
	}
	items := []*coredata.APIRecord{{Operation: basev1.OperationAiMessageServiceListAiMessageEvent, Method: "GET"}}
	if err = adapter.ReplaceAll(ctx, items); err != nil {
		t.Fatal(err)
	}
	var actual []*models.BaseMenu
	queryMenu := provider.Query(ctx).BaseMenu
	actual, err = queryMenu.WithContext(ctx).Unscoped().Order(queryMenu.ID).Find()
	if err != nil {
		t.Fatal(err)
	}
	for index, menu := range actual {
		if menu.API != menus[index].API || menu.Status != menus[index].Status || menu.DeletedAt != menus[index].DeletedAt {
			t.Fatal("已撤销、禁用或删除的入口被重新授权")
		}
	}
	menu := provider.Query(ctx).BaseMenu
	_, err = menu.WithContext(ctx).Where(menu.ID.Eq(90990000)).UpdateSimple(menu.API.Value("invalid-json"))
	if err != nil {
		t.Fatal(err)
	}
	err = provider.Transaction(ctx, func(txCtx context.Context) error {
		return adapter.ReplaceAll(txCtx, append(items, &coredata.APIRecord{Operation: "/must/rollback", Method: "GET"}))
	})
	if err == nil {
		t.Fatal("无效菜单 JSON 被静默覆盖")
	}
	query := provider.Query(ctx).BaseAPI
	var count int64
	count, err = query.WithContext(ctx).Where(query.Operation.Eq("/must/rollback")).Count()
	if err != nil || count != 0 {
		t.Fatalf("资源同步失败没有回滚: %d, %v", count, err)
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
