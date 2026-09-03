package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseTableArchiveService 管理表归档配置。
type BaseTableArchiveService struct {
	adminv1.UnimplementedBaseTableArchiveServiceServer
	baseTableArchiveCase *biz.BaseTableArchiveCase
}

// NewBaseTableArchiveService 创建表归档配置服务。
func NewBaseTableArchiveService(baseTableArchiveCase *biz.BaseTableArchiveCase) *BaseTableArchiveService {
	return &BaseTableArchiveService{baseTableArchiveCase: baseTableArchiveCase}
}

// PageBaseTableArchive 分页查询表归档配置。
func (s *BaseTableArchiveService) PageBaseTableArchive(ctx context.Context, req *adminv1.PageBaseTableArchiveRequest) (*adminv1.PageBaseTableArchiveResponse, error) {
	result, err := s.baseTableArchiveCase.PageBaseTableArchive(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableArchive %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档配置失败")
	}
	return result, nil
}

// GetBaseTableArchive 查询表归档配置。
func (s *BaseTableArchiveService) GetBaseTableArchive(ctx context.Context, req *adminv1.GetBaseTableArchiveRequest) (*adminv1.BaseTableArchiveForm, error) {
	result, err := s.baseTableArchiveCase.GetBaseTableArchive(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableArchive %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档配置失败")
	}
	return result, nil
}

// CreateBaseTableArchive 创建表归档配置。
func (s *BaseTableArchiveService) CreateBaseTableArchive(ctx context.Context, req *adminv1.CreateBaseTableArchiveRequest) (*emptypb.Empty, error) {
	err := s.baseTableArchiveCase.CreateBaseTableArchive(ctx, req.GetBaseTableArchive())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTableArchive %v", err))
		return nil, errorsx.WrapInternal(err, "创建归档配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTableArchive 更新表归档配置。
func (s *BaseTableArchiveService) UpdateBaseTableArchive(ctx context.Context, req *adminv1.UpdateBaseTableArchiveRequest) (*emptypb.Empty, error) {
	err := s.baseTableArchiveCase.UpdateBaseTableArchive(ctx, req.GetBaseTableArchive())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTableArchive %v", err))
		return nil, errorsx.WrapInternal(err, "更新归档配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTableArchive 删除表归档配置。
func (s *BaseTableArchiveService) DeleteBaseTableArchive(ctx context.Context, req *adminv1.DeleteBaseTableArchiveRequest) (*emptypb.Empty, error) {
	err := s.baseTableArchiveCase.DeleteBaseTableArchive(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTableArchive %v", err))
		return nil, errorsx.WrapInternal(err, "删除归档配置失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseTableArchiveStatus 设置表归档配置状态。
func (s *BaseTableArchiveService) SetBaseTableArchiveStatus(ctx context.Context, req *adminv1.SetBaseTableArchiveStatusRequest) (*emptypb.Empty, error) {
	err := s.baseTableArchiveCase.SetBaseTableArchiveStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseTableArchiveStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置归档配置状态失败")
	}
	return new(emptypb.Empty), nil
}
