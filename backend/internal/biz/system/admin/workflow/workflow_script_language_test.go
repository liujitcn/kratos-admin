package workflow

import "testing"

// TestScriptLanguageSupported 校验流程执行仅允许无宿主能力的 JavaScript 引擎。
func TestScriptLanguageSupported(t *testing.T) {
	for _, language := range []string{"JAVASCRIPT"} {
		if !ScriptLanguageSupported(language) {
			t.Fatalf("应支持 %s", language)
		}
	}
	for _, language := range []string{"LUA", "WASM", "GO", "PYTHON", "STARLARK", "TCL", "CEL", "EXPR", "javascript", ""} {
		if ScriptLanguageSupported(language) {
			t.Fatalf("不应支持 %s", language)
		}
	}
}
