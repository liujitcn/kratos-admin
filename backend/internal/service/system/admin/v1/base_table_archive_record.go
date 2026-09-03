package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseTableArchiveRecordService 查询表归档执行记录。
type BaseTableArchiveRecordService struct {
	adminv1.UnimplementedBaseTableArchiveRecordServiceServer
	baseTableArchiveRecordCase *biz.BaseTableArchiveRecordCase
}

// NewBaseTableArchiveRecordService 创建表归档记录服务。
func NewBaseTableArchiveRecordService(baseTableArchiveRecordCase *biz.BaseTableArchiveRecordCase) *BaseTableArchiveRecordService {
	return &BaseTableArchiveRecordService{baseTableArchiveRecordCase: baseTableArchiveRecordCase}
}

// PageBaseTableArchiveRecord 分页查询表归档记录。
func (s *BaseTableArchiveRecordService) PageBaseTableArchiveRecord(ctx context.Context, req *adminv1.PageBaseTableArchiveRecordRequest) (*adminv1.PageBaseTableArchiveRecordResponse, error) {
	result, err := s.baseTableArchiveRecordCase.PageBaseTableArchiveRecord(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseTableArchiveRecord %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档记录失败")
	}
	return result, nil
}

// GetBaseTableArchiveRecord 查询表归档记录。
func (s *BaseTableArchiveRecordService) GetBaseTableArchiveRecord(ctx context.Context, req *adminv1.GetBaseTableArchiveRecordRequest) (*adminv1.BaseTableArchiveRecord, error) {
	result, err := s.baseTableArchiveRecordCase.GetBaseTableArchiveRecord(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseTableArchiveRecord %v", err))
		return nil, errorsx.WrapInternal(err, "查询归档记录失败")
	}
	return result, nil
}
