package logmiddleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/adminagreement"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginaudit"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/runtimeconfig"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
	"github.com/liujitcn/kratos-kit/key"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueMemory "github.com/liujitcn/kratos-kit/queue/memory"
	"github.com/liujitcn/kratos-kit/sdk"
)

// TestBusinessAuditClassification 验证业务模块的变更、导出与查询按事实类型审计。
func TestBusinessAuditClassification(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		method       string
		operationLog bool
		accessLog    bool
	}{
		{"宿主新增", "/manager.admin.v1.CustomerService/CreateCustomer", http.MethodPost, true, false},
		{"宿主修改RPC", "/manager.admin.v1.CustomerService/UpdateCustomer", "RPC", true, false},
		{"宿主审核RPC", "/manager.admin.v1.OrderService/ApproveOrder", "RPC", true, false},
		{"工作流审批RPC", "/system.admin.v1.WorkflowTaskService/AuditWorkflowTask", "RPC", true, false},
		{"工作流撤回RPC", "/system.admin.v1.WorkflowInstanceService/WithdrawWorkflowInstance", "RPC", true, false},
		{"工作流重试RPC", "/system.admin.v1.WorkflowInstanceService/RetryWorkflowScript", "RPC", true, false},
		{"知识文档重处理RPC", "/rag.admin.v1.AiKnowledgeDocService/ReprocessAiKnowledgeDoc", "RPC", true, false},
		{"未知业务RPC", "/manager.admin.v1.CustomerService/ProcessCustomer", "RPC", true, false},
		{"导出RPC", "/manager.admin.v1.CustomerService/ExportCustomer", "RPC", true, true},
		{"敏感查看RPC", "/manager.admin.v1.CustomerService/GetCustomer", "RPC", false, true},
		{"POST查询", "/system.admin.v1.BaseUserService/PageBaseUser", http.MethodPost, false, true},
		{"POST搜索", "/manager.admin.v1.CustomerService/QueryCustomer", http.MethodPost, false, true},
		{"日志查询", "/system.admin.v1.BaseOperationLogService/GetBaseOperationLog", http.MethodGet, false, false},
		{"SSE订阅", "/base.v1.SseService/SubscribeSse", http.MethodGet, false, false},
		{"流请求", "/manager.admin.v1.EventService/StreamEvents", http.MethodPost, false, false},
		{"登录", "/base.v1.LoginService/Login", http.MethodPost, false, false},
		{"验证码校验", "/base.v1.LoginService/VerifyCaptcha", http.MethodPost, false, false},
		{"无操作", "", http.MethodPost, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := request{Operation: test.operation, Method: test.method}
			if got := isOperation(info); got != test.operationLog {
				t.Errorf("操作审计 = %v，期望 %v", got, test.operationLog)
			}
			if got := isDataAccess(info); got != test.accessLog {
				t.Errorf("数据访问审计 = %v，期望 %v", got, test.accessLog)
			}
		})
	}
}

// TestAuditSnapshotRedactsCredentials 验证业务请求中的不同字段命名不会泄漏凭据。
func TestAuditSnapshotRedactsCredentials(t *testing.T) {
	snapshot, _ := logSnapshot(map[string]interface{}{
		"name": "customer", "code": "ORDER-1",
		"nested": []interface{}{map[string]interface{}{
			"password": "password-value", "accessToken": "access-value", "refresh_token": "refresh-value",
			"clientSecret": "client-value", "secret": "totp-value", "token": "token-value", "apiKey": "api-value",
			"cookie": "session-value", "recoveryCodes": []string{"recovery-value"},
		}},
	})
	for _, secret := range []string{"password-value", "access-value", "refresh-value", "client-value", "totp-value", "token-value", "api-value", "session-value", "recovery-value"} {
		if strings.Contains(snapshot, secret) {
			t.Errorf("快照泄漏凭据 %q: %s", secret, snapshot)
		}
	}
	var payload map[string]interface{}
	err := json.Unmarshal([]byte(snapshot), &payload)
	if err != nil {
		t.Fatal(err)
	}
	if payload["name"] != "customer" || payload["code"] != "ORDER-1" {
		t.Fatalf("普通业务定位字段丢失: %s", snapshot)
	}
}

// TestSecretRevealAuditsOnlyTarget 验证明文查看只记录目标资源、编号与字段，不保存响应明文。
func TestSecretRevealAuditsOnlyTarget(t *testing.T) {
	logMiddleware := &Middleware{}
	event, ok, err := logMiddleware.buildEvent(adminTask{
		Kind: "data_access", Request: request{Operation: "/base.v1.SecretCryptoService/RevealSecretField", Method: http.MethodPost},
		Payload: &basev1.RevealSecretFieldRequest{Resource: "ai_provider", Id: 12, Field: "api_key"},
		Reply:   &basev1.RevealSecretFieldResponse{Text: "plain-key-never-in-audit"},
	})
	if err != nil || !ok {
		t.Fatalf("查看密钥未构造访问审计: %v", err)
	}
	if strings.Contains(string(event.Payload), "plain-key-never-in-audit") {
		t.Fatal("查看密钥的响应明文泄漏到审计载荷")
	}
	var access models.BaseDataAccessLog
	err = json.Unmarshal(event.Payload, &access)
	if err != nil {
		t.Fatal(err)
	}
	if access.ResourceType != "ai_provider" || access.ResourceID != "12" || access.FieldScope != `["api_key"]` || access.Sensitive != 1 {
		t.Fatalf("密钥查看未记录准确目标: %+v", access)
	}
}

// TestConsumerPersistsFailedAudit 验证写库失败的事件在队列确认前完整落盘并带有效签名。
func TestConsumerPersistsFailedAudit(t *testing.T) {
	directory := t.TempDir()
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	var configJSON []byte
	configJSON, err = json.Marshal(map[string]string{"file_path": directory})
	if err != nil {
		t.Fatal(err)
	}
	err = runtimeconfig.SaveJSON(store, runtimeconfig.BaseLogFallbackKey, string(configJSON))
	if err != nil {
		t.Fatal(err)
	}
	var integrityKey key.Key
	keyPath := directory + "/root.key"
	integrityKey, err = key.NewKey(context.Background(), &configv1.Key{
		Type: "file", Scope: "audit-test", RootName: keyPath, File: &configv1.Key_File{Path: keyPath},
	})
	if err != nil {
		t.Fatal(err)
	}
	previousCache := sdk.Runtime.GetCache()
	previousKey := sdk.Runtime.GetKey()
	sdk.Runtime.SetCache(store)
	sdk.Runtime.SetKey(integrityKey)
	t.Cleanup(func() {
		sdk.Runtime.SetCache(previousCache)
		sdk.Runtime.SetKey(previousKey)
	})

	logMiddleware := &Middleware{}
	var event adminEvent
	event, _, err = logMiddleware.buildEvent(adminTask{
		EventID: "audit-event-1", Kind: "operation",
		Request: request{Operation: "/manager.admin.v1.CustomerService/CreateCustomer", Method: http.MethodPost},
		Payload: map[string]string{"name": "customer", "password": "never-persist-plain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rawEvent []byte
	rawEvent, err = json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	operationRepo := &data.BaseOperationLogRepository{BaseRepository: unavailableOperationRepository{}}
	consumer := NewConsumer(nil, operationRepo, nil, nil)
	err = consumer(queueData.Message{ID: "queue-1", Values: map[string]interface{}{"data": string(rawEvent)}})
	if err != nil {
		t.Fatalf("写库失败应在持久回退后确认消息: %v", err)
	}
	var rawRecord []byte
	rawRecord, err = os.ReadFile(runtimeconfig.BaseLogFallbackFilePath(directory))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawRecord), "never-persist-plain") {
		t.Fatal("回退文件泄漏密码")
	}
	var record baseLogFallbackRecord
	err = json.Unmarshal(rawRecord, &record)
	if err != nil {
		t.Fatal(err)
	}
	var integritySecret string
	integritySecret, err = runtimeconfig.ResolveBaseLogFallbackIntegrityKey()
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(integritySecret))
	_, err = mac.Write(record.Content)
	if err != nil {
		t.Fatal(err)
	}
	if record.HMAC != hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("回退记录签名不匹配")
	}
	var content baseLogFallbackContent
	err = json.Unmarshal(record.Content, &content)
	if err != nil {
		t.Fatal(err)
	}
	var persisted adminEvent
	err = json.Unmarshal(content.Payload, &persisted)
	if err != nil {
		t.Fatal(err)
	}
	if content.Stage != "consumer_failed" || persisted.EventID != "audit-event-1" || string(persisted.Payload) != string(event.Payload) {
		t.Fatalf("回退记录不能完整重放: %s", rawRecord)
	}
}

// TestAgreementConfirmationProducesAudit 验证完成登录的协议确认通过真实缓冲与队列独立留痕。
func TestAgreementConfirmationProducesAudit(t *testing.T) {
	queue := queueMemory.NewMemory(4)
	t.Cleanup(func() {
		queue.Shutdown()
		queue.Wait()
	})
	events := make(chan queueData.Message, 4)
	queue.Register(string(AdminEventStream()), func(message queueData.Message) error {
		events <- message
		return nil
	})
	logMiddleware := newMiddleware(queue, nil)
	t.Cleanup(logMiddleware.close)
	acceptedAt := time.Date(2026, 11, 6, 2, 0, 0, 0, time.UTC)
	receipt := adminagreement.Receipt{
		ServiceVersion: "service-v1", PrivacyVersion: "privacy-v1", ServiceDigest: "service-sha256", PrivacyDigest: "privacy-sha256", AcceptedAt: acceptedAt,
	}
	ctx := transport.NewServerContext(context.Background(), &httpTransport.Transport{})
	httpTransport.SetOperation(ctx, "/base.v1.LoginService/AdminLogin")
	handler := logMiddleware.Handle(func(ctx context.Context, _ interface{}) (interface{}, error) {
		loginaudit.Record(ctx, &authData.UserTokenPayload{TenantId: 7, TenantCode: "1007", UserId: 42, UserName: "alice"})
		ctx = adminagreement.WithReceipt(ctx, receipt)
		adminagreement.Record(ctx)
		return nil, nil
	})
	_, err := handler(ctx, map[string]string{"password": "never-audit-password"})
	if err != nil {
		t.Fatal(err)
	}
	logMiddleware.close()
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case message := <-events:
			raw := message.Values["data"].(string)
			if strings.Contains(raw, "never-audit-password") {
				t.Fatal("登录请求密码泄漏到审计队列")
			}
			var event adminEvent
			err = json.Unmarshal([]byte(raw), &event)
			if err != nil {
				t.Fatal(err)
			}
			if event.Kind != "operation" {
				continue
			}
			var operation models.BaseOperationLog
			err = json.Unmarshal(event.Payload, &operation)
			if err != nil {
				t.Fatal(err)
			}
			if operation.ResourceType != "admin_agreement" || operation.ResourceID != "service-v1|privacy-v1" || operation.UserID != 42 || operation.TenantID != 7 {
				t.Fatalf("协议审计未绑定真实身份与版本: %+v", operation)
			}
			var persisted adminagreement.Receipt
			err = json.Unmarshal([]byte(operation.AfterData), &persisted)
			if err != nil {
				t.Fatal(err)
			}
			if persisted != receipt {
				t.Fatalf("协议版本、摘要或确认时间丢失: %+v", persisted)
			}
			return
		case <-timeout.C:
			t.Fatal("完成登录后的协议确认未进入操作审计队列")
		}
	}
}

type unavailableOperationRepository struct {
	repository.BaseRepository[models.BaseOperationLog]
}

// Create 模拟数据库不可用，保留真实事件构造和持久回退路径。
func (unavailableOperationRepository) Create(context.Context, *models.BaseOperationLog) error {
	return errors.New("database unavailable")
}

// FindByID 模拟写库与重复事件确认同时失败。
func (unavailableOperationRepository) FindByID(context.Context, int64) (*models.BaseOperationLog, error) {
	return nil, errors.New("database unavailable")
}
