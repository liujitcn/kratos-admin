package scripting

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/securityconfig"
)

// TestIsolatedLanguages 验证九种语言在真实隔离进程中仍可执行并返回可序列化结果。
func TestIsolatedLanguages(t *testing.T) {
	wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0, 1, 4, 1, 96, 0, 0, 3, 2, 1, 0, 7, 10, 1, 6, 95, 115, 116, 97, 114, 116, 0, 0, 10, 4, 1, 2, 0, 11}
	for language, source := range map[string]string{JavaScriptLanguage: "ctx.value + 1", LuaLanguage: "ctx.result = ctx.value + 1", PythonLanguage: "1 + 1", GoLanguage: "1 + 1", CELLanguage: "1 + 1", ExprLanguage: "1 + 1", StarlarkLanguage: "result = ctx['value'] + 1", TclLanguage: "expr {[dict get $ctx value] + 1}", WasmLanguage: base64.StdEncoding.EncodeToString(wasm)} {
		t.Run(language, func(t *testing.T) {
			result, err := NewRuntime().Execute(t.Context(), language, source, map[string]any{"value": 1})
			if err != nil {
				t.Fatal(err)
			}
			var output string
			output, err = MarshalOutput(result)
			if err != nil || output == "" {
				t.Fatalf("脚本输出无效: %s, %v", output, err)
			}
			if language != StarlarkLanguage && language != WasmLanguage && language != LuaLanguage && language != PythonLanguage && output != "2" {
				t.Fatalf("计算结果回归: %s", output)
			}
		})
	}
}

// TestIsolatedFileAccess 验证 Lua 和 Tcl 文件能力保留，但只能读取部署明确授权的文件。
func TestIsolatedFileAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.txt")
	err := os.WriteFile(path, []byte("approved-content"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	for language, source := range map[string]string{LuaLanguage: "local f = assert(io.open(ctx.path)); ctx.output = f:read('*a')", TclLanguage: "set f [open [dict get $ctx path]]; read $f"} {
		t.Run(language, func(t *testing.T) {
			runtime := NewRuntime()
			_, err := runtime.Execute(t.Context(), language, source, map[string]any{"path": path})
			if err == nil {
				t.Fatal("未授权文件可读")
			}
			securityRuntimeConfig := securityconfig.Current()
			securityRuntimeConfig.ScriptReadPaths = []string{path}
			securityconfig.Set(securityRuntimeConfig)
			t.Cleanup(func() { securityconfig.Set(securityconfig.Default()) })
			var result any
			result, err = runtime.Execute(t.Context(), language, source, map[string]any{"path": path})
			encoded, marshalErr := MarshalOutput(result)
			if err != nil || marshalErr != nil || !strings.Contains(encoded, "approved-content") {
				t.Fatalf("授权文件访问回归: %v, %v", result, err)
			}
		})
	}
}

// TestIsolatedHTTP 验证原 Lua HTTP API 经受控代理仍可使用，未授权目标没有实际出站请求。
func TestIsolatedHTTP(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits.Add(1)
		_, err := writer.Write([]byte("approved-response"))
		if err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	source := "local h=require('http'); local c=h.client(); local r,e=c:do_request(h.request('GET',ctx.url)); if e then error(e) end; ctx.output=r.body"
	_, err := NewRuntime().Execute(t.Context(), LuaLanguage, source, map[string]any{"url": server.URL})
	if hits.Load() != 0 {
		t.Fatal("未授权请求离开了网络代理")
	}
	if err != nil && !strings.Contains(err.Error(), "未授权") && !strings.Contains(err.Error(), "connect") {
		t.Fatalf("HTTP 能力未正常初始化: %v", err)
	}
	securityRuntimeConfig := securityconfig.Current()
	securityRuntimeConfig.ScriptAllowedCIDRs = []string{"127.0.0.1/32"}
	securityconfig.Set(securityRuntimeConfig)
	t.Cleanup(func() { securityconfig.Set(securityconfig.Default()) })
	var result any
	result, err = NewRuntime().Execute(t.Context(), LuaLanguage, source, map[string]any{"url": server.URL})
	encoded, marshalErr := MarshalOutput(result)
	if err != nil || marshalErr != nil || !strings.Contains(encoded, "approved-response") || hits.Load() != 1 {
		t.Fatalf("授权 HTTP 请求失败: %v, %v, hits=%d", result, err, hits.Load())
	}
}

// TestIsolatedIntegerPrecision 验证进程通信不会损坏 Tcl 的 64 位整型结果。
func TestIsolatedIntegerPrecision(t *testing.T) {
	result, err := NewRuntime().Execute(t.Context(), TclLanguage, "expr {9007199254740993}", nil)
	if err != nil {
		t.Fatal(err)
	}
	var output string
	output, err = MarshalOutput(result)
	if err != nil || output != strconv.FormatInt(9007199254740993, 10) {
		t.Fatalf("整型精度丢失: %s, %v", output, err)
	}
}
