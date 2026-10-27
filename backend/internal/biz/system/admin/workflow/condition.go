package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"
)

// EvalCondition 在沙箱中对条件表达式求值。formDataJSON 是申请表单数据 JSON 文本，
// 解析为 map 后作为唯一表达式环境；语法错、类型错、运行错均返回 error，由调用方 fail-closed 处理。
func EvalCondition(condition, formDataJSON string) (bool, error) {
	if condition == "" {
		return false, nil
	}

	env := make(map[string]any)
	if formDataJSON != "" {
		if err := json.Unmarshal([]byte(formDataJSON), &env); err != nil {
			return false, fmt.Errorf("表单数据不是合法 JSON: %w", err)
		}
		if env == nil {
			return false, fmt.Errorf("表单数据必须是 JSON 对象")
		}
	}

	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("条件表达式编译失败: %w", err)
	}
	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("条件表达式求值失败: %w", err)
	}
	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("条件表达式结果不是布尔值")
	}
	return result, nil
}
