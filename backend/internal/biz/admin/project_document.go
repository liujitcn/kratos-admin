package biz

import (
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/projectdoc"
)

// ProjectDocumentCase 提供多项目文档目录查询业务。
type ProjectDocumentCase struct {
	catalog *projectdoc.Catalog
}

// NewProjectDocumentCase 创建多项目文档目录查询业务实例。
func NewProjectDocumentCase(catalog *projectdoc.Catalog) *ProjectDocumentCase {
	return &ProjectDocumentCase{catalog: catalog}
}

// TreeProjectDocument 查询全部项目文档树。
func (c *ProjectDocumentCase) TreeProjectDocument() *systemadminv1.TreeProjectDocumentResponse {
	catalogProjects := c.catalog.Projects()
	projects := make([]*systemadminv1.ProjectDocumentProject, 0, len(catalogProjects))
	for _, project := range catalogProjects {
		projects = append(projects, mapProjectDocumentProject(project))
	}
	return &systemadminv1.TreeProjectDocumentResponse{Projects: projects}
}

// GetProjectDocument 按稳定 ID 查询项目文档详情。
func (c *ProjectDocumentCase) GetProjectDocument(id string) (*systemadminv1.ProjectDocument, error) {
	document, exists := c.catalog.Get(id)
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

// ProjectDocuments 返回可继续向外部宿主贡献的项目文档。
func (c *ProjectDocumentCase) ProjectDocuments() []projectdoc.Document {
	return c.catalog.Documents()
}

// mapProjectDocumentProject 将领域项目目录转换为接口项目目录。
func mapProjectDocumentProject(project projectdoc.Project) *systemadminv1.ProjectDocumentProject {
	return &systemadminv1.ProjectDocumentProject{
		Key:         project.Key,
		Name:        project.Name,
		Documents:   mapProjectDocumentListItems(project.Documents),
		Directories: mapProjectDocumentDirectories(project.Directories),
	}
}

// mapProjectDocumentDirectory 递归转换领域文档目录。
func mapProjectDocumentDirectory(directory projectdoc.Directory) *systemadminv1.ProjectDocumentDirectory {
	return &systemadminv1.ProjectDocumentDirectory{
		Name:        directory.Name,
		Path:        directory.Path,
		Documents:   mapProjectDocumentListItems(directory.Documents),
		Directories: mapProjectDocumentDirectories(directory.Directories),
	}
}

// mapProjectDocumentListItems 转换文档目录项集合。
func mapProjectDocumentListItems(documents []projectdoc.Document) []*systemadminv1.ProjectDocumentListItem {
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

// mapProjectDocumentDirectories 转换递归文档目录集合。
func mapProjectDocumentDirectories(directories []projectdoc.Directory) []*systemadminv1.ProjectDocumentDirectory {
	items := make([]*systemadminv1.ProjectDocumentDirectory, 0, len(directories))
	for _, directory := range directories {
		items = append(items, mapProjectDocumentDirectory(directory))
	}
	return items
}
