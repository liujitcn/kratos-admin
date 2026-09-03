package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// ProjectDocumentService 提供项目文档只读接口。
type ProjectDocumentService struct {
	adminv1.UnimplementedProjectDocumentServiceServer
	projectDocumentCase *biz.ProjectDocumentCase
}

// NewProjectDocumentService 创建项目文档服务。
func NewProjectDocumentService(projectDocumentCase *biz.ProjectDocumentCase) *ProjectDocumentService {
	return &ProjectDocumentService{projectDocumentCase: projectDocumentCase}
}

// TreeProjectDocument 查询项目文档树。
func (s *ProjectDocumentService) TreeProjectDocument(
	ctx context.Context,
	req *adminv1.TreeProjectDocumentRequest,
) (*adminv1.TreeProjectDocumentResponse, error) {
	response, err := s.projectDocumentCase.TreeProjectDocument(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TreeProjectDocument %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目文档树失败")
	}
	return response, nil
}

// GetProjectDocument 查询项目文档详情。
func (s *ProjectDocumentService) GetProjectDocument(
	ctx context.Context,
	req *adminv1.GetProjectDocumentRequest,
) (*adminv1.ProjectDocument, error) {
	document, err := s.projectDocumentCase.GetProjectDocument(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetProjectDocument %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目文档失败")
	}
	return document, nil
}
