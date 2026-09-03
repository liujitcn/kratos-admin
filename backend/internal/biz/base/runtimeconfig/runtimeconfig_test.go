package runtimeconfig

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestValidateProtoRuntimeConfig 验证日志入库回退配置由 Proto 规则统一校验。
func TestValidateProtoRuntimeConfig(t *testing.T) {
	valid := DefaultBaseLogFallbackConfig()
	if err := ValidateJSON(BaseLogFallbackKey, mustJSON(t, valid)); err != nil {
		t.Fatalf("valid base log fallback config rejected: %v", err)
	}
}

// TestDefaultJSONDoesNotExposeIntegrityKey 验证日志回退配置不再暴露可持久化的完整性密钥字段。
func TestDefaultJSONDoesNotExposeIntegrityKey(t *testing.T) {
	value, err := DefaultJSON(BaseLogFallbackKey)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(value, `"file_path"`) || strings.Contains(value, `"integrity_key"`) {
		t.Fatalf("unexpected base log fallback config: %s", value)
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
