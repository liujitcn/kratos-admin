package message

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// TestImageBudgetDoesNotCountBase64AsText 验证图片按视觉输入预算，不把传输编码误算为几十万文本Token。
func TestImageBudgetDoesNotCountBase64AsText(t *testing.T) {
	input := UserParts([]*schema.ContentBlock{schema.NewContentBlock(&schema.UserInputImage{Base64Data: strings.Repeat("a", 1024*1024), MIMEType: "image/png"})})
	tokens := EstimateTokens([]*schema.AgenticMessage{input}, nil)
	if tokens < 8192 || tokens > 16384 {
		t.Fatalf("图片估算混入Base64文本: %d", tokens)
	}
	if input.ContentBlocks[0].UserInputImage.Base64Data == "" {
		t.Fatal("估算修改了原始图片")
	}
}
