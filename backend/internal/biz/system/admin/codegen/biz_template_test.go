package codegen

import (
	"go/parser"
	"go/token"
	"strconv"
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
