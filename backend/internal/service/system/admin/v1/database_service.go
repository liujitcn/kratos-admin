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

// DatabaseService 提供数据源和数据库表元数据查询服务。
type DatabaseService struct {
	adminv1.UnimplementedDatabaseServiceServer
	databaseCase *biz.DatabaseCase
}

// NewDatabaseService 创建数据源元数据查询服务。
func NewDatabaseService(databaseCase *biz.DatabaseCase) *DatabaseService {
	return &DatabaseService{databaseCase: databaseCase}
}

// OptionDatabase 查询已初始化的数据源名称。
func (s *DatabaseService) OptionBaseTableSource(ctx context.Context, req *adminv1.OptionBaseTableSourceRequest) (*commonv1.StringValues, error) {
	result, err := s.databaseCase.OptionBaseTableSource(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("OptionDatabase %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据源失败")
	}
	return result, nil
}

// OptionBaseTable 查询指定数据源中的数据库表选项。
func (s *DatabaseService) OptionBaseTable(ctx context.Context, req *adminv1.OptionBaseTableRequest) (*adminv1.OptionBaseTableResponse, error) {
	result, err := s.databaseCase.OptionBaseTable(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("OptionBaseTable %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库表失败")
	}
	return result, nil
}

// ListTableColumns 查询数据库字段。
func (s *DatabaseService) ListTableColumns(ctx context.Context, req *adminv1.ListTableColumnsRequest) (*adminv1.ListTableColumnsResponse, error) {
	res, err := s.databaseCase.ListTableColumns(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("ListTableColumns %v", err))
		return nil, errorsx.WrapInternal(err, "查询数据库字段失败")
	}
	return res, nil
}

// TestConnection 测试数据库连接。
func (s *DatabaseService) TestConnection(ctx context.Context, req *adminv1.TestConnectionRequest) (*adminv1.TestConnectionResponse, error) {
	res, err := s.databaseCase.TestConnection(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TestConnection %v", err))
		return nil, errorsx.WrapInternal(err, "测试数据库连接失败")
	}
	return res, nil
}
