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

// TestGetI18nCustomTenantIsolation 验证自定义翻译只返回当前登录租户的数据。
func TestGetI18nCustomTenantIsolation(t *testing.T) {
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
	if err = db.AutoMigrate(&models.BaseI18NCustom{}); err != nil {
		t.Fatal(err)
	}
	items := []*models.BaseI18NCustom{
		{TenantID: 1, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "租户一项目", Status: 1},
		{TenantID: 2, Site: int32(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN), Key: "common.field.project", Locale: "zh-CN", Value: "租户二项目", Status: 1},
	}
	for _, item := range items {
		if err = db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	configCase := &ConfigCase{BaseCase: &biz.BaseCase{}, i18nCustomRepo: data.NewBaseI18NCustomRepository(store)}
	identity := &authdata.UserTokenPayload{TenantId: 2, TenantCode: "tenant-two", UserId: 20}
	ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
	var response *basev1.GetI18nCustomResponse
	response, err = configCase.GetI18nCustom(ctx, &basev1.GetI18nCustomRequest{Site: basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetItems()) != 1 || response.GetItems()[0].GetValue() != "租户二项目" {
		t.Fatalf("当前租户翻译 = %+v，期望只返回租户二翻译", response.GetItems())
	}
}
