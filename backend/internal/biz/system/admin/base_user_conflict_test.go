package biz

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/errorsx"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestBaseUserUniqueConflictReportsActualIndex 验证用户账号与编号冲突分别指出具体字段。
func TestBaseUserUniqueConflictReportsActualIndex(t *testing.T) {
	catalog, err := corei18n.NewI18n("base-user-conflict-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		index      string
		field      string
		messageKey string
		message    string
	}{
		{
			name:       "username",
			index:      "unique_base_user_ user_name",
			field:      "user_name",
			messageKey: "system.base.user.error.duplicate_user_name",
			message:    "This username is already in use in the tenant",
		},
		{
			name:       "user code",
			index:      "unique_base_user_ user_code",
			field:      "user_code",
			messageKey: "system.base.user.error.duplicate_user_code",
			message:    "This user code is already in use in the tenant",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conflict := baseUserUniqueConflict(&mysql.MySQLError{
				Number:  1062,
				Message: "Duplicate entry 'masked-value' for key '" + test.index + "'",
			})
			structured := errors.FromError(conflict)
			if structured.Metadata[errorsx.METADATA_KEY_FIELD] != test.field {
				t.Fatalf("field = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_FIELD], test.field)
			}
			if structured.Metadata[errorsx.METADATA_KEY_CONSTRAINT] != test.index {
				t.Fatalf("constraint = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_CONSTRAINT], test.index)
			}
			if structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != test.messageKey {
				t.Fatalf("message key = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY], test.messageKey)
			}
			localized := errors.FromError(corei18n.LocalizeError(catalog, "en-US", "zh-CN", conflict))
			if localized.Message != test.message {
				t.Errorf("message = %q, want %q", localized.Message, test.message)
			}
		})
	}
}
