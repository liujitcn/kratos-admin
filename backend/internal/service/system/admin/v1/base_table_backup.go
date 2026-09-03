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

// BaseTableBackupService 管理数据库备份配置。
type BaseTableBackupService struct {
	adminv1.UnimplementedBaseTableBackupServiceServer
	baseTableBackupCase *biz.BaseTableBackupCase
}

// NewBaseTableBackupService 创建数据库备份配置服务。
func NewBaseTableBackupService(baseTableBackupCase *biz.BaseTableBackupCase) *BaseTableBackupService {
	return &BaseTableBackupService{baseTableBackupCase: baseTableBackupCase}
}

// PageBaseTableBackup 分页查询数据库备份配置。
func (s *BaseTableBackupService) PageBaseTableBackup(ctx context.Context, req *adminv1.PageBaseTableBackupRequest) (*adminv1.PageBaseTableBackupResponse, error) {
	result, err := s.baseTableBackupCase.PageBaseTableBackup(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableBackup %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份配置失败")
	}
	return result, nil
}

// GetBaseTableBackup 查询数据库备份配置。
func (s *BaseTableBackupService) GetBaseTableBackup(ctx context.Context, req *adminv1.GetBaseTableBackupRequest) (*adminv1.BaseTableBackupForm, error) {
	result, err := s.baseTableBackupCase.GetBaseTableBackup(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableBackup %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份配置失败")
	}
	return result, nil
}

// CreateBaseTableBackup 创建数据库备份配置。
func (s *BaseTableBackupService) CreateBaseTableBackup(ctx context.Context, req *adminv1.CreateBaseTableBackupRequest) (*emptypb.Empty, error) {
	err := s.baseTableBackupCase.CreateBaseTableBackup(ctx, req.GetBaseTableBackup())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseTableBackup %v", err))
		return nil, errorsx.WrapInternal(err, "创建备份配置失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseTableBackup 更新数据库备份配置。
func (s *BaseTableBackupService) UpdateBaseTableBackup(ctx context.Context, req *adminv1.UpdateBaseTableBackupRequest) (*emptypb.Empty, error) {
	err := s.baseTableBackupCase.UpdateBaseTableBackup(ctx, req.GetBaseTableBackup())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseTableBackup %v", err))
		return nil, errorsx.WrapInternal(err, "更新备份配置失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseTableBackup 删除数据库备份配置。
func (s *BaseTableBackupService) DeleteBaseTableBackup(ctx context.Context, req *adminv1.DeleteBaseTableBackupRequest) (*emptypb.Empty, error) {
	err := s.baseTableBackupCase.DeleteBaseTableBackup(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseTableBackup %v", err))
		return nil, errorsx.WrapInternal(err, "删除备份配置失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseTableBackupStatus 设置数据库备份配置状态。
func (s *BaseTableBackupService) SetBaseTableBackupStatus(ctx context.Context, req *adminv1.SetBaseTableBackupStatusRequest) (*emptypb.Empty, error) {
	err := s.baseTableBackupCase.SetBaseTableBackupStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseTableBackupStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置备份配置状态失败")
	}
	return new(emptypb.Empty), nil
}
