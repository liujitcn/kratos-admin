package dto

// LocaleInfo 描述一种启用语言及其当前请求状态。
type LocaleInfo struct {
	// Locale 是语言代码。
	Locale string
	// IsPrimary 表示该语言是否为主语言。
	IsPrimary bool
	// IsCurrent 表示该语言是否为当前请求语言。
	IsCurrent bool
}
