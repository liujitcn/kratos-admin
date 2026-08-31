package biz

import (
	"context"
	"strings"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	messagepublisher "github.com/liujitcn/kratos-admin/backend/pkg/notification"
)

func TestSanitizeMessageContent(t *testing.T) {
	content, err := sanitizeMessageContent(
		`<p>hello</p><script>alert(1)</script><a href="javascript:alert(2)">link</a>`,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "script") || strings.Contains(content, "javascript:") {
		t.Fatalf("sanitized content still contains executable markup: %s", content)
	}
	if !strings.Contains(content, "hello") {
		t.Fatalf("sanitized content lost safe text: %s", content)
	}
}

// TestSanitizeSafeMarkdown 验证 Markdown 语法保留且原生可执行 HTML 被清除。
func TestSanitizeSafeMarkdown(t *testing.T) {
	content, err := sanitizeMessageContent(
		"# title\n\n**bold** <script>alert(1)</script><img src=x onerror=alert(2)>",
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "# title") || !strings.Contains(content, "**bold**") {
		t.Fatalf("sanitized content lost markdown syntax: %s", content)
	}
	if strings.Contains(content, "<script") || strings.Contains(content, "<img") || strings.Contains(content, "onerror") {
		t.Fatalf("sanitized content still contains raw executable HTML: %s", content)
	}
}

// TestNormalizeMessageActionParams 验证空动作参数使用合法 JSON 对象落库。
func TestNormalizeMessageActionParams(t *testing.T) {
	if normalized := normalizeMessageActionParams(""); normalized != "{}" {
		t.Fatalf("empty action params normalized to %q", normalized)
	}
	if normalized := normalizeMessageActionParams(`{"id":1}`); normalized != `{"id":1}` {
		t.Fatalf("valid action params changed to %q", normalized)
	}
}

func TestValidatePublishedMessage(t *testing.T) {
	valid := messagepublisher.Message{CategoryCode: "SYSTEM", Title: "title", Content: "content", Source: "source", IdempotencyKey: "key", SenderName: "system"}
	if err := validatePublishedMessage(valid); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}
	invalid := valid
	invalid.Title = ""
	if err := validatePublishedMessage(invalid); err == nil {
		t.Fatal("empty title should be rejected")
	}
	invalid = valid
	invalid.ContentFormat = basev1.MessageContentFormat(99)
	if err := validatePublishedMessage(invalid); err == nil {
		t.Fatal("unknown content format should be rejected")
	}
}

func TestMessageAudienceAndEnumValidation(t *testing.T) {
	caseValue := &BaseMessageCase{}
	form := &adminv1.BaseMessageForm{
		CategoryId:    1,
		Title:         "title",
		Content:       "content",
		ContentFormat: basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT,
		Priority:      basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL,
		ActionType:    basev1.MessageActionType_MESSAGE_ACTION_TYPE_UNSPECIFIED,
	}
	if err := caseValue.validateMessageForm(context.Background(), 1, form); err == nil {
		t.Fatal("empty audiences should be rejected")
	}
}

func TestMessageAudienceDuplicateIgnoresNonDepartmentChildrenFlag(t *testing.T) {
	first := &adminv1.BaseMessageAudienceForm{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER, Id: 7, IncludeChildren: false}
	second := &adminv1.BaseMessageAudienceForm{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER, Id: 7, IncludeChildren: true}
	if messageAudienceKey(first) != messageAudienceKey(second) {
		t.Fatal("duplicate non-department audiences should use the same key")
	}
	deptWithoutChildren := &adminv1.BaseMessageAudienceForm{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_DEPT, Id: 7, IncludeChildren: false}
	deptWithChildren := &adminv1.BaseMessageAudienceForm{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_DEPT, Id: 7, IncludeChildren: true}
	if messageAudienceKey(deptWithoutChildren) == messageAudienceKey(deptWithChildren) {
		t.Fatal("department audience keys should preserve include_children")
	}
}
