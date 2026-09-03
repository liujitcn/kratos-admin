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

// BaseTableBackupRestoreService 手工恢复数据库备份。
type BaseTableBackupRestoreService struct {
	adminv1.UnimplementedBaseTableBackupRestoreServiceServer
	baseTableBackupRestoreCase *biz.BaseTableBackupRestoreCase
}

// NewBaseTableBackupRestoreService 创建数据库备份恢复服务。
func NewBaseTableBackupRestoreService(baseTableBackupRestoreCase *biz.BaseTableBackupRestoreCase) *BaseTableBackupRestoreService {
	return &BaseTableBackupRestoreService{baseTableBackupRestoreCase: baseTableBackupRestoreCase}
}

// PageBaseTableBackupRestore 分页查询数据库备份恢复记录。
func (s *BaseTableBackupRestoreService) PageBaseTableBackupRestore(ctx context.Context, req *adminv1.PageBaseTableBackupRestoreRequest) (*adminv1.PageBaseTableBackupRestoreResponse, error) {
	result, err := s.baseTableBackupRestoreCase.PageBaseTableBackupRestore(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableBackupRestore %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份恢复记录失败")
	}
	return result, nil
}

// GetBaseTableBackupRestore 查询数据库备份恢复记录。
func (s *BaseTableBackupRestoreService) GetBaseTableBackupRestore(ctx context.Context, req *adminv1.GetBaseTableBackupRestoreRequest) (*adminv1.BaseTableBackupRestore, error) {
	result, err := s.baseTableBackupRestoreCase.GetBaseTableBackupRestore(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableBackupRestore %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份恢复记录失败")
	}
	return result, nil
}

// ExecuteBaseTableBackupRestore 手工创建数据库备份恢复请求。
func (s *BaseTableBackupRestoreService) ExecuteBaseTableBackupRestore(ctx context.Context, req *adminv1.ExecuteBaseTableBackupRestoreRequest) (*emptypb.Empty, error) {
	err := s.baseTableBackupRestoreCase.ExecuteBaseTableBackupRestore(ctx, req.GetBaseTableBackupRestore())
	if err != nil {
		log.Error(fmt.Sprintf("ExecuteBaseTableBackupRestore %v", err))
		return nil, errorsx.WrapInternal(err, "执行备份恢复失败")
	}
	return new(emptypb.Empty), nil
}
