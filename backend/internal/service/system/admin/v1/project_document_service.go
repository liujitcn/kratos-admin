package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/errorsx"
	coreDocs "github.com/liujitcn/kratos-core/resource/docs"
	docsdto "github.com/liujitcn/kratos-core/resource/docs/dto"

	"github.com/go-kratos/kratos/v3/log"
)

// ProjectDocumentService 提供项目文档只读接口。
type ProjectDocumentService struct {
	systemadminv1.UnimplementedProjectDocumentServiceServer
	docs *coreDocs.Docs
}

// NewProjectDocumentService 创建项目文档服务。
func NewProjectDocumentService(docs *coreDocs.Docs) *ProjectDocumentService {
	return &ProjectDocumentService{docs: docs}
}

// TreeProjectDocument 查询项目文档树。
func (s *ProjectDocumentService) TreeProjectDocument(
	ctx context.Context,
	_ *systemadminv1.TreeProjectDocumentRequest,
) (*systemadminv1.TreeProjectDocumentResponse, error) {
	coreProjects := s.docs.Projects(ctx)
	projects := make([]*systemadminv1.ProjectDocumentProject, 0, len(coreProjects))
	for _, project := range coreProjects {
		projects = append(projects, mapProjectDocumentProject(project))
	}
	return &systemadminv1.TreeProjectDocumentResponse{Projects: projects}, nil
}

// GetProjectDocument 查询项目文档详情。
func (s *ProjectDocumentService) GetProjectDocument(
	ctx context.Context,
	req *systemadminv1.GetProjectDocumentRequest,
) (*systemadminv1.ProjectDocument, error) {
	document, exists := s.docs.Get(ctx, req.GetId())
	if !exists {
		err := errorsx.ResourceNotFound("项目文档不存在")
		log.Error(fmt.Sprintf("GetProjectDocument %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目文档失败")
	}
	return mapProjectDocument(document), nil
}

// mapProjectDocumentProject 将 Core 项目目录转换为 Admin 接口项目目录。
func mapProjectDocumentProject(project docsdto.Project) *systemadminv1.ProjectDocumentProject {
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
func mapProjectDocumentDirectory(directory docsdto.Directory) *systemadminv1.ProjectDocumentDirectory {
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
func mapProjectDocumentListItems(documents []docsdto.DocumentListItem) []*systemadminv1.ProjectDocumentListItem {
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

// mapProjectDocument 转换 Core 文档详情。
func mapProjectDocument(document docsdto.Document) *systemadminv1.ProjectDocument {
	return &systemadminv1.ProjectDocument{
		Id:          document.ID,
		ProjectKey:  document.ProjectKey,
		ProjectName: document.ProjectName,
		Path:        document.Path,
		Content:     document.Content,
		Locale:      document.Locale,
		UpdatedAt:   document.UpdatedAt,
	}
}
