package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CodeGenTableService Admin代码生成表配置服务。
type CodeGenTableService struct {
	systemadminv1.UnimplementedCodeGenTableServiceServer
	codeGenTableCase *biz.CodeGenTableCase
}

// NewCodeGenTableService 创建Admin代码生成表配置服务。
func NewCodeGenTableService(codeGenTableCase *biz.CodeGenTableCase) *CodeGenTableService {
	return &CodeGenTableService{codeGenTableCase: codeGenTableCase}
}

// PageCodeGenTable 查询代码生成表配置列表。
func (s *CodeGenTableService) PageCodeGenTable(ctx context.Context, req *systemadminv1.PageCodeGenTableRequest) (*systemadminv1.PageCodeGenTableResponse, error) {
	page, err := s.codeGenTableCase.PageCodeGenTable(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成表配置列表失败")
	}
	return page, nil
}

// ListCodeGenDatabaseTable 查询数据库表列表。
func (s *CodeGenTableService) ListCodeGenDatabaseTable(ctx context.Context, _ *systemadminv1.ListCodeGenDatabaseTableRequest) (*systemadminv1.ListCodeGenDatabaseTableResponse, error) {
	res, err := s.codeGenTableCase.ListCodeGenDatabaseTable(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenDatabaseTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表列表失败")
	}
	return res, nil
}

// GetCodeGenTable 查询代码生成表配置。
func (s *CodeGenTableService) GetCodeGenTable(ctx context.Context, req *systemadminv1.GetCodeGenTableRequest) (*systemadminv1.CodeGenTableForm, error) {
	item, err := s.codeGenTableCase.GetCodeGenTable(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成表配置失败")
	}
	return item, nil
}

// CreateCodeGenTable 创建代码生成表配置。
func (s *CodeGenTableService) CreateCodeGenTable(ctx context.Context, req *systemadminv1.CreateCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.CreateCodeGenTable(ctx, req.GetCodeGenTable())
	if err != nil {
		log.Error(fmt.Sprintf("CreateCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "创建代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateCodeGenTable 更新代码生成表配置。
func (s *CodeGenTableService) UpdateCodeGenTable(ctx context.Context, req *systemadminv1.UpdateCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.UpdateCodeGenTable(ctx, req.GetId(), req.GetCodeGenTable())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "更新代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteCodeGenTable 删除代码生成表配置。
func (s *CodeGenTableService) DeleteCodeGenTable(ctx context.Context, req *systemadminv1.DeleteCodeGenTableRequest) (*emptypb.Empty, error) {
	err := s.codeGenTableCase.DeleteCodeGenTable(ctx, req.GetIds())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteCodeGenTable %v", err))
		return nil, errorsx.WrapInternal(err, "删除代码生成表配置失败")
	}
	return new(emptypb.Empty), nil
}
