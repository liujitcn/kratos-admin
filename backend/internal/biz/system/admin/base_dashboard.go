package biz

import (
	"context"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
)

// BaseDashboardCase 提供后台首页的只读业务统计。
type BaseDashboardCase struct {
	*biz.BaseCase
	userRepo         *data.BaseUserRepository
	roleRepo         *data.BaseRoleRepository
	loginLogRepo     *data.BaseLoginLogRepository
	operationLogRepo *data.BaseOperationLogRepository
}

// NewBaseDashboardCase 创建后台首页统计业务实例。
func NewBaseDashboardCase(
	baseCase *biz.BaseCase,
	userRepo *data.BaseUserRepository,
	roleRepo *data.BaseRoleRepository,
	loginLogRepo *data.BaseLoginLogRepository,
	operationLogRepo *data.BaseOperationLogRepository,
) *BaseDashboardCase {
	return &BaseDashboardCase{
		BaseCase:         baseCase,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		loginLogRepo:     loginLogRepo,
		operationLogRepo: operationLogRepo,
	}
}

// GetBaseDashboardOverview 查询用户、角色和当天审计数量。
func (c *BaseDashboardCase) GetBaseDashboardOverview(ctx context.Context) (*adminv1.BaseDashboardOverview, error) {
	var userCount int64
	var err error
	userCount, err = c.userRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	var roleCount int64
	roleCount, err = c.roleRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	start, end := dashboardDayRange(time.Now())
	loginQuery := c.loginLogRepo.Query(ctx).BaseLoginLog
	loginOpts := []repository.QueryOption{
		repository.Where(loginQuery.OccurredAt.Gte(start)),
		repository.Where(loginQuery.OccurredAt.Lt(end)),
	}
	var todayLoginCount int64
	todayLoginCount, err = c.loginLogRepo.Count(ctx, loginOpts...)
	if err != nil {
		return nil, err
	}
	operationQuery := c.operationLogRepo.Query(ctx).BaseOperationLog
	operationOpts := []repository.QueryOption{
		repository.Where(operationQuery.OccurredAt.Gte(start)),
		repository.Where(operationQuery.OccurredAt.Lt(end)),
	}
	var todayOperationCount int64
	todayOperationCount, err = c.operationLogRepo.Count(ctx, operationOpts...)
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseDashboardOverview{
		UserCount:           uint64(userCount),
		RoleCount:           uint64(roleCount),
		TodayLoginCount:     uint64(todayLoginCount),
		TodayOperationCount: uint64(todayOperationCount),
	}, nil
}

// GetBaseDashboardLoginTrend 查询指定天数的登录次数趋势。
func (c *BaseDashboardCase) GetBaseDashboardLoginTrend(ctx context.Context, req *adminv1.GetBaseDashboardLoginTrendRequest) (*adminv1.BaseDashboardTrendResponse, error) {
	days := int(req.GetDays())
	if days == 0 {
		days = 7
	}
	now := time.Now()
	start := dashboardStartOfDay(now).AddDate(0, 0, -(days - 1))
	end := dashboardStartOfDay(now).AddDate(0, 0, 1)
	query := c.loginLogRepo.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{
		repository.Where(query.OccurredAt.Gte(start)),
		repository.Where(query.OccurredAt.Lt(end)),
		repository.Select(query.OccurredAt),
	}
	var err error
	var rows []*models.BaseLoginLog
	rows, err = c.loginLogRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]uint64, len(rows))
	for _, row := range rows {
		counts[row.OccurredAt.Format("2006-01-02")]++
	}
	points := make([]*adminv1.BaseDashboardTrendPoint, 0, days)
	for offset := 0; offset < days; offset++ {
		date := start.AddDate(0, 0, offset)
		key := date.Format("2006-01-02")
		points = append(points, &adminv1.BaseDashboardTrendPoint{Date: key, Count: counts[key]})
	}
	return &adminv1.BaseDashboardTrendResponse{Points: points}, nil
}

// GetBaseDashboardOperationDistribution 查询操作动作分布。
func (c *BaseDashboardCase) GetBaseDashboardOperationDistribution(ctx context.Context) (*adminv1.BaseDashboardDistributionResponse, error) {
	query := c.operationLogRepo.Query(ctx).BaseOperationLog
	opts := []repository.QueryOption{repository.Select(query.Action)}
	var err error
	var rows []*models.BaseOperationLog
	rows, err = c.operationLogRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	counts := make(map[int32]uint64)
	for _, row := range rows {
		counts[row.Action]++
	}
	items := make([]*adminv1.BaseDashboardDistributionItem, 0, len(counts))
	for action, count := range counts {
		items = append(items, &adminv1.BaseDashboardDistributionItem{
			Label: adminv1.BaseOperationAction(action).String(),
			Count: count,
		})
	}
	return &adminv1.BaseDashboardDistributionResponse{Items: items}, nil
}

// GetBaseDashboardLoginDistribution 查询登录结果分布。
func (c *BaseDashboardCase) GetBaseDashboardLoginDistribution(ctx context.Context) (*adminv1.BaseDashboardDistributionResponse, error) {
	query := c.loginLogRepo.Query(ctx).BaseLoginLog
	opts := []repository.QueryOption{repository.Select(query.Result)}
	var err error
	var rows []*models.BaseLoginLog
	rows, err = c.loginLogRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	counts := make(map[int32]uint64)
	for _, row := range rows {
		counts[row.Result]++
	}
	items := make([]*adminv1.BaseDashboardDistributionItem, 0, len(counts))
	for result, count := range counts {
		items = append(items, &adminv1.BaseDashboardDistributionItem{
			Label: adminv1.BaseLogResult(result).String(),
			Count: count,
		})
	}
	return &adminv1.BaseDashboardDistributionResponse{Items: items}, nil
}

// dashboardStartOfDay 返回指定时间所在自然日的起点。
func dashboardStartOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

// dashboardDayRange 返回指定时间所在自然日的左闭右开区间。
func dashboardDayRange(value time.Time) (time.Time, time.Time) {
	start := dashboardStartOfDay(value)
	return start, start.AddDate(0, 0, 1)
}
