package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	httpTransport "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/go-utils/id"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreQueue "github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-kit/auth"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

const adminEventStream queueTransport.Stream = "admin.audit.event"

// AdminEventStream 返回 Admin 业务审计事件队列名称。
func AdminEventStream() queueTransport.Stream {
	return adminEventStream
}

// adminEvent 是 Admin 业务审计队列的事件包装，不与 Core 审计事件混用。
type adminEvent struct {
	Kind    string `json:"kind"`    // 业务审计类型。
	Payload string `json:"payload"` // 对应 Admin 日志模型的 JSON 载荷。
}

// Middleware 记录 Admin 请求对应的登录、操作、数据访问和权限审计事实。
// Core 产生的 API/策略事件走 Core Sink；本中间件只处理 Admin 自身业务事件。
type Middleware struct{}

// NewMiddleware 创建审计日志中间件。
func NewMiddleware() middleware.Middleware {
	return (&Middleware{}).Handle
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
		ctx := context.Background()
		var envelope adminEvent
		if err := json.Unmarshal([]byte(rawValue), &envelope); err != nil {
			return fmt.Errorf("解析 Admin 审计事件失败: %w", err)
		}
		if envelope.Payload == "" {
			return fmt.Errorf("Admin 审计事件载荷为空")
		}
		event := envelope
		switch event.Kind {
		case "login":
			item := &models.BaseLoginLog{}
			if err := json.Unmarshal([]byte(event.Payload), item); err != nil {
				return fmt.Errorf("解析 Admin 登录审计事件失败: %w", err)
			}
			return loginLogRepo.Create(ctx, item)
		case "operation":
			item := &models.BaseOperationLog{}
			if err := json.Unmarshal([]byte(event.Payload), item); err != nil {
				return fmt.Errorf("解析 Admin 操作审计事件失败: %w", err)
			}
			return operationLogRepo.Create(ctx, item)
		case "data_access":
			item := &models.BaseDataAccessLog{}
			if err := json.Unmarshal([]byte(event.Payload), item); err != nil {
				return fmt.Errorf("解析 Admin 数据访问审计事件失败: %w", err)
			}
			return dataAccessLogRepo.Create(ctx, item)
		case "permission":
			item := &models.BasePermissionLog{}
			if err := json.Unmarshal([]byte(event.Payload), item); err != nil {
				return fmt.Errorf("解析 Admin 权限审计事件失败: %w", err)
			}
			return permissionLogRepo.Create(ctx, item)
		default:
			return fmt.Errorf("未知 Admin 审计事件类型: %s", event.Kind)
		}
	}
}

// Handle 记录一次请求的登录、操作、数据访问和权限审计信息。
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
		startedAt := time.Now()
		request := requestInfo(ctx)
		reply, err = next(ctx, req)
		result, reasonCode, reason, statusCode := resultInfo(err)
		request.LatencyMs = int32(time.Since(startedAt).Milliseconds())
		request.StatusCode = statusCode
		request.Result = result
		request.ReasonCode = reasonCode
		request.Reason = reason
		request.OccurredAt = startedAt
		m.writeLogin(request, req)
		m.writeOperation(request, req)
		m.writeDataAccess(request)
		m.writePermission(request, req)
		return reply, err
	}
}

// request 汇总一次 HTTP 或 gRPC 请求的审计上下文。
// 该结构只在中间件生命周期内使用，最终会被转换为具体审计日志模型。
type request struct {
	Operation   string    // Kratos operation 标识。
	ServiceName string    // 从 operation 提取的服务名称。
	Method      string    // HTTP 方法或 RPC 标识。
	Path        string    // HTTP 路径模板。
	RequestID   string    // 请求追踪编号。
	TraceID     string    // 分布式链路追踪编号。
	TenantID    int64     // 当前租户编号。
	TenantCode  string    // 当前租户编码。
	UserID      int64     // 当前用户编号。
	UserName    string    // 当前用户账号。
	ClientIP    string    // 对端 IP 地址。
	UserAgent   string    // 客户端 User-Agent。
	StatusCode  int32     // 对外返回状态码。
	Result      int32     // 审计结果枚举值。
	ReasonCode  string    // 稳定错误原因码。
	Reason      string    // 脱敏后的错误描述。
	LatencyMs   int32     // 请求耗时，单位毫秒。
	OccurredAt  time.Time // 请求开始时间。
}

// requestInfo 从服务端传输和认证上下文提取通用审计字段。
func requestInfo(ctx context.Context) request {
	info := request{RequestID: id.NewGUIDv4NoHyphen(), TenantCode: databaseGorm.DefaultTenantCode, Method: "RPC"}
	if serverTransport, ok := transport.FromServerContext(ctx); ok {
		info.Operation = serverTransport.Operation()
		info.ServiceName = serviceName(info.Operation)
		if htr, ok := serverTransport.(*httpTransport.Transport); ok && htr.Request() != nil {
			httpRequest := htr.Request()
			info.Method = httpRequest.Method
			info.Path = htr.PathTemplate()
			if value := httpRequest.Header.Get("X-Request-ID"); value != "" {
				info.RequestID = value
			}
			info.TraceID = httpRequest.Header.Get("traceparent")
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
func resultInfo(err error) (int32, string, string, int32) {
	if err == nil {
		return int32(adminv1.BaseAuditResult_BASE_AUDIT_RESULT_SUCCESS), "", "", http.StatusOK
	}
	statusCode := int32(http.StatusInternalServerError)
	reasonCode := "INTERNAL_ERROR"
	reason := err.Error()
	if structuredErr := kratosErrors.FromError(err); structuredErr != nil {
		statusCode = structuredErr.Code
		reasonCode = structuredErr.Reason
		reason = structuredErr.Message
	}
	result := adminv1.BaseAuditResult_BASE_AUDIT_RESULT_FAILURE
	if statusCode >= http.StatusInternalServerError {
		result = adminv1.BaseAuditResult_BASE_AUDIT_RESULT_ERROR
	}
	return int32(result), reasonCode, reason, statusCode
}

// writeLogin 写入登录、退出、刷新令牌和 OAuth 认证日志。
func (m *Middleware) writeLogin(info request, req interface{}) {
	loginType, ok := loginType(info.Operation)
	if !ok {
		return
	}
	userName := info.UserName
	tenantCode := info.TenantCode
	if loginRequest, ok := req.(interface {
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
	if clientRequest, ok := req.(interface{ GetClientId() string }); ok && clientRequest.GetClientId() != "" {
		userName = clientRequest.GetClientId()
	}
	item := &models.BaseLoginLog{
		TenantID: info.TenantID, TenantCode: tenantCode, UserID: info.UserID, UserName: userName,
		LoginType: int32(loginType), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
		ClientIP: info.ClientIP, UserAgent: info.UserAgent, RequestID: info.RequestID, TraceID: info.TraceID,
		OccurredAt: info.OccurredAt, CreatedAt: time.Now(),
	}
	m.emit("login", item, info.Operation)
}

// writeOperation 写入具有业务变更语义的操作日志。
func (m *Middleware) writeOperation(info request, req interface{}) {
	if !isOperation(info) {
		return
	}
	afterData, changedFields := auditSnapshot(req)
	resourceID, resourceName := auditResource(req)
	item := &models.BaseOperationLog{
		TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName,
		RequestID: info.RequestID, TraceID: info.TraceID, ResourceType: resourceType(info.Operation), ResourceID: resourceID, ResourceName: resourceName,
		ChangedFields: changedFields, BeforeData: "{}", AfterData: afterData,
		Action: int32(operationAction(info)), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
		OccurredAt: info.OccurredAt, CreatedAt: time.Now(),
	}
	m.emit("operation", item, info.Operation)
}

// writeDataAccess 写入查询、详情、导入导出等数据访问摘要。
func (m *Middleware) writeDataAccess(info request) {
	if !isDataAccess(info) {
		return
	}
	item := &models.BaseDataAccessLog{
		TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName,
		RequestID: info.RequestID, TraceID: info.TraceID, ResourceType: resourceType(info.Operation),
		AccessType: int32(accessType(info)), DataSource: databaseGorm.DefaultClientName, FieldScope: "{}", Result: info.Result,
		ReasonCode: info.ReasonCode, LatencyMs: info.LatencyMs, OccurredAt: info.OccurredAt, CreatedAt: time.Now(),
	}
	m.emit("data_access", item, info.Operation)
}

// writePermission 写入角色、菜单和 API 权限相关变更日志。
func (m *Middleware) writePermission(info request, req interface{}) {
	if !isPermissionOperation(info) {
		return
	}
	newValue, _ := auditSnapshot(req)
	targetID, targetName := auditResource(req)
	item := &models.BasePermissionLog{
		TenantID: info.TenantID, TenantCode: info.TenantCode, UserID: info.UserID, UserName: info.UserName,
		RequestID: info.RequestID, TraceID: info.TraceID, TargetType: int32(permissionTargetType(info.Operation)), TargetID: targetID, TargetName: targetName,
		OldValue: "{}", NewValue: newValue,
		Action: int32(permissionAction(info)), Result: info.Result, ReasonCode: info.ReasonCode, Reason: info.Reason,
		OccurredAt: info.OccurredAt, CreatedAt: time.Now(),
	}
	m.emit("permission", item, info.Operation)
}

// auditSnapshot 将请求对象转换为脱敏 JSON，并提取顶层变更字段。
func auditSnapshot(value interface{}) (string, string) {
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
	payload = redactAuditValue(payload)
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

// redactAuditValue 递归移除审计快照中的凭据、令牌和验证码。
func redactAuditValue(value interface{}) interface{} {
	switch item := value.(type) {
	case map[string]interface{}:
		for key, child := range item {
			if isSensitiveAuditField(key) {
				item[key] = "[REDACTED]"
				continue
			}
			item[key] = redactAuditValue(child)
		}
	case []interface{}:
		for index, child := range item {
			item[index] = redactAuditValue(child)
		}
	}
	return value
}

// isSensitiveAuditField 判断字段名是否包含不应进入审计快照的敏感值。
func isSensitiveAuditField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{"password", "old_pwd", "new_pwd", "pwd", "client_secret", "crypto_key", "encrypted_key", "ciphertext", "nonce", "access_token", "refresh_token", "authorization", "captcha_code", "verification_code", "private_key", "content", "action_params"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "iv" || strings.HasSuffix(normalized, "_iv")
}

// auditResource 从请求快照中提取资源编号和名称，便于审计人员定位对象。
func auditResource(value interface{}) (string, string) {
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

// emit 将 Admin 业务审计事件投递到异步入库队列。
func (m *Middleware) emit(kind string, item interface{}, operation string) {
	payload, err := json.Marshal(item)
	if err != nil {
		log.Error("序列化 Admin 审计事件失败", "error", err, "operation", operation, "kind", kind)
		return
	}
	if !coreQueue.AddQueue(adminEventStream, adminEvent{Kind: kind, Payload: string(payload)}) {
		log.Error("投递 Admin 审计事件失败", "operation", operation, "kind", kind)
	}
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
		for _, action := range []string{"Create", "Update", "Delete", "Set", "Reset", "Revoke", "Rotate", "Mark", "Archive", "Restore", "Send", "Confirm", "Disable", "Enable", "Publish"} {
			if strings.Contains(info.Operation, "/"+action) {
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
