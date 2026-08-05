package dto

// LocaleState 描述当前请求可使用的运行时语言。
type LocaleState struct {
	// Current 是当前请求语言。
	Current string
	// Primary 是主语言；数据库没有有效主语言时回退到默认语言。
	Primary string
	// Enabled 是数据库中启用的语言。
	Enabled []string
}

// IsCurrentPrimary 判断当前请求语言是否为主语言。
func (s *LocaleState) IsCurrentPrimary() bool {
	return s.Current == s.Primary
}

// IsEnabled 判断语言是否处于运行时启用列表。
func (s *LocaleState) IsEnabled(locale string) bool {
	for _, value := range s.Enabled {
		if value == locale {
			return true
		}
	}
	return false
}

// IsEditable 判断语言是否为启用的非主语言。
func (s *LocaleState) IsEditable(locale string) bool {
	return locale != s.Primary && s.IsEnabled(locale)
}

// EditableLocales 返回启用的非主语言副本。
func (s *LocaleState) EditableLocales() []string {
	locales := make([]string, 0, len(s.Enabled))
	for _, locale := range s.Enabled {
		if locale != s.Primary {
			locales = append(locales, locale)
		}
	}
	return locales
}
