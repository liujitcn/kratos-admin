package biz

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestValidateFormConfig 验证统一配置接口拒绝非法表单和绕过表单类型的请求。
func TestValidateFormConfig(t *testing.T) {
	valid := models.BaseConfig{Site: 1, Type: int32(adminv1.BaseConfigType_BASE_CONFIG_TYPE_FORM), Key: "baseLogFallback", Value: `{"file_path":"./logs/custom"}`, Status: 1}
	cases := []struct {
		name   string
		change func(*models.BaseConfig)
	}{
		{"未知表单", func(value *models.BaseConfig) { value.Key = "unknown" }},
		{"非系统位置", func(value *models.BaseConfig) { value.Site = 2 }},
		{"停用表单", func(value *models.BaseConfig) { value.Status = 2 }},
		{"非法 JSON", func(value *models.BaseConfig) { value.Value = "{" }},
		{"必填字段为空", func(value *models.BaseConfig) { value.Value = `{"file_path":""}` }},
		{"普通类型冒用表单编码", func(value *models.BaseConfig) { value.Type = 1 }},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			entity := valid
			item.change(&entity)
			if err := validateFormConfig(&entity, nil); err == nil {
				t.Fatal("非法配置未被拒绝")
			}
		})
	}
	var err error
	err = validateFormConfig(&valid, nil)
	if err != nil {
		t.Fatalf("有效表单被拒绝: %v", err)
	}
	ordinary := models.BaseConfig{Site: 2, Type: 1, Key: "sysName", Value: "系统名称", Status: 1}
	err = validateFormConfig(&ordinary, nil)
	if err != nil {
		t.Fatalf("普通配置被拒绝: %v", err)
	}
}

// TestValidateFormConfigIdentity 验证编辑表单时不能改变已有运行配置的身份。
func TestValidateFormConfigIdentity(t *testing.T) {
	previous := models.BaseConfig{Site: 1, Type: 6, Key: "baseLogFallback", Value: `{"file_path":"./logs/original"}`, Status: 1}
	var err error
	for _, change := range []func(*models.BaseConfig){
		func(value *models.BaseConfig) { value.Site = 2 },
		func(value *models.BaseConfig) { value.Type = 1 },
		func(value *models.BaseConfig) { value.Key = "other" },
	} {
		entity := previous
		change(&entity)
		err = validateFormConfig(&entity, &previous)
		if err == nil {
			t.Fatal("已有表单身份被允许修改")
		}
	}
	entity := previous
	entity.Value = `{"file_path":"./logs/updated"}`
	err = validateFormConfig(&entity, &previous)
	if err != nil {
		t.Fatalf("表单值更新被拒绝: %v", err)
	}
}
