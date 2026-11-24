package biz

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestGetNextBaseTenantCode 验证租户编号按数字递增，计入软删除记录并检查编号上限。
func TestGetNextBaseTenantCode(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseTenant))
	c := &BaseTenantCase{BaseTenantRepository: data.NewBaseTenantRepository(database)}
	ctx := t.Context()
	code, err := c.getNextBaseTenantCode(ctx)
	if err != nil || code != "1000" {
		t.Fatalf("初始租户编号 = %q，期望 1000，错误 = %v", code, err)
	}

	tenants := []models.BaseTenant{
		{Code: "0000"},
		{Code: "1999"},
		{Code: "002000", DeletedAt: 1},
		{Code: "custom"},
		{Code: "+9999"},
	}
	if err = client.DB.Create(&tenants).Error; err != nil {
		t.Fatal(err)
	}
	code, err = c.getNextBaseTenantCode(ctx)
	if err != nil || code != "2001" {
		t.Fatalf("下一个租户编号 = %q，期望 2001，错误 = %v", code, err)
	}

	if err = client.DB.Create(&models.BaseTenant{Code: "9999"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err = c.getNextBaseTenantCode(ctx); errors.Code(err) != 409 {
		t.Fatalf("租户编号用完应返回冲突错误，实际 = %v", err)
	}
}
