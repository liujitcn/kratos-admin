package logmiddleware

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/cache/memory"
	"github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

type testQueue struct {
	stream  string
	message data.Message
	started chan struct{}
	release chan struct{}
}

type logTestKey struct {
	value []byte
}

// Derive 返回测试使用的运行时派生密钥。
func (k logTestKey) Derive(context.Context, string) ([]byte, error) { return k.value, nil }

// TestLogResponseMetadata 验证数据访问审计从真实响应提取字段、行数和敏感标识。
func TestLogResponseMetadata(t *testing.T) {
	response := &adminv1.PageBaseUserResponse{
		BaseUsers: []*adminv1.BaseUser{{Phone: "138****0000"}, {Phone: "139****0000"}},
		Total:     2,
	}
	fields, rows, sensitive := logResponseMetadata(response)
	if rows != 2 || !sensitive {
		t.Fatalf("unexpected response metadata: fields=%s rows=%d sensitive=%v", fields, rows, sensitive)
	}
	if fields == "[]" {
		t.Fatal("expected populated response field scope")
	}
}

// TestLogResourceReadsNestedBusinessFields 验证嵌套表单中的资源编号和业务名称可以被识别。
func TestLogResourceReadsNestedBusinessFields(t *testing.T) {
	resourceID, resourceName := logResource(map[string]interface{}{
		"base_message": map[string]interface{}{
			"id":    1,
			"title": "系统维护通知",
		},
	})
	if resourceID != "1" || resourceName != "系统维护通知" {
		t.Fatalf("unexpected nested resource: id=%s name=%s", resourceID, resourceName)
	}
}

// TestBuildOperationEventUsesResourceSnapshot 验证操作日志使用请求前的资源名称和快照。
func TestBuildOperationEventUsesResourceSnapshot(t *testing.T) {
	logMiddleware := newMiddleware(nil, nil)
	defer logMiddleware.close()
	event, ok, err := logMiddleware.buildEvent(adminTask{
		Kind:     "operation",
		Request:  request{Operation: "/system.admin.v1.BaseMessageService/UpdateBaseMessage", OccurredAt: time.Now()},
		Payload:  map[string]interface{}{"base_message": map[string]interface{}{"id": 1, "title": "新标题"}},
		Snapshot: resourceSnapshot{ResourceID: "1", ResourceName: "旧标题", BeforeData: `{"id":1,"title":"旧标题"}`},
	})
	if err != nil || !ok {
		t.Fatalf("build operation event failed: ok=%v err=%v", ok, err)
	}
	item := &models.BaseOperationLog{}
	if err = json.Unmarshal(event.Payload, item); err != nil {
		t.Fatal(err)
	}
	if item.ResourceID != "1" || item.ResourceName != "旧标题" || item.BeforeData != `{"id":1,"title":"旧标题"}` {
		t.Fatalf("operation snapshot was not applied: %+v", item)
	}
}

// TestBuildDataAccessEventIncludesTableAndRequestIdentity 验证数据访问日志写入资源表和关联请求字段。
func TestBuildDataAccessEventIncludesTableAndRequestIdentity(t *testing.T) {
	logMiddleware := newMiddleware(nil, nil)
	defer logMiddleware.close()
	event, ok, err := logMiddleware.buildEvent(adminTask{
		Kind:    "data_access",
		Request: request{Operation: "/system.admin.v1.BaseMessageService/PageBaseMessage", RequestID: "request-1", TraceID: "trace-1", OccurredAt: time.Now()},
		Payload: map[string]interface{}{"tenant_id": 1},
		Reply: &adminv1.PageBaseMessageResponse{
			BaseMessages: []*adminv1.BaseMessage{{Title: "系统通知"}},
			Total:        1,
		},
	})
	if err != nil || !ok {
		t.Fatalf("build data access event failed: ok=%v err=%v", ok, err)
	}
	item := &models.BaseDataAccessLog{}
	if err = json.Unmarshal(event.Payload, item); err != nil {
		t.Fatal(err)
	}
	if item.TableName_ != "base_message" || item.RequestID != "request-1" || item.TraceID != "trace-1" {
		t.Fatalf("data access metadata was not applied: %+v", item)
	}
}

// TestBuildPermissionEventUsesRoleSnapshot 验证权限日志使用操作前角色快照。
func TestBuildPermissionEventUsesRoleSnapshot(t *testing.T) {
	logMiddleware := newMiddleware(nil, nil)
	defer logMiddleware.close()
	event, ok, err := logMiddleware.buildEvent(adminTask{
		Kind:     "permission",
		Request:  request{Operation: "/system.admin.v1.BaseRoleService/UpdateBaseRole", RequestID: "request-2", TraceID: "trace-2", OccurredAt: time.Now()},
		Payload:  map[string]interface{}{"base_role": map[string]interface{}{"id": 2, "name": "新角色"}},
		Snapshot: resourceSnapshot{ResourceID: "2", ResourceName: "旧角色", BeforeData: `{"id":2,"name":"旧角色","menus":"[1]"}`},
	})
	if err != nil || !ok {
		t.Fatalf("build permission event failed: ok=%v err=%v", ok, err)
	}
	item := &models.BasePermissionLog{}
	if err = json.Unmarshal(event.Payload, item); err != nil {
		t.Fatal(err)
	}
	if item.TargetID != "2" || item.TargetName != "旧角色" || item.OldValue != `{"id":2,"name":"旧角色","menus":"[1]"}` || item.RequestID != "request-2" || item.TraceID != "trace-2" {
		t.Fatalf("permission metadata was not applied: %+v", item)
	}
}

// TestLogSnapshotRedactsRuntimeConfigValue 验证隐藏配置更新的 JSON 字符串不会进入审计快照。
func TestLogSnapshotRedactsRuntimeConfigValue(t *testing.T) {
	snapshot, _ := logSnapshot(map[string]string{"key": "baseLogFallback", "value_json": `{"file_path":"./logs/base-log-fallback","password":"secret"}`})
	if strings.Contains(snapshot, "secret") || !strings.Contains(snapshot, "[REDACTED]") {
		t.Fatalf("runtime config value was not redacted: %s", snapshot)
	}
}

// Append 记录测试期间投递的队列消息。
func (q *testQueue) Append(stream string, message data.Message) error {
	q.stream = stream
	q.message = message
	if q.started != nil {
		close(q.started)
		<-q.release
	}
	return nil
}

// Register 忽略测试不需要的消费者注册。
func (*testQueue) Register(string, data.ConsumerFunc) {}

// Run 忽略测试不需要的队列启动。
func (*testQueue) Run() {}

// Shutdown 忽略测试不需要的队列停止。
func (*testQueue) Shutdown() {}

type testHeader map[string]string

// Get 读取测试请求头。
func (h testHeader) Get(key string) string { return h[key] }

// Set 写入测试请求头。
func (h testHeader) Set(key string, value string) { h[key] = value }

// Add 写入测试请求头。
func (h testHeader) Add(key string, value string) { h[key] = value }

// Keys 返回测试请求头名称。
func (h testHeader) Keys() []string { return nil }

// Values 返回测试请求头值。
func (h testHeader) Values(key string) []string { return []string{h[key]} }

type testTransport struct {
	header testHeader
}

// Kind 返回测试传输类型。
func (*testTransport) Kind() transport.Kind { return transport.KindGRPC }

// Endpoint 返回测试传输端点。
func (*testTransport) Endpoint() string { return "grpc://test" }

// Operation 返回用于触发操作审计的测试方法。
func (*testTransport) Operation() string { return "/system.admin.v1.BaseUserService/UpdateBaseUser" }

// RequestHeader 返回测试请求头。
func (t *testTransport) RequestHeader() transport.Header { return t.header }

// ReplyHeader 返回测试响应头。
func (t *testTransport) ReplyHeader() transport.Header { return t.header }

// Method 返回测试请求方法。
func (*testTransport) Method() string { return "POST" }

// TestMiddlewareEmitsAdminEventToQueue 验证 Admin 业务审计事件只投递到异步入库队列。
func TestMiddlewareEmitsAdminEventToQueue(t *testing.T) {
	queue := &testQueue{}
	logMiddleware := newMiddleware(queue, nil)
	logMiddleware.enqueue(adminTask{
		Kind: "login",
		Request: request{
			Operation:  "/base.v1.LoginService/Login",
			RequestID:  "test",
			TenantCode: "default",
			OccurredAt: time.Now(),
		},
		Payload: map[string]string{"user_name": "admin"},
	})
	logMiddleware.close()
	if queue.stream != string(adminEventStream) {
		t.Fatalf("unexpected stream: %s", queue.stream)
	}
	rawBody, ok := queue.message.Values["data"].(string)
	if !ok {
		t.Fatal("expected string queue payload")
	}
	var event adminEvent
	err := json.Unmarshal([]byte(rawBody), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != "login" {
		t.Fatalf("unexpected event kind: %s", event.Kind)
	}
	if len(event.Payload) == 0 {
		t.Fatal("expected event payload")
	}
}

// TestMiddlewareDoesNotWaitForQueueIO 验证队列阻塞不会延迟主业务处理结果。
func TestMiddlewareDoesNotWaitForQueueIO(t *testing.T) {
	queue := &testQueue{started: make(chan struct{}), release: make(chan struct{})}
	logMiddleware := newMiddleware(queue, nil)
	defer func() {
		close(queue.release)
		logMiddleware.close()
	}()
	ctx := transport.NewServerContext(context.Background(), &testTransport{header: testHeader{"X-Request-ID": "request-1"}})
	handler := logMiddleware.Handle(func(context.Context, interface{}) (interface{}, error) {
		return "business-result", nil
	})
	startedAt := time.Now()
	reply, err := handler(ctx, map[string]string{"id": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "business-result" {
		t.Fatalf("unexpected business result: %v", reply)
	}
	if time.Since(startedAt) > 100*time.Millisecond {
		t.Fatal("business handler waited for log queue IO")
	}
	select {
	case <-queue.started:
	case <-time.After(time.Second):
		t.Fatal("log worker did not dispatch queued event")
	}
}

// TestIsOperationRecognizesRestoreRPC 验证 gRPC Execute...Restore 会被记录为业务操作。
func TestIsOperationRecognizesRestoreRPC(t *testing.T) {
	info := request{Method: "RPC", Operation: "/system.admin.v1.BaseTableBackupRestoreService/ExecuteBaseTableBackupRestore"}
	if !isOperation(info) {
		t.Fatal("restore RPC was not recognized as an operation")
	}
}

// TestBaseLogFallbackFileHasValidHMAC 验证日志入库回退文件同步落盘并携带有效 HMAC。
func TestBaseLogFallbackFileHasValidHMAC(t *testing.T) {
	directory := t.TempDir()
	key := "base-log-fallback-integrity-key-32-bytes"
	previousKey := sdk.Runtime.GetKey()
	sdk.Runtime.SetKey(logTestKey{value: []byte(key)})
	t.Cleanup(func() { sdk.Runtime.SetKey(previousKey) })
	configCache := newLogStorageTestCache(t, directory)
	logMiddleware := newMiddleware(nil, configCache)
	err := logMiddleware.writeBaseLogFallback("queue_unavailable", "login", "/base.v1.LoginService/Login", map[string]string{"user_name": "admin"})
	logMiddleware.close()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, runtimeconfig.BaseLogFallbackFileName)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("日志入库回退文件没有记录")
	}
	var record baseLogFallbackRecord
	if err = json.Unmarshal(scanner.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(base64.RawStdEncoding.EncodeToString([]byte(key))))
	_, _ = mac.Write(record.Content)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(record.HMAC)) {
		t.Fatal("日志入库回退 HMAC 校验失败")
	}
}

// TestBaseLogFallbackUsesLastConfigWhenCacheUnavailable 验证缓存不可用时仍使用最近一次有效回退配置写文件。
func TestBaseLogFallbackUsesLastConfigWhenCacheUnavailable(t *testing.T) {
	directory := t.TempDir()
	key := "base-log-fallback-integrity-key-32-bytes"
	previousKey := sdk.Runtime.GetKey()
	sdk.Runtime.SetKey(logTestKey{value: []byte(key)})
	t.Cleanup(func() { sdk.Runtime.SetKey(previousKey) })
	configCache := newLogStorageTestCache(t, directory)
	logMiddleware := newMiddleware(nil, configCache)
	err := logMiddleware.writeBaseLogFallback("queue_unavailable", "login", "/base.v1.LoginService/Login", map[string]string{"user_name": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	err = configCache.Del(runtimeconfig.CacheKey(runtimeconfig.BaseLogFallbackKey))
	if err != nil {
		t.Fatal(err)
	}
	err = logMiddleware.writeBaseLogFallback("queue_append_failed", "login", "/base.v1.LoginService/Login", map[string]string{"user_name": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	logMiddleware.close()

	path := filepath.Join(directory, runtimeconfig.BaseLogFallbackFileName)
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if err = scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if lines != 2 {
		t.Fatalf("expected two fallback records, got %d", lines)
	}
}

// newLogStorageTestCache 创建包含日志入库回退配置的内存缓存。
func newLogStorageTestCache(t *testing.T, directory string) cache.Cache {
	t.Helper()
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	config := runtimeconfig.DefaultBaseLogFallbackConfig()
	config.FilePath = directory
	value, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Set(runtimeconfig.CacheKey(runtimeconfig.BaseLogFallbackKey), string(value), runtimeconfig.CacheExpire); err != nil {
		t.Fatal(err)
	}
	return store
}

type logTestRecord struct {
	ID int64
}

// TestCreateLogTreatsDuplicateDeliveryAsSuccess 验证同一队列消息重复投递不会生成第二条日志。
func TestCreateLogTreatsDuplicateDeliveryAsSuccess(t *testing.T) {
	record := &logTestRecord{ID: 10}
	var stored *logTestRecord
	create := func(_ context.Context, item *logTestRecord) error {
		if stored != nil {
			return errors.New("duplicate primary key")
		}
		stored = item
		return nil
	}
	find := func(_ context.Context, id int64) (*logTestRecord, error) {
		if stored != nil && stored.ID == id {
			return stored, nil
		}
		return nil, errors.New("not found")
	}
	err := createLogRecord(context.Background(), record.ID, record, create, find)
	if err != nil {
		t.Fatal(err)
	}
	err = createLogRecord(context.Background(), record.ID, record, create, find)
	if err != nil {
		t.Fatalf("duplicate delivery must be idempotent: %v", err)
	}
}
