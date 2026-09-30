package backup

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

// TestPostgresSQLSplitter 验证按 PostgreSQL 语义切分语句。
func TestPostgresSQLSplitter(t *testing.T) {
	content := `-- 头部注释
SET statement_timeout = 0;
INSERT INTO "public"."base_user" (name, memo) VALUES ('张''三', E'行一\n行二');
CREATE FUNCTION tick() RETURNS trigger AS $body$
BEGIN
  NEW.updated_at := now(); -- 函数体内的分号
END;
$body$ LANGUAGE plpgsql;
/* 块注释; 内的分号 */
SELECT 1;
`
	db := &statementCollector{}
	splitter := &postgresSQLSplitter{reader: bufio.NewReader(strings.NewReader(content))}
	if err := splitter.run(context.Background(), db.collect); err != nil {
		t.Fatal(err)
	}
	statements := db.statements
	if len(statements) != 4 {
		t.Fatalf("期望 4 条语句实际 %d：%v", len(statements), statements)
	}
	if !strings.HasSuffix(statements[0], "SET statement_timeout = 0") {
		t.Fatalf("普通语句切分错误: %s", statements[0])
	}
	if !strings.Contains(statements[1], `'张''三'`) || !strings.Contains(statements[1], `E'行一\n行二'`) {
		t.Fatalf("字符串转义处理错误: %s", statements[1])
	}
	if !strings.Contains(statements[2], "NEW.updated_at := now(); -- 函数体内的分号") || !strings.HasSuffix(statements[2], "LANGUAGE plpgsql") {
		t.Fatalf("dollar-quote 函数体切分错误: %s", statements[2])
	}
	if !strings.HasSuffix(statements[3], "SELECT 1") {
		t.Fatalf("块注释后语句错误: %s", statements[3])
	}
}

// TestDumpPostgresValueFormatting 验证各类型值的 SQL 字面量编码。
func TestDumpPostgresValueFormatting(t *testing.T) {
	cases := []struct {
		value    interface{}
		expected string
	}{
		{nil, "NULL"},
		{"文本'带引号", "'文本''带引号'"},
		{true, "true"},
		{false, "false"},
		{int64(42), "42"},
		{[]byte{0xde, 0xad}, "'\\xdead'::bytea"},
	}
	for _, testCase := range cases {
		value := formatPostgresValue(testCase.value, "")
		if value != testCase.expected {
			t.Fatalf("值 %v 期望 %s 实际 %s", testCase.value, testCase.expected, value)
		}
	}
}

type statementCollector struct {
	statements []string
}

func (c *statementCollector) collect(statement string) error {
	c.statements = append(c.statements, statement)
	return nil
}

// TestPostgresSQLSplitterRoundTrip 验证包含分号字符串与 dollar-quote 的导出内容可被完整还原。
func TestPostgresSQLSplitterRoundTrip(t *testing.T) {
	content := "INSERT INTO \"t\" (a) VALUES ('a;b');\nCREATE FUNCTION f() RETURNS void AS $$ BEGIN RETURN; END; $$;\n"
	db := &statementCollector{}
	splitter := &postgresSQLSplitter{reader: bufio.NewReader(strings.NewReader(content))}
	if err := splitter.run(context.Background(), db.collect); err != nil {
		t.Fatal(err)
	}
	if len(db.statements) != 2 {
		t.Fatalf("期望 2 条语句实际 %d：%v", len(db.statements), db.statements)
	}
	if db.statements[0] != `INSERT INTO "t" (a) VALUES ('a;b')` {
		t.Fatalf("含分号字符串切分错误: %s", db.statements[0])
	}
	if !strings.HasPrefix(db.statements[1], "CREATE FUNCTION f()") || !strings.HasSuffix(db.statements[1], "LANGUAGE plpgsql") && !strings.HasSuffix(db.statements[1], "$$") {
		t.Fatalf("dollar-quote 语句切分错误: %s", db.statements[1])
	}
}
