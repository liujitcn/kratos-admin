package dto

import systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

// TranslationKey 标识一个资源的单语言翻译记录。
type TranslationKey struct {
	ResourceID int64
	Locale     string
}

// TranslationDraftSource 描述草稿操作从服务端读取的受控源文。
type TranslationDraftSource struct {
	ResourceType systemadminv1.TranslationResourceType
	ResourceID   int64
	Text         string
	Field        systemadminv1.BaseConfigTranslationField
}

// ConfigTranslationSource 描述系统配置翻译所需的中文源文。
type ConfigTranslationSource struct {
	Name  string
	Value string
	Type  int32
}

// MenuMetadata 承载菜单 JSON 元信息中需要国际化的字段。
type MenuMetadata struct {
	Title string `json:"title"`
}
