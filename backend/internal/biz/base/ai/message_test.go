package ai

import (
	"encoding/json"
	"testing"
)

// TestMarshalReplyContentNilReturnsValidJSON 验证空助手回复仍返回可入库的 JSON。
func TestMarshalReplyContentNilReturnsValidJSON(t *testing.T) {
	raw := MarshalReplyContent(nil)
	if raw != MarshalEmptyOutputContent() {
		t.Fatalf("nil 回复 = %q，期望 %q", raw, MarshalEmptyOutputContent())
	}
	if !json.Valid([]byte(raw)) {
		t.Fatalf("nil 回复不是合法 JSON: %q", raw)
	}
}
