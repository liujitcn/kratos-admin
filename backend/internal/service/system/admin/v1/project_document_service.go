package admin

import (
	"context"
	"fmt"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	coreDocs "github.com/liujitcn/kratos-core/resource/docs"
	"github.com/liujitcn/kratos-core/resource/docs/dto"
	"github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/go-kratos/kratos/v3/log"
)

// ProjectDocumentService 提供项目文档只读接口。
type ProjectDocumentService struct {
	adminv1.UnimplementedProjectDocumentServiceServer
	docs    *coreDocs.Docs
	catalog *i18n.I18n
}

// NewProjectDocumentService 创建项目文档服务。
func NewProjectDocumentService(docs *coreDocs.Docs, catalog *i18n.I18n) *ProjectDocumentService {
	return &ProjectDocumentService{docs: docs, catalog: catalog}
}

// TreeProjectDocument 查询项目文档树。
func (s *ProjectDocumentService) TreeProjectDocument(
	ctx context.Context,
	_ *adminv1.TreeProjectDocumentRequest,
) (*adminv1.TreeProjectDocumentResponse, error) {
	coreProjects := s.docs.Projects(ctx)
	projects := make([]*adminv1.ProjectDocumentProject, 0, len(coreProjects))
	for _, project := range coreProjects {
		projects = append(projects, s.mapProjectDocumentProject(ctx, project))
	}
	return &adminv1.TreeProjectDocumentResponse{Projects: projects}, nil
}

// GetProjectDocument 查询项目文档详情。
func (s *ProjectDocumentService) GetProjectDocument(
	ctx context.Context,
	req *adminv1.GetProjectDocumentRequest,
) (*adminv1.ProjectDocument, error) {
	document, exists := s.docs.Get(ctx, req.GetId())
	if !exists {
		err := errorsx.ResourceNotFound("项目文档不存在")
		log.Error(fmt.Sprintf("GetProjectDocument %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目文档失败")
	}
	return s.mapProjectDocument(ctx, document), nil
}

// mapProjectDocumentProject 将 Core 项目目录转换为 Admin 接口项目目录。
func (s *ProjectDocumentService) mapProjectDocumentProject(ctx context.Context, project dto.Project) *adminv1.ProjectDocumentProject {
	item := &adminv1.ProjectDocumentProject{
		Key:       project.Key,
		Name:      s.localizeProjectName(ctx, project.Name),
		Documents: mapProjectDocumentListItems(project.Documents),
	}
	for _, directory := range project.Directories {
		item.Directories = append(item.Directories, mapProjectDocumentDirectory(directory))
	}
	return item
}

// mapProjectDocument 转换 Core 文档详情。
func (s *ProjectDocumentService) mapProjectDocument(ctx context.Context, document dto.Document) *adminv1.ProjectDocument {
	return &adminv1.ProjectDocument{
		Id:          document.ID,
		ProjectKey:  document.ProjectKey,
		ProjectName: s.localizeProjectName(ctx, document.ProjectName),
		Path:        document.Path,
		Name:        document.Name,
		Content:     document.Content,
		UpdatedAt:   document.UpdatedAt,
	}
}

// localizeProjectName 返回请求语言对应的项目展示名称。
func (s *ProjectDocumentService) localizeProjectName(ctx context.Context, name string) string {
	messageKey, found := s.catalog.KeyForSource(name)
	if !found {
		return name
	}
	return s.catalog.Localize(biz.LocaleFromContext(ctx), "", messageKey, nil, name)
}

// mapProjectDocumentDirectory 递归转换 Core 文档目录。
func mapProjectDocumentDirectory(directory dto.Directory) *adminv1.ProjectDocumentDirectory {
	item := &adminv1.ProjectDocumentDirectory{
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
func mapProjectDocumentListItems(documents []dto.DocumentListItem) []*adminv1.ProjectDocumentListItem {
	items := make([]*adminv1.ProjectDocumentListItem, 0, len(documents))
	for _, document := range documents {
		items = append(items, &adminv1.ProjectDocumentListItem{
			Id:        document.ID,
			Path:      document.Path,
			Name:      document.Name,
			UpdatedAt: document.UpdatedAt,
		})
	}
	return items
}
