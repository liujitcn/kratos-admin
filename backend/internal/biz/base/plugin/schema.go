package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// CompileSchema 验证工具输入契约且不允许从网络加载外部引用。
func CompileSchema(raw string) (*jsonschema.Resolved, error) {
	if len(raw) > 128<<10 {
		return nil, fmt.Errorf("工具参数Schema超过128KiB限制")
	}
	var schema jsonschema.Schema
	err := json.Unmarshal([]byte(raw), &schema)
	if err != nil {
		return nil, fmt.Errorf("工具参数Schema无效: %w", err)
	}
	if schema.Type != "object" {
		return nil, fmt.Errorf("工具参数Schema必须为object")
	}
	return schema.Resolve(&jsonschema.ResolveOptions{})
}

// ContractHash 使用规范JSON绑定名称、说明和输入契约；描述变化同样需要重新审查。
func ContractHash(name, description, raw string) (string, error) {
	var value any
	err := json.Unmarshal([]byte(raw), &value)
	if err != nil {
		return "", err
	}
	var canonical []byte
	canonical, err = json.Marshal([]any{name, description, value})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// ValidateArguments 在实际执行边界验证 JSON 参数。
func ValidateArguments(rawSchema, arguments string) error {
	resolved, err := CompileSchema(rawSchema)
	if err != nil {
		return err
	}
	if len(arguments) > 1<<20 {
		return fmt.Errorf("工具参数超过1MiB限制")
	}
	var value any
	err = json.Unmarshal([]byte(arguments), &value)
	if err != nil {
		return fmt.Errorf("工具参数JSON无效: %w", err)
	}
	return resolved.Validate(value)
}
