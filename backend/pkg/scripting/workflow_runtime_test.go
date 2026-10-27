package scripting

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWorkflowRuntimeIsolation 校验流程引擎不暴露宿主环境或模块加载。
func TestWorkflowRuntimeIsolation(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, `({process:typeof process,require:typeof require,os:typeof os,http:typeof http})`, nil)
	if err != nil {
		t.Fatal(err)
	}
	output := result.(map[string]any)
	for _, key := range []string{"process", "require", "os", "http"} {
		if output[key] != "undefined" {
			t.Fatalf("暴露宿主能力 %s: %v", key, output[key])
		}
	}
	path := filepath.Join(t.TempDir(), "benign.json")
	err = os.WriteFile(path, []byte(`{"benign":true}`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, `require("`+filepath.ToSlash(path)+`")`, nil)
	if err == nil {
		t.Fatal("不得读取宿主JSON")
	}
}

// TestWorkflowRuntimeContext 校验上下文深拷贝和结构化结果契约。
func TestWorkflowRuntimeContext(t *testing.T) {
	input := map[string]any{"form": map[string]any{"amount": 1}, "rows": []any{map[string]any{"value": "old"}}}
	runtime := NewRuntime()
	result, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, `ctx.form.amount=2;ctx.rows[0].value="new";({allowed:true,variables:{total:ctx.form.amount}})`, input)
	if err != nil {
		t.Fatal(err)
	}
	if input["form"].(map[string]any)["amount"] != 1 || input["rows"].([]any)[0].(map[string]any)["value"] != "old" {
		t.Fatal("输入被修改")
	}
	if result.(map[string]any)["allowed"] != true {
		t.Fatalf("结果契约错误: %#v", result)
	}
}

// TestWorkflowRuntimeTimeout 校验纯JavaScript无限循环可被超时取消。
func TestWorkflowRuntimeTimeout(t *testing.T) {
	runtime := &Runtime{timeout: 20 * time.Millisecond}
	_, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, `while(true){}`, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("未按期取消: %v", err)
	}
}

// TestWorkflowRuntimeSerializationTimeout 校验结果中的访问器与代理也受同一次超时约束。
func TestWorkflowRuntimeSerializationTimeout(t *testing.T) {
	cases := map[string]string{
		"getter":        `({get variables(){for(;;){}}})`,
		"proxy":         `new Proxy({}, {ownKeys(){for(;;){}}})`,
		"proxyGet":      `new Proxy({}, {get(){for(;;){}}})`,
		"contextGetter": `Object.defineProperty(globalThis,"ctx",{get(){for(;;){}}});undefined`,
		"toJSON":        `({toJSON(){for(;;){}}})`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			runtime := &Runtime{timeout: 20 * time.Millisecond}
			done := make(chan error, 1)
			go func() {
				_, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, source, nil)
				done <- err
			}()
			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("序列化未按期取消: %v", err)
				}
			case <-time.After(500 * time.Millisecond):
				t.Fatal("序列化逃逸执行超时")
			}
		})
	}
}

// TestWorkflowRuntimeSerializationJSON 校验保存的原生序列化器不受脚本覆盖且兼容上下文返回。
func TestWorkflowRuntimeSerializationJSON(t *testing.T) {
	runtime := NewRuntime()
	cases := []string{
		`JSON.stringify=()=>"{}"; ({allowed:true,variables:{total:3}})`,
		`ctx.allowed=true;ctx.variables={total:3};undefined`,
	}
	for _, source := range cases {
		result, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, source, nil)
		if err != nil {
			t.Fatal(err)
		}
		output := result.(map[string]any)
		if output["allowed"] != true || output["variables"].(map[string]any)["total"] != float64(3) {
			t.Fatalf("结果不符: %#v", result)
		}
	}
	_, err := runtime.ExecuteWorkflow(context.Background(), JavaScriptLanguage, `({get variables(){throw new Error("getter failure")}})`, nil)
	if err == nil {
		t.Fatal("抛错访问器必须返回错误")
	}
}
