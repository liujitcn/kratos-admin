package dto

import systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

// TranslationKey 标识一个资源的单语言翻译记录。
type TranslationKey struct {
	TargetID int64
	Locale   string
}

// TranslationDraftSource 描述草稿操作从服务端读取的受控源文。
type TranslationDraftSource struct {
	TargetType systemadminv1.TranslationTargetType
	TargetID   int64
	Text       string
}

// TranslationQueueMessage 描述一次动态资源机器翻译队列消息。
type TranslationQueueMessage struct {
	TargetType   systemadminv1.TranslationTargetType `json:"target_type"`
	TargetID     int64                               `json:"target_id"`
	SourceLocale string                              `json:"source_locale"`
	TargetLocale string                              `json:"target_locale"`
}

// MenuMetadata 承载菜单 JSON 元信息中需要国际化的字段。
type MenuMetadata struct {
	Title string `json:"title"`
}
