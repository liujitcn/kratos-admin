package biz

import (
	"sync"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/pkg/docs"
	"github.com/liujitcn/kratos-core/pkg/errorsx"
)

// ProjectDocumentCase 提供 Core 项目文档注册表到接口模型的转换。
type ProjectDocumentCase struct {
	mu       sync.RWMutex
	registry *docs.Registry
}

// NewProjectDocumentCase 创建项目文档查询业务实例。
func NewProjectDocumentCase() *ProjectDocumentCase {
	return &ProjectDocumentCase{}
}

// SetProjectDocumentRegistry 接收 Core 汇总后的项目文档注册表。
func (c *ProjectDocumentCase) SetProjectDocumentRegistry(registry *docs.Registry) {
	c.mu.Lock()
	c.registry = registry
	c.mu.Unlock()
}

// TreeProjectDocument 查询 Core 已注册项目文档树。
func (c *ProjectDocumentCase) TreeProjectDocument() *systemadminv1.TreeProjectDocumentResponse {
	registry := c.registrySnapshot()
	coreProjects := registry.Projects()
	projects := make([]*systemadminv1.ProjectDocumentProject, 0, len(coreProjects))
	for _, project := range coreProjects {
		projects = append(projects, mapProjectDocumentProject(project))
	}
	return &systemadminv1.TreeProjectDocumentResponse{Projects: projects}
}

// GetProjectDocument 按稳定 ID 查询 Core 已注册项目文档详情。
func (c *ProjectDocumentCase) GetProjectDocument(id string) (*systemadminv1.ProjectDocument, error) {
	document, exists := c.registrySnapshot().Get(id)
	if !exists {
		return nil, errorsx.ResourceNotFound("项目文档不存在")
	}
	return &systemadminv1.ProjectDocument{
		Id:          document.ID,
		ProjectKey:  document.ProjectKey,
		ProjectName: document.ProjectName,
		Path:        document.Path,
		Content:     document.Content,
	}, nil
}

// registrySnapshot 返回当前 Core 文档注册表，空注册表用于兼容启动早期查询。
func (c *ProjectDocumentCase) registrySnapshot() *docs.Registry {
	c.mu.RLock()
	registry := c.registry
	c.mu.RUnlock()
	if registry != nil {
		return registry
	}
	return &docs.Registry{}
}

// mapProjectDocumentProject 将 Core 项目目录转换为接口项目目录。
func mapProjectDocumentProject(project docs.Project) *systemadminv1.ProjectDocumentProject {
	item := &systemadminv1.ProjectDocumentProject{
		Key:       project.Key,
		Name:      project.Name,
		Documents: mapProjectDocumentListItems(project.Documents),
	}
	for _, directory := range project.Directories {
		item.Directories = append(item.Directories, mapProjectDocumentDirectory(directory))
	}
	return item
}

// mapProjectDocumentDirectory 递归转换 Core 文档目录。
func mapProjectDocumentDirectory(directory docs.Directory) *systemadminv1.ProjectDocumentDirectory {
	item := &systemadminv1.ProjectDocumentDirectory{
		Name:      directory.Name,
		Path:      directory.Path,
		Documents: mapProjectDocumentListItems(directory.Documents),
	}
	for _, child := range directory.Directories {
		item.Directories = append(item.Directories, mapProjectDocumentDirectory(child))
	}
	return item
}

// mapProjectDocumentListItems 转换 Core 文档摘要集合。
func mapProjectDocumentListItems(documents []docs.DocumentListItem) []*systemadminv1.ProjectDocumentListItem {
	items := make([]*systemadminv1.ProjectDocumentListItem, 0, len(documents))
	for _, document := range documents {
		items = append(items, &systemadminv1.ProjectDocumentListItem{
			Id:        document.ID,
			Path:      document.Path,
			UpdatedAt: document.UpdatedAt,
		})
	}
	return items
}
