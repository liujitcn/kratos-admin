package codegen

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// backendPackageName 复用目标目录的包声明，目录尚未创建时采用对应层的默认包名。
func backendPackageName(directory, fallback string) string {
	content, err := os.ReadFile(filepath.Join(repoRoot(), directory, "init.go"))
	if err != nil {
		return fallback
	}
	var file *ast.File
	file, err = parser.ParseFile(token.NewFileSet(), "init.go", content, parser.PackageClauseOnly)
	if err != nil {
		return fallback
	}
	return file.Name.Name
}

// backendModulePath 读取宿主模块路径，缺失时由生成目标校验拒绝生成。
func backendModulePath() string {
	content, err := os.ReadFile(filepath.Join(repoRoot(), "backend", "go.mod"))
	if err != nil {
		return ""
	}
	return modfile.ModulePath(content)
}

// frontendPackageNameForBusinessModule 从目标业务模块的清单读取真实包名。
func frontendPackageNameForBusinessModule(module string) string {
	content, err := os.ReadFile(filepath.Join(repoRoot(), "frontend", "admin", "packages", "modules", module, "package.json"))
	if err != nil {
		return ""
	}
	var manifest struct {
		Name string `json:"name"` // Name 前端业务模块的 npm 包名。
	}
	err = json.Unmarshal(content, &manifest)
	if err != nil {
		return ""
	}
	return manifest.Name
}
