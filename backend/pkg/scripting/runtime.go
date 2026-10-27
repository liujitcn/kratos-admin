package scripting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dop251/goja"

	"github.com/google/wire"
	script_engine "github.com/liujitcn/go-scripts"
	_ "github.com/liujitcn/go-scripts/cel"
	_ "github.com/liujitcn/go-scripts/expr"
	_ "github.com/liujitcn/go-scripts/gpython"
	_ "github.com/liujitcn/go-scripts/javascript"
	_ "github.com/liujitcn/go-scripts/lua"
	_ "github.com/liujitcn/go-scripts/starlark"
	_ "github.com/liujitcn/go-scripts/tcl"
	_ "github.com/liujitcn/go-scripts/wazero"
	_ "github.com/liujitcn/go-scripts/yaegi"
)

// ProviderSet 提供脚本运行时的依赖注入集合。
var ProviderSet = wire.NewSet(NewRuntime)

const (
	// JavaScriptLanguage 是脚本表中 JavaScript 的稳定语言标识。
	JavaScriptLanguage = "JAVASCRIPT"
	// LuaLanguage 是脚本表中 Lua 的稳定语言标识。
	LuaLanguage = "LUA"
	// PythonLanguage 是脚本表中 Python 的稳定语言标识。
	PythonLanguage = "PYTHON"
	// GoLanguage 是脚本表中 Go 的稳定语言标识。
	GoLanguage = "GO"
	// WasmLanguage 是脚本表中 WebAssembly 的稳定语言标识。
	WasmLanguage = "WASM"
	// CELLanguage 是脚本表中 CEL 的稳定语言标识。
	CELLanguage = "CEL"
	// ExprLanguage 是脚本表中 Expr 的稳定语言标识。
	ExprLanguage = "EXPR"
	// StarlarkLanguage 是脚本表中 Starlark 的稳定语言标识。
	StarlarkLanguage = "STARLARK"
	// TclLanguage 是脚本表中 TCL 的稳定语言标识。
	TclLanguage = "TCL"
	// DefaultTimeout 是单次脚本执行的最大耗时。
	DefaultTimeout = 5 * time.Second
	// MaxContextBytes 是单次脚本上下文允许的最大 JSON 大小。
	MaxContextBytes = 200 * 1024
)

// Runtime 负责创建隔离的脚本执行实例。
type Runtime struct {
	timeout time.Duration
}

// NewRuntime 创建脚本执行运行时。
func NewRuntime() *Runtime {
	return &Runtime{timeout: DefaultTimeout}
}

// Execute 执行一段指定语言的源码并返回脚本输出。
// 输入按 JSON 深拷贝隔离；支持结构化全局变量的引擎通过 ctx 读取上下文。
// 超时依赖引擎支持取消；GPython、Starlark、TCL 等现有适配器不保证硬中断。
func (r *Runtime) Execute(ctx context.Context, language string, source string, input map[string]any) (any, error) {
	if r == nil {
		return nil, errors.New("脚本运行时未初始化")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if source == "" {
		return nil, errors.New("脚本源码不能为空")
	}
	if input == nil {
		input = make(map[string]any)
	}

	timeout := r.timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	rawInput, err := MarshalOutput(input)
	if err != nil {
		return nil, fmt.Errorf("脚本输入无效: %w", err)
	}
	var isolatedInput map[string]any
	isolatedInput, err = ParseContext(rawInput)
	if err != nil {
		return nil, fmt.Errorf("脚本输入无效: %w", err)
	}
	var typ script_engine.Type
	typ, err = scriptLanguageType(language)
	if err != nil {
		return nil, err
	}
	contextGlobalName := "ctx"
	if language == GoLanguage {
		contextGlobalName = "Ctx"
	}
	if language == WasmLanguage {
		var decoded []byte
		decoded, err = base64.StdEncoding.DecodeString(source)
		if err != nil {
			return nil, fmt.Errorf("WASM 源码必须是有效的 Base64: %w", err)
		}
		source = string(decoded)
	}

	var engine script_engine.Engine
	engine, err = script_engine.NewScriptEngine(typ)
	if err != nil {
		return nil, fmt.Errorf("创建 %s 引擎失败: %w", language, err)
	}
	defer engine.Close()
	if err = engine.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化 %s 引擎失败: %w", language, err)
	}

	quota := script_engine.AsQuotaController(engine)
	if quota != nil {
		quota.SetQuota(script_engine.Quota{Timeout: timeout})
	}
	globalAccessor := script_engine.AsGlobalAccessor(engine)
	if globalAccessor != nil {
		if err = globalAccessor.RegisterGlobal(contextGlobalName, isolatedInput); err != nil {
			return nil, fmt.Errorf("注入脚本上下文失败: %w", err)
		}
	}

	var result any
	loader := script_engine.AsLoader(engine)
	syncExecutor := script_engine.AsSyncExecutor(engine)
	if loader != nil && syncExecutor != nil {
		if err = loader.LoadString(ctx, "base_script", source); err != nil {
			return nil, fmt.Errorf("加载脚本失败: %w", err)
		}
		result, err = syncExecutor.ExecuteSync(ctx)
		if values, ok := result.([]any); ok && len(values) > 0 {
			result = values[len(values)-1]
		}
	} else {
		executor := script_engine.AsExecutor(engine)
		if executor == nil {
			return nil, errors.New("脚本引擎不支持执行能力")
		}
		result, err = executor.ExecuteString(ctx, "base_script", source)
	}
	if err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}
	if result != nil || language == WasmLanguage || language == GoLanguage || globalAccessor == nil {
		return result, nil
	}
	var value any
	value, err = globalAccessor.GetGlobal(contextGlobalName)
	if err != nil {
		return nil, fmt.Errorf("读取脚本上下文失败: %w", err)
	}
	return value, nil
}

// ExecuteWorkflow 使用不注册宿主模块的纯 JavaScript VM 执行流程脚本。
// 仅注入原生 JSON 上下文，不提供 process、require、文件、网络或 Go 对象能力。
func (r *Runtime) ExecuteWorkflow(ctx context.Context, language string, source string, input map[string]any) (any, error) {
	if r == nil {
		return nil, errors.New("脚本运行时未初始化")
	}
	if language != JavaScriptLanguage {
		return nil, errors.New("流程仅支持 JavaScript 脚本")
	}
	if source == "" {
		return nil, errors.New("脚本源码不能为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := r.timeout
	if timeout <= 0 || timeout > DefaultTimeout {
		timeout = DefaultTimeout
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	if input == nil {
		input = map[string]any{}
	}
	raw, err := MarshalOutput(input)
	if err != nil {
		return nil, fmt.Errorf("脚本输入无效: %w", err)
	}
	// JSON.parse 产生纯 JS 对象，避免把 Go map、切片或方法反射到脚本环境。
	vm := goja.New()
	// 在用户脚本执行前保存原生函数，脚本覆盖 JSON.stringify 不得绕过结果检查。
	stringify, ok := goja.AssertFunction(vm.Get("JSON").ToObject(vm).Get("stringify"))
	if !ok {
		return nil, errors.New("初始化流程 JSON 序列化器失败")
	}
	var value goja.Value
	value, err = vm.RunString("JSON.parse(" + strconv.Quote(raw) + ")")
	if err != nil {
		return nil, fmt.Errorf("解析流程上下文失败: %w", err)
	}
	err = vm.Set("ctx", value)
	if err != nil {
		return nil, fmt.Errorf("注入流程上下文失败: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { vm.Interrupt(ctx.Err()) })
	defer stop()
	value, err = vm.RunString(source)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("执行流程脚本失败: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("执行流程脚本失败: %w", err)
	}
	// 结果访问器、Proxy 和 toJSON 均可能运行用户代码，序列化完成前不能停止中断。
	var serialized goja.Value
	var exception *goja.Exception
	func() {
		// Try 捕获普通 JS 异常，Interrupt 属于不可捕获异常，需在 Go 边界单独转换。
		defer func() {
			if failure := recover(); failure != nil {
				switch interrupted := failure.(type) {
				case *goja.InterruptedError:
					err = interrupted
				case *goja.Exception:
					err = interrupted
				default:
					panic(failure)
				}
			}
		}()
		exception = vm.Try(func() {
			if goja.IsUndefined(value) || goja.IsNull(value) {
				value = vm.Get("ctx")
			}
			serialized, err = stringify(goja.Undefined(), value)
		})
	}()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("序列化流程结果失败: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("序列化流程结果失败: %w", err)
	}
	if exception != nil {
		return nil, fmt.Errorf("序列化流程结果失败: %w", exception)
	}
	if goja.IsUndefined(serialized) {
		return nil, errors.New("流程结果不是有效 JSON")
	}
	// 原生 stringify 只返回字符串或 undefined，不再读取脚本对象或调用 Export。
	outputJSON := serialized.String()
	if len(outputJSON) > MaxContextBytes {
		return nil, fmt.Errorf("脚本输出不能超过 %d 字节", MaxContextBytes)
	}
	var output any
	err = json.Unmarshal([]byte(outputJSON), &output)
	stop()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("序列化流程结果失败: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("流程结果不是有效 JSON: %w", err)
	}
	return output, nil
}

// scriptLanguageType 将数据库语言标识转换为 go-scripts 引擎类型。
func scriptLanguageType(language string) (script_engine.Type, error) {
	types := map[string]script_engine.Type{
		JavaScriptLanguage: script_engine.JavaScriptType,
		LuaLanguage:        script_engine.LuaType,
		PythonLanguage:     script_engine.GPythonType,
		GoLanguage:         script_engine.YaegiType,
		WasmLanguage:       script_engine.WazeroType,
		CELLanguage:        script_engine.CELType,
		ExprLanguage:       script_engine.ExprType,
		StarlarkLanguage:   script_engine.StarlarkType,
		TclLanguage:        script_engine.TclType,
	}
	typ, ok := types[language]
	if !ok {
		return script_engine.UnknownType, fmt.Errorf("不支持的脚本语言: %s", language)
	}
	return typ, nil
}

// ParseContext 将 JSON 上下文解析为脚本可用的对象。
func ParseContext(raw string) (map[string]any, error) {
	if len(raw) > MaxContextBytes {
		return nil, fmt.Errorf("脚本上下文不能超过 %d 字节", MaxContextBytes)
	}
	if raw == "" {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("脚本上下文必须是 JSON 对象: %w", err)
	}
	if value == nil {
		return map[string]any{}, nil
	}
	return value, nil
}

// MarshalOutput 将脚本结果编码为管理端展示用 JSON。
func MarshalOutput(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("脚本输出不是有效 JSON: %w", err)
	}
	if len(data) > MaxContextBytes {
		return "", fmt.Errorf("脚本输出不能超过 %d 字节", MaxContextBytes)
	}
	return string(data), nil
}
