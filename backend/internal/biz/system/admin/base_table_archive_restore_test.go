package biz

import (
	"database/sql"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
)

// TestRestoreInternalArchiveWithAuthenticatedContext 验证登录用户可选择、全量及幂等恢复，且旁路不泄漏至原客户端。
func TestRestoreInternalArchiveWithAuthenticatedContext(t *testing.T) {
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
	name := "e2e_restore_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	archive := "base_table_archive_" + name
	_, err = sqlDB.ExecContext(t.Context(), "CREATE TABLE `"+name+"` (id BIGINT PRIMARY KEY, title VARCHAR(64))")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, cleanupErr := sqlDB.Exec("DROP TABLE IF EXISTS `" + archive + "`, `" + name + "`")
		if cleanupErr != nil {
			t.Error(cleanupErr)
		}
	})
	_, err = sqlDB.ExecContext(t.Context(), "CREATE TABLE `"+archive+"` LIKE `"+name+"`")
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(t.Context(), "INSERT INTO `"+archive+"` VALUES (1,'归档一'),(2,'归档二')")
	if err != nil {
		t.Fatal(err)
	}
	identity := &data.UserTokenPayload{UserId: 1, TenantId: 1, TenantCode: "0000", RoleCode: "super"}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	baseCase := &biz.BaseCase{GormClients: map[string]*gorm.Client{"default": client}}
	record := &models.BaseTableArchiveRecord{SourceName: "default", TableName_: name, ArchiveTableName: archive}
	var count int64
	count, err = restoreInternalArchive(ctx, baseCase, record, adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_SELECTED, `{"start_id":1,"end_id":1}`)
	if err != nil || count != 1 {
		t.Fatalf("选择性恢复 = %d, err = %v", count, err)
	}
	count, err = restoreInternalArchive(ctx, baseCase, record, adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL, "")
	if err != nil || count != 1 {
		t.Fatalf("全量补齐恢复 = %d, err = %v", count, err)
	}
	count, err = restoreInternalArchive(ctx, baseCase, record, adminv1.BaseTableArchiveRestoreMode_BASE_TABLE_ARCHIVE_RESTORE_MODE_ALL, "")
	if err != nil || count != 0 {
		t.Fatalf("重复恢复 = %d, err = %v", count, err)
	}
	if err = client.DB.WithContext(ctx).Exec("SELECT 1").Error; !errors.Is(err, gorm.ErrRawDataIsolationUnsupported) {
		t.Fatalf("原客户端仍应拒绝无旁路的 SQL，实际 = %v", err)
	}
}
