package logmiddleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/go-utils/id"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/runtimeconfig"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreBiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/server/requestmeta"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/cache"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	adminEventStream queueTransport.Stream = "admin.log.event"
	adminBufferSize                        = 2048
)

// AdminEventStream 返回 Admin 业务审计事件队列名称。
func AdminEventStream() queueTransport.Stream {
	return adminEventStream
}

// adminEvent 是 Admin 业务审计队列的事件包装，不与 Core 审计事件混用。
type adminEvent struct {
	EventID string          `json:"event_id,omitempty"` // 业务审计事件的稳定编号。
	Kind    string          `json:"kind"`               // 业务审计类型。
	Payload json.RawMessage `json:"payload"`            // 对应 Admin 日志模型的 JSON 载荷。
}

// adminTask 保存请求线程投递给后台审计工作协程的最小上下文。
type adminTask struct {
	EventID string
	Kind    string
	Request request
	Payload interface{}
	Reply   interface{}
}

// Middleware 记录 Admin 请求对应的登录、操作、数据访问和权限审计事实。
// Core 产生的 API/策略事件走 Core Sink；本中间件只处理 Admin 自身业务事件。
type Middleware struct {
	queue       kitQueue.Queue
	configCache cache.Cache
	tasks       chan adminTask
	worker      sync.WaitGroup
	stateMu     sync.RWMutex
	fileMu      sync.Mutex
	closed      bool
}

// auditLogSpoolContent 是日志入库回退文件中参与完整性签名的稳定内容。
type auditLogSpoolContent struct {
	RecordedAt string          `json:"recorded_at"`
	Stage      string          `json:"stage"`
	Kind       string          `json:"kind"`
	Operation  string          `json:"operation"`
	Payload    json.RawMessage `json:"payload"`
}

// auditLogSpoolRecord 包装日志入库回退内容及其 HMAC。
type auditLogSpoolRecord struct {
	Content json.RawMessage `json:"content"`
	HMAC    string          `json:"hmac"`
}

// NewMiddleware 创建审计日志中间件。
func NewMiddleware(baseCase *coreBiz.BaseCase) (middleware.Middleware, func()) {
	logMiddleware := newMiddleware(nil, baseCase.Cache)
	return logMiddleware.Handle, logMiddleware.close
}

// NewConsumer 创建 Admin 业务审计事件消费者并写入对应日志表。
func NewConsumer(
	loginLogRepo *data.BaseLoginLogRepository,
	operationLogRepo *data.BaseOperationLogRepository,
	dataAccessLogRepo *data.BaseDataAccessLogRepository,
	permissionLogRepo *data.BasePermissionLogRepository,
) queueData.ConsumerFunc {
	return func(message queueData.Message) error {
		rawValue, ok := message.Values["data"].(string)
		if !ok || rawValue == "" {
			return fmt.Errorf("Admin 审计事件消息体为空")
		}
		return ReplayAdminEvent(context.Background(), message.ID, []byte(rawValue), loginLogRepo, operationLogRepo, dataAccessLogRepo, permissionLogRepo)
	}
}

// ReplayAdminEvent 将一条 Admin 审计事件按稳定编号幂等写入对应日志表。
func ReplayAdminEvent(
	ctx context.Context,
	messageID string,
	rawValue []byte,
	loginLogRepo *data.BaseLoginLogRepository,
	operationLogRepo *data.BaseOperationLogRepository,
	dataAccessLogRepo *data.BaseDataAccessLogRepository,
	permissionLogRepo *data.BasePermissionLogRepository,
) error {
	var envelope adminEvent
	err := json.Unmarshal(rawValue, &envelope)
	if err != nil {
		return fmt.Errorf("解析 Admin 审计事件失败: %w", err)
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("Admin 审计事件载荷为空")
	}
	eventID := messageID
	if envelope.EventID != "" {
		eventID = envelope.EventID
	}
	recordID := coreBiz.LogMessagePrimaryKey(eventID)
	switch envelope.Kind {
	case "login":
		item := &models.BaseLoginLog{}
		err = json.Unmarshal(envelope.Payload, item)
		if err != nil {
			return fmt.Errorf("解析 Admin 登录审计事件失败: %w", err)
		}
		item.ID = recordID
		item.CreatedAt = time.Now()
		return createLogRecord(ctx, item.ID, item, loginLogRepo.Create, loginLogRepo.FindByID)
	case "operation":
		item := &models.BaseOperationLog{}
		err = json.Unmarshal(envelope.Payload, item)
		if err != nil {
			return fmt.Errorf("解析 Admin 操作审计事件失败: %w", err)
		}
		item.ID = recordID
		item.CreatedAt = time.Now()
		return createLogRecord(ctx, item.ID, item, operationLogRepo.Create, operationLogRepo.FindByID)
	case "data_access":
		item := &models.BaseDataAccessLog{}
		err = json.Unmarshal(envelope.Payload, item)
		if err != nil {
			return fmt.Errorf("解析 Admin 数据访问审计事件失败: %w", err)
		}
		item.ID = recordID
		item.CreatedAt = time.Now()
		return createLogRecord(ctx, item.ID, item, dataAccessLogRepo.Create, dataAccessLogRepo.FindByID)
	case "permission":
		item := &models.BasePermissionLog{}
		err = json.Unmarshal(envelope.Payload, item)
		if err != nil {
			return fmt.Errorf("解析 Admin 权限审计事件失败: %w", err)
		}
		item.ID = recordID
		item.CreatedAt = time.Now()
		return createLogRecord(ctx, item.ID, item, permissionLogRepo.Create, permissionLogRepo.FindByID)
	default:
		return fmt.Errorf("未知 Admin 审计事件类型: %s", envelope.Kind)
	}
}

// Handle 记录一次请求的登录、操作、数据访问和权限审计信息。
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
		startedAt := time.Now()
		request := requestInfo(ctx)
		reply, err = next(ctx, req)
		result, reasonCode, reason := resultInfo(err)
		request.Result = result
		request.ReasonCode = reasonCode
		request.Reason = reason
		request.OccurredAt = startedAt
		if _, ok := loginType(request.Operation); ok {
			m.enqueue(adminTask{Kind: "login", Request: request, Payload: req})
		}
		permissionOperation := isPermissionOperation(request)
		if isOperation(request) && !permissionOperation {
			m.enqueue(adminTask{Kind: "operation", Request: request, Payload: req})
		}
		if isDataAccess(request) {
			m.enqueue(adminTask{Kind: "data_access", Request: request, Reply: reply})
		}
		if permissionOperation {
			m.enqueue(adminTask{Kind: "permission", Request: request, Payload: req})
		}
		return reply, err
	}
}

// enqueue 将 Admin 审计任务非阻塞写入进程内缓冲。
func (m *Middleware) enqueue(task adminTask) {
	if task.EventID == "" {
		task.EventID = id.NewGUIDv7NoHyphen()
	}
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	if m.closed {
		log.Error("Admin 审计任务缓冲已关闭", "kind", task.Kind, "operation", task.Request.Operation)
		m.writeAuditLogSpoolTask("buffer_closed", task)
		return
	}
	select {
	case m.tasks <- task:
	default:
		log.Error("Admin 审计任务缓冲已满", "kind", task.Kind, "operation", task.Request.Operation)
		m.writeAuditLogSpoolTask("buffer_full", task)
	}
}

// writeAuditLogSpoolTask 将未进入进程缓冲的请求按统一脱敏规则写入日志入库回退文件。
func (m *Middleware) writeAuditLogSpoolTask(stage string, task adminTask) {
	event, ok, err := m.buildEvent(task)
	if err != nil {
		log.Error("构造 Admin 日志入库回退事件失败", "error", err, "stage", stage, "kind", task.Kind, "operation", task.Request.Operation)
		return
	}
	if !ok {
		return
	}
	var rawBody []byte
	rawBody, err = json.Marshal(event)
	if err != nil {
		log.Error("封装 Admin 日志入库回退事件失败", "error", err, "stage", stage, "kind", task.Kind, "operation", task.Request.Operation)
		return
	}
	if err = m.writeAuditLogSpool(stage, event.Kind, task.Request.Operation, json.RawMessage(rawBody)); err != nil {
		log.Error("写入 Admin 日志入库回退文件失败", "error", err, "stage", stage, "kind", task.Kind, "operation", task.Request.Operation)
	}
}

// run 在后台构造、脱敏并投递 Admin 业务审计事件。
func (m *Middleware) run() {
	defer m.worker.Done()
	for task := range m.tasks {
		event, ok, err := m.buildEvent(task)
		if err != nil {
			log.Error("构造 Admin 审计事件失败", "error", err, "kind", task.Kind, "operation", task.Request.Operation)
			continue
		}
		if ok {
			m.emitEvent(event, task.Request.Operation)
		}
	}
}

// close 关闭任务缓冲并等待已接收的审计任务完成队列投递。
func (m *Middleware) close() {
	m.stateMu.Lock()
	if !m.closed {
		m.closed = true
		close(m.tasks)
	}
	m.stateMu.Unlock()
	m.worker.Wait()
}

// buildEvent 构造一条脱敏后的 Admin 审计事件。
func (m *Middleware) buildEvent(task adminTask) (adminEvent, bool, error) {
	info := task.Request
	var item interface{}
	switch task.Kind {
	case "login":
		loginTypeValue, ok := loginType(info.Operation)
		if !ok {
			return adminEvent{}, false, nil
		}
		userName := info.UserName
		tenantCode := info.TenantCode
		if loginRequest, ok := task.Payload.(interface {
			GetTenantCode() string
			GetUserName() string
		}); ok {
			if loginRequest.GetTenantCode() != "" {
				tenantCode = loginRequest.GetTenantCode()
			}
			if loginRequest.GetUserName() != "" {
				userName = loginRequest.GetUserName()
			}
		}
		if clientRequest, ok := task.Payload.(interface{ GetClientId() string }); ok && clientRequest.GetClientId() != "" {
			userName = clientRequest.GetClientId()
		}
		item = &models.BaseLoginLog{
			TenantID: info.TenantID, TenantCode: tenantCode, UserID: info.UserID, UserName: userName,
			LoginType: int32(loginTypeValue), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
			ClientIP: info.ClientIP, UserAgent: info.UserAgent, RequestID: info.RequestID, TraceID: info.TraceID,
			OccurredAt: info.OccurredAt,
		}
	case "operation":
		afterData, changedFields := logSnapshot(task.Payload)
		resourceID, resourceName := logResource(task.Payload)
		item = &models.BaseOperationLog{
			TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName,
			RequestID: info.RequestID, TraceID: info.TraceID, ResourceType: resourceType(info.Operation), ResourceID: resourceID, ResourceName: resourceName,
			ChangedFields: changedFields, BeforeData: "{}", AfterData: afterData,
			Action: int32(operationAction(info)), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
			OccurredAt: info.OccurredAt,
		}
	case "data_access":
		fieldScope, affectedRows, sensitive := logResponseMetadata(task.Reply)
		item = &models.BaseDataAccessLog{
			TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName, ResourceType: resourceType(info.Operation),
			AccessType: int32(accessType(info)), DataSource: databaseGorm.DefaultClientName, FieldScope: fieldScope, AffectedRows: affectedRows, Sensitive: boolToInt32(sensitive), Result: info.Result,
			ReasonCode: info.ReasonCode, OccurredAt: info.OccurredAt,
		}
	case "permission":
		newValue, _ := logSnapshot(task.Payload)
		targetID, targetName := logResource(task.Payload)
		item = &models.BasePermissionLog{
			TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName, TargetType: int32(permissionTargetType(info.Operation)), TargetID: targetID, TargetName: targetName,
			OldValue: "{}", NewValue: newValue,
			Action: int32(permissionAction(info)), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
			OccurredAt: info.OccurredAt,
		}
	default:
		return adminEvent{}, false, fmt.Errorf("未知 Admin 审计事件类型: %s", task.Kind)
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return adminEvent{}, false, fmt.Errorf("序列化 Admin 审计事件失败: %w", err)
	}
	return adminEvent{EventID: task.EventID, Kind: task.Kind, Payload: payload}, true, nil
}

// request 汇总一次 HTTP 或 gRPC 请求的审计上下文。
// 该结构只在中间件生命周期内使用，最终会被转换为具体审计日志模型。
type request struct {
	Operation  string    // Kratos operation 标识。
	Method     string    // HTTP 方法或 RPC 标识。
	Path       string    // HTTP 路径模板。
	RequestID  string    // 请求追踪编号。
	TraceID    string    // 分布式链路追踪编号。
	TenantID   int64     // 当前租户编号。
	TenantCode string    // 当前租户编码。
	UserID     int64     // 当前用户编号。
	UserName   string    // 当前用户账号。
	ClientIP   string    // 对端 IP 地址。
	UserAgent  string    // 客户端 User-Agent。
	Result     int32     // 审计结果枚举值。
	ReasonCode string    // 稳定错误原因码。
	Reason     string    // 脱敏后的错误描述。
	OccurredAt time.Time // 请求开始时间。
}

// requestInfo 从服务端传输和认证上下文提取通用审计字段。
func requestInfo(ctx context.Context) request {
	info := request{RequestID: requestmeta.RequestID(ctx), TraceID: requestmeta.TraceID(ctx), TenantCode: databaseGorm.DefaultTenantCode, Method: "RPC"}
	if info.RequestID == "" {
		info.RequestID = id.NewGUIDv4NoHyphen()
	}
	if serverTransport, ok := transport.FromServerContext(ctx); ok {
		info.Operation = serverTransport.Operation()
		if htr, ok := serverTransport.(*httpTransport.Transport); ok && htr.Request() != nil {
			httpRequest := htr.Request()
			info.Method = httpRequest.Method
			info.Path = htr.PathTemplate()
			info.ClientIP = clientIP(httpRequest)
			info.UserAgent = httpRequest.UserAgent()
		}
	}
	if authInfo, err := auth.FromContext(ctx); err == nil && authInfo != nil {
		info.TenantID = authInfo.TenantId
		info.TenantCode = authInfo.TenantCode
		info.UserID = authInfo.UserId
		info.UserName = authInfo.UserName
	}
	return info
}

// resultInfo 将业务错误转换为审计结果和稳定原因。
func resultInfo(err error) (int32, string, string) {
	if err == nil {
		return int32(adminv1.BaseLogResult_BASE_LOG_RESULT_SUCCESS), "", ""
	}
	statusCode := int32(http.StatusInternalServerError)
	reasonCode := "INTERNAL_ERROR"
	reason := err.Error()
	if structuredErr := kratosErrors.FromError(err); structuredErr != nil {
		statusCode = structuredErr.Code
		reasonCode = structuredErr.Reason
		reason = structuredErr.Message
	}
	result := adminv1.BaseLogResult_BASE_LOG_RESULT_FAILURE
	if statusCode >= http.StatusInternalServerError {
		result = adminv1.BaseLogResult_BASE_LOG_RESULT_ERROR
	}
	return int32(result), reasonCode, reason
}

// logResponseMetadata 从真实 Proto 响应提取字段范围、返回行数和敏感字段触达标识。
func logResponseMetadata(value interface{}) (string, int32, bool) {
	message, ok := value.(proto.Message)
	if !ok || message == nil {
		return "[]", 0, false
	}
	reflection := message.ProtoReflect()
	fields := make([]string, 0)
	rows := int64(0)
	sensitive := false
	reflection.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fields = append(fields, field.JSONName())
		if field.IsList() && int64(value.List().Len()) > rows {
			rows = int64(value.List().Len())
		}
		if field.JSONName() == "total" {
			switch field.Kind() {
			case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
				rows = value.Int()
			case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
				if value.Uint() > math.MaxInt32 {
					rows = math.MaxInt32
				} else {
					rows = int64(value.Uint())
				}
			}
		}
		if logFieldSensitive(field) {
			sensitive = true
		}
		return true
	})
	sort.Strings(fields)
	fieldScope, err := json.Marshal(fields)
	if err != nil {
		fieldScope = []byte("[]")
	}
	if rows > math.MaxInt32 {
		rows = math.MaxInt32
	}
	return string(fieldScope), int32(rows), sensitive
}

// logFieldSensitive 判断响应字段名称或嵌套消息是否包含个人敏感信息。
func logFieldSensitive(field protoreflect.FieldDescriptor) bool {
	name := strings.ToLower(field.JSONName())
	for _, marker := range []string{"phone", "mobile", "email", "idcard", "identity", "bank", "password", "secret", "token"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	if field.Kind() != protoreflect.MessageKind {
		return false
	}
	fields := field.Message().Fields()
	for index := 0; index < fields.Len(); index++ {
		child := fields.Get(index)
		childName := strings.ToLower(child.JSONName())
		for _, marker := range []string{"phone", "mobile", "email", "idcard", "identity", "bank", "password", "secret", "token"} {
			if strings.Contains(childName, marker) {
				return true
			}
		}
	}
	return false
}

// boolToInt32 将审计布尔标识转换为数据库 tinyint。
func boolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// logSnapshot 将请求对象转换为脱敏 JSON，并提取顶层变更字段。
func logSnapshot(value interface{}) (string, string) {
	if value == nil {
		return "{}", "[]"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}", "[]"
	}
	var payload interface{}
	if err = json.Unmarshal(raw, &payload); err != nil {
		return "{}", "[]"
	}
	payload = redactLogValue(payload)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}", "[]"
	}
	fields := make([]string, 0)
	if object, ok := payload.(map[string]interface{}); ok {
		for field := range object {
			fields = append(fields, field)
		}
		sort.Strings(fields)
	}
	fieldJSON, err := json.Marshal(fields)
	if err != nil {
		return string(encoded), "[]"
	}
	return string(encoded), string(fieldJSON)
}

// redactLogValue 递归移除审计快照中的凭据、令牌和验证码。
func redactLogValue(value interface{}) interface{} {
	switch item := value.(type) {
	case map[string]interface{}:
		for key, child := range item {
			if isSensitiveLogField(key) {
				item[key] = "[REDACTED]"
				continue
			}
			item[key] = redactLogValue(child)
		}
	case []interface{}:
		for index, child := range item {
			item[index] = redactLogValue(child)
		}
	}
	return value
}

// isSensitiveLogField 判断字段名是否包含不应进入审计快照的敏感值。
func isSensitiveLogField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{"password", "old_pwd", "new_pwd", "pwd", "client_secret", "crypto_key", "encrypted_key", "ciphertext", "nonce", "access_token", "refresh_token", "authorization", "captcha_code", "verification_code", "private_key", "content", "action_params", "value_json", "phone", "mobile", "email", "id_card", "identity_number"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "iv" || strings.HasSuffix(normalized, "_iv")
}

// logResource 从请求快照中提取资源编号和名称，便于审计人员定位对象。
func logResource(value interface{}) (string, string) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", ""
	}
	var payload interface{}
	if err = json.Unmarshal(raw, &payload); err != nil {
		return "", ""
	}
	object, ok := payload.(map[string]interface{})
	if !ok {
		return "", ""
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	resourceID := ""
	resourceName := ""
	for _, key := range keys {
		value := object[key]
		switch {
		case resourceID == "" && (key == "id" || strings.HasSuffix(key, "_id")):
			resourceID = fmt.Sprint(value)
		case resourceName == "" && (key == "name" || strings.HasSuffix(key, "_name")):
			resourceName = fmt.Sprint(value)
		}
	}
	return resourceID, resourceName
}

// emitEvent 将构造完成的 Admin 审计事件投递到异步入库队列。
func (m *Middleware) emitEvent(event adminEvent, operation string) {
	rawBody, err := json.Marshal(event)
	if err != nil {
		log.Error("封装 Admin 审计事件失败", "error", err, "operation", operation, "kind", event.Kind)
		return
	}
	eventQueue := m.queue
	if eventQueue == nil {
		eventQueue = sdk.Runtime.GetQueue()
	}
	if eventQueue == nil {
		log.Error("Admin 审计事件队列不可用", "operation", operation, "kind", event.Kind)
		if spoolErr := m.writeAuditLogSpool("queue_unavailable", event.Kind, operation, json.RawMessage(rawBody)); spoolErr != nil {
			log.Error("写入 Admin 日志入库回退文件失败", "error", spoolErr, "stage", "queue_unavailable", "kind", event.Kind, "operation", operation)
		}
		return
	}
	err = eventQueue.Append(string(adminEventStream), queueData.Message{Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		log.Error("投递 Admin 审计事件失败", "error", err, "operation", operation, "kind", event.Kind)
		if spoolErr := m.writeAuditLogSpool("queue_append_failed", event.Kind, operation, json.RawMessage(rawBody)); spoolErr != nil {
			log.Error("写入 Admin 日志入库回退文件失败", "error", spoolErr, "stage", "queue_append_failed", "kind", event.Kind, "operation", operation)
		}
	}
}

// writeAuditLogSpool 将单条失败事件追加到日志入库回退文件并同步落盘。
func (m *Middleware) writeAuditLogSpool(stage, kind, operation string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化日志入库回退载荷失败: %w", err)
	}
	var contentJSON []byte
	contentJSON, err = json.Marshal(auditLogSpoolContent{
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Stage:      stage,
		Kind:       kind,
		Operation:  operation,
		Payload:    payloadJSON,
	})
	if err != nil {
		return fmt.Errorf("序列化日志入库回退内容失败: %w", err)
	}
	config := runtimeconfig.DefaultAuditLogSpoolConfig()
	if m.configCache != nil {
		if err = runtimeconfig.LoadJSON(m.configCache, runtimeconfig.AuditLogSpoolKey, &config); err != nil {
			return fmt.Errorf("读取日志入库回退配置失败: %w", err)
		}
	}
	var key string
	key, err = runtimeconfig.ResolveAuditLogIntegrityKey()
	if err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("日志入库回退完整性密钥为空")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(contentJSON)
	var recordJSON []byte
	recordJSON, err = json.Marshal(auditLogSpoolRecord{Content: contentJSON, HMAC: hex.EncodeToString(mac.Sum(nil))})
	if err != nil {
		return fmt.Errorf("序列化日志入库回退记录失败: %w", err)
	}
	path := runtimeconfig.AuditLogSpoolFilePath(config.FilePath)
	m.fileMu.Lock()
	defer m.fileMu.Unlock()
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("创建日志入库回退目录失败: %w", err)
	}
	var file *os.File
	file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("打开日志入库回退文件失败: %w", err)
	}
	defer file.Close()
	if _, err = file.Write(append(recordJSON, '\n')); err != nil {
		return fmt.Errorf("追加日志入库回退记录失败: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("同步日志入库回退文件失败: %w", err)
	}
	return nil
}

// loginType 判断认证操作类型。
func loginType(operation string) (adminv1.BaseLoginLogType, bool) {
	switch {
	case strings.Contains(operation, "LoginService/Login"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_PASSWORD, true
	case strings.Contains(operation, "MfaService/"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_MFA, true
	case strings.Contains(operation, "LoginService/RefreshToken"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_TOKEN_REFRESH, true
	case strings.Contains(operation, "LoginService/Logout"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_LOGOUT, true
	case strings.Contains(operation, "OauthClientService/IssueOauthClientToken"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_OAUTH, true
	case strings.Contains(operation, "OauthService/"):
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_OAUTH, true
	default:
		return adminv1.BaseLoginLogType_BASE_LOGIN_LOG_TYPE_UNSPECIFIED, false
	}
}

// isOperation 判断请求是否属于业务变更操作。
func isOperation(info request) bool {
	if !strings.HasPrefix(info.Operation, "/system.admin.v1.") {
		return false
	}
	if strings.Contains(info.Operation, "LogService/") || strings.Contains(info.Operation, "MigrationService/") {
		return false
	}
	if info.Method == "RPC" {
		methodName := info.Operation
		if index := strings.LastIndex(methodName, "/"); index >= 0 {
			methodName = methodName[index+1:]
		}
		for _, action := range []string{"Create", "Update", "Delete", "Set", "Reset", "Revoke", "Rotate", "Mark", "Archive", "Restore", "Send", "Confirm", "Disable", "Enable", "Publish"} {
			if strings.Contains(info.Operation, "/"+action) || strings.HasSuffix(methodName, action) {
				return true
			}
		}
		return false
	}
	return info.Method == http.MethodPost || info.Method == http.MethodPut || info.Method == http.MethodPatch || info.Method == http.MethodDelete
}

// isDataAccess 判断请求是否产生数据访问事实。
func isDataAccess(info request) bool {
	if info.Operation == "" {
		return false
	}
	if strings.Contains(info.Operation, "LogService/") || strings.Contains(info.Operation, "MigrationService/") {
		return false
	}
	if info.Method == "RPC" {
		for _, action := range []string{"Get", "List", "Page", "Option", "Summary", "Tree", "Count", "Preview"} {
			if strings.Contains(info.Operation, "/"+action) {
				return true
			}
		}
		return false
	}
	return info.Method == http.MethodGet || strings.Contains(info.Operation, "Export") || strings.Contains(info.Operation, "Download") || strings.Contains(info.Operation, "Import")
}

// isPermissionOperation 判断请求是否属于权限对象或权限规则变更。
func isPermissionOperation(info request) bool {
	if !isOperation(info) {
		return false
	}
	return strings.Contains(info.Operation, "RoleService/") || strings.Contains(info.Operation, "MenuService/") || strings.Contains(info.Operation, "ApiService/") || strings.Contains(info.Operation, "Casbin") || strings.Contains(info.Operation, "Permission")
}

// serviceName 提取 operation 中的服务名称。
func serviceName(operation string) string {
	parts := strings.Split(strings.TrimPrefix(operation, "/"), "/")
	if len(parts) != 2 {
		return operation
	}
	serviceParts := strings.Split(parts[0], ".")
	return serviceParts[len(serviceParts)-1]
}

// resourceType 将服务名称转换为稳定的资源类型。
func resourceType(operation string) string {
	name := strings.TrimSuffix(serviceName(operation), "Service")
	if name == "" {
		return "unknown"
	}
	var builder strings.Builder
	for index, char := range name {
		if index > 0 && char >= 'A' && char <= 'Z' {
			builder.WriteByte('_')
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

// operationAction 根据请求方法和 operation 名称映射业务动作。
func operationAction(info request) adminv1.BaseOperationAction {
	if info.Method == "RPC" {
		switch {
		case strings.Contains(info.Operation, "/Create"):
			return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_CREATE
		case strings.Contains(info.Operation, "/Update"), strings.Contains(info.Operation, "/Set"), strings.Contains(info.Operation, "/Reset"):
			return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_UPDATE
		case strings.Contains(info.Operation, "/Delete"):
			return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_DELETE
		}
	}
	switch {
	case strings.Contains(info.Operation, "Publish"):
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_PUBLISH
	case strings.Contains(info.Operation, "Revoke"):
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_REVOKE
	case strings.Contains(info.Operation, "Import"):
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_IMPORT
	case strings.Contains(info.Operation, "Export"):
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_EXPORT
	case info.Method == http.MethodPost:
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_CREATE
	case info.Method == http.MethodPut || info.Method == http.MethodPatch:
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_UPDATE
	case info.Method == http.MethodDelete:
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_DELETE
	default:
		return adminv1.BaseOperationAction_BASE_OPERATION_ACTION_OTHER
	}
}

// accessType 根据 operation 名称映射数据访问类型。
func accessType(info request) adminv1.BaseDataAccessType {
	switch {
	case strings.Contains(info.Operation, "Export"):
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_EXPORT
	case strings.Contains(info.Operation, "Download"):
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_DOWNLOAD
	case strings.Contains(info.Operation, "Import"):
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_IMPORT
	case strings.Contains(info.Operation, "Get"):
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_DETAIL
	case strings.Contains(info.Operation, "Query"):
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_QUERY
	default:
		return adminv1.BaseDataAccessType_BASE_DATA_ACCESS_TYPE_LIST
	}
}

// permissionTargetType 根据 operation 名称映射权限目标类型。
func permissionTargetType(operation string) adminv1.BasePermissionTargetType {
	switch {
	case strings.Contains(operation, "RoleService/"):
		return adminv1.BasePermissionTargetType_BASE_PERMISSION_TARGET_TYPE_ROLE
	case strings.Contains(operation, "MenuService/"):
		return adminv1.BasePermissionTargetType_BASE_PERMISSION_TARGET_TYPE_MENU
	case strings.Contains(operation, "ApiService/"):
		return adminv1.BasePermissionTargetType_BASE_PERMISSION_TARGET_TYPE_API
	case strings.Contains(operation, "TenantService/"):
		return adminv1.BasePermissionTargetType_BASE_PERMISSION_TARGET_TYPE_TENANT
	default:
		return adminv1.BasePermissionTargetType_BASE_PERMISSION_TARGET_TYPE_USER
	}
}

// permissionAction 根据请求方法映射权限变更动作。
func permissionAction(info request) adminv1.BasePermissionAction {
	if info.Method == "RPC" {
		switch {
		case strings.Contains(info.Operation, "/Create"):
			return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_CREATE
		case strings.Contains(info.Operation, "/Update"), strings.Contains(info.Operation, "/Set"):
			return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_UPDATE
		case strings.Contains(info.Operation, "/Delete"):
			return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_DELETE
		}
	}
	switch info.Method {
	case http.MethodPost:
		return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_CREATE
	case http.MethodPut, http.MethodPatch:
		return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_UPDATE
	case http.MethodDelete:
		return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_DELETE
	default:
		return adminv1.BasePermissionAction_BASE_PERMISSION_ACTION_ASSIGN
	}
}

// clientIP 提取客户端地址并去掉端口。
func clientIP(req *http.Request) string {
	// 只记录 TCP 对端地址；未经可信代理校验的转发头可由客户端伪造。
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		return host
	}
	return req.RemoteAddr
}

// newMiddleware 创建并启动 Admin 审计后台工作协程。
func newMiddleware(queue kitQueue.Queue, configCache cache.Cache) *Middleware {
	logMiddleware := &Middleware{queue: queue, configCache: configCache, tasks: make(chan adminTask, adminBufferSize)}
	logMiddleware.worker.Add(1)
	go logMiddleware.run()
	return logMiddleware
}

// createLogRecord 写入审计日志，并将同一队列消息的重复投递视为成功。
func createLogRecord[T any](ctx context.Context, id int64, item *T, create func(context.Context, *T) error, find func(context.Context, int64) (*T, error)) error {
	err := create(ctx, item)
	if err == nil || id == 0 {
		return err
	}
	var existing *T
	var lookupErr error
	existing, lookupErr = find(ctx, id)
	if lookupErr == nil && existing != nil {
		return nil
	}
	return err
}
