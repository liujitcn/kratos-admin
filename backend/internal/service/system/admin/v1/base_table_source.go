package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BaseTableSourceService 提供数据源和数据库表元数据查询服务。
type BaseTableSourceService struct {
	adminv1.UnimplementedBaseTableSourceServiceServer
	baseTableSourceCase *biz.BaseTableSourceCase
}

// NewBaseTableSourceService 创建数据源元数据查询服务。
func NewBaseTableSourceService(baseTableSourceCase *biz.BaseTableSourceCase) *BaseTableSourceService {
	return &BaseTableSourceService{baseTableSourceCase: baseTableSourceCase}
}

// OptionBaseTableSource 查询已初始化的数据源名称。
func (s *BaseTableSourceService) OptionBaseTableSource(ctx context.Context, req *adminv1.OptionBaseTableSourceRequest) (*commonv1.StringValues, error) {
	result, err := s.baseTableSourceCase.OptionBaseTableSource(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTableSource %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据源失败")
	}
	return result, nil
}

// OptionBaseTable 查询指定数据源中的数据库表名称。
func (s *BaseTableSourceService) OptionBaseTable(ctx context.Context, req *adminv1.OptionBaseTableRequest) (*commonv1.StringValues, error) {
	result, err := s.baseTableSourceCase.OptionBaseTable(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表失败")
	}
	return result, nil
}
