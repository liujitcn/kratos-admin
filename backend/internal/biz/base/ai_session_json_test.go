package biz

import (
	"encoding/json"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestNormalizeAiMessageJSONUsesValidDefaults 验证分支复制的空消息 JSON 使用正确默认值。
func TestNormalizeAiMessageJSONUsesValidDefaults(t *testing.T) {
	item := &models.AiMessage{}
	normalizeAiMessageJSON(item)
	values := map[string]string{
		"input_content":  item.InputContent,
		"output_content": item.OutputContent,
		"attachments":    item.Attachments,
		"tools":          item.Tools,
		"token":          item.Token,
	}
	for field, value := range values {
		if !json.Valid([]byte(value)) {
			t.Fatalf("%s 不是合法 JSON: %q", field, value)
		}
	}
	if item.Attachments != "[]" || item.Tools != "[]" {
		t.Fatalf("数组字段默认值错误: attachments=%q tools=%q", item.Attachments, item.Tools)
	}
}
