package admin

import (
	"database/sql"
	"errors"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
)

// TestArchiveWithAuthenticatedContext 验证手工调度归档时允许受控 SQL，且不污染原客户端的数据隔离。
func TestArchiveWithAuthenticatedContext(t *testing.T) {
	dsn := os.Getenv("KRATOS_ARCHIVE_TEST_DSN")
	if dsn == "" {
		t.Skip("设置 KRATOS_ARCHIVE_TEST_DSN 指向独立 MySQL 测试库后运行")
	}
	client, cleanup, err := gorm.NewGormClient(&configv1.Data_Database{Driver: "mysql", Source: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	var sqlDB *sql.DB
	sqlDB, err = client.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	name := "e2e_archive_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	resource := archiveResourceDefinition{tableName: name, timeColumn: "created_at", archiveTableName: "base_table_archive_" + name}
	_, err = sqlDB.ExecContext(t.Context(), "CREATE TABLE `"+name+"` (id BIGINT PRIMARY KEY, created_at DATETIME NOT NULL)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, cleanupErr := sqlDB.Exec("DROP TABLE IF EXISTS `" + resource.archiveTableName + "`, `" + name + "`")
		if cleanupErr != nil {
			t.Error(cleanupErr)
		}
	})
	_, err = sqlDB.ExecContext(t.Context(), "INSERT INTO `"+name+"` VALUES (1,'2000-01-01'),(2,'2000-01-01'),(3,'2099-01-01')")
	if err != nil {
		t.Fatal(err)
	}
	identity := &authdata.UserTokenPayload{UserId: 1, TenantId: 1, TenantCode: "0000", RoleCode: "super"}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	var archived, deleted int64
	archived, deleted, err = archiveIntoCurrentDatabase(ctx, client, resource, time.Now(), 1, false)
	if err != nil || archived != 2 || deleted != 0 {
		t.Fatalf("保留在线数据归档 = %d/%d, err = %v", archived, deleted, err)
	}
	archived, deleted, err = archiveIntoCurrentDatabase(ctx, client, resource, time.Now(), 1, true)
	if err != nil || archived != 2 || deleted != 2 {
		t.Fatalf("校验后删除归档 = %d/%d, err = %v", archived, deleted, err)
	}
	var ids []int64
	ids, err = listArchiveIDs(ctx, client, resource, time.Now().AddDate(100, 0, 0), 10, 0)
	if err != nil || !slices.Equal(ids, []int64{3}) {
		t.Fatalf("剩余在线主键 = %v, err = %v", ids, err)
	}
	var count int64
	count, err = countArchiveRows(ctx, client, resource, time.Now())
	if err != nil || count != 0 {
		t.Fatalf("剩余历史行数 = %d, err = %v", count, err)
	}
	if err = client.DB.WithContext(ctx).Exec("SELECT 1").Error; !errors.Is(err, gorm.ErrRawDataIsolationUnsupported) {
		t.Fatalf("原客户端仍应拒绝无旁路的 SQL，实际 = %v", err)
	}
}
