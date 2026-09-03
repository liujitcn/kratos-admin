package biz

import (
	"context"
	"fmt"
	"sort"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	corebiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseTableSourceCase 提供数据备份、数据归档和代码生成共用的数据源元数据。
type BaseTableSourceCase struct {
	*corebiz.BaseCase
}

// NewBaseTableSourceCase 创建数据源元数据业务实例。
func NewBaseTableSourceCase(baseCase *corebiz.BaseCase) *BaseTableSourceCase {
	return &BaseTableSourceCase{BaseCase: baseCase}
}

// OptionBaseTableSource 查询已初始化的数据源名称。
func (c *BaseTableSourceCase) OptionBaseTableSource(context.Context) (*commonv1.StringValues, error) {
	names := make([]string, 0, len(c.GormClients))
	for name, client := range c.GormClients {
		if name == "" || client == nil || client.DB == nil {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return &commonv1.StringValues{Value: names}, nil
}

// OptionBaseTable 查询指定数据源中的数据库表名称。
func (c *BaseTableSourceCase) OptionBaseTable(ctx context.Context, req *adminv1.OptionBaseTableRequest) (*commonv1.StringValues, error) {
	client, err := GormClientBySourceName(c.BaseCase, req.GetSourceName())
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	var tables []string
	tables, err = client.Migrator().GetTables()
	if err != nil {
		return nil, fmt.Errorf("查询数据源 %s 的数据库表失败: %w", req.GetSourceName(), err)
	}
	sort.Strings(tables)
	return &commonv1.StringValues{Value: tables}, nil
}

// GormClientBySourceName 按数据源名称获取已初始化的 GORM 客户端。
func GormClientBySourceName(baseCase *corebiz.BaseCase, sourceName string) (*gorm.Client, error) {
	if sourceName == "" {
		return nil, fmt.Errorf("数据源名称不能为空")
	}
	client := baseCase.GormClients[sourceName]
	if client == nil || client.DB == nil {
		return nil, fmt.Errorf("数据源 %s 未初始化", sourceName)
	}
	return client, nil
}

// sourceTableKey 返回数据源和数据表组成的稳定索引键。
func sourceTableKey(sourceName, tableName string) string {
	return sourceName + "\x00" + tableName
}
