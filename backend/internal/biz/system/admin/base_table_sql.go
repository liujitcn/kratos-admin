package biz

// 表归档与恢复的跨数据库方言辅助：内部归档模式需要在 MySQL 与 PostgreSQL 上生成等价的
// 建表、复制和标识符引用 SQL；OSS 导出依赖 mysqldump，仅 MySQL 系数据源支持。

const (
	// sqlDriverMySQL 是 MySQL 驱动名。
	sqlDriverMySQL = "mysql"
	// SqlDriverPostgres 是 PostgreSQL 驱动名。
	SqlDriverPostgres = "postgres"
	// sqlDriverDoris 是 Doris 驱动名，与 MySQL 共用方言。
	sqlDriverDoris = "doris"
)

// MysqlFamilyDriver 判断驱动是否属于 MySQL 方言系（含复用 MySQL 方言的 Doris）。
func MysqlFamilyDriver(driverName string) bool {
	return driverName == sqlDriverMySQL || driverName == sqlDriverDoris
}

// QuoteSQLIdentifier 按数据库方言为表名、字段名加引号：MySQL 系使用反引号，其余使用 SQL 标准双引号。
func QuoteSQLIdentifier(driverName string, name string) string {
	if MysqlFamilyDriver(driverName) {
		return "`" + name + "`"
	}
	return `"` + name + `"`
}

// CreateArchiveTableSQL 生成与源表结构一致的归档表建表 SQL。
func CreateArchiveTableSQL(driverName string, archive string, source string) string {
	if driverName == SqlDriverPostgres {
		// PostgreSQL 使用 LIKE 子句；INCLUDING ALL 连同主键、唯一索引一并复制，保证幂等归档。
		return "CREATE TABLE IF NOT EXISTS " + archive + " (LIKE " + source + " INCLUDING ALL)"
	}
	return "CREATE TABLE IF NOT EXISTS " + archive + " LIKE " + source
}

// CopyRowsIgnoreConflictSQL 生成跨表复制数据的 SQL，目标表已存在的记录按方言跳过冲突。
func CopyRowsIgnoreConflictSQL(driverName string, target string, source string, condition string) string {
	if driverName == SqlDriverPostgres {
		return "INSERT INTO " + target + " SELECT * FROM " + source + condition + " ON CONFLICT DO NOTHING"
	}
	return "INSERT IGNORE INTO " + target + " SELECT * FROM " + source + condition
}
