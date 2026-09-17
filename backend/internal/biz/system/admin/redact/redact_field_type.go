package redact

import "strings"

// IsRedactStringDatabaseType 判断数据库字段是否为脱敏主表可保存的字符串类型。
func IsRedactStringDatabaseType(databaseType string) bool {
	switch strings.ToLower(databaseType) {
	case "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "enum", "set", "json":
		return true
	default:
		return false
	}
}
