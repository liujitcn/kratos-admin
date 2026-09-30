package biz

import "testing"

// TestQuoteSQLIdentifier 验证按方言引用标识符。
func TestQuoteSQLIdentifier(t *testing.T) {
	cases := []struct {
		driver   string
		name     string
		expected string
	}{
		{driver: "mysql", name: "base_user", expected: "`base_user`"},
		{driver: "doris", name: "base_user", expected: "`base_user`"},
		{driver: "postgres", name: "base_user", expected: `"base_user"`},
	}
	for _, testCase := range cases {
		if value := QuoteSQLIdentifier(testCase.driver, testCase.name); value != testCase.expected {
			t.Fatalf("驱动 %s 期望 %s 实际 %s", testCase.driver, testCase.expected, value)
		}
	}
}

// TestCreateArchiveTableSQL 验证归档建表 SQL 的方言分支。
func TestCreateArchiveTableSQL(t *testing.T) {
	pg := CreateArchiveTableSQL("postgres", `"base_user_archive"`, `"base_user"`)
	if pg != `CREATE TABLE IF NOT EXISTS "base_user_archive" (LIKE "base_user" INCLUDING ALL)` {
		t.Fatalf("PostgreSQL 建表 SQL 不符合预期: %s", pg)
	}
	mysql := CreateArchiveTableSQL("mysql", "`base_user_archive`", "`base_user`")
	if mysql != "CREATE TABLE IF NOT EXISTS `base_user_archive` LIKE `base_user`" {
		t.Fatalf("MySQL 建表 SQL 不符合预期: %s", mysql)
	}
}

// TestCopyRowsIgnoreConflictSQL 验证跨表复制 SQL 的方言分支。
func TestCopyRowsIgnoreConflictSQL(t *testing.T) {
	pg := CopyRowsIgnoreConflictSQL("postgres", `"a"`, `"b"`, ` WHERE "id" IN (1,2)`)
	if pg != `INSERT INTO "a" SELECT * FROM "b" WHERE "id" IN (1,2) ON CONFLICT DO NOTHING` {
		t.Fatalf("PostgreSQL 复制 SQL 不符合预期: %s", pg)
	}
	mysql := CopyRowsIgnoreConflictSQL("mysql", "`a`", "`b`", " WHERE `id` IN (1,2)")
	if mysql != "INSERT IGNORE INTO `a` SELECT * FROM `b` WHERE `id` IN (1,2)" {
		t.Fatalf("MySQL 复制 SQL 不符合预期: %s", mysql)
	}
}
