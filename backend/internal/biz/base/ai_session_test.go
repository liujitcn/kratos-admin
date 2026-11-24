package biz

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestAiSessionUpdatedAtPostgres 验证 PostgreSQL 会话更新只写入一次框架生成的更新时间。
func TestAiSessionUpdatedAtPostgres(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=test dbname=test sslmode=disable"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true,
		NowFunc: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	var updateSQL string
	var updateVars []any
	err = db.Callback().Update().After("gorm:update").Register("ai_session_test:update", func(tx *gorm.DB) {
		updateSQL = tx.Statement.SQL.String()
		updateVars = tx.Statement.Vars
	})
	if err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*databasegorm.Client{databasegorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	c := &AiSessionCase{AiSessionRepository: data.NewAiSessionRepository(store)}
	session := &models.AiSession{ID: 2, TenantID: 1}
	if err = c.UpdateSessionSummary(t.Context(), session, "你好"); err != nil {
		t.Fatal(err)
	}
	set, _, _ := strings.Cut(updateSQL, " WHERE ")
	if count := strings.Count(set, `"updated_at"=`); count != 1 {
		t.Fatalf("updated_at 赋值次数为 %d，期望 1: %s", count, updateSQL)
	}
	if len(updateVars) < 2 || updateVars[1] != now || session.Summary != "你好" || !session.UpdatedAt.Equal(now) {
		t.Fatalf("更新后的会话摘要或时间戳不匹配: 参数 %v，会话摘要 %q，会话更新时间 %v", updateVars, session.Summary, session.UpdatedAt)
	}
	now = now.Add(time.Second)
	if err = c.RefreshSessionUpdatedAt(t.Context(), session); err != nil {
		t.Fatal(err)
	}
	set, _, _ = strings.Cut(updateSQL, " WHERE ")
	if set != `UPDATE "ai_session" SET "updated_at"=$1` || !session.UpdatedAt.Equal(now) {
		t.Fatalf("会话排序时间未按预期刷新: %s, %v", updateSQL, session.UpdatedAt)
	}
}
