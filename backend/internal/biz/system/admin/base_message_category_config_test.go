package biz

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestMessageProviderIDsJSON 验证消息分类 Provider ID JSON 的稳定转换。
func TestMessageProviderIDsJSON(t *testing.T) {
	raw, err := encodeProviderIDs([]int64{3, 7})
	if err != nil {
		t.Fatal(err)
	}
	if raw != `[3,7]` {
		t.Fatalf("encodeProviderIDs returned %q", raw)
	}
	ids, err := decodeMessageProviderIDs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 7 {
		t.Fatalf("decodeMessageProviderIDs returned %#v", ids)
	}
}

// TestBaseMessageCategoryToFormProviderIDs 验证编辑表单返回 Provider ID 数组。
func TestBaseMessageCategoryToFormProviderIDs(t *testing.T) {
	categoryCase := NewBaseMessageCategoryCase(nil, nil, nil, nil, nil, nil, nil)
	form, err := categoryCase.toForm(&models.BaseMessageCategory{ID: 42, ProviderID: `[7,9]`, InboxEnabled: int32(adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE)})
	if err != nil {
		t.Fatal(err)
	}
	if form.GetId() != 42 || len(form.GetProviderId()) != 2 || form.GetProviderId()[0] != 7 || form.GetProviderId()[1] != 9 {
		t.Fatalf("unexpected category form: id=%d provider_ids=%v", form.GetId(), form.GetProviderId())
	}
}
