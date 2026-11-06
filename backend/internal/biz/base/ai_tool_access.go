package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/server/middleware"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// PrepareToolCall 在工具实际执行前重新校验 API 权限、租户和 Proto 参数。
func (h *McpCase) PrepareToolCall(ctx context.Context, terminal string, name string, arguments string) (string, bool, error) {
	if !matchToolTerminal(terminal, name) {
		return "", false, errorsx.PermissionDenied("工具不属于当前终端")
	}
	query := h.baseAPIRepo.Query(ctx).BaseAPI
	list, err := h.baseAPIRepo.List(ctx, repository.Where(query.ToolName.Eq(name)))
	if err != nil {
		return "", false, err
	}
	if len(list) == 0 {
		return "", false, errorsx.PermissionDenied("Agent工具未启用")
	}
	requiresApproval := false
	for _, item := range list {
		if agentToolBlocked(item) || item.AgentStatus != _const.STATUS_STATUS_ENABLE {
			return "", false, errorsx.PermissionDenied("Agent工具未启用或执行能力被禁止")
		}
		err = h.authorizeToolCall(ctx, item)
		if err != nil {
			return "", false, err
		}
		requiresApproval = requiresApproval || !agentToolReadOnly(item)
	}
	if terminal == "app" && requiresApproval {
		return "", false, errorsx.PermissionDenied("请在管理端确认业务写入操作")
	}
	var value string
	value, err = prepareAgentToolArguments(ctx, list[0].Operation, arguments)
	return value, requiresApproval, err
}

// prepareAgentToolArguments 在进程内调用边界应用 HTTP/gRPC 同等的 Proto 校验和租户补全。
func prepareAgentToolArguments(ctx context.Context, operation string, arguments string) (string, error) {
	operation = strings.ReplaceAll(strings.TrimPrefix(operation, "/"), "/", ".")
	var err error
	var descriptor protoreflect.Descriptor
	descriptor, err = protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(operation))
	if err != nil {
		return "", errorsx.Internal("工具接口契约不存在").WithCause(err)
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return "", errorsx.Internal("工具接口契约无效")
	}
	var messageType protoreflect.MessageType
	messageType, err = protoregistry.GlobalTypes.FindMessageByName(method.Input().FullName())
	if err != nil {
		return "", errorsx.Internal("工具请求契约不存在").WithCause(err)
	}
	request := messageType.New().Interface()
	if err = protojson.Unmarshal([]byte(arguments), request); err != nil {
		return "", errorsx.InvalidArgument("工具参数无效").WithCause(err)
	}
	validate := middleware.NewTenantScopeMiddleware()(middleware.NewValidateMiddleware()(func(_ context.Context, request any) (any, error) {
		return json.Marshal(request)
	}))
	var value any
	value, err = validate(ctx, request)
	if err != nil {
		return "", err
	}
	return string(value.([]byte)), nil
}

// agentToolReadOnly 仅自动放行具有读取语义的 HTTP 接口，其他接口保守要求确认。
func agentToolReadOnly(api *models.BaseAPI) bool {
	return api.Method == "GET" || api.Method == "HEAD"
}

// agentToolBlocked 封闭未经沙箱隔离的源码执行，以及能替模型操作审批会话的入口。
func agentToolBlocked(api *models.BaseAPI) bool {
	return strings.HasPrefix(api.ToolName, "base_v1_ai_message_service_") ||
		strings.HasPrefix(api.ToolName, "base_v1_ai_session_service_") ||
		(!agentToolReadOnly(api) && (strings.Contains(api.Operation, ".BaseScriptService/") || strings.Contains(api.Operation, ".CodeGenService/") || strings.Contains(api.Operation, ".BaseJobService/") || strings.Contains(api.Operation, ".Workflow")))
}
