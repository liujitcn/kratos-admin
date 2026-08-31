package dto

import adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

// I18nKey 标识一个资源的单语言翻译记录。
type I18nKey struct {
	TargetID int64
	Locale   string
}

// I18nDraftSource 描述草稿操作从服务端读取的受控源文。
type I18nDraftSource struct {
	TargetType adminv1.I18nTargetType
	TargetID   int64
	Text       string
}

// MenuMetadata 承载菜单 JSON 元信息中需要国际化的字段。
type MenuMetadata struct {
	Title string `json:"title"`
}
