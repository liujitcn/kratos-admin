package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseJobLogService Admin定时任务日志服务
type BaseJobLogService struct {
	adminv1.UnimplementedBaseJobLogServiceServer
	baseJobLogCase *biz.BaseJobLogCase
}

// NewBaseJobLogService 创建Admin定时任务日志服务
func NewBaseJobLogService(baseJobLogCase *biz.BaseJobLogCase) *BaseJobLogService {
	return &BaseJobLogService{baseJobLogCase: baseJobLogCase}
}

// PageBaseJobLog 查询定时任务日志分页列表
func (s *BaseJobLogService) PageBaseJobLog(ctx context.Context, req *adminv1.PageBaseJobLogRequest) (*adminv1.PageBaseJobLogResponse, error) {
	page, err := s.baseJobLogCase.PageBaseJobLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseJobLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询定时任务日志分页列表失败")
	}
	return page, nil
}

// GetBaseJobLog 查询定时任务日志
func (s *BaseJobLogService) GetBaseJobLog(ctx context.Context, req *adminv1.GetBaseJobLogRequest) (*adminv1.BaseJobLog, error) {
	baseLog, err := s.baseJobLogCase.GetBaseJobLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseJobLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询定时任务日志失败")
	}
	return baseLog, nil
}
