package biz

import (
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestAdminProtocolConfigIsolation 验证管理端协议独立下发、停用和刷新，移动端缓存互不影响。
func TestAdminProtocolConfigIsolation(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseConfig), new(models.BaseLanguage), new(models.BaseI18N))
	cache, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	err = client.DB.Create([]*models.BaseConfig{
		{ID: 1161, Site: 2, Name: "隐私政策", Type: 3, Key: "privacyProtocol", Value: "管理端隐私政策", Status: 1},
		{ID: 1162, Site: 2, Name: "服务条款", Type: 3, Key: "serviceProtocol", Value: "管理端服务条款", Status: 0},
		{ID: 1230, Site: 3, Name: "隐私政策", Type: 3, Key: "privacyProtocol", Value: "移动端隐私政策", Status: 1},
		{ID: 1240, Site: 3, Name: "服务条款", Type: 3, Key: "serviceProtocol", Value: "移动端服务条款", Status: 1},
	}).Error
	if err != nil {
		t.Fatal(err)
	}
	baseCase := &biz.BaseCase{Cache: cache}
	repository := data.NewBaseConfigRepository(database)
	public := basebiz.NewConfigCase(baseCase, repository, data.NewBaseI18NRepository(database), nil, data.NewBaseLanguageRepository(database), nil, nil)
	admin := &BaseConfigCase{BaseCase: baseCase, BaseConfigRepository: repository}
	check := func(site basev1.BaseConfigSite, privacy, service string) {
		t.Helper()
		var response *basev1.GetConfigResponse
		response, err = public.GetConfig(t.Context(), &basev1.GetConfigRequest{Site: site})
		if err != nil {
			t.Fatal(err)
		}
		values := make(map[string]string)
		for _, item := range response.GetConfigs() {
			values[item.GetKey()] = item.GetValue()
		}
		if values["privacyProtocol"] != privacy || values["serviceProtocol"] != service {
			t.Fatalf("站点 %d 协议串用：%v", site, values)
		}
	}
	for range 2 {
		check(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN, "管理端隐私政策", "")
		check(basev1.BaseConfigSite_BASE_CONFIG_SITE_APP, "移动端隐私政策", "移动端服务条款")
	}
	query := repository.Query(t.Context()).BaseConfig
	_, err = query.WithContext(t.Context()).Where(query.ID.Eq(1162)).UpdateSimple(query.Status.Value(1))
	if err != nil {
		t.Fatal(err)
	}
	err = admin.refreshBaseConfigSite(t.Context(), 2)
	if err != nil {
		t.Fatal(err)
	}
	check(basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN, "管理端隐私政策", "管理端服务条款")
	var snapshot string
	snapshot, err = cache.Get(_const.BaseConfigCacheKey(2))
	if err != nil {
		t.Fatal(err)
	}
	err = admin.refreshBaseConfigSite(t.Context(), 3)
	if err != nil {
		t.Fatal(err)
	}
	var refreshed string
	refreshed, err = cache.Get(_const.BaseConfigCacheKey(2))
	if err != nil || refreshed != snapshot {
		t.Fatalf("移动端刷新影响了管理端配置缓存：%v", err)
	}
}
