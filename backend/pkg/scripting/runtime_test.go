package scripting

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRuntimeRejectsHostCapabilities 验证脚本无法使用进程、文件和网络能力。
func TestRuntimeRejectsHostCapabilities(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "script-data.txt")
	err := os.WriteFile(fixture, []byte("script-file-marker"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	cases := []struct {
		name     string
		language string
		source   string
	}{
		{"javascript_process", JavaScriptLanguage, "process.env"},
		{"javascript_file", JavaScriptLanguage, fmt.Sprintf("require('fs').readFileSync(%q)", fixture)},
		{"javascript_network", JavaScriptLanguage, fmt.Sprintf("fetch(%q)", server.URL)},
		{"lua_process", LuaLanguage, `local pipe = io.popen("printf script-process-marker"); return pipe:read("*a")`},
		{"lua_file", LuaLanguage, fmt.Sprintf("local file = io.open(%q); return file:read('*a')", fixture)},
		{"lua_network", LuaLanguage, fmt.Sprintf("local http = require('http'); local client = http.client(); local request = http.request('GET', %q); return client:do_request(request)", server.URL)},
		{"tcl_process", TclLanguage, "exec echo script-process-marker"},
		{"tcl_file", TclLanguage, fmt.Sprintf("set file [open {%s} r]; read $file", fixture)},
		{"python_file", PythonLanguage, fmt.Sprintf("open(%q).read()", fixture)},
	}
	runtime := NewRuntime()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			output, err := runtime.Execute(context.Background(), test.language, test.source, nil)
			if err == nil {
				t.Fatalf("宿主能力应被拒绝，实际输出：%v", output)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("脚本发起了 %d 次网络请求", requests.Load())
	}
}

// TestRuntimeRejectsUnsafeLanguages 验证不受限语言在创建引擎前被统一拒绝。
func TestRuntimeRejectsUnsafeLanguages(t *testing.T) {
	cases := []struct {
		language string
		source   string
	}{
		{LuaLanguage, "return 2"},
		{PythonLanguage, "1 + 1"},
		{GoLanguage, "1 + 1"},
		{WasmLanguage, "AGFzbQEAAAA="},
		{StarlarkLanguage, "1 + 1"},
		{TclLanguage, "expr {1 + 1}"},
	}
	runtime := NewRuntime()
	for _, test := range cases {
		t.Run(test.language, func(t *testing.T) {
			_, err := runtime.Execute(context.Background(), test.language, test.source, nil)
			if err == nil || !strings.Contains(err.Error(), "不支持的脚本语言") {
				t.Fatalf("应拒绝不受限语言，实际错误：%v", err)
			}
		})
	}
}

// TestRuntimeKeepsPureDataScripts 验证正常 JavaScript 和数据表达式继续工作。
func TestRuntimeKeepsPureDataScripts(t *testing.T) {
	cases := []struct {
		language string
		source   string
	}{
		{JavaScriptLanguage, "({total: ctx.value + 1})"},
		{CELLanguage, "{'total': ctx.value + 1.0}"},
		{ExprLanguage, "{total: ctx.value + 1}"},
	}
	runtime := NewRuntime()
	for _, test := range cases {
		t.Run(test.language, func(t *testing.T) {
			output, err := runtime.Execute(context.Background(), test.language, test.source, map[string]any{"value": 2})
			if err != nil {
				t.Fatal(err)
			}
			var raw string
			raw, err = MarshalOutput(output)
			if err != nil || raw != `{"total":3}` {
				t.Fatalf("表达式结果错误：%s，%v", raw, err)
			}
		})
	}
}

// TestRuntimeJavascriptBounds 验证普通脚本复用流程脚本的输入、输出和取消边界。
func TestRuntimeJavascriptBounds(t *testing.T) {
	runtime := &Runtime{timeout: 20 * time.Millisecond}
	cases := []struct {
		name   string
		source string
	}{
		{"timeout", "while (true) {}"},
		{"result_getter_timeout", "({get value() { while (true) {} }})"},
		{"result_to_json_timeout", "({toJSON() { while (true) {} }})"},
		{"output_size", fmt.Sprintf("'x'.repeat(%d)", MaxContextBytes+1)},
		{"overridden_stringify", fmt.Sprintf("JSON.stringify = () => '{}'; ({value: 'x'.repeat(%d)})", MaxContextBytes+1)},
		{"source_size", strings.Repeat(";", MaxContextBytes+1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := runtime.Execute(context.Background(), JavaScriptLanguage, test.source, nil)
			if err == nil {
				t.Fatal("超出脚本边界的执行应失败")
			}
		})
	}
}
