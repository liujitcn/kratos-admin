package biz

import (
	"context"
	"database/sql"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newI18nCustomTestConfigCase 创建带内存库的配置业务实例，并写入默认租户与两个测试租户。
func newI18nCustomTestConfigCase(t *testing.T, customs []*models.BaseI18NCustom) *ConfigCase {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseI18NCustom{}, &models.BaseTenant{}); err != nil {
		t.Fatal(err)
	}
	tenants := []*models.BaseTenant{
		{ID: 1, Code: kitgorm.DefaultTenantCode, Name: "默认租户", Status: 1},
		{ID: 2, Code: "tenant-two", Name: "租户二", Status: 1},
	}
	for _, tenant := range tenants {
		if err = db.Create(tenant).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range customs {
		if err = db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	return &ConfigCase{
		BaseCase:       &biz.BaseCase{},
		i18nCustomRepo: data.NewBaseI18NCustomRepository(store),
		tenantRepo:     data.NewBaseTenantRepository(store),
	}
}

// requestAdminI18nCustom 以指定租户身份请求管理端自定义翻译。
func requestAdminI18nCustom(t *testing.T, configCase *ConfigCase, tenantID int64, tenantCode string) []*basev1.I18nCustomItem {
	t.Helper()
	identity := &authdata.UserTokenPayload{TenantId: tenantID, TenantCode: tenantCode, UserId: 20}
	ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
	var response *basev1.GetI18nCustomResponse
	var err error
	response, err = configCase.GetI18nCustom(ctx, &basev1.GetI18nCustomRequest{Site: basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN})
	if err != nil {
		t.Fatal(err)
	}
	return response.GetItems()
}

// TestGetI18nCustomTenantIsolation 验证自定义翻译只返回当前登录租户的数据，默认租户不混入其他租户翻译。
func TestGetI18nCustomTenantIsolation(t *testing.T) {
	configCase := newI18nCustomTestConfigCase(t, []*models.BaseI18NCustom{
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "默认租户项目", Status: 1},
		{TenantID: 2, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "租户二项目", Status: 1},
	})
	items := requestAdminI18nCustom(t, configCase, 1, kitgorm.DefaultTenantCode)
	if len(items) != 1 || items[0].GetValue() != "默认租户项目" {
		t.Fatalf("默认租户翻译 = %+v，期望只返回默认租户翻译", items)
	}
	items = requestAdminI18nCustom(t, configCase, 2, "tenant-two")
	if len(items) != 1 || items[0].GetValue() != "租户二项目" {
		t.Fatalf("当前租户翻译 = %+v，期望只返回租户二翻译", items)
	}
}

// TestGetI18nCustomFallbackToDefaultTenant 验证租户未自定义的键回退到默认租户文案，已自定义的键保持覆盖。
func TestGetI18nCustomFallbackToDefaultTenant(t *testing.T) {
	configCase := newI18nCustomTestConfigCase(t, []*models.BaseI18NCustom{
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "默认租户项目", Status: 1},
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "system.base.tenant_project.all", Locale: "zh-CN", Value: "全部项目", Status: 1},
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "system.base.tenant_project.all", Locale: "en-US", Value: "All projects", Status: 1},
		// 默认租户的停用覆盖项不参与回退。
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.tenant", Locale: "zh-CN", Value: "停用文案", Status: 2},
		// 其他站点和其他租户的数据不参与回退。
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_APP), Key: "common.field.status", Locale: "zh-CN", Value: "APP 站点状态", Status: 1},
		{TenantID: 2, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "租户二项目", Status: 1},
	})
	values := map[string]string{}
	for _, item := range requestAdminI18nCustom(t, configCase, 2, "tenant-two") {
		values[item.GetLocale()+"\x00"+item.GetKey()] = item.GetValue()
	}
	if values["zh-CN\x00common.field.project"] != "租户二项目" {
		t.Fatalf("租户已自定义键 = %v，期望保持租户二自己的文案", values)
	}
	if values["zh-CN\x00system.base.tenant_project.all"] != "全部项目" || values["en-US\x00system.base.tenant_project.all"] != "All projects" {
		t.Fatalf("租户未自定义键 = %v，期望回退到默认租户文案", values)
	}
	if _, ok := values["zh-CN\x00common.field.tenant"]; ok {
		t.Fatalf("停用覆盖项 = %v，期望不参与回退", values)
	}
	if _, ok := values["zh-CN\x00common.field.status"]; ok {
		t.Fatalf("其他站点覆盖项 = %v，期望不参与回退", values)
	}
}
