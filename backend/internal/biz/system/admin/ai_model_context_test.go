package biz

import "testing"

// TestChatModelContextWindow 验证上下文容量配置可以进入现有模型管理链路，且不会吞掉输出预算。
func TestChatModelContextWindow(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		raw   string
		valid bool
	}{
		{name: "small_model", raw: `{"api_type":"CHAT_COMPLETIONS","context_window":8192,"max_tokens":1024}`, valid: true},
		{name: "legacy_default", raw: `{"api_type":"RESPONSES","max_tokens":1024}`, valid: true},
		{name: "negative", raw: `{"api_type":"CHAT_COMPLETIONS","context_window":-1}`, valid: false},
		{name: "output_exhausts_window", raw: `{"api_type":"CHAT_COMPLETIONS","context_window":8192,"max_tokens":8192}`, valid: false},
		{name: "oversized", raw: `{"api_type":"CHAT_COMPLETIONS","context_window":20000000}`, valid: false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := parseAiModelChatConfig(scenario.raw)
			if (err == nil) != scenario.valid {
				t.Fatalf("配置接受状态不符合预期: valid=%v err=%v", scenario.valid, err)
			}
		})
	}
}
