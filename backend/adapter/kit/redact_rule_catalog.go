package kit

import (
	"fmt"
	"strings"

	"github.com/liujitcn/kratos-kit/redact"
)

// ValidateRedactRule 校验规则编码、运行时类型和数据库中的规则参数。
func ValidateRedactRule(code, ruleType, rule string) error {
	if code == "" {
		return fmt.Errorf("规则编码不能为空")
	}
	normalizedType := strings.ToUpper(strings.TrimSpace(ruleType))
	if normalizedType == "" {
		return fmt.Errorf("规则 %s 的规则类型不能为空", code)
	}
	_, err := redact.NewFieldPolicy(redact.PolicyModeApplyRule, normalizedType, rule)
	if err != nil {
		return fmt.Errorf("规则 %s 类型或参数无效: %w", code, err)
	}
	return nil
}
