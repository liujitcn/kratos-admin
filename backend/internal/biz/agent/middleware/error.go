package middleware

import (
	"encoding/json"
	"errors"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-core/errorsx"
)

// ErrToolOutcomeUnknown 表示工具已调用但没有得到可确认的结果。
var ErrToolOutcomeUnknown = errors.New("工具调用结果未知")

// toolErrorWithDetails 携带业务显式允许展示给模型和用户的失败说明。
type toolErrorWithDetails struct {
	cause   error
	message string
	details map[string]any
}

// WithToolErrorDetails 附加已按当前用户授权核实的工具回应字段，保留原错误分类。
func WithToolErrorDetails(err error, message string, details map[string]any) error {
	return &toolErrorWithDetails{cause: err, message: message, details: details}
}

// Error 返回原始业务错误说明。
func (e *toolErrorWithDetails) Error() string { return e.cause.Error() }

// Unwrap 供错误分类检查原始原因。
func (e *toolErrorWithDetails) Unwrap() error { return e.cause }

// MarshalToolError 将工具错误转换成稳定 JSON 文本。
func MarshalToolError(message string) string {
	raw, err := json.Marshal(map[string]string{"error": message})
	// 理论上 map[string]string 不会序列化失败；保留兜底是为了稳定工具协议。
	if err != nil {
		return `{"error":"tool execution failed"}`
	}
	return string(raw)
}

// describeToolFailure 保留可信权限原因及显式公开字段，隐藏底层错误文案和元数据。
func describeToolFailure(err error, message string) (string, string) {
	var reason string
	if kratosErrors.Reason(err) == errorsx.ReasonPermissionDenied {
		reason = errorsx.ReasonPermissionDenied
	}
	var details map[string]any
	if visible, ok := errors.AsType[*toolErrorWithDetails](err); ok {
		message = visible.message
		details = visible.details
	}
	if reason == "" && len(details) == 0 {
		return message, MarshalToolError(message)
	}
	raw, marshalErr := json.Marshal(struct {
		Error   string         `json:"error"`
		Reason  string         `json:"reason,omitempty"`
		Details map[string]any `json:"details,omitempty"`
	}{Error: message, Reason: reason, Details: details})
	if marshalErr != nil {
		return message, MarshalToolError(message)
	}
	return message, string(raw)
}
