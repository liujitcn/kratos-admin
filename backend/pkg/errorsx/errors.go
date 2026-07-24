package errorsx

import (
	stderrs "errors"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	// METADATA_KEY_CONFLICT_TYPE 标识冲突类型。
	METADATA_KEY_CONFLICT_TYPE = "conflict_type"
	// METADATA_KEY_RESOURCE 标识资源名称。
	METADATA_KEY_RESOURCE = "resource"
	// METADATA_KEY_FIELD 标识字段名称。
	METADATA_KEY_FIELD = "field"
	// METADATA_KEY_CONSTRAINT 标识数据库约束名称。
	METADATA_KEY_CONSTRAINT = "constraint"
	// METADATA_KEY_CHILD_RESOURCE 标识子资源名称。
	METADATA_KEY_CHILD_RESOURCE = "child_resource"
	// METADATA_KEY_CURRENT_STATE 标识当前状态。
	METADATA_KEY_CURRENT_STATE = "current_state"
	// METADATA_KEY_EXPECTED_STATE 标识期望状态。
	METADATA_KEY_EXPECTED_STATE = "expected_state"

	// CONFLICT_TYPE_UNIQUE_VIOLATION 表示唯一约束冲突。
	CONFLICT_TYPE_UNIQUE_VIOLATION = "unique_violation"
	// CONFLICT_TYPE_HAS_CHILDREN 表示仍存在子资源。
	CONFLICT_TYPE_HAS_CHILDREN = "has_children"
	// CONFLICT_TYPE_STATE_CONFLICT 表示状态冲突。
	CONFLICT_TYPE_STATE_CONFLICT = "state_conflict"
	// CONFLICT_TYPE_PROTECTED_RESOURCE 表示受保护资源。
	CONFLICT_TYPE_PROTECTED_RESOURCE = "protected_resource"
)

// InvalidArgument 构造请求参数错误。
func InvalidArgument(message string) *kratosErrors.Error {
	return kratosErrors.New(400, commonv1.ErrorReason_INVALID_ARGUMENT.String(), message)
}

// Unauthenticated 构造认证失败错误。
func Unauthenticated(message string) *kratosErrors.Error {
	return kratosErrors.New(401, commonv1.ErrorReason_UNAUTHENTICATED.String(), message)
}

// PermissionDenied 构造权限不足错误。
func PermissionDenied(message string) *kratosErrors.Error {
	return kratosErrors.New(403, commonv1.ErrorReason_PERMISSION_DENIED.String(), message)
}

// ResourceNotFound 构造资源不存在错误。
func ResourceNotFound(message string) *kratosErrors.Error {
	return kratosErrors.New(404, commonv1.ErrorReason_RESOURCE_NOT_FOUND.String(), message)
}

// Conflict 构造状态冲突错误。
func Conflict(message string) *kratosErrors.Error {
	return kratosErrors.New(409, commonv1.ErrorReason_CONFLICT.String(), message)
}

// Internal 构造内部错误。
func Internal(message string) *kratosErrors.Error {
	return kratosErrors.New(500, commonv1.ErrorReason_INTERNAL_ERROR.String(), message)
}

// UniqueConflict 构造唯一约束冲突错误。
func UniqueConflict(message, resource, field, constraint string) *kratosErrors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_UNIQUE_VIOLATION,
		METADATA_KEY_RESOURCE:      resource,
		METADATA_KEY_FIELD:         field,
	}
	// 提供了约束名时，再补充到错误元数据中。
	if constraint != "" {
		metadata[METADATA_KEY_CONSTRAINT] = constraint
	}
	return Conflict(message).WithMetadata(metadata)
}

// HasChildrenConflict 构造存在子资源的冲突错误。
func HasChildrenConflict(message, resource, childResource string) *kratosErrors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_HAS_CHILDREN,
		METADATA_KEY_RESOURCE:      resource,
	}
	// 已知子资源名称时，再补充到错误元数据中。
	if childResource != "" {
		metadata[METADATA_KEY_CHILD_RESOURCE] = childResource
	}
	return Conflict(message).WithMetadata(metadata)
}

// StateConflict 构造状态冲突错误。
func StateConflict(message, resource, currentState, expectedState string) *kratosErrors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_STATE_CONFLICT,
		METADATA_KEY_RESOURCE:      resource,
	}
	// 提供了当前状态时，再补充到错误元数据中。
	if currentState != "" {
		metadata[METADATA_KEY_CURRENT_STATE] = currentState
	}
	// 提供了期望状态时，再补充到错误元数据中。
	if expectedState != "" {
		metadata[METADATA_KEY_EXPECTED_STATE] = expectedState
	}
	return Conflict(message).WithMetadata(metadata)
}

// ProtectedResourceConflict 构造受保护资源冲突错误。
func ProtectedResourceConflict(message, resource string) *kratosErrors.Error {
	metadata := map[string]string{
		METADATA_KEY_CONFLICT_TYPE: CONFLICT_TYPE_PROTECTED_RESOURCE,
	}
	// 提供了资源名称时，再补充到错误元数据中。
	if resource != "" {
		metadata[METADATA_KEY_RESOURCE] = resource
	}
	return Conflict(message).WithMetadata(metadata)
}

// IsStructuredError 判断错误是否已经携带稳定 reason。
func IsStructuredError(err error) bool {
	var kratosErr *kratosErrors.Error
	// 已经是 Kratos 错误且 reason 非空时，视为已完成分类。
	return stderrs.As(err, &kratosErr) && kratosErr.Reason != ""
}

// WrapIfNeeded 在错误尚未完成分类时，使用兜底错误包装。
func WrapIfNeeded(err error, fallback *kratosErrors.Error) error {
	if err == nil {
		return nil
	}
	// 已经完成分类的错误直接透传，避免覆盖原始语义。
	if IsStructuredError(err) {
		return err
	}
	if fallback == nil {
		return err
	}
	return fallback.WithCause(err)
}

// WrapInternal 在错误尚未完成分类时，包装成内部错误。
func WrapInternal(err error, message string) error {
	// 仓储层统一返回 gorm.ErrRecordNotFound，这类可预期查询空结果应对外表现为资源不存在。
	if stderrs.Is(err, gorm.ErrRecordNotFound) {
		return ResourceNotFound(message).WithCause(err)
	}
	return WrapIfNeeded(err, Internal(message))
}

// IsMySQLDuplicateKey 判断是否命中了 MySQL 唯一键冲突。
func IsMySQLDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return stderrs.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
