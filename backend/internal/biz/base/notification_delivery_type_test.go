package biz

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

// TestNotificationVisibleOptionsExcludesProviderDeliveries 验证 Provider 投递不会进入站内收件箱。
func TestNotificationVisibleOptionsExcludesProviderDeliveries(t *testing.T) {
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
	now := time.Now()
	if err = db.Create([]*models.BaseMessageDelivery{
		{ID: 1, TenantID: 1, MessageID: 10, DeliveryType: _const.MessageDeliveryTypeInbox, Status: _const.MessageDeliveryStatusSucceeded, UserID: 20, ReceivedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: 2, TenantID: 1, MessageID: 10, DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: 7, Status: _const.MessageDeliveryStatusPending, UserID: 20, ReceivedAt: now, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	notificationCase := &NotificationCase{BaseMessageDeliveryRepository: data.NewBaseMessageDeliveryRepository(store)}
	list, err := notificationCase.List(context.Background(), notificationCase.visibleOptions(context.Background(), 20)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].DeliveryType != _const.MessageDeliveryTypeInbox {
		t.Fatalf("visible inbox deliveries = %+v, want one inbox record only", list)
	}
}
