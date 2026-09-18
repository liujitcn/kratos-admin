package data

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMessageDeliveryWriterSupportsMixedMessages 验证混合消息批次和软删除记录均能正确幂等写入。
func TestMessageDeliveryWriterSupportsMixedMessages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseMessageDelivery{}); err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	writer := NewMessageDeliveryWriter(store)
	ctx := context.Background()
	now := time.Now()
	items := []*models.BaseMessageDelivery{
		{ID: 1, TenantID: 1, MessageID: 11, UserID: 101, ReceivedAt: now},
		{ID: 2, TenantID: 1, MessageID: 12, UserID: 101, ReceivedAt: now},
		{ID: 3, TenantID: 1, MessageID: 12, UserID: 102, ReceivedAt: now},
	}
	created, err := writer.CreateIgnore(ctx, items, 10)
	if err != nil || created != 3 {
		t.Fatalf("混合消息批次写入错误: created=%d err=%v", created, err)
	}
	deleted := &models.BaseMessageDelivery{TenantID: 1, MessageID: 12, UserID: 102}
	query := store.Query(ctx).BaseMessageDelivery
	var existing *models.BaseMessageDelivery
	existing, err = query.WithContext(ctx).Where(
		query.TenantID.Eq(deleted.TenantID),
		query.MessageID.Eq(deleted.MessageID),
		query.UserID.Eq(deleted.UserID),
	).First()
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Delete(existing).Error; err != nil {
		t.Fatal(err)
	}
	created, err = writer.CreateIgnore(ctx, []*models.BaseMessageDelivery{{TenantID: 1, MessageID: 12, UserID: 102, ReceivedAt: now}}, 10)
	if err != nil || created != 1 {
		t.Fatalf("软删除投递记录重建错误: created=%d err=%v", created, err)
	}
}
