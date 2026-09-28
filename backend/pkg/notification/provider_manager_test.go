package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/notify"
)

// TestProviderManagerSendsConfiguredWebhook 验证数据库 Provider 配置可通过 kit Manager 完成外部发送。
func TestProviderManagerSendsConfiguredWebhook(t *testing.T) {
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
		writer.Header().Set("X-Message-ID", "receipt-1")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	manager := NewProviderManager(&biz.BaseCase{})
	receipt, err := manager.Send(context.Background(), &models.BaseMessageProvider{
		ID: 7, Provider: "webhook", ClientID: server.URL,
		Config: `{"allow_http":true,"allow_private_network":true}`,
	}, notify.Message{Title: "Notice", Content: "Body"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt == nil || receipt.MessageID != "receipt-1" {
		t.Fatalf("unexpected webhook receipt: %+v", receipt)
	}
	select {
	case payload := <-received:
		if payload.Title != "Notice" || payload.Content != "Body" {
			t.Fatalf("unexpected webhook payload: %+v", payload)
		}
	default:
		t.Fatal("webhook endpoint did not receive the message")
	}
}
