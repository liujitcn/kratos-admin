package biz

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-sql-driver/mysql"
	"github.com/liujitcn/kratos-core/errorsx"
)

// TestThirdAccountUniqueConflictUsesConstraintSpecificMessageKey 验证冲突唯一索引对应准确的绑定提示。
func TestThirdAccountUniqueConflictUsesConstraintSpecificMessageKey(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		message    string
		messageKey string
		field      string
		constraint string
	}{
		{
			name:       "duplicate binding",
			err:        mysqlErr("duplicate key"),
			message:    "三方账号绑定关系已存在",
			messageKey: "base.third_account.error.binding_exists",
			field:      "provider,identifier",
		},
		{
			name:       "same user provider",
			err:        mysqlErr("duplicate key unique_base_third_account_user"),
			message:    "当前用户已绑定该登录方式",
			messageKey: "base.third_account.error.current_user_provider_bound",
			field:      "tenant_id,user_id,provider",
			constraint: "unique_base_third_account_user",
		},
		{
			name:       "account bound to another user",
			err:        mysqlErr("duplicate key unique_base_third_account"),
			message:    "三方账号已被其他用户绑定",
			messageKey: "base.third_account.error.account_bound_to_other_user",
			field:      "provider,identifier",
			constraint: "unique_base_third_account",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			structured := errors.FromError(thirdAccountUniqueConflict(test.err))
			if structured.Message != test.message {
				t.Errorf("conflict message = %q, want %q", structured.Message, test.message)
			}
			if structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != test.messageKey {
				t.Errorf("message key = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY], test.messageKey)
			}
			if structured.Metadata[errorsx.METADATA_KEY_CONSTRAINT] != test.constraint {
				t.Errorf("constraint = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_CONSTRAINT], test.constraint)
			}
			if structured.Metadata[errorsx.METADATA_KEY_FIELD] != test.field {
				t.Errorf("field = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_FIELD], test.field)
			}
		})
	}
}

// mysqlErr 创建只包含驱动原始错误消息的唯一键测试错误。
func mysqlErr(message string) error {
	return &mysql.MySQLError{Number: 1062, Message: message}
}
