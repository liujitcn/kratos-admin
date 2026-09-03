package backup

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
)

// RestoreMySQL 使用 Go 逐条执行 MySQL SQL 导入文件。
func RestoreMySQL(ctx context.Context, db *sql.DB, database string, source io.Reader) error {
	if db == nil {
		return fmt.Errorf("MySQL 数据库连接为空")
	}
	if source == nil {
		return fmt.Errorf("SQL 恢复输入为空")
	}
	if database != "" {
		if _, err := db.ExecContext(ctx, "USE "+quoteIdentifier(database)); err != nil {
			return fmt.Errorf("切换 MySQL 恢复数据库失败: %w", err)
		}
	}
	splitter := &mysqlSQLSplitter{reader: bufio.NewReaderSize(source, 64*1024), delimiter: ";"}
	if err := splitter.run(ctx, func(statement string) error {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行 SQL 恢复语句失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

type mysqlSQLSplitter struct {
	reader          *bufio.Reader
	delimiter       string
	statement       bytes.Buffer
	quote           byte
	escaped         bool
	lineComment     bool
	blockComment    bool
	blockCommentEnd bool
}

// run 读取 SQL 文件并按 MySQL 分隔符执行完整语句。
func (s *mysqlSQLSplitter) run(ctx context.Context, execute func(string) error) error {
	var err error
	for {
		var line string
		line, err = s.reader.ReadString('\n')
		if len(line) > 0 {
			if s.canReadDelimiterDirective() {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToUpper(trimmed), "DELIMITER ") {
					delimiter := strings.TrimSpace(trimmed[len("DELIMITER "):])
					if delimiter == "" {
						return fmt.Errorf("MySQL DELIMITER 指令为空")
					}
					s.statement.Reset()
					s.delimiter = delimiter
					if err == io.EOF {
						return nil
					}
					continue
				}
			}
			if err = s.consumeLine(ctx, []byte(line), execute); err != nil {
				return err
			}
		}
		if err == io.EOF {
			return s.flush(ctx, execute)
		}
		if err != nil {
			return fmt.Errorf("读取 SQL 恢复文件失败: %w", err)
		}
	}
}

// consumeLine 解析一行 SQL 并在遇到当前分隔符时执行语句。
func (s *mysqlSQLSplitter) consumeLine(ctx context.Context, line []byte, execute func(string) error) error {
	for index := 0; index < len(line); {
		if s.lineComment {
			s.statement.WriteByte(line[index])
			if line[index] == '\n' {
				s.lineComment = false
			}
			index++
			continue
		}
		if s.blockComment {
			s.statement.WriteByte(line[index])
			if s.blockCommentEnd && line[index] == '/' {
				s.blockComment = false
				s.blockCommentEnd = false
			} else {
				s.blockCommentEnd = line[index] == '*'
			}
			index++
			continue
		}
		if s.quote != 0 {
			s.statement.WriteByte(line[index])
			if s.escaped {
				s.escaped = false
			} else if line[index] == '\\' && s.quote != '`' {
				s.escaped = true
			} else if line[index] == s.quote && index+1 < len(line) && line[index+1] == s.quote {
				s.statement.WriteByte(line[index+1])
				index++
			} else if line[index] == s.quote {
				s.quote = 0
			}
			index++
			continue
		}
		if strings.HasPrefix(string(line[index:]), s.delimiter) {
			if err := s.flush(ctx, execute); err != nil {
				return err
			}
			index += len(s.delimiter)
			continue
		}
		if line[index] == '\'' || line[index] == '"' || line[index] == '`' {
			s.quote = line[index]
			s.statement.WriteByte(line[index])
			index++
			continue
		}
		if line[index] == '#' {
			s.lineComment = true
			s.statement.WriteByte(line[index])
			index++
			continue
		}
		if line[index] == '/' && index+1 < len(line) && line[index+1] == '*' {
			s.blockComment = true
			s.blockCommentEnd = false
			s.statement.WriteByte(line[index])
			s.statement.WriteByte(line[index+1])
			index += 2
			continue
		}
		if line[index] == '-' && index+1 < len(line) && line[index+1] == '-' && (index+2 == len(line) || line[index+2] == ' ' || line[index+2] == '\t' || line[index+2] == '\r' || line[index+2] == '\n') {
			s.lineComment = true
			s.statement.WriteByte(line[index])
			s.statement.WriteByte(line[index+1])
			index += 2
			continue
		}
		s.statement.WriteByte(line[index])
		index++
	}
	return nil
}

// flush 执行当前已收集的 SQL 语句并清空缓冲区。
func (s *mysqlSQLSplitter) flush(ctx context.Context, execute func(string) error) error {
	statement := strings.TrimSpace(s.statement.String())
	s.statement.Reset()
	if statement == "" {
		return nil
	}
	if isOnlySQLComments(statement) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := execute(statement); err != nil {
		return err
	}
	return nil
}

// canReadDelimiterDirective 判断当前是否位于一条新 SQL 语句的开头。
func (s *mysqlSQLSplitter) canReadDelimiterDirective() bool {
	return s.quote == 0 && !s.lineComment && !s.blockComment && isOnlySQLComments(s.statement.String())
}

// isOnlySQLComments 判断 SQL 文本是否只包含空白和普通注释。
func isOnlySQLComments(value string) bool {
	for {
		value = strings.TrimSpace(value)
		if value == "" {
			return true
		}
		if strings.HasPrefix(value, "--") && (len(value) == 2 || value[2] == ' ' || value[2] == '\t' || value[2] == '\r' || value[2] == '\n') {
			lineEnd := strings.IndexByte(value, '\n')
			if lineEnd < 0 {
				return true
			}
			value = value[lineEnd+1:]
			continue
		}
		if strings.HasPrefix(value, "#") {
			lineEnd := strings.IndexByte(value, '\n')
			if lineEnd < 0 {
				return true
			}
			value = value[lineEnd+1:]
			continue
		}
		if strings.HasPrefix(value, "/*") && !strings.HasPrefix(value, "/*!") {
			commentEnd := strings.Index(value[2:], "*/")
			if commentEnd < 0 {
				return false
			}
			value = value[commentEnd+4:]
			continue
		}
		return false
	}
}
