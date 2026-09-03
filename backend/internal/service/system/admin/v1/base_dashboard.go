package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseDashboardService 提供后台首页业务统计接口。
type BaseDashboardService struct {
	adminv1.UnimplementedBaseDashboardServiceServer
	baseDashboardCase *biz.BaseDashboardCase
}

// NewBaseDashboardService 创建后台首页统计服务。
func NewBaseDashboardService(baseDashboardCase *biz.BaseDashboardCase) *BaseDashboardService {
	return &BaseDashboardService{baseDashboardCase: baseDashboardCase}
}

// GetBaseDashboardOverview 查询首页概览统计。
func (s *BaseDashboardService) GetBaseDashboardOverview(ctx context.Context, _ *adminv1.GetBaseDashboardOverviewRequest) (*adminv1.BaseDashboardOverview, error) {
	res, err := s.baseDashboardCase.GetBaseDashboardOverview(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDashboardOverview %v", err))
		return nil, errorsx.WrapInternal(err, "查询首页概览统计失败")
	}
	return res, nil
}

// GetBaseDashboardLoginTrend 查询登录趋势。
func (s *BaseDashboardService) GetBaseDashboardLoginTrend(ctx context.Context, req *adminv1.GetBaseDashboardLoginTrendRequest) (*adminv1.BaseDashboardTrendResponse, error) {
	res, err := s.baseDashboardCase.GetBaseDashboardLoginTrend(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDashboardLoginTrend %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录趋势失败")
	}
	return res, nil
}

// GetBaseDashboardOperationDistribution 查询操作动作分布。
func (s *BaseDashboardService) GetBaseDashboardOperationDistribution(ctx context.Context, _ *adminv1.GetBaseDashboardOperationDistributionRequest) (*adminv1.BaseDashboardDistributionResponse, error) {
	res, err := s.baseDashboardCase.GetBaseDashboardOperationDistribution(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDashboardOperationDistribution %v", err))
		return nil, errorsx.WrapInternal(err, "查询操作动作分布失败")
	}
	return res, nil
}

// GetBaseDashboardLoginDistribution 查询登录结果分布。
func (s *BaseDashboardService) GetBaseDashboardLoginDistribution(ctx context.Context, _ *adminv1.GetBaseDashboardLoginDistributionRequest) (*adminv1.BaseDashboardDistributionResponse, error) {
	res, err := s.baseDashboardCase.GetBaseDashboardLoginDistribution(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDashboardLoginDistribution %v", err))
		return nil, errorsx.WrapInternal(err, "查询登录结果分布失败")
	}
	return res, nil
}
