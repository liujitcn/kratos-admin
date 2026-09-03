package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestValidateProtoRuntimeConfig 验证日志入库回退配置由 Proto 规则统一校验。
func TestValidateProtoRuntimeConfig(t *testing.T) {
	valid := DefaultAuditLogSpoolConfig()
	if err := ValidateJSON(AuditLogSpoolKey, mustJSON(t, valid)); err != nil {
		t.Fatalf("valid audit log spool config rejected: %v", err)
	}
}

// TestMigrateJSON 清理已从运行配置 Proto 移除的历史字段。
func TestMigrateJSON(t *testing.T) {
	value, err := MigrateJSON(AuditLogSpoolKey, `{"spool_file":"./data/audit-log-spool/admin-log.jsonl","integrity_key":"stored-secret","retention_days":180}`)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err = json.Unmarshal([]byte(value), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["file_path"] != "./data/audit-log-spool" {
		t.Fatalf("legacy spool file was not converted to directory: %s", value)
	}
	for _, field := range []string{"spool_file", "integrity_key", "retention_days"} {
		if _, exists := payload[field]; exists {
			t.Fatalf("deprecated field was not removed: %s", field)
		}
	}
}

// TestDefaultJSONDoesNotExposeIntegrityKey 验证日志回退配置不再暴露可持久化的完整性密钥字段。
func TestDefaultJSONDoesNotExposeIntegrityKey(t *testing.T) {
	value, err := DefaultJSON(AuditLogSpoolKey)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(value, `"file_path"`) || strings.Contains(value, `"integrity_key"`) {
		t.Fatalf("unexpected audit log spool config: %s", value)
	}
}

// mustJSON 将测试配置编码为 JSON。
func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
