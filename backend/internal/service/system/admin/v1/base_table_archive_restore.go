package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseTableArchiveRestoreService 手工恢复归档数据。
type BaseTableArchiveRestoreService struct {
	adminv1.UnimplementedBaseTableArchiveRestoreServiceServer
	baseTableArchiveRestoreCase *biz.BaseTableArchiveRestoreCase
}

// NewBaseTableArchiveRestoreService 创建归档恢复服务。
func NewBaseTableArchiveRestoreService(baseTableArchiveRestoreCase *biz.BaseTableArchiveRestoreCase) *BaseTableArchiveRestoreService {
	return &BaseTableArchiveRestoreService{baseTableArchiveRestoreCase: baseTableArchiveRestoreCase}
}

// PageBaseTableArchiveRestore 分页查询归档恢复记录。
func (s *BaseTableArchiveRestoreService) PageBaseTableArchiveRestore(ctx context.Context, req *adminv1.PageBaseTableArchiveRestoreRequest) (*adminv1.PageBaseTableArchiveRestoreResponse, error) {
	result, err := s.baseTableArchiveRestoreCase.PageBaseTableArchiveRestore(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableArchiveRestore %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档恢复记录失败")
	}
	return result, nil
}

// GetBaseTableArchiveRestore 查询归档恢复记录。
func (s *BaseTableArchiveRestoreService) GetBaseTableArchiveRestore(ctx context.Context, req *adminv1.GetBaseTableArchiveRestoreRequest) (*adminv1.BaseTableArchiveRestore, error) {
	result, err := s.baseTableArchiveRestoreCase.GetBaseTableArchiveRestore(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableArchiveRestore %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档恢复记录失败")
	}
	return result, nil
}

// ExecuteBaseTableArchiveRestore 手工创建归档恢复请求。
func (s *BaseTableArchiveRestoreService) ExecuteBaseTableArchiveRestore(ctx context.Context, req *adminv1.ExecuteBaseTableArchiveRestoreRequest) (*emptypb.Empty, error) {
	err := s.baseTableArchiveRestoreCase.ExecuteBaseTableArchiveRestore(ctx, req.GetBaseTableArchiveRestore())
	if err != nil {
		log.Error(fmt.Sprintf("ExecuteBaseTableArchiveRestore %v", err))
		return nil, errorsx.WrapInternal(err, "执行归档恢复失败")
	}
	return new(emptypb.Empty), nil
}
