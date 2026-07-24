package codegen

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

var sourceTemplates = template.Must(
	template.New("codegen").
		Delims("[[", "]]").
		Option("missingkey=error").
		ParseFS(templateFiles, "templates/*.tmpl"),
)

// backendBizTemplateData 描述后端业务文件模板输入。
type backendBizTemplateData struct {
	Entity       string
	EntityVar    string
	BusinessName string
	APIImport    string
	JSONImport   string
	ErrorImport  string
	Repository   string
	FormType     string
	ModelType    string
	DTOType      string
	CommonImport string
	Methods      string
}

// backendServiceTemplateData 描述后端服务文件模板输入。
type backendServiceTemplateData struct {
	Entity       string
	EntityVar    string
	BusinessName string
	APIAlias     string
	APIImport    string
	BizImport    string
	CommonImport string
	Methods      string
}

// protoTemplateData 描述 Proto 文件模板输入。
type protoTemplateData struct {
	Entity       string
	TableComment string
	PackageName  string
	RPCs         string
	Messages     string
}

// frontendAPITemplateData 描述前端 API 文件模板输入。
type frontendAPITemplateData struct {
	Entity       string
	BusinessName string
	TypeImports  string
	RPCImport    string
	EmptyImport  string
	OptionImport string
	URLConst     string
	ResourcePath string
	Methods      string
}

// frontendPageTemplateData 描述 Vue 页面模板输入。
type frontendPageTemplateData struct {
	Entity          string
	BusinessName    string
	HasTenantOption bool
	Script          string
}

// renderTemplate 使用内嵌模板渲染完整文件骨架。
func renderTemplate(name string, data any) string {
	var content bytes.Buffer
	if err := sourceTemplates.ExecuteTemplate(&content, name, data); err != nil {
		panic(err)
	}
	return content.String()
}
