package biz

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbconn"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbmeta"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

var databaseTableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// DatabaseCase 提供管理端连接探测和物理元数据。
type DatabaseCase struct {
	*biz.BaseCase
	biSourceCase *admin.BiSourceCase
}

// NewDatabaseCase 创建数据源元数据业务实例。
func NewDatabaseCase(baseCase *biz.BaseCase, biSourceCase *admin.BiSourceCase) *DatabaseCase {
	return &DatabaseCase{BaseCase: baseCase, biSourceCase: biSourceCase}
}

// OptionBaseTableSource 查询已初始化的数据源名称。
func (c *DatabaseCase) OptionBaseTableSource(context.Context) (*commonv1.StringValues, error) {
	names := make([]string, 0, len(c.GormClients))
	for name, client := range c.GormClients {
		if name == "" || client == nil || client.DB == nil {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return &commonv1.StringValues{Value: names}, nil
}

// OptionBaseTable 查询指定数据源中的数据库表选项。
func (c *DatabaseCase) OptionBaseTable(ctx context.Context, req *adminv1.OptionBaseTableRequest) (*adminv1.OptionBaseTableResponse, error) {
	client, err := GormClientBySourceName(c.BaseCase, req.GetSourceName())
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	var tables []dbmeta.Table
	tables, err = listDatabaseTableMetadata(ctx, client, nil)
	if err != nil {
		return nil, fmt.Errorf("查询数据源 %s 的数据库表失败: %w", req.GetSourceName(), err)
	}
	options := make([]*adminv1.BaseTableOption, 0, len(tables))
	for _, table := range tables {
		options = append(options, &adminv1.BaseTableOption{Name: table.TableName, Comment: table.TableComment})
	}
	return &adminv1.OptionBaseTableResponse{Tables: options}, nil
}

// ListTableColumns 查询指定数据库表的字段元数据。
func (c *DatabaseCase) ListTableColumns(ctx context.Context, req *adminv1.ListTableColumnsRequest) (*adminv1.ListTableColumnsResponse, error) {
	databaseColumns, err := listDatabaseColumnMetadata(ctx, c.BaseCase, req.GetSourceName(), req.GetTableName())
	if err != nil {
		return nil, err
	}
	columns := make([]*adminv1.DatabaseColumn, 0, len(databaseColumns))
	for _, item := range databaseColumns {
		columnComment := item.Comment
		// 数据库未配置字段注释时回退显示字段名。
		if columnComment == "" {
			columnComment = item.Name
		}
		columnType := item.ColumnType
		// 数据库未返回完整类型时回退到基础数据类型。
		if columnType == "" {
			columnType = item.DataType
		}
		columns = append(columns, &adminv1.DatabaseColumn{
			Name:       item.Name,
			Comment:    columnComment,
			DbType:     item.DataType,
			ColumnType: columnType,
			IsPrimary:  item.ColumnKey == "PRI",
			IsNullable: item.IsNullable == "YES",
		})
	}
	return &adminv1.ListTableColumnsResponse{Columns: columns}, nil
}

// TestConnection 测试当前账号获准使用的宿主连接或表单中的独立连接。
func (c *DatabaseCase) TestConnection(ctx context.Context, req *adminv1.TestConnectionRequest) (*adminv1.TestConnectionResponse, error) {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var pool *sql.DB
	if req.GetHost() == "" {
		if identity.TenantCode != gorm.DefaultTenantCode {
			return nil, errorsx.PermissionDenied("只有默认租户可以测试服务端数据库客户端")
		}
		var client *gorm.Client
		client, err = GormClientBySourceName(c.BaseCase, req.GetSourceName())
		if err != nil {
			return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
		}
		pool, err = client.DB.DB()
		if err != nil {
			return nil, errorsx.Internal("获取数据库连接池失败").WithCause(err)
		}
	} else {
		config := dbconn.Config{Driver: req.GetDriver(), Host: req.GetHost(), Port: req.GetPort(), DatabaseName: req.GetDatabaseName(), Username: req.GetUsername(), SSLMode: req.GetSslMode()}
		password := req.GetPassword().GetText()
		if password == "" {
			if req.GetBiSourceId() <= 0 {
				return nil, errorsx.InvalidArgument("独立数据库连接的密码不能为空")
			}
			password, err = c.biSourceCase.ConnectionTestPassword(ctx, req.GetBiSourceId(), config)
			if err != nil {
				return nil, err
			}
		}
		pool, err = dbconn.Open(config, password)
		if err != nil {
			return nil, errorsx.InvalidArgument("无法打开数据库，请检查连接参数")
		}
		defer func() {
			if closeErr := pool.Close(); closeErr != nil {
				log.Warn(fmt.Sprintf("关闭数据库测试连接失败: %v", closeErr))
			}
		}()
	}
	if err = dbconn.Ping(ctx, pool); err != nil {
		return &adminv1.TestConnectionResponse{Success: false, Message: "数据库连接失败，请检查连接配置和数据库状态"}, nil
	}
	return &adminv1.TestConnectionResponse{Success: true, Message: "数据源连接测试通过"}, nil
}

// listDatabaseTableMetadata 查询指定客户端当前库中的实体表。
func listDatabaseTableMetadata(ctx context.Context, database *gorm.Client, tableNames []string) ([]dbmeta.Table, error) {
	return dbmeta.ListTables(ctx, database.DB, database.Driver(), dbmeta.TableScope{Names: tableNames})
}

// listDatabaseColumnMetadata 校验物理表名并读取完整字段元数据。
func listDatabaseColumnMetadata(ctx context.Context, baseCase *biz.BaseCase, sourceName, tableName string) ([]dbmeta.Column, error) {
	if tableName == "" {
		return nil, errorsx.InvalidArgument("数据库表名不能为空")
	}
	if !databaseTableNamePattern.MatchString(tableName) {
		return nil, errorsx.InvalidArgument("数据库表名格式不正确")
	}
	database, err := GormClientBySourceName(baseCase, sourceName)
	if err != nil {
		return nil, err
	}
	return dbmeta.ListColumns(ctx, database.DB, database.Driver(), "", []string{tableName}, true)
}

// GormClientBySourceName 按数据源名称获取已初始化的 GORM 客户端。
func GormClientBySourceName(baseCase *biz.BaseCase, sourceName string) (*gorm.Client, error) {
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
