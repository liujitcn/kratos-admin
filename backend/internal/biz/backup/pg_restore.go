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

// RestorePostgres 使用 Go 逐条执行 PostgreSQL SQL 导入文件。
//
// 切分器按 PostgreSQL 语义处理单引号、双引号标识符、E” 反斜杠转义、
// dollar-quoted 函数体与注释，pg_dump 与 Go 导出的文件均可恢复。
func RestorePostgres(ctx context.Context, db *sql.DB, source io.Reader) error {
	if db == nil {
		return fmt.Errorf("PostgreSQL 数据库连接为空")
	}
	if source == nil {
		return fmt.Errorf("SQL 恢复输入为空")
	}
	splitter := &postgresSQLSplitter{reader: bufio.NewReaderSize(source, 64*1024)}
	return splitter.run(ctx, func(statement string) error {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行 SQL 恢复语句失败: %w", err)
		}
		return nil
	})
}

type postgresSQLSplitter struct {
	reader        *bufio.Reader
	statement     bytes.Buffer
	quote         byte
	escaped       bool
	eString       bool
	dollarTag     string
	lineComment   bool
	blockComment  bool
	blockCommentD bool
}

// run 读取 SQL 文件并按 PostgreSQL 分隔符执行完整语句。
func (s *postgresSQLSplitter) run(ctx context.Context, execute func(string) error) error {
	var err error
	for {
		var line string
		line, err = s.reader.ReadString('\n')
		if len(line) > 0 {
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

// consumeLine 解析一行 SQL 并在遇到分号时执行语句。
func (s *postgresSQLSplitter) consumeLine(ctx context.Context, line []byte, execute func(string) error) error {
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
			if s.blockCommentD && line[index] == '/' {
				s.blockComment = false
				s.blockCommentD = false
			} else {
				s.blockCommentD = line[index] == '*'
			}
			index++
			continue
		}
		if s.dollarTag != "" {
			if end := strings.Index(string(line[index:]), s.dollarTag); end >= 0 {
				s.statement.WriteString(s.dollarTag)
				index += end + len(s.dollarTag)
				s.dollarTag = ""
				continue
			}
			s.statement.Write(line[index:])
			index = len(line)
			continue
		}
		if s.quote != 0 {
			s.statement.WriteByte(line[index])
			if s.escaped {
				s.escaped = false
			} else if s.quote == '\'' && line[index] == '\\' && s.eString {
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
		if line[index] == ';' {
			if err := s.flush(ctx, execute); err != nil {
				return err
			}
			index++
			continue
		}
		if line[index] == '$' {
			if tag := postgresDollarTag(line[index:]); tag != "" {
				s.dollarTag = tag
				s.statement.Write([]byte(tag))
				index += len(tag)
				continue
			}
			s.statement.WriteByte(line[index])
			index++
			continue
		}
		if line[index] == '\'' || line[index] == '"' {
			s.quote = line[index]
			// E'' 字符串中的反斜杠是转义符，普通标准模式字符串的反斜杠是字面量。
			s.eString = s.quote == '\'' && s.prevNonSpaceByte() == 'E'
			s.statement.WriteByte(line[index])
			index++
			continue
		}
		if line[index] == '-' && index+1 < len(line) && line[index+1] == '-' && (index+2 == len(line) || line[index+2] == ' ' || line[index+2] == '\t' || line[index+2] == '\r' || line[index+2] == '\n') {
			s.lineComment = true
			s.statement.WriteByte(line[index])
			s.statement.WriteByte(line[index+1])
			index += 2
			continue
		}
		if line[index] == '/' && index+1 < len(line) && line[index+1] == '*' {
			s.blockComment = true
			s.blockCommentD = false
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
func (s *postgresSQLSplitter) flush(ctx context.Context, execute func(string) error) error {
	statement := strings.TrimSpace(s.statement.String())
	s.statement.Reset()
	if statement == "" || isOnlySQLComments(statement) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return execute(statement)
}

// prevNonSpaceByte 返回缓冲区中最后一个非空白字节；缓冲为空时返回 0。
func (s *postgresSQLSplitter) prevNonSpaceByte() byte {
	value := s.statement.Bytes()
	for index := len(value) - 1; index >= 0; index-- {
		switch value[index] {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return value[index]
		}
	}
	return 0
}

// postgresDollarTag 识别行内以当前位置开始的 dollar-quote 起始标记；非标记返回空串。
func postgresDollarTag(line []byte) string {
	if len(line) < 2 || line[0] != '$' {
		return ""
	}
	for index := 1; index < len(line); index++ {
		switch {
		case line[index] == '$':
			return string(line[:index+1])
		case line[index] == '_' || line[index] >= 'a' && line[index] <= 'z' || line[index] >= 'A' && line[index] <= 'Z' || (index > 1 && line[index] >= '0' && line[index] <= '9'):
			continue
		default:
			return ""
		}
	}
	return ""
}
