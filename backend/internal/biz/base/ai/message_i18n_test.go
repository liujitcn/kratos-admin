package ai

import (
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// TestBuildUserContentUsesLocalizedAttachmentFallback 验证纯附件消息保留调用方提供的本地化正文。
func TestBuildUserContentUsesLocalizedAttachmentFallback(t *testing.T) {
	attachments := []*basev1.AiAttachment{{}}
	if got, want := BuildUserContent("", attachments, "Analyze these attachments"), "Analyze these attachments"; got != want {
		t.Fatalf("attachment-only content = %q, want %q", got, want)
	}
	if got, want := BuildUserContent("Question", attachments, "Analyze these attachments"), "Question"; got != want {
		t.Fatalf("text content = %q, want %q", got, want)
	}
	if got := BuildUserContent("", nil, "Analyze these attachments"); got != "" {
		t.Fatalf("empty content without attachments = %q, want empty", got)
	}
}

// TestBuildDynamicSummaryUsesLocalizedFallback 验证空会话摘要由调用方按当前语言传入。
func TestBuildDynamicSummaryUsesLocalizedFallback(t *testing.T) {
	if got, want := BuildDynamicSummary("", nil, "New conversation", "2 attachments"), "New conversation"; got != want {
		t.Fatalf("default summary = %q, want %q", got, want)
	}
	if got, want := BuildDynamicSummary("", []*basev1.AiAttachment{{}}, "New conversation", "1 attachment"), "1 attachment"; got != want {
		t.Fatalf("attachment summary = %q, want %q", got, want)
	}
	if got, want := BuildDynamicSummary("A question", nil, "New conversation", "2 attachments"), "A question"; got != want {
		t.Fatalf("question summary = %q, want %q", got, want)
	}
}
