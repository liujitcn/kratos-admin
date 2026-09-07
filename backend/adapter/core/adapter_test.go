package core

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	coredata "github.com/liujitcn/kratos-core/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestAdaptersShareTransactionContext 验证独立创建的仓储仍通过请求上下文复用同一事务连接。
func TestAdaptersShareTransactionContext(t *testing.T) {
	connector := &transactionConnector{}
	connection := sql.OpenDB(connector)
	t.Cleanup(func() {
		closeErr := connection.Close()
		if closeErr != nil {
			t.Error(closeErr)
		}
	})
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	databases := map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}}
	var api *APIStoreAdapter
	api, err = NewAPIStoreAdapter(databases)
	if err != nil {
		t.Fatal(err)
	}
	var permission *PermissionStoreAdapter
	permission, err = NewPermissionStoreAdapter(databases)
	if err != nil {
		t.Fatal(err)
	}
	var job *JobStoreAdapter
	job, err = NewJobStoreAdapter(databases)
	if err != nil {
		t.Fatal(err)
	}
	var log *LogStoreAdapter
	log, err = NewLogStoreAdapter(databases)
	if err != nil {
		t.Fatal(err)
	}
	var transaction coredata.Transaction
	transaction, err = NewTransaction(databases)
	if err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("回滚测试事务")
	err = transaction.Transaction(context.Background(), func(ctx context.Context) error {
		var err error
		_, err = api.ListForPolicy(ctx)
		if err != nil {
			return err
		}
		_, err = permission.ListTenants(ctx)
		if err != nil {
			return err
		}
		_, err = job.List(ctx)
		if err != nil {
			return err
		}
		_, err = log.ExistsAPI(ctx, 1)
		if err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) || connector.queries != 4 {
		t.Fatalf("适配器未共享事务: queries=%d, err=%v", connector.queries, err)
	}
}

// TestAdaptersRejectInvalidDatabases 验证各公开适配器构造入口均返回数据源初始化错误。
func TestAdaptersRejectInvalidDatabases(t *testing.T) {
	for name, databases := range map[string]map[string]*kitgorm.Client{
		"empty":           nil,
		"missing-default": {"other": {}},
		"nil-client":      {kitgorm.DefaultClientName: nil},
		"missing-db":      {kitgorm.DefaultClientName: {}},
	} {
		t.Run(name, func(t *testing.T) {
			api, err := NewAPIStoreAdapter(databases)
			if err == nil || api != nil {
				t.Fatal("API 适配器应拒绝无效数据源")
			}
			var permission *PermissionStoreAdapter
			permission, err = NewPermissionStoreAdapter(databases)
			if err == nil || permission != nil {
				t.Fatal("权限适配器应拒绝无效数据源")
			}
			var job *JobStoreAdapter
			job, err = NewJobStoreAdapter(databases)
			if err == nil || job != nil {
				t.Fatal("任务适配器应拒绝无效数据源")
			}
			var log *LogStoreAdapter
			log, err = NewLogStoreAdapter(databases)
			if err == nil || log != nil {
				t.Fatal("日志适配器应拒绝无效数据源")
			}
			var transaction coredata.Transaction
			transaction, err = NewTransaction(databases)
			if err == nil || transaction != nil {
				t.Fatal("事务接口应拒绝无效数据源")
			}
		})
	}
}

type transactionConnector struct {
	queries int
}

// Connect 为当前应用创建独立的测试连接。
func (c *transactionConnector) Connect(context.Context) (driver.Conn, error) {
	return &transactionConnection{connector: c}, nil
}

// Driver 返回连接器所属测试驱动。
func (*transactionConnector) Driver() driver.Driver { return transactionDriver{} }

type transactionDriver struct{}

// Open 禁止绕过连接器创建测试连接。
func (transactionDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("测试仅允许通过连接器创建连接")
}

type transactionConnection struct {
	connector     *transactionConnector
	inTransaction bool
}

// Prepare 禁止测试使用预处理语句。
func (*transactionConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("测试不支持预处理语句")
}

// Close 关闭无外部资源的测试连接。
func (*transactionConnection) Close() error { return nil }

// Begin 标记当前连接为事务连接。
func (c *transactionConnection) Begin() (driver.Tx, error) {
	c.inTransaction = true
	return transactionHandle{connection: c}, nil
}

// QueryContext 拒绝未使用事务连接的仓储查询。
func (c *transactionConnection) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if !c.inTransaction {
		return nil, errors.New("仓储查询未复用事务连接")
	}
	c.connector.queries++
	return emptyRows{}, nil
}

type transactionHandle struct {
	connection *transactionConnection
}

// Commit 结束测试事务。
func (tx transactionHandle) Commit() error {
	tx.connection.inTransaction = false
	return nil
}

// Rollback 回滚测试事务。
func (tx transactionHandle) Rollback() error {
	tx.connection.inTransaction = false
	return nil
}

type emptyRows struct{}

// Columns 返回仓储查询的最小结果列。
func (emptyRows) Columns() []string { return []string{"id"} }

// Close 关闭空结果集。
func (emptyRows) Close() error { return nil }

// Next 返回空结果集的结束标识。
func (emptyRows) Next([]driver.Value) error { return io.EOF }
