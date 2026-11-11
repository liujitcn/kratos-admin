package biz

import (
	"encoding/json"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestNormalizeCodeGenTableJSONUsesObject 验证代码生成表的空左树配置使用合法 JSON 对象。
func TestNormalizeCodeGenTableJSONUsesObject(t *testing.T) {
	for _, value := range []string{"", "null"} {
		item := &models.CodeGenTable{LeftTreeConfig: value, I18NConfig: value}
		normalizeCodeGenTableJSON(item)
		if item.LeftTreeConfig != "{}" || item.I18NConfig != "{}" {
			t.Fatalf("输入 %q 归一化结果 = left_tree_config:%q i18n_config:%q，期望 {}", value, item.LeftTreeConfig, item.I18NConfig)
		}
		if !json.Valid([]byte(item.LeftTreeConfig)) || !json.Valid([]byte(item.I18NConfig)) {
			t.Fatalf("输入 %q 归一化后不是合法 JSON", value)
		}
	}
}

// TestBaseMenuFormMapperUsesValidJSON 验证空菜单表单映射不会生成非法 JSON。
func TestBaseMenuFormMapperUsesValidJSON(t *testing.T) {
	menuCase := NewBaseMenuCase(nil, nil, nil, nil, nil, nil)
	item := menuCase.formMapper.ToEntity(&adminv1.BaseMenuForm{})
	normalizeBaseMenuJSON(item)
	if item.Meta != "{}" || item.API != "[]" {
		t.Fatalf("空表单归一化结果 = meta:%q api:%q，期望 {} 和 []", item.Meta, item.API)
	}
	for _, value := range []string{"", "null"} {
		item.Meta = value
		item.API = value
		normalizeBaseMenuJSON(item)
		if item.Meta != "{}" || item.API != "[]" {
			t.Fatalf("输入 %q 归一化结果 = meta:%q api:%q，期望 {} 和 []", value, item.Meta, item.API)
		}
		for field, value := range map[string]string{"meta": item.Meta, "api": item.API} {
			if !json.Valid([]byte(value)) {
				t.Fatalf("%s 不是合法 JSON: %q", field, value)
			}
		}
	}
}
