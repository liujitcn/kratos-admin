package biz

import (
	"strings"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestClaimDispatchUpdatesTimestampOnce 验证抢占投递任务时 PostgreSQL 更新语句只设置一次更新时间。
func TestClaimDispatchUpdatesTimestampOnce(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 port=1"}), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var updateSQL string
	err = db.Callback().Update().After("gorm:update").Register("test:capture_dispatch_update", func(tx *gorm.DB) {
		updateSQL = tx.Statement.SQL.String()
	})
	if err != nil {
		t.Fatal(err)
	}
	var database *data.Data
	database, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	messageCase := &BaseMessageCase{dispatchRepo: data.NewBaseMessageDispatchRepository(database)}
	_, err = messageCase.claimDispatch(t.Context(), &models.BaseMessageDispatch{ID: 1, Version: 1}, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	set, _, _ := strings.Cut(updateSQL, " WHERE ")
	if count := strings.Count(set, `"updated_at"=`); count != 1 {
		t.Fatalf("抢占更新包含 %d 次 updated_at 赋值，期望 1 次: %s", count, updateSQL)
	}
}
