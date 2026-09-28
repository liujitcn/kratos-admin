package data

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
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
	query := store.Query(ctx).BaseMessageDelivery
	items := []*models.BaseMessageDelivery{
		{ID: 1, TenantID: 1, MessageID: 11, DeliveryType: _const.MessageDeliveryTypeInbox, Status: _const.MessageDeliveryStatusSucceeded, UserID: 101, ReceivedAt: now},
		{ID: 2, TenantID: 1, MessageID: 12, DeliveryType: _const.MessageDeliveryTypeInbox, Status: _const.MessageDeliveryStatusSucceeded, UserID: 101, ReceivedAt: now},
		{ID: 3, TenantID: 1, MessageID: 12, DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: 7, Status: _const.MessageDeliveryStatusPending, UserID: 101, ReceivedAt: now},
		{ID: 4, TenantID: 1, MessageID: 12, DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: 8, Status: _const.MessageDeliveryStatusPending, UserID: 101, ReceivedAt: now},
	}
	created, err := writer.CreateIgnore(ctx, items, 10)
	if err != nil || created != 4 {
		t.Fatalf("合并投递批次写入错误: created=%d err=%v", created, err)
	}
	created, err = writer.CreateIgnore(ctx, items, 10)
	if err != nil || created != 0 {
		t.Fatalf("重复投递写入未幂等: created=%d err=%v", created, err)
	}
	readAt := time.Now()
	if err = writer.SetReadAt(ctx, 101, []int64{3}, &readAt); err != nil {
		t.Fatal(err)
	}
	var providerDelivery *models.BaseMessageDelivery
	providerDelivery, err = query.WithContext(ctx).Where(
		query.DeliveryType.Eq(_const.MessageDeliveryTypeProvider),
		query.ProviderID.Eq(7),
		query.MessageID.Eq(12),
		query.UserID.Eq(101),
	).First()
	if err != nil {
		t.Fatal(err)
	}
	if providerDelivery.ReadAt != 0 {
		t.Fatalf("站内信已读操作修改了 Provider 投递记录: read_at=%d", providerDelivery.ReadAt)
	}
	if err = writer.SetReadAt(ctx, 101, []int64{2}, &readAt); err != nil {
		t.Fatal(err)
	}
	var inboxDelivery *models.BaseMessageDelivery
	inboxDelivery, err = query.WithContext(ctx).Where(
		query.DeliveryType.Eq(_const.MessageDeliveryTypeInbox),
		query.MessageID.Eq(12),
		query.UserID.Eq(101),
	).First()
	if err != nil {
		t.Fatal(err)
	}
	if inboxDelivery.ReadAt != readAt.UnixMilli() {
		t.Fatalf("站内信已读时间未更新: read_at=%d", inboxDelivery.ReadAt)
	}
	deleted := &models.BaseMessageDelivery{TenantID: 1, MessageID: 12, DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: 8, UserID: 101}
	var existing *models.BaseMessageDelivery
	existing, err = query.WithContext(ctx).Where(
		query.TenantID.Eq(deleted.TenantID),
		query.MessageID.Eq(deleted.MessageID),
		query.DeliveryType.Eq(deleted.DeliveryType),
		query.ProviderID.Eq(deleted.ProviderID),
		query.UserID.Eq(deleted.UserID),
	).First()
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Delete(existing).Error; err != nil {
		t.Fatal(err)
	}
	created, err = writer.CreateIgnore(ctx, []*models.BaseMessageDelivery{{TenantID: 1, MessageID: 12, DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: 8, Status: _const.MessageDeliveryStatusPending, UserID: 101, ReceivedAt: now}}, 10)
	if err != nil || created != 1 {
		t.Fatalf("软删除投递记录重建错误: created=%d err=%v", created, err)
	}
}
