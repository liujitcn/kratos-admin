package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseLogService 提供公共日志聚合查询接口。
type BaseLogService struct {
	adminv1.UnimplementedBaseLogServiceServer // 提供未实现 RPC 的默认返回。
	baseLogCase                               *biz.BaseLogCase
}

// NewBaseLogService 创建公共日志聚合查询服务。
func NewBaseLogService(baseLogCase *biz.BaseLogCase) *BaseLogService {
	return &BaseLogService{baseLogCase: baseLogCase}
}

// GetBaseLogTrace 查询同一请求或链路关联的审计时间线。
func (s *BaseLogService) GetBaseLogTrace(ctx context.Context, req *adminv1.GetBaseLogTraceRequest) (*adminv1.GetBaseLogTraceResponse, error) {
	result, err := s.baseLogCase.GetBaseLogTrace(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseLogTrace %v", err))
		return nil, errorsx.WrapInternal(err, "查询关联审计时间线失败")
	}
	return result, nil
}
