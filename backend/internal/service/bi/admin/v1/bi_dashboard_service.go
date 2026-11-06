package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BiDashboardService 提供 BI 仪表盘草稿和发布管理。
type BiDashboardService struct {
	adminv1.UnimplementedBiDashboardServiceServer
	biDashboardCase *admin.BiDashboardCase
}

// NewBiDashboardService 创建 BI 仪表盘服务。
func NewBiDashboardService(biDashboardCase *admin.BiDashboardCase) *BiDashboardService {
	return &BiDashboardService{biDashboardCase: biDashboardCase}
}

// PageBiDashboard 分页查询 BI 仪表盘。
func (s *BiDashboardService) PageBiDashboard(ctx context.Context, req *adminv1.PageBiDashboardRequest) (*adminv1.PageBiDashboardResponse, error) {
	result, err := s.biDashboardCase.PageBiDashboard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBiDashboard %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 仪表盘失败")
	}
	return result, nil
}

// GetBiDashboard 查询 BI 仪表盘。
func (s *BiDashboardService) GetBiDashboard(ctx context.Context, req *adminv1.GetBiDashboardRequest) (*adminv1.BiDashboard, error) {
	result, err := s.biDashboardCase.GetBiDashboard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiDashboard %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 仪表盘失败")
	}
	return result, nil
}

// SaveBiDashboard 保存 BI 仪表盘草稿。
func (s *BiDashboardService) SaveBiDashboard(ctx context.Context, req *adminv1.SaveBiDashboardRequest) (*adminv1.BiDashboard, error) {
	result, err := s.biDashboardCase.SaveBiDashboard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveBiDashboard %v", err))
		return nil, errorsx.WrapInternal(err, "保存 BI 仪表盘失败")
	}
	return result, nil
}

// PublishBiDashboard 发布 BI 仪表盘。
func (s *BiDashboardService) PublishBiDashboard(ctx context.Context, req *adminv1.PublishBiDashboardRequest) (*adminv1.BiDashboard, error) {
	result, err := s.biDashboardCase.PublishBiDashboard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PublishBiDashboard %v", err))
		return nil, errorsx.WrapInternal(err, "发布 BI 仪表盘失败")
	}
	return result, nil
}

// DeleteBiDashboard 删除 BI 仪表盘。
func (s *BiDashboardService) DeleteBiDashboard(ctx context.Context, req *adminv1.DeleteBiDashboardRequest) (*adminv1.DeleteBiDashboardResponse, error) {
	result, err := s.biDashboardCase.DeleteBiDashboard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBiDashboard %v", err))
		return nil, errorsx.WrapInternal(err, "删除 BI 仪表盘失败")
	}
	return result, nil
}

// GetBiDashboardData 执行已发布仪表盘中的图表查询。
func (s *BiDashboardService) GetBiDashboardData(ctx context.Context, req *adminv1.GetBiDashboardDataRequest) (*adminv1.BiDashboardDataResponse, error) {
	result, err := s.biDashboardCase.GetBiDashboardData(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiDashboardData %v", err))
		return nil, errorsx.WrapInternal(err, "查询 BI 仪表盘数据失败")
	}
	return result, nil
}
