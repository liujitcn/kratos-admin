package scripting

import (
	"context"
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
	// DefaultTimeout 是单次脚本执行的默认时间预算。
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

// Execute 仅执行无宿主能力的 JavaScript 和 JSON 数据表达式。
// CEL、Expr 不注册函数或模块；只接收 JSON 深拷贝，不能反射调用宿主对象。
func (r *Runtime) Execute(ctx context.Context, language string, source string, input map[string]any) (result any, err error) {
	if !LanguageSupported(language) {
		return nil, fmt.Errorf("不支持的脚本语言: %s，仅支持 JavaScript、CEL、Expr", language)
	}
	if language == JavaScriptLanguage {
		return r.ExecuteWorkflow(ctx, language, source, input)
	}
	if r == nil {
		return nil, errors.New("脚本运行时未初始化")
	}
	if source == "" {
		return nil, errors.New("脚本源码不能为空")
	}
	if len(source) > MaxContextBytes {
		return nil, fmt.Errorf("脚本源码不能超过 %d 字节", MaxContextBytes)
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
	// shortcut: CEL/Expr 适配器仅支持执行前后检查取消，强制中断需升级支持执行预算的引擎。
	if err = ctx.Err(); err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}
	if input == nil {
		input = map[string]any{}
	}
	var rawInput string
	rawInput, err = MarshalOutput(input)
	if err != nil {
		return nil, fmt.Errorf("脚本输入无效: %w", err)
	}
	var isolatedInput map[string]any
	isolatedInput, err = ParseContext(rawInput)
	if err != nil {
		return nil, fmt.Errorf("脚本输入无效: %w", err)
	}
	typ := script_engine.CELType
	if language == ExprLanguage {
		typ = script_engine.ExprType
	}
	var engine script_engine.Engine
	engine, err = script_engine.NewScriptEngine(typ)
	if err != nil {
		return nil, fmt.Errorf("创建 %s 引擎失败: %w", language, err)
	}
	if err = engine.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化 %s 引擎失败: %w", language, err)
	}
	defer func() {
		closeErr := engine.Close()
		if closeErr != nil {
			err = errors.Join(err, fmt.Errorf("关闭脚本引擎失败: %w", closeErr))
		}
	}()
	if err = engine.RegisterGlobal("ctx", isolatedInput); err != nil {
		return nil, fmt.Errorf("注入脚本上下文失败: %w", err)
	}
	result, err = engine.ExecuteString(ctx, "base_script", source)
	if err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return nil, fmt.Errorf("执行脚本失败: %w", err)
	}
	if result == nil {
		result = isolatedInput
	}
	if _, err = MarshalOutput(result); err != nil {
		return nil, err
	}
	return result, nil
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
	if len(source) > MaxContextBytes {
		return nil, fmt.Errorf("脚本源码不能超过 %d 字节", MaxContextBytes)
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

// LanguageSupported 判断语言是否具有不注册宿主模块的受限执行入口。
func LanguageSupported(language string) bool {
	return language == JavaScriptLanguage || language == CELLanguage || language == ExprLanguage
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
