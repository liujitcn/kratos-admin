package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseOperationLogService 提供业务操作日志的只读管理接口。
type BaseOperationLogService struct {
	adminv1.UnimplementedBaseOperationLogServiceServer // 提供未实现 RPC 的默认返回。
	baseOperationLogCase                               *biz.BaseOperationLogCase
}

// NewBaseOperationLogService 创建业务操作日志管理服务。
func NewBaseOperationLogService(baseOperationLogCase *biz.BaseOperationLogCase) *BaseOperationLogService {
	return &BaseOperationLogService{baseOperationLogCase: baseOperationLogCase}
}

// PageBaseOperationLog 查询业务操作日志分页列表。
func (s *BaseOperationLogService) PageBaseOperationLog(ctx context.Context, req *adminv1.PageBaseOperationLogRequest) (*adminv1.PageBaseOperationLogResponse, error) {
	page, err := s.baseOperationLogCase.PageBaseOperationLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseOperationLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询业务操作日志分页列表失败")
	}
	return page, nil
}

// GetBaseOperationLog 查询业务操作日志详情。
func (s *BaseOperationLogService) GetBaseOperationLog(ctx context.Context, req *adminv1.GetBaseOperationLogRequest) (*adminv1.BaseOperationLog, error) {
	item, err := s.baseOperationLogCase.GetBaseOperationLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseOperationLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询业务操作日志详情失败")
	}
	return item, nil
}
