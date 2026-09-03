package backup

import (
	"bufio"
	"context"
	"reflect"
	"strings"
	"testing"
)

// TestMySQLSQLSplitterSupportsQuotesAndDelimiter 验证 SQL 分隔符不会误切字符串和存储程序。
func TestMySQLSQLSplitterSupportsQuotesAndDelimiter(t *testing.T) {
	script := "-- before delimiter\nDELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 'a;;b'; SELECT 1; END;;\nDELIMITER ;\nINSERT INTO t VALUES ('x;y'), ('a'';b');\n"
	splitter := &mysqlSQLSplitter{reader: bufio.NewReader(strings.NewReader(script)), delimiter: ";"}
	var statements []string
	if err := splitter.run(context.Background(), func(statement string) error {
		statements = append(statements, statement)
		return nil
	}); err != nil {
		t.Fatalf("splitter.run() error = %v", err)
	}
	want := []string{
		"CREATE PROCEDURE p() BEGIN SELECT 'a;;b'; SELECT 1; END",
		"INSERT INTO t VALUES ('x;y'), ('a'';b')",
	}
	if !reflect.DeepEqual(statements, want) {
		t.Fatalf("statements = %#v, want %#v", statements, want)
	}
}

// TestMySQLSQLSplitterPreservesVersionComment 验证 MySQL 版本条件注释不会被当作普通注释丢弃。
func TestMySQLSQLSplitterPreservesVersionComment(t *testing.T) {
	splitter := &mysqlSQLSplitter{reader: bufio.NewReader(strings.NewReader("/*!40101 SET NAMES utf8mb4 */;")), delimiter: ";"}
	var statements []string
	if err := splitter.run(context.Background(), func(statement string) error {
		statements = append(statements, statement)
		return nil
	}); err != nil {
		t.Fatalf("splitter.run() error = %v", err)
	}
	if len(statements) != 1 || statements[0] != "/*!40101 SET NAMES utf8mb4 */" {
		t.Fatalf("statements = %#v", statements)
	}
}

// TestIsOnlySQLComments 验证普通注释可以被安全忽略而版本注释仍视为 SQL。
func TestIsOnlySQLComments(t *testing.T) {
	if !isOnlySQLComments("-- comment\n# another comment\n/* block */") {
		t.Fatal("ordinary comments were not recognized")
	}
	if isOnlySQLComments("/*!40101 SET NAMES utf8mb4 */") {
		t.Fatal("version comment was incorrectly ignored")
	}
}
