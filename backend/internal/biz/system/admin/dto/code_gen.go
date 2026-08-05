package dto

import "database/sql"

// CodeGenDatabaseTable 承载数据库表元数据查询结果。
type CodeGenDatabaseTable struct {
	TableName    string `gorm:"column:table_name"`    // 数据库表名
	TableComment string `gorm:"column:table_comment"` // 业务表描述
}

// CodeGenDatabaseColumn 承载数据库字段元数据查询结果。
type CodeGenDatabaseColumn struct {
	Name                   string         `gorm:"column:column_name"`              // 数据库字段名
	Comment                string         `gorm:"column:column_comment"`           // 数据库字段注释
	DataType               string         `gorm:"column:data_type"`                // 数据库字段类型
	ColumnType             string         `gorm:"column:column_type"`              // 数据库完整字段类型
	ColumnKey              string         `gorm:"column:column_key"`               // 数据库字段索引类型
	IsNullable             string         `gorm:"column:is_nullable"`              // 是否允许为空
	Extra                  string         `gorm:"column:extra"`                    // 数据库字段扩展信息
	OrdinalPosition        int32          `gorm:"column:ordinal_position"`         // 数据库字段顺序
	CharacterMaximumLength sql.NullInt64  `gorm:"column:character_maximum_length"` // 字符字段长度
	NumericPrecision       sql.NullInt64  `gorm:"column:numeric_precision"`        // 数值字段精度
	NumericScale           sql.NullInt64  `gorm:"column:numeric_scale"`            // 数值字段小数位
	ColumnDefault          sql.NullString `gorm:"column:column_default"`           // 数据库字段默认值
}
