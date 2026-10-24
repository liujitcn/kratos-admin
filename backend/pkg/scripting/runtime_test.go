package scripting

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRuntimeExecute(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.Execute(context.Background(), JavaScriptLanguage, `ctx.answer = ctx.answer + 1; ctx.message = "ok"; ctx;`, map[string]any{"answer": 1})
	if err != nil {
		t.Fatalf("execute script: %v", err)
	}
	output, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected output type %T", result)
	}
	answer, err := strconv.ParseFloat(fmt.Sprint(output["answer"]), 64)
	if err != nil || math.Abs(answer-2) > 0.001 {
		t.Fatalf("unexpected answer: %#v", output["answer"])
	}
	if output["message"] != "ok" {
		t.Fatalf("unexpected message: %#v", output["message"])
	}
}

func TestRuntimeTimeout(t *testing.T) {
	runtime := &Runtime{timeout: 20 * time.Millisecond}
	_, err := runtime.Execute(context.Background(), JavaScriptLanguage, `for (;;) {}`, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !containsQuotaError(err) {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestRuntimeLua(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.Execute(context.Background(), LuaLanguage, `ctx.answer = ctx.answer + 1`, map[string]any{"answer": 3})
	if err != nil {
		t.Fatalf("execute lua script: %v", err)
	}
	output, ok := result.(map[string]any)
	if !ok || fmt.Sprint(output["answer"]) != "4" {
		t.Fatalf("unexpected lua output: %#v", result)
	}
}

func TestRuntimeGo(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.Execute(context.Background(), GoLanguage, `1 + 1`, map[string]any{"answer": 3})
	if err != nil {
		t.Fatalf("execute go script: %v", err)
	}
	if fmt.Sprint(result) != "2" {
		t.Fatalf("unexpected go output: %#v", result)
	}
}

func TestRuntimeOtherLanguages(t *testing.T) {
	runtime := NewRuntime()
	tests := []struct {
		name     string
		language string
		source   string
	}{
		{name: "python", language: PythonLanguage, source: `1 + 2`},
		{name: "cel", language: CELLanguage, source: `1 + 2`},
		{name: "expr", language: ExprLanguage, source: `1 + 2`},
		{name: "starlark", language: StarlarkLanguage, source: `ctx["answer"] = 4`},
		{name: "tcl", language: TclLanguage, source: `expr {1 + 2}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := runtime.Execute(context.Background(), test.language, test.source, map[string]any{"answer": 3})
			if err != nil {
				t.Fatalf("execute %s script: %v", test.language, err)
			}
		})
	}
}

func TestRuntimeRejectsUnknownLanguage(t *testing.T) {
	runtime := NewRuntime()
	_, err := runtime.Execute(context.Background(), "UNKNOWN", `return 1`, nil)
	if err == nil || !strings.Contains(err.Error(), "不支持的脚本语言") {
		t.Fatalf("expected unsupported language error, got %v", err)
	}
}

func TestRuntimeRejectsInvalidWasmBase64(t *testing.T) {
	runtime := NewRuntime()
	_, err := runtime.Execute(context.Background(), "WASM", "not-base64", nil)
	if err == nil || !strings.Contains(err.Error(), "WASM 源码必须是有效的 Base64") {
		t.Fatalf("expected invalid wasm base64 error, got %v", err)
	}
}

func TestParseContext(t *testing.T) {
	value, err := ParseContext(`{"name":"demo"}`)
	if err != nil {
		t.Fatalf("parse context: %v", err)
	}
	if value["name"] != "demo" {
		t.Fatalf("unexpected context: %#v", value)
	}
	if _, err = ParseContext(`[]`); err == nil {
		t.Fatal("expected object validation error")
	}
}

func containsQuotaError(err error) bool {
	return err != nil && (errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "execution quota exceeded"))
}
