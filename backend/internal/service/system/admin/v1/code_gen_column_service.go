package admin

import (
	"context"
	"fmt"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CodeGenColumnService Admin代码生成字段服务。
type CodeGenColumnService struct {
	systemadminv1.UnimplementedCodeGenColumnServiceServer
	codeGenColumnCase *biz.CodeGenColumnCase
}

// NewCodeGenColumnService 创建Admin代码生成字段服务。
func NewCodeGenColumnService(codeGenColumnCase *biz.CodeGenColumnCase) *CodeGenColumnService {
	return &CodeGenColumnService{codeGenColumnCase: codeGenColumnCase}
}

// ListCodeGenColumn 查询代码生成字段配置。
func (s *CodeGenColumnService) ListCodeGenColumn(ctx context.Context, req *systemadminv1.ListCodeGenColumnRequest) (*systemadminv1.ListCodeGenColumnResponse, error) {
	res, err := s.codeGenColumnCase.ListCodeGenColumn(ctx, req.GetTableId())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenColumn %v", err))
		return nil, errorsx.WrapInternal(err, "查询代码生成字段配置失败")
	}
	return res, nil
}

// ListCodeGenDatabaseColumn 查询数据库表字段列表。
func (s *CodeGenColumnService) ListCodeGenDatabaseColumn(ctx context.Context, req *systemadminv1.ListCodeGenDatabaseColumnRequest) (*systemadminv1.ListCodeGenDatabaseColumnResponse, error) {
	res, err := s.codeGenColumnCase.ListCodeGenDatabaseColumn(ctx, req.GetTableName())
	if err != nil {
		log.Error(fmt.Sprintf("ListCodeGenDatabaseColumn %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表字段列表失败")
	}
	return res, nil
}

// SaveCodeGenColumn 保存代码生成字段配置。
func (s *CodeGenColumnService) SaveCodeGenColumn(ctx context.Context, req *systemadminv1.SaveCodeGenColumnRequest) (*emptypb.Empty, error) {
	err := s.codeGenColumnCase.SaveCodeGenColumn(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveCodeGenColumn %v", err))
		return nil, errorsx.WrapInternal(err, "保存代码生成字段配置失败")
	}
	return new(emptypb.Empty), nil
}
