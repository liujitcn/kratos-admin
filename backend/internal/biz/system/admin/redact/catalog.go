package redact

import (
	"fmt"
	"strings"

	"github.com/liujitcn/kratos-kit/redact"
)

// RuleTemplate 描述一个由代码固定支持的脱敏规则模板。
type RuleTemplate struct {
	// Code 是模板稳定编码。
	Code string
	// RuleType 是运行时规则类型。
	RuleType string
	// NameKey 是模板名称国际化键。
	NameKey string
	// FallbackName 是模板名称的中文回退值。
	FallbackName string
	// DefaultRule 是模板默认参数 JSON。
	DefaultRule string
	// DescriptionKey 是模板说明国际化键。
	DescriptionKey string
}

var ruleTemplates = []RuleTemplate{
	{Code: "phone_mask", RuleType: "MASK", NameKey: "system.admin.base.redact.rule.template.phone_mask", FallbackName: "手机号掩码", DefaultRule: `{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}`, DescriptionKey: "system.admin.base.redact.rule.description.phone_mask"},
	{Code: "email_mask", RuleType: "EMAIL", NameKey: "system.admin.base.redact.rule.template.email_mask", FallbackName: "邮箱掩码", DefaultRule: `{"email":{"keep_local_first":2,"mask_domain":false,"mask_char":"*"}}`, DescriptionKey: "system.admin.base.redact.rule.description.email_mask"},
	{Code: "regex_replace", RuleType: "REGEX", NameKey: "system.admin.base.redact.rule.template.regex_replace", FallbackName: "正则替换", DefaultRule: `{"regex":{"pattern":"(?s).+","replacement":"[REDACTED]"}}`, DescriptionKey: "system.admin.base.redact.rule.description.regex_replace"},
	{Code: "truncate_text", RuleType: "TRUNCATE", NameKey: "system.admin.base.redact.rule.template.truncate_text", FallbackName: "文本截断", DefaultRule: `{"truncate":{"length":10,"suffix":"..."}}`, DescriptionKey: "system.admin.base.redact.rule.description.truncate_text"},
	{Code: "hash_digest", RuleType: "HASH", NameKey: "system.admin.base.redact.rule.template.hash_digest", FallbackName: "哈希摘要", DefaultRule: `{"hash":{"algo":"SHA256"}}`, DescriptionKey: "system.admin.base.redact.rule.description.hash_digest"},
	{Code: "stable_uuid", RuleType: "UUID", NameKey: "system.admin.base.redact.rule.template.stable_uuid", FallbackName: "稳定 UUID", DefaultRule: `{"uuid":{}}`, DescriptionKey: "system.admin.base.redact.rule.description.stable_uuid"},
	{Code: "ip_mask", RuleType: "IP", NameKey: "system.admin.base.redact.rule.template.ip_mask", FallbackName: "IP 地址掩码", DefaultRule: `{"ip":{"keep_octets":2,"mask_char":"x"}}`, DescriptionKey: "system.admin.base.redact.rule.description.ip_mask"},
	{Code: "url_mask", RuleType: "URL", NameKey: "system.admin.base.redact.rule.template.url_mask", FallbackName: "URL 参数掩码", DefaultRule: `{"url":{"mask_query":true,"mask_char":"*"}}`, DescriptionKey: "system.admin.base.redact.rule.description.url_mask"},
	{Code: "fixed_length_mask", RuleType: "FIXED_LENGTH", NameKey: "system.admin.base.redact.rule.template.fixed_length_mask", FallbackName: "等长掩码", DefaultRule: `{"fixed_length":{"char":"X"}}`, DescriptionKey: "system.admin.base.redact.rule.description.fixed_length_mask"},
}

// RuleTemplates 返回固定规则模板的副本。
func RuleTemplates() []RuleTemplate {
	result := make([]RuleTemplate, len(ruleTemplates))
	copy(result, ruleTemplates)
	return result
}

// RuleTemplateCodes 返回固定规则模板编码。
func RuleTemplateCodes() []string {
	result := make([]string, 0, len(ruleTemplates))
	for _, template := range ruleTemplates {
		result = append(result, template.Code)
	}
	return result
}

// RuleTemplateByCode 按固定编码查找规则模板。
func RuleTemplateByCode(code string) (RuleTemplate, bool) {
	for _, template := range ruleTemplates {
		if template.Code == code {
			return template, true
		}
	}
	return RuleTemplate{}, false
}

// ValidateRuleTemplate 校验规则模板编码、类型和参数。
func ValidateRuleTemplate(code, ruleType, rule string) (RuleTemplate, error) {
	template, ok := RuleTemplateByCode(code)
	if !ok {
		return RuleTemplate{}, fmt.Errorf("规则模板编码不受支持: %s", code)
	}
	if !strings.EqualFold(template.RuleType, ruleType) {
		return RuleTemplate{}, fmt.Errorf("规则模板 %s 的规则类型必须为 %s", code, template.RuleType)
	}
	_, err := redact.NewFieldPolicy(redact.PolicyModeApplyRule, template.RuleType, rule)
	if err != nil {
		return RuleTemplate{}, fmt.Errorf("规则模板 %s 参数无效: %w", code, err)
	}
	return template, nil
}
