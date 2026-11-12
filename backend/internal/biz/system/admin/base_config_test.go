package biz

import (
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestRefreshFormBaseConfigUsesDescriptionName 验证表单配置名称使用文字描述，并修正历史键名。
func TestRefreshFormBaseConfigUsesDescriptionName(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseConfig))
	cache, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	value, err := runtimeconfig.DefaultJSON(runtimeconfig.SecurityScriptKey)
	if err != nil {
		t.Fatal(err)
	}
	err = client.DB.Create(&models.BaseConfig{
		Site:   _const.BASE_CONFIG_SITE_SYSTEM,
		Name:   runtimeconfig.SecurityScriptKey,
		Type:   int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM),
		Key:    runtimeconfig.SecurityScriptKey,
		Value:  value,
		Status: 1,
	}).Error
	if err != nil {
		t.Fatal(err)
	}

	configCase := &BaseConfigCase{
		BaseCase:             &biz.BaseCase{Cache: cache},
		BaseConfigRepository: data.NewBaseConfigRepository(database),
	}
	err = configCase.refreshFormBaseConfig(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	query := database.Query(t.Context()).BaseConfig
	item, err := query.WithContext(t.Context()).Where(query.Key.Eq(runtimeconfig.SecurityScriptKey)).First()
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "脚本沙箱配置" {
		t.Fatalf("表单配置名称 = %q，期望文字描述", item.Name)
	}
}

// TestValidateBaseConfigValue 验证新增 JSON、动态列表和多文本配置的持久化结构。
func TestValidateBaseConfigValue(t *testing.T) {
	for _, test := range []struct {
		name  string
		typ   adminv1.BaseConfigType
		value string
		valid bool
	}{
		{name: "json", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_JSON, value: `{"hosts":[]}`, valid: true},
		{name: "dynamic list", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_DYNAMIC_LIST, value: `["127.0.0.1/32"]`, valid: true},
		{name: "multi text", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_MULTI_TEXT, value: `[{"key":"name","value":"value"}]`, valid: true},
		{name: "invalid json", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_JSON, value: `{`, valid: false},
		{name: "invalid list", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_DYNAMIC_LIST, value: `{"value":"x"}`, valid: false},
		{name: "empty multi text key", typ: adminv1.BaseConfigType_BASE_CONFIG_TYPE_MULTI_TEXT, value: `[{"key":"","value":"value"}]`, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateBaseConfigValue(&models.BaseConfig{Type: int32(test.typ), Value: test.value})
			if (err == nil) != test.valid {
				t.Fatalf("校验结果 = %v, valid = %v", err, test.valid)
			}
		})
	}
}

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
