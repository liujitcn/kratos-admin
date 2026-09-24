package kit

import "testing"

// TestValidateRedactRuleAllowsCustomCode 验证规则编码可由数据库动态定义。
func TestValidateRedactRuleAllowsCustomCode(t *testing.T) {
	rule := `{"mask":{"keep_first":1,"keep_last":1,"min_mask":1,"mask_char":"*"}}`
	err := ValidateRedactRule("employee_number_mask", "MASK", rule)
	if err != nil {
		t.Fatalf("custom rule code should be accepted: %v", err)
	}
	err = ValidateRedactRule("custom_mask", "MASK", `{}`)
	if err == nil {
		t.Fatal("invalid rule parameters should be rejected")
	}
	err = ValidateRedactRule("unknown_type", "UNKNOWN", `{}`)
	if err == nil {
		t.Fatal("unknown rule type should be rejected")
	}
}

// TestValidateRedactRuleAllowsDirectEncryption 验证直接字段加密规则能够进入运行时策略。
func TestValidateRedactRuleAllowsDirectEncryption(t *testing.T) {
	err := ValidateRedactRule("encrypt", "ENCRYPT", `{"encrypt":{"algorithm":"AES_GCM"}}`)
	if err != nil {
		t.Fatal(err)
	}
}
