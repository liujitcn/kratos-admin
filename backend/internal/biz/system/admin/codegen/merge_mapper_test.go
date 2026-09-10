package codegen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestMergeBizMapperFields 验证已有 Case 补齐标准字段和构造初始化，重复生成不重复追加。
func TestMergeBizMapperFields(t *testing.T) {
	r := &renderer{}
	table := &Table{EntityName: "CasbinRule", BusinessName: "casbin_rule", BusinessModule: "system", APIPath: "system/admin/v1"}
	for _, constructor := range []string{
		"func NewCasbinRuleCase() *CasbinRuleCase { return &CasbinRuleCase{existing: true} }",
		"func NewCasbinRuleCase() (*CasbinRuleCase, error) { return &CasbinRuleCase{existing: true}, nil }",
	} {
		original := "package biz\ntype CasbinRuleCase struct { existing bool }\n" + constructor
		methods := []*Proto{{GenerateWhenMissing: 1, MethodName: "PageCasbinRule", ProtoFilePath: r.defaultProtoPath(table)}}
		result := r.appendMainBizMethods(original, table, nil, methods)
		for _, marker := range []string{"formMapper *mapper.CopierMapper", "mapper     *mapper.CopierMapper", "formMapper:", "mapper:", "c.mapper.ToDTO", "existing: true"} {
			if !strings.Contains(result, marker) {
				t.Errorf("缺少 %s", marker)
			}
		}
		if _, err := parser.ParseFile(token.NewFileSet(), "merged.go", result, parser.AllErrors); err != nil {
			t.Fatal(err)
		}
		repeated := r.appendMainBizMethods(result, table, nil, methods)
		if strings.Count(repeated, "mapper.NewCopierMapper") != 2 {
			t.Fatal("重复生成添加了重复初始化")
		}
	}
}
