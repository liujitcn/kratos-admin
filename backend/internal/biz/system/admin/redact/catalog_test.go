package redact

import "testing"

// TestRuleTemplates 验证所有固定模板都能被当前运行时解析。
func TestRuleTemplates(t *testing.T) {
	templates := RuleTemplates()
	if len(templates) != 9 {
		t.Fatalf("unexpected template count: %d", len(templates))
	}
	var err error
	for _, template := range templates {
		if template.Code == "" || template.RuleType == "" || template.DefaultRule == "" {
			t.Fatalf("incomplete rule template: %+v", template)
		}
		_, err = ValidateRuleTemplate(template.Code, template.RuleType, template.DefaultRule)
		if err != nil {
			t.Fatalf("template %s is invalid: %v", template.Code, err)
		}
	}
}
