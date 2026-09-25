package biz

import (
	"context"
	"errors"
	"strings"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-core/biz"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestAiFallbackReasonDoesNotExposeProviderError 验证聊天错误详情使用可翻译提示且不泄露底层错误文本。
func TestAiFallbackReasonDoesNotExposeProviderError(t *testing.T) {
	cause := errors.New("provider rejected request: secret-token")
	aiCase := &AiMessageCase{}
	want := i18n.EncodeMessage("system.ai.chat.error.details_unavailable", nil)

	response := aiCase.buildAiFallbackResponse(context.Background(), "", nil, cause)
	if response.FallbackReason != want {
		t.Fatalf("fallback reason = %q, want %q", response.FallbackReason, want)
	}
	if strings.Contains(response.FallbackReason, "secret-token") {
		t.Fatal("fallback reason must not include provider error details")
	}

	failedReply := aiCase.buildAiFailedReply(context.Background(), response, cause)
	if failedReply.FallbackReason != want {
		t.Fatalf("failed reply reason = %q, want %q", failedReply.FallbackReason, want)
	}
}

// TestAiFallbackReplyUsesRequestLocale 验证助手降级正文按请求语言写入会话。
func TestAiFallbackReplyUsesRequestLocale(t *testing.T) {
	catalog, err := corei18n.NewI18n("ai-fallback-reply-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	runtime := ai.NewRuntime(nil, nil, nil, nil, catalog)
	tests := []struct {
		locale      string
		prompt      string
		content     string
		attachments string
	}{
		{
			locale:      "zh-CN",
			prompt:      "请总结",
			content:     "已收到你的问题：请总结。但当前 AI 助手暂时不可用，无法生成完整回复，请稍后再试。",
			attachments: "已收到你的问题和 2 个附件，但当前 AI 助手暂时不可用，无法生成完整回复，请稍后再试。",
		},
		{
			locale:      "en-US",
			prompt:      "Summarize this",
			content:     "Your question was received: Summarize this. However, the AI assistant is currently unavailable and cannot generate a complete reply. Please try again later.",
			attachments: "Your question was received with 2 attachment(s), but the AI assistant is currently unavailable and cannot generate a complete reply. Please try again later.",
		},
		{
			locale:      "ja-JP",
			prompt:      "要約してください",
			content:     "質問：要約してください を受け取りましたが、AIアシスタントは現在利用できず、完全な回答を生成できません。しばらくしてからもう一度お試しください。",
			attachments: "質問と添付ファイル2件を受け取りましたが、AIアシスタントは現在利用できず、完全な回答を生成できません。しばらくしてからもう一度お試しください。",
		},
		{
			locale:      "zh-TW",
			prompt:      "請總結",
			content:     "已收到您的問題：請總結。但目前 AI 助理暫時無法使用，無法產生完整回覆，請稍後再試。",
			attachments: "已收到您的問題和 2 個附件，但目前 AI 助理暫時無法使用，無法產生完整回覆，請稍後再試。",
		},
	}
	for _, test := range tests {
		ctx := biz.WithLocale(context.Background(), test.locale)
		aiCase := &AiMessageCase{aiRuntime: runtime}
		textReply := aiCase.buildAiFallbackResponse(ctx, test.prompt, nil, errors.New("provider error"))
		if textReply.Content != test.content {
			t.Errorf("locale %s text fallback = %q, want %q", test.locale, textReply.Content, test.content)
		}
		attachmentReply := aiCase.buildAiFallbackResponse(ctx, "", make([]*basev1.AiAttachment, 2), errors.New("provider error"))
		if attachmentReply.Content != test.attachments {
			t.Errorf("locale %s attachment fallback = %q, want %q", test.locale, attachmentReply.Content, test.attachments)
		}
	}
}
