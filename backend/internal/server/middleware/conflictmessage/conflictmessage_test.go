package conflictmessage

import (
	"context"
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"
)

type conflictTestTransport struct {
	transport.Transporter
	operation string
}

// Operation 返回用于验证冲突路径回退的请求操作。
func (t conflictTestTransport) Operation() string {
	return t.operation
}

// TestConflictMessageKeyMiddlewarePreservesUniqueConflictDetail 验证唯一冲突的具体文案和元数据不会被通用提示覆盖。
func TestConflictMessageKeyMiddlewarePreservesUniqueConflictDetail(t *testing.T) {
	message := "手机号已被占用"
	messageKey := errorsx.MessageKey(message)
	locales := []struct {
		locale   string
		generic  string
		detailed string
	}{
		{locale: "zh-CN", generic: "资源状态冲突", detailed: message},
		{locale: "en-US", generic: "Resource state conflict", detailed: "The phone number is already in use"},
		{locale: "zh-TW", generic: "資源狀態衝突", detailed: "手機號已被佔用"},
		{locale: "ja-JP", generic: "リソースの状態がこの操作と競合しています", detailed: "電話番号はすでに使用されています"},
	}
	localeFiles := make(fstest.MapFS, len(locales))
	for _, locale := range locales {
		data, marshalErr := json.Marshal(map[string]map[string]string{
			"common.error.conflict": {"other": locale.generic},
			messageKey:              {"other": locale.detailed},
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		localeFiles[locale.locale+".json"] = &fstest.MapFile{Data: data}
	}
	catalog, err := i18n.NewI18n("conflict-message-test", localeFiles)
	if err != nil {
		t.Fatal(err)
	}
	middleware := NewConflictMessageKeyMiddleware()
	handler := middleware(func(context.Context, any) (any, error) {
		return nil, errorsx.UniqueConflict(message, "base_user", "phone", "unique_base_user_phone")
	})
	_, err = handler(context.Background(), nil)
	if err == nil {
		t.Fatal("唯一冲突错误不能丢失")
	}
	for _, locale := range locales {
		localized := errors.FromError(i18n.LocalizeError(catalog, locale.locale, "zh-CN", err))
		if localized.Message != locale.detailed {
			t.Errorf("locale %s conflict message = %q, want %q", locale.locale, localized.Message, locale.detailed)
		}
	}
	localized := errors.FromError(i18n.LocalizeError(catalog, "zh-CN", "zh-CN", err))
	if localized.Metadata[errorsx.METADATA_KEY_CONFLICT_TYPE] != errorsx.CONFLICT_TYPE_UNIQUE_VIOLATION {
		t.Fatalf("conflict type = %q", localized.Metadata[errorsx.METADATA_KEY_CONFLICT_TYPE])
	}
	if localized.Metadata[errorsx.METADATA_KEY_RESOURCE] != "base_user" {
		t.Fatalf("conflict resource = %q", localized.Metadata[errorsx.METADATA_KEY_RESOURCE])
	}
	if localized.Metadata[errorsx.METADATA_KEY_FIELD] != "phone" {
		t.Fatalf("conflict field = %q", localized.Metadata[errorsx.METADATA_KEY_FIELD])
	}
}

// TestConflictMessageKeyMiddlewarePreservesExplicitMessageKey 验证已有业务消息键不会被兼容处理覆盖。
func TestConflictMessageKeyMiddlewarePreservesExplicitMessageKey(t *testing.T) {
	const messageKey = "system.code.gen.error.route_duplicate"
	conflict := errorsx.WithMessageKey(errorsx.Conflict("生成接口与已有 API 重复"), messageKey, map[string]string{"Method": "CreateUser"})
	handler := NewConflictMessageKeyMiddleware()(func(context.Context, any) (any, error) {
		return nil, conflict
	})
	_, err := handler(context.Background(), nil)
	if err == nil {
		t.Fatal("冲突错误不能丢失")
	}
	structured := errors.FromError(err)
	if structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != messageKey {
		t.Fatalf("message key = %q, want %q", structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY], messageKey)
	}
	if structured.Metadata[errorsx.METADATA_KEY_MESSAGE_ARGS] != `{"Method":"CreateUser"}` {
		t.Fatalf("message args = %q", structured.Metadata[errorsx.METADATA_KEY_MESSAGE_ARGS])
	}
}

// TestConflictMessageKeyMiddlewareAddsOperationWhenLocationIsMissing 验证缺少业务位置时至少返回具体接口路径。
func TestConflictMessageKeyMiddlewareAddsOperationWhenLocationIsMissing(t *testing.T) {
	const operation = "/api/v1/system/admin/users"
	ctx := transport.NewServerContext(context.Background(), conflictTestTransport{operation: operation})
	handler := NewConflictMessageKeyMiddleware()(func(context.Context, any) (any, error) {
		return nil, errorsx.Conflict("资源状态冲突")
	})
	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatal("冲突错误不能丢失")
	}
	structured := errors.FromError(err)
	if structured.Metadata["operation"] != operation {
		t.Fatalf("operation = %q, want %q", structured.Metadata["operation"], operation)
	}
	if structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != errorsx.MessageKey(structured.Message) {
		t.Fatalf("message key = %q", structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY])
	}
}
