package structured

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

// Schema 表示结构化输出 JSON Schema。
type Schema = jsonschema.Schema

// SchemaFor 根据结果类型生成 JSON Schema。
func SchemaFor[T any]() (*Schema, error) {
	return jsonschema.For[T](nil)
}

// SchemaPrompt 构造结构化输出的 JSON Schema 文本约束。
func SchemaPrompt(outputSchema *Schema) string {
	// 无 Schema 时仍明确要求 JSON 对象，避免模型返回 Markdown 或解释性文本。
	if outputSchema == nil {
		return "只返回一个合法 JSON 对象，不要使用 Markdown 代码块。"
	}
	raw, err := json.Marshal(outputSchema)
	// Schema 序列化失败时降级为通用 JSON 约束，调用方仍可继续做结构化解析。
	if err != nil {
		return "只返回一个合法 JSON 对象，不要使用 Markdown 代码块。"
	}
	return "请严格按以下 JSON Schema 返回一个合法 JSON 对象，不要输出 Markdown 代码块或额外说明：\n" + string(raw)
}
