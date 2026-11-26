package biz

import (
	"encoding/json"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
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

// TestAiBranchOriginSurvivesRegeneration 验证编辑输入和替换回复后，历史读取仍返回分支来源。
func TestAiBranchOriginSurvivesRegeneration(t *testing.T) {
	message := &models.AiMessage{
		InputContent: `{"content":"原来的问题","branch_origin":{"type":"branch_origin","title":"库存预警","excerpt":"平日与活动日分别统计"}}`,
	}
	input := ai.ParseInputContent(message.InputContent)
	input.Content = "新问题"
	message.InputContent = ai.MarshalInputContentPayload(input)
	message.OutputContent = ai.MarshalReplyContent(&ai.Response{Content: "重新生成的正文", BlocksJSON: `[{"type":"bi_query_result"}]`})
	output := toAiMessageDTO(message).GetOutputContent()

	var blocks []map[string]string
	err := json.Unmarshal([]byte(output.GetBlocksJson()), &blocks)
	if err != nil || len(blocks) != 2 || blocks[0]["type"] != "bi_query_result" ||
		blocks[1]["type"] != "branch_origin" || blocks[1]["title"] != "库存预警" || blocks[1]["excerpt"] != "平日与活动日分别统计" {
		t.Fatalf("分支来源或业务块丢失: %s, %v", output.GetBlocksJson(), err)
	}
}
