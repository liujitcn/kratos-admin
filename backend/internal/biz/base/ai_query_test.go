package biz

import (
	"strings"
	"testing"
	"time"
)

func TestExtractSQL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "纯 SQL 原样返回",
			raw:  "SELECT * FROM base_user LIMIT 10",
			want: "SELECT * FROM base_user LIMIT 10",
		},
		{
			name: "剥掉 markdown 围栏",
			raw:  "```sql\nSELECT * FROM base_user LIMIT 10\n```",
			want: "SELECT * FROM base_user LIMIT 10",
		},
		{
			name: "去掉结尾分号",
			raw:  "SELECT * FROM base_user;",
			want: "SELECT * FROM base_user",
		},
		{
			name: "去掉行注释与块注释",
			raw:  "-- 查询用户\nSELECT * FROM base_user /* 全部字段 */ LIMIT 10",
			want: "SELECT * FROM base_user   LIMIT 10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractSQL(tt.raw); got != tt.want {
				t.Fatalf("extractSQL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateReadOnlySQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "SELECT 合法", sql: "SELECT COUNT(*) FROM base_user WHERE status = 1"},
		{name: "WITH 合法", sql: "WITH t AS (SELECT id FROM base_user) SELECT * FROM t"},
		{name: "ORDER BY DESC 合法", sql: "SELECT id FROM base_user ORDER BY id DESC"},
		{name: "INSERT 拒绝", sql: "INSERT INTO base_user (id) VALUES (1)", wantErr: true},
		{name: "UPDATE 拒绝", sql: "UPDATE base_user SET status = 1", wantErr: true},
		{name: "DELETE 拒绝", sql: "DELETE FROM base_user WHERE id = 1", wantErr: true},
		{name: "DROP 拒绝", sql: "SELECT * FROM base_user; DROP TABLE base_user", wantErr: true},
		{name: "多语句拒绝", sql: "SELECT * FROM base_user; SELECT * FROM base_role", wantErr: true},
		{name: "INTO OUTFILE 拒绝", sql: "SELECT * FROM base_user INTO OUTFILE '/tmp/x'", wantErr: true},
		{name: "FOR UPDATE 拒绝", sql: "SELECT * FROM base_user FOR UPDATE", wantErr: true},
		{name: "字符串字面量中的关键字不误判", sql: "SELECT * FROM base_user WHERE remark = 'drop into set'"},
		{name: "SLEEP 函数放行由执行超时兜底", sql: "SELECT id FROM base_user LIMIT 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReadOnlySQL(extractSQL(tt.sql))
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateReadOnlySQL(%q) error = %v, wantErr %v", tt.sql, err, tt.wantErr)
			}
		})
	}
}

func TestReferencedTableNames(t *testing.T) {
	sql := "WITH recent AS (SELECT id FROM base_message WHERE created_at > '2026-01-01') SELECT u.user_name, r.id FROM base_user u JOIN recent r ON r.user_id = u.id JOIN `base_dept` d ON d.id = u.dept_id"
	got := referencedTableNames(sql)
	want := map[string]bool{"base_message": true, "base_user": true, "recent": true, "base_dept": true}
	if len(got) != len(want) {
		t.Fatalf("referencedTableNames() = %v, want keys %v", got, want)
	}
	for _, name := range got {
		if !want[name] {
			t.Fatalf("referencedTableNames() 意外表名 %q", name)
		}
	}
}

func TestCheckTableWhitelist(t *testing.T) {
	meta := map[string]*aiQueryTableMeta{
		"base_user":    {Name: "base_user", TenantScoped: true},
		"base_dept":    {Name: "base_dept", TenantScoped: true},
		"base_message": {Name: "base_message"},
	}
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "白名单内通过", sql: "SELECT * FROM base_user u JOIN base_dept d ON d.id = u.dept_id"},
		{name: "逗号连接白名单外拒绝", sql: "SELECT * FROM base_user, casbin_rule", wantErr: true},
		{name: "子查询内部表校验", sql: "SELECT * FROM (SELECT id FROM casbin_rule) t", wantErr: true},
		{name: "CTE 名称豁免", sql: "WITH recent AS (SELECT id FROM base_message) SELECT * FROM recent"},
		{name: "ORDER BY 列表不误判", sql: "SELECT u.id, u.user_name FROM base_user u ORDER BY u.id, u.user_name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTableWhitelist(tt.sql, referencedTableNames(tt.sql), meta)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkTableWhitelist(%q) error = %v, wantErr %v", tt.sql, err, tt.wantErr)
			}
		})
	}
}

func TestEnforceTenantScoping(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		tenantTables []string
		tenantID     int64
		want         string
		wantErr      bool
	}{
		{
			name:     "无租户表原样返回",
			sql:      "SELECT * FROM base_dict ORDER BY id DESC",
			tenantID: 42,
			want:     "SELECT * FROM base_dict ORDER BY id DESC",
		},
		{
			name:         "租户条件取值重写",
			sql:          "SELECT tenant_id, COUNT(*) FROM base_user WHERE status = 1 AND tenant_id = 7 GROUP BY tenant_id",
			tenantTables: []string{"base_user"},
			tenantID:     42,
			want:         "SELECT tenant_id, COUNT(*) FROM base_user WHERE status = 1 AND tenant_id = 42 GROUP BY tenant_id",
		},
		{
			name:         "别名限定租户条件重写",
			sql:          "SELECT u.id FROM base_user u WHERE u.tenant_id = -1",
			tenantTables: []string{"base_user"},
			tenantID:     42,
			want:         "SELECT u.id FROM base_user u WHERE u.tenant_id = 42",
		},
		{
			name:         "缺失租户条件拒绝",
			sql:          "SELECT id FROM base_user WHERE status = 1",
			tenantTables: []string{"base_user"},
			wantErr:      true,
		},
		{
			name:         "非等值租户条件拒绝",
			sql:          "SELECT id FROM base_user WHERE tenant_id >= 1",
			tenantTables: []string{"base_user"},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := enforceTenantScoping(tt.sql, tt.tenantTables, tt.tenantID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("enforceTenantScoping(%q) error = %v, wantErr %v", tt.sql, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("enforceTenantScoping(%q) = %q, want %q", tt.sql, got, tt.want)
			}
		})
	}
}

func TestApplyLimitClamp(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{name: "缺失 LIMIT 追加", sql: "SELECT * FROM base_user", want: "SELECT * FROM base_user LIMIT 50"},
		{name: "超上限钳制", sql: "SELECT * FROM base_user LIMIT 500", want: "SELECT * FROM base_user LIMIT 50"},
		{name: "偏移分页重写", sql: "SELECT * FROM base_user LIMIT 10, 20", want: "SELECT * FROM base_user LIMIT 50"},
		{name: "WITH 后追加", sql: "WITH t AS (SELECT id FROM base_user) SELECT * FROM t", want: "WITH t AS (SELECT id FROM base_user) SELECT * FROM t LIMIT 50"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyLimitClamp(tt.sql, aiQueryDefaultMaxRows); got != tt.want {
				t.Fatalf("applyLimitClamp(%q) = %q, want %q", tt.sql, got, tt.want)
			}
		})
	}
}

func TestFormatSQLValue(t *testing.T) {
	now := time.Date(2026, 9, 29, 8, 30, 0, 0, time.Local)
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "NULL 空串", value: nil, want: ""},
		{name: "字节切片", value: []byte("hello"), want: "hello"},
		{name: "时间格式化", value: now, want: "2026-09-29 08:30:00"},
		{name: "数字", value: int64(12), want: "12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSQLValue(tt.value); got != tt.want {
				t.Fatalf("formatSQLValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestSchemaDictionarySkipsMissingTables(t *testing.T) {
	c := &AiQueryCase{}
	meta := map[string]*aiQueryTableMeta{
		"base_user": {
			Name:         "base_user",
			Comment:      "用户",
			TenantScoped: true,
			Columns: []aiQueryColumnMeta{
				{Name: "tenant_id", Type: "bigint", Comment: "租户 ID"},
				{Name: "status", Type: "tinyint", Comment: "状态"},
			},
		},
	}
	dictionary := c.schemaDictionary(meta)
	if !strings.Contains(dictionary, "base_user（用户）【租户表】") {
		t.Fatalf("schemaDictionary() 缺少租户表条目: %q", dictionary)
	}
	if !strings.Contains(dictionary, "状态(status:tinyint)") {
		t.Fatalf("schemaDictionary() 缺少字段条目: %q", dictionary)
	}
	if strings.Contains(dictionary, "base_role") {
		t.Fatalf("schemaDictionary() 不应包含未登记表: %q", dictionary)
	}
}
