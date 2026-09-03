package module

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/docs"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-admin/backend/internal/openapi"
	"github.com/liujitcn/kratos-admin/backend/migration"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

type resource struct {
	projectKey  string
	projectName string
	models      module.Models
	docs        fs.FS
	i18n        fs.FS
	openAPI     fs.FS
	migrations  module.Migrations
}

// localizedDocsFS 在项目文档默认目录可用时过滤过期的本地化目录，避免翻译资源阻断启动。
type localizedDocsFS struct {
	source fs.FS
}

type projectDocsCatalog struct {
	Documents   []projectDocsDocument  `json:"documents"`
	Directories []projectDocsDirectory `json:"directories"`
}

type projectDocsDirectory struct {
	Documents   []projectDocsDocument  `json:"documents"`
	Directories []projectDocsDirectory `json:"directories"`
}

type projectDocsDocument struct {
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

// ProjectKey 返回 Admin 项目的稳定标识。
func (r *resource) ProjectKey() string {
	return r.projectKey
}

// ProjectName 返回 Admin 项目的展示名称。
func (r *resource) ProjectName() string {
	return r.projectName
}

// Models 返回 Admin 数据库自动迁移所需的模型。
func (r *resource) Models() module.Models {
	return r.models
}

// Docs 返回 Admin 项目文档文件系统。
func (r *resource) Docs() fs.FS {
	return r.docs
}

// I18n 返回 Admin 项目语言文件系统。
func (r *resource) I18n() fs.FS {
	return r.i18n
}

// OpenAPI 返回 Admin OpenAPI 文件系统。
func (r *resource) OpenAPI() fs.FS {
	return r.openAPI
}

// Migrations 返回 Admin 数据库迁移资源。
func (r *resource) Migrations() module.Migrations {
	return r.migrations
}

var _ module.Resource = (*resource)(nil)

// NewModuleResources 创建 Admin 在业务对象构建前提供给 Core 的静态资源。
func NewModuleResources() module.Resources {
	docsFS, err := fs.Sub(docs.DocsFS, "assets")
	if err != nil {
		panic(fmt.Errorf("读取内嵌项目文档资源: %w", err))
	}
	models := data.Models()
	return module.Resources{
		&resource{
			projectKey:  fmt.Sprintf("%s-%s", _const.Project, _const.AppID),
			projectName: _const.Name,
			models:      module.Models{gorm.DefaultClientName: models},
			openAPI:     openapi.Assets(),
			docs:        &localizedDocsFS{source: docsFS},
			migrations: module.Migrations{
				{Name: migration.ModuleName, FS: migration.Assets(), Path: "."},
			},
			i18n: i18n.Assets(),
		},
	}
}

// Open 打开文档资源文件，并保留 fs.FS 的默认行为。
func (f *localizedDocsFS) Open(name string) (fs.File, error) {
	return f.source.Open(name)
}

// ReadDir 读取文档资源目录并过滤无法匹配默认更新时间的本地化目录。
func (f *localizedDocsFS) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	var err error
	entries, err = fs.ReadDir(f.source, name)
	if err != nil || name != "." {
		return entries, err
	}

	var defaultUpdatedAt map[string]string
	defaultUpdatedAt, err = readProjectDocsUpdatedAt(f.source, "docs.json")
	if err != nil {
		return entries, nil
	}
	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isLocalizedDocsCatalog(entry.Name()) {
			filtered = append(filtered, entry)
			continue
		}
		var localizedUpdatedAt map[string]string
		localizedUpdatedAt, err = readProjectDocsUpdatedAt(f.source, entry.Name())
		if err == nil && sameProjectDocsUpdatedAt(defaultUpdatedAt, localizedUpdatedAt) {
			filtered = append(filtered, entry)
			continue
		}
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "项目文档语言资源 %q 暂不可用，启动时回退默认语言: %v\n", entry.Name(), err)
			continue
		}
		_, _ = fmt.Fprintf(os.Stderr, "项目文档语言资源 %q 与默认目录不一致，启动时回退默认语言\n", entry.Name())
	}
	return filtered, nil
}

// isLocalizedDocsCatalog 判断文件名是否为带语言后缀的项目文档目录。
func isLocalizedDocsCatalog(name string) bool {
	return strings.HasPrefix(name, "docs.") && strings.HasSuffix(name, ".json") && name != "docs.json"
}

// readProjectDocsUpdatedAt 读取项目文档目录中所有文档的更新时间。
func readProjectDocsUpdatedAt(files fs.FS, name string) (map[string]string, error) {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		return nil, err
	}
	var catalog projectDocsCatalog
	err = json.Unmarshal(data, &catalog)
	if err != nil {
		return nil, err
	}
	updatedAt := make(map[string]string)
	err = appendProjectDocsUpdatedAt(updatedAt, catalog.Documents, catalog.Directories)
	if err != nil {
		return nil, err
	}
	return updatedAt, nil
}

// appendProjectDocsUpdatedAt 递归收集项目文档的稳定路径和更新时间。
func appendProjectDocsUpdatedAt(
	updatedAt map[string]string,
	documents []projectDocsDocument,
	directories []projectDocsDirectory,
) error {
	for _, document := range documents {
		if document.Path == "" || document.UpdatedAt == "" {
			return fmt.Errorf("文档缺少路径或更新时间")
		}
		if _, exists := updatedAt[document.Path]; exists {
			return fmt.Errorf("文档路径重复: %s", document.Path)
		}
		updatedAt[document.Path] = document.UpdatedAt
	}
	for _, directory := range directories {
		err := appendProjectDocsUpdatedAt(updatedAt, directory.Documents, directory.Directories)
		if err != nil {
			return err
		}
	}
	return nil
}

// sameProjectDocsUpdatedAt 判断本地化文档是否完整复用默认目录的更新时间。
func sameProjectDocsUpdatedAt(defaultUpdatedAt, localizedUpdatedAt map[string]string) bool {
	if len(defaultUpdatedAt) != len(localizedUpdatedAt) {
		return false
	}
	for path, updatedAt := range defaultUpdatedAt {
		if localizedUpdatedAt[path] != updatedAt {
			return false
		}
	}
	return true
}
