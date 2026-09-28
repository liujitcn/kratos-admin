package biz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestValidateRouteChangeKeepsActiveDeliveriesBoundToTheirTemplate 验证派发和重试期间分类不能删除当前路由。
func TestValidateRouteChangeKeepsActiveDeliveriesBoundToTheirTemplate(t *testing.T) {
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
	if err = db.AutoMigrate(&models.BaseMessage{}, &models.BaseMessageDelivery{}); err != nil {
		t.Fatal(err)
	}
	store, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	messages := []*models.BaseMessage{
		{ID: 1, TenantID: 1, CategoryID: 2, Source: "test", IdempotencyKey: "dispatching", ActionParams: "{}", Status: int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING), CreatedAt: now, UpdatedAt: now},
		{ID: 2, TenantID: 1, CategoryID: 2, Source: "test", IdempotencyKey: "published", ActionParams: "{}", Status: int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED), CreatedAt: now, UpdatedAt: now},
	}
	if err = db.Create(messages).Error; err != nil {
		t.Fatal(err)
	}
	delivery := &models.BaseMessageDelivery{
		ID: 1, TenantID: 1, MessageID: messages[1].ID, DeliveryType: _const.MessageDeliveryTypeProvider,
		ProviderID: 7, UserID: 20, Status: _const.MessageDeliveryStatusPending,
		ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err = db.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}
	messageRepo := data.NewBaseMessageRepository(store)
	deliveryRepo := data.NewBaseMessageDeliveryRepository(store)
	categoryCase := &BaseMessageCategoryCase{baseMessageRepo: messageRepo, deliveryRepo: deliveryRepo}
	ctx := context.Background()
	if err = categoryCase.validateRouteChange(ctx, 2, []int64{7}); err == nil {
		t.Fatal("route change passed while a dispatch was in progress")
	}
	messageQuery := store.Query(ctx).BaseMessage
	_, err = messageQuery.WithContext(ctx).
		Where(messageQuery.ID.Eq(messages[0].ID)).
		UpdateSimple(messageQuery.Status.Value(int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED)))
	if err != nil {
		t.Fatal(err)
	}
	if err = categoryCase.validateRouteChange(ctx, 2, []int64{7}); err == nil {
		t.Fatal("route change passed while a provider delivery was pending")
	}
	deliveryQuery := store.Query(ctx).BaseMessageDelivery
	_, err = deliveryQuery.WithContext(ctx).
		Where(deliveryQuery.ID.Eq(delivery.ID)).
		UpdateSimple(deliveryQuery.Status.Value(_const.MessageDeliveryStatusSucceeded))
	if err != nil {
		t.Fatal(err)
	}
	if err = categoryCase.validateRouteChange(ctx, 2, []int64{7}); err != nil {
		t.Fatalf("route change rejected after provider delivery completed: %v", err)
	}
}
