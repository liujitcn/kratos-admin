package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseTableBackupRecordService 查询数据库备份执行记录。
type BaseTableBackupRecordService struct {
	adminv1.UnimplementedBaseTableBackupRecordServiceServer
	baseTableBackupRecordCase *biz.BaseTableBackupRecordCase
}

// NewBaseTableBackupRecordService 创建数据库备份记录服务。
func NewBaseTableBackupRecordService(baseTableBackupRecordCase *biz.BaseTableBackupRecordCase) *BaseTableBackupRecordService {
	return &BaseTableBackupRecordService{baseTableBackupRecordCase: baseTableBackupRecordCase}
}

// PageBaseTableBackupRecord 分页查询数据库备份记录。
func (s *BaseTableBackupRecordService) PageBaseTableBackupRecord(ctx context.Context, req *adminv1.PageBaseTableBackupRecordRequest) (*adminv1.PageBaseTableBackupRecordResponse, error) {
	result, err := s.baseTableBackupRecordCase.PageBaseTableBackupRecord(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableBackupRecord %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份记录失败")
	}
	return result, nil
}

// GetBaseTableBackupRecord 查询数据库备份记录。
func (s *BaseTableBackupRecordService) GetBaseTableBackupRecord(ctx context.Context, req *adminv1.GetBaseTableBackupRecordRequest) (*adminv1.BaseTableBackupRecord, error) {
	result, err := s.baseTableBackupRecordCase.GetBaseTableBackupRecord(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableBackupRecord %v", err))
		return nil, errorsx.WrapInternal(err, "查询备份记录失败")
	}
	return result, nil
}
