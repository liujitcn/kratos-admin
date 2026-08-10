package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// ProjectDocumentService 提供项目文档只读接口。
type ProjectDocumentService struct {
	systemadminv1.UnimplementedProjectDocumentServiceServer
	projectDocumentCase *biz.ProjectDocumentCase
}

// NewProjectDocumentService 创建项目文档服务。
func NewProjectDocumentService(projectDocumentCase *biz.ProjectDocumentCase) *ProjectDocumentService {
	return &ProjectDocumentService{projectDocumentCase: projectDocumentCase}
}

// TreeProjectDocument 查询项目文档树。
func (s *ProjectDocumentService) TreeProjectDocument(
	_ context.Context,
	_ *systemadminv1.TreeProjectDocumentRequest,
) (*systemadminv1.TreeProjectDocumentResponse, error) {
	return s.projectDocumentCase.TreeProjectDocument(), nil
}

// GetProjectDocument 查询项目文档详情。
func (s *ProjectDocumentService) GetProjectDocument(
	_ context.Context,
	req *systemadminv1.GetProjectDocumentRequest,
) (*systemadminv1.ProjectDocument, error) {
	document, err := s.projectDocumentCase.GetProjectDocument(req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetProjectDocument %v", err))
		return nil, errorsx.WrapInternal(err, "查询项目文档失败")
	}
	return document, nil
}
