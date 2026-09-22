package biz

import (
	"database/sql"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/migrator"
)

// TestIsUniqueStorageIndexColumn 验证复合唯一索引字段不能配置入库脱敏。
func TestIsUniqueStorageIndexColumn(t *testing.T) {
	indexes := []gorm.Index{
		migrator.Index{
			NameValue:   "unique_base_user",
			ColumnList:  []string{"tenant_id", "user_name"},
			UniqueValue: sql.NullBool{Bool: true, Valid: true},
		},
		migrator.Index{
			NameValue:   "idx_base_user_phone",
			ColumnList:  []string{"phone"},
			UniqueValue: sql.NullBool{Bool: false, Valid: true},
		},
	}
	if !isUniqueStorageIndexColumn("user_name", indexes) {
		t.Fatal("复合唯一索引字段应禁止配置入库脱敏")
	}
	if isUniqueStorageIndexColumn("phone", indexes) {
		t.Fatal("普通索引字段应允许配置入库脱敏")
	}
}
