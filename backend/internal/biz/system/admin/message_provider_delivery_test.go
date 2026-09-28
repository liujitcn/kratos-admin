package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data"
	gendata "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/notify"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBuildMessageProviderPayloadRendersSmsParams 验证短信模板使用显式顺序渲染参数。
func TestBuildMessageProviderPayloadRendersSmsParams(t *testing.T) {
	payload, err := buildMessageProviderPayload(
		&models.BaseMessage{ID: 10, CategoryID: 3, Title: "Notice", Content: "Body", ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT)},
		&models.BaseMessageTemplate{
			Title:         "Hello {{.UserName}}",
			Content:       "{{.Content}}",
			ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT),
			Config:        `{"template_code":"SMS_100","ordered_params":["{{.UserName}}","{{.Title}}"]}`,
		},
		&models.BaseMessageProvider{ID: 2, Provider: "sms"},
		&models.BaseUser{UserName: "user-a", Phone: "+15555550100"},
		notify.Recipient{Phone: "+15555550100"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Title != "Hello user-a" || payload.Content != "Body" {
		t.Fatalf("unexpected rendered message: %+v", payload)
	}
	if payload.Template == nil || payload.Template.Code != "SMS_100" {
		t.Fatalf("unexpected provider template: %+v", payload.Template)
	}
	if len(payload.Template.OrderedParams) != 2 || payload.Template.OrderedParams[0] != "user-a" || payload.Template.OrderedParams[1] != "Notice" {
		t.Fatalf("unexpected ordered params: %#v", payload.Template.OrderedParams)
	}
	if len(payload.Recipients) != 1 || payload.Recipients[0].Phone != "+15555550100" {
		t.Fatalf("unexpected recipients: %#v", payload.Recipients)
	}
}

// TestIsBroadcastMessageProvider 验证 Webhook 和群机器人使用单条广播投递。
func TestIsBroadcastMessageProvider(t *testing.T) {
	if !isBroadcastMessageProvider(&models.BaseMessageProvider{Provider: "webhook"}) {
		t.Fatal("webhook should be treated as a broadcast provider")
	}
	if !isBroadcastMessageProvider(&models.BaseMessageProvider{Provider: "dingtalk", Config: `{"implementation":"group"}`}) {
		t.Fatal("group bot should be treated as a broadcast provider")
	}
	if isBroadcastMessageProvider(&models.BaseMessageProvider{Provider: "dingtalk", Config: `{"implementation":"app"}`}) {
		t.Fatal("application provider should retain per-user deliveries")
	}
}

// TestProcessDispatchUsesCurrentCategoryAndUnifiedDeliveries 验证派发读取当前分类并将两类投递写入同一张表。
func TestProcessDispatchUsesCurrentCategoryAndUnifiedDeliveries(t *testing.T) {
	type webhookPayload struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	received := make(chan webhookPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload webhookPayload
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode webhook payload: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- payload
		writer.Header().Set("X-Message-ID", "provider-receipt")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

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
	if err = db.AutoMigrate(
		&models.BaseMessage{}, &models.BaseMessageDispatch{}, &models.BaseMessageDelivery{},
		&models.BaseMessageCategory{}, &models.BaseMessageProvider{},
		&models.BaseMessageTemplate{}, &models.BaseUser{}, &models.BaseThirdAccount{},
	); err != nil {
		t.Fatal(err)
	}
	store, err := gendata.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	user := &models.BaseUser{
		ID: 20, TenantID: 1, UserName: "user-a", UserCode: "user-a", RoleID: 1, DeptID: 1,
		PasswordChangedAt: now, PasswordHistory: "[]", MustChangePassword: 0, Gender: 0, Status: 1,
		CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	message := &models.BaseMessage{
		ID: 10, TenantID: 1, CategoryID: 1, SourceType: int32(basev1.MessageSourceType_MESSAGE_SOURCE_TYPE_ADMIN),
		Source: "admin", IdempotencyKey: "message-10", SenderType: int32(basev1.MessageSenderType_MESSAGE_SENDER_TYPE_USER),
		SenderID: 1, SenderName: "Admin", Priority: int32(basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL),
		Title: "Notice", Content: "Body", ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT),
		ActionParams: "{}", ExpiresAt: now.Add(time.Hour).UnixMilli(), Version: 1,
		Status: int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING), CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	category := &models.BaseMessageCategory{
		ID: 1, Code: "EXTERNAL", Name: "External", Sort: 1, DefaultPriority: int32(basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL),
		RetentionDays: 180, ProviderID: "[8]", InboxEnabled: int32(adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_DISABLE),
		Status: 1, CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	provider := &models.BaseMessageProvider{
		ID: 7, Provider: "webhook", Name: "test-webhook", ClientID: server.URL,
		Config: `{"allow_http":true,"allow_private_network":true}`, Status: 1,
		CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	messageTemplate := &models.BaseMessageTemplate{
		ID: 1, CategoryID: category.ID, ProviderID: provider.ID,
		Title: "{{.Title}}", Content: "{{.Content}}", ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT),
		Config: "{}", CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	dispatch := &models.BaseMessageDispatch{
		ID: 30, TenantID: 1, MessageID: message.ID,
		AudienceType: int32(basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER), AudienceID: user.ID,
		BatchSize: 500, Version: 1, Status: int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING),
		CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err = db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(message).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(category).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(provider).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(messageTemplate).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(dispatch).Error; err != nil {
		t.Fatal(err)
	}
	tx := gendata.NewTransaction(store)
	messageRepo := gendata.NewBaseMessageRepository(store)
	categoryRepo := gendata.NewBaseMessageCategoryRepository(store)
	providerRepo := gendata.NewBaseMessageProviderRepository(store)
	templateRepo := gendata.NewBaseMessageTemplateRepository(store)
	categoryCase := NewBaseMessageCategoryCase(&biz.BaseCase{}, tx, categoryRepo, messageRepo, gendata.NewBaseMessageDeliveryRepository(store), providerRepo, templateRepo)
	messageCase := NewBaseMessageCase(
		&biz.BaseCase{}, tx, messageRepo,
		gendata.NewBaseMessageDispatchRepository(store), gendata.NewBaseMessageDeliveryRepository(store), data.NewMessageDeliveryWriter(store),
		categoryCase,
		gendata.NewBaseUserRepository(store), gendata.NewBaseThirdAccountRepository(store), nil, nil, nil, nil, nil,
	)
	if err = db.Model(category).Update("ProviderID", "[7]").Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Model(category).Update("InboxEnabled", int32(adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE)).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Model(messageTemplate).Update("Content", "Changed body").Error; err != nil {
		t.Fatal(err)
	}
	err = messageCase.ProcessDispatch(context.Background(), &dto.MessageDispatchTask{DispatchID: dispatch.ID, TenantID: dispatch.TenantID, ExpectedVersion: dispatch.Version})
	if err != nil {
		t.Fatal(err)
	}
	currentDispatch, err := gendata.NewBaseMessageDispatchRepository(store).FindByID(context.Background(), dispatch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if currentDispatch.Status != int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_SUCCEEDED) {
		t.Fatalf("dispatch status = %d version=%d lock=%q, want succeeded", currentDispatch.Status, currentDispatch.Version, currentDispatch.LockToken)
	}
	finishedMessage, err := messageRepo.FindByID(context.Background(), message.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finishedMessage.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED) {
		t.Fatalf("message status = %d, want published", finishedMessage.Status)
	}
	deliveryQuery := store.Query(context.Background()).BaseMessageDelivery
	deliveries, err := gendata.NewBaseMessageDeliveryRepository(store).List(context.Background(),
		repository.Where(deliveryQuery.MessageID.Eq(message.ID)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if finishedMessage.RecipientTotal != 2 || finishedMessage.DeliveredTotal != 2 || finishedMessage.FailedTotal != 0 {
		t.Fatalf("message totals = recipients:%d delivered:%d failed:%d, want 2/2/0", finishedMessage.RecipientTotal, finishedMessage.DeliveredTotal, finishedMessage.FailedTotal)
	}
	if len(deliveries) != 2 {
		t.Fatalf("dispatch created %d delivery records, want inbox + provider", len(deliveries))
	}
	inboxCount := 0
	providerCount := 0
	for _, delivery := range deliveries {
		if delivery.DeliveryType == _const.MessageDeliveryTypeInbox {
			inboxCount++
			if delivery.UserID != user.ID || delivery.Status != _const.MessageDeliveryStatusSucceeded {
				t.Fatalf("unexpected inbox delivery: %+v", delivery)
			}
		} else if delivery.DeliveryType == _const.MessageDeliveryTypeProvider {
			providerCount++
			if delivery.ProviderID != provider.ID || delivery.Status != _const.MessageDeliveryStatusSucceeded || delivery.ReceiptID != "provider-receipt" {
				t.Fatalf("dispatch did not persist successful Provider delivery: %+v", delivery)
			}
		}
	}
	if inboxCount != 1 || providerCount != 1 {
		t.Fatalf("delivery types = inbox:%d provider:%d, want 1/1", inboxCount, providerCount)
	}
	select {
	case payload := <-received:
		if payload.Title != "Notice" || payload.Content != "Changed body" {
			t.Fatalf("unexpected webhook payload: %+v", payload)
		}
	default:
		t.Fatal("dispatch did not send the configured webhook")
	}
}

// TestFinishProviderDeliveryUpdatesPublishedMessageTotals 验证发布后 Provider 最终失败会计入消息统计。
func TestFinishProviderDeliveryUpdatesPublishedMessageTotals(t *testing.T) {
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
	store, err := gendata.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	message := &models.BaseMessage{
		ID: 10, TenantID: 1, CategoryID: 1, SourceType: int32(basev1.MessageSourceType_MESSAGE_SOURCE_TYPE_ADMIN),
		Source: "admin", IdempotencyKey: "message-10", SenderType: int32(basev1.MessageSenderType_MESSAGE_SENDER_TYPE_USER),
		SenderID: 1, SenderName: "Admin", Priority: int32(basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL),
		Title: "Notice", Content: "Body", ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT),
		ActionParams: "{}",
		Status:       int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED), CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now,
	}
	delivery := &models.BaseMessageDelivery{
		TenantID: 1, MessageID: message.ID, DeliveryType: _const.MessageDeliveryTypeProvider,
		ProviderID: 7, UserID: 20, Status: _const.MessageDeliveryStatusRunning,
		LockToken: "active-lock", ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err = db.Create(message).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(delivery).Error; err != nil {
		t.Fatal(err)
	}
	store, err = gendata.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	messageCase := &BaseMessageCase{
		BaseCase: &biz.BaseCase{}, tx: gendata.NewTransaction(store),
		BaseMessageRepository: gendata.NewBaseMessageRepository(store),
		deliveryRepo:          gendata.NewBaseMessageDeliveryRepository(store),
	}
	delivery.LockToken = "active-lock"
	if err = messageCase.finishProviderDelivery(context.Background(), delivery, _const.MessageDeliveryStatusFailed, "failed", "", 0); err != nil {
		t.Fatal(err)
	}
	updated, err := gendata.NewBaseMessageRepository(store).FindByID(context.Background(), message.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.FailedTotal != 1 {
		deliveryQuery := store.Query(context.Background()).BaseMessageDelivery
		updatedDelivery, findErr := gendata.NewBaseMessageDeliveryRepository(store).Find(context.Background(),
			repository.Where(deliveryQuery.TenantID.Eq(delivery.TenantID)),
			repository.Where(deliveryQuery.MessageID.Eq(delivery.MessageID)),
			repository.Where(deliveryQuery.DeliveryType.Eq(delivery.DeliveryType)),
			repository.Where(deliveryQuery.ProviderID.Eq(delivery.ProviderID)),
			repository.Where(deliveryQuery.UserID.Eq(delivery.UserID)),
		)
		if findErr != nil {
			t.Fatal(findErr)
		}
		t.Fatalf("failed_total = %d, want 1; message_status=%d delivery=%+v", updated.FailedTotal, updated.Status, updatedDelivery)
	}
}
