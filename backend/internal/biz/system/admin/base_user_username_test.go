package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
)

// TestReservedUsernamesReadFromConfig 验证保留用户名校验随 base_config 的当前值变化。
func TestReservedUsernamesReadFromConfig(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseConfig))
	err := client.DB.Create(&models.BaseConfig{
		ID:     1010,
		Site:   _const.BASE_CONFIG_SITE_SYSTEM,
		Name:   "系统保留用户名",
		Type:   6,
		Key:    runtimeconfig.SecurityIdentityKey,
		Value:  `{"reserved_usernames":["admin","root"],"account_renames":[]}`,
		Status: 1,
	}).Error
	if err != nil {
		t.Fatal(err)
	}

	configCase := &BaseConfigCase{
		BaseCase:             &biz.BaseCase{},
		BaseConfigRepository: data.NewBaseConfigRepository(database),
	}
	userCase := &BaseUserCase{BaseCase: &biz.BaseCase{}, baseConfigCase: configCase}
	if userCase.validateUserName(t.Context(), "admin") == nil {
		t.Fatal("数据库配置的保留用户名未被拒绝")
	}
	if err = userCase.validateUserName(t.Context(), "guest"); err != nil {
		t.Fatalf("未配置的用户名被拒绝: %v", err)
	}

	query := database.Query(t.Context()).BaseConfig
	_, err = query.WithContext(t.Context()).Where(query.ID.Eq(1010)).UpdateSimple(query.Value.Value(`{"reserved_usernames":["guest"],"account_renames":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = userCase.validateUserName(t.Context(), "admin"); err != nil {
		t.Fatalf("数据库配置更新后仍拒绝正常用户名: %v", err)
	}
	if userCase.validateUserName(t.Context(), "guest") == nil {
		t.Fatal("数据库配置更新后的保留用户名未被拒绝")
	}
}
