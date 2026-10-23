package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseScriptLogService Admin脚本执行日志服务。
type BaseScriptLogService struct {
	adminv1.UnimplementedBaseScriptLogServiceServer
	baseScriptLogCase *biz.BaseScriptLogCase
}

// NewBaseScriptLogService 创建Admin脚本执行日志服务。
func NewBaseScriptLogService(baseScriptLogCase *biz.BaseScriptLogCase) *BaseScriptLogService {
	return &BaseScriptLogService{baseScriptLogCase: baseScriptLogCase}
}

// PageBaseScriptLog 查询脚本执行日志分页列表。
func (s *BaseScriptLogService) PageBaseScriptLog(ctx context.Context, req *adminv1.PageBaseScriptLogRequest) (*adminv1.PageBaseScriptLogResponse, error) {
	result, err := s.baseScriptLogCase.PageBaseScriptLog(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseScriptLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询脚本执行日志失败")
	}
	return result, nil
}

// GetBaseScriptLog 查询脚本执行日志详情。
func (s *BaseScriptLogService) GetBaseScriptLog(ctx context.Context, req *adminv1.GetBaseScriptLogRequest) (*adminv1.BaseScriptLog, error) {
	result, err := s.baseScriptLogCase.GetBaseScriptLog(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseScriptLog %v", err))
		return nil, errorsx.WrapInternal(err, "查询脚本执行日志详情失败")
	}
	return result, nil
}
