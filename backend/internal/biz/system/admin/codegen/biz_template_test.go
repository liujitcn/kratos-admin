package codegen

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TestBizTemplateBaseCaseImport 验证两类业务模板依赖 Core 基类，不反向依赖业务聚合包。
func TestBizTemplateBaseCaseImport(t *testing.T) {
	for _, name := range []string{"backend_biz.tmpl", "backend_external_biz.tmpl"} {
		content := renderTemplate(name, backendBizTemplateData{Entity: "TenantProject", EntityVar: "tenantProject", Repository: "TenantProjectRepository", FormType: "Form", ModelType: "Model", DTOType: "DTO"})
		file, err := parser.ParseFile(token.NewFileSet(), name, content, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, spec := range file.Imports {
			var path string
			path, err = strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if path == "github.com/liujitcn/kratos-admin/backend/internal/biz" {
				t.Errorf("%s 导入业务聚合包，导致循环依赖", name)
			}
			if path == "github.com/liujitcn/kratos-core/biz" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s 未使用 Core BaseCase", name)
		}
	}
}

// TestRenderDeleteBizMethodLocalizesChildrenConflict 验证生成的删除校验携带稳定翻译键和资源名称参数。
func TestRenderDeleteBizMethodLocalizesChildrenConflict(t *testing.T) {
	table := &Table{EntityName: "Category", BusinessName: "分类", TableName_: "base_category", ParentColumn: "parent_id"}
	content := (&renderer{}).renderDeleteBizMethod(table, nil, true)
	for _, expected := range []string{
		`errorsx.WithMessageKey(`,
		`errorsx.HasChildrenConflict("删除{{.Parent}}失败，下面有{{.Child}}", "base_category", "base_category")`,
		`"system.code.gen.error.has_children"`,
		`map[string]string{"Parent": "分类", "Child": "分类"}`,
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("生成的删除方法缺少 %q", expected)
		}
	}
}
