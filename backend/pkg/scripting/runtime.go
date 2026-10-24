package scripting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/google/wire"
	scriptengine "github.com/liujitcn/go-scripts"
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
// 支持结构化全局变量的引擎可通过 ctx 读取和修改上下文，返回值为空时使用上下文作为输出。
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

	typ, err := scriptLanguageType(language)
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

	var engine scriptengine.Engine
	engine, err = scriptengine.NewScriptEngine(typ)
	if err != nil {
		return nil, fmt.Errorf("创建 %s 引擎失败: %w", language, err)
	}
	defer engine.Close()
	if err = engine.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化 %s 引擎失败: %w", language, err)
	}

	quota := scriptengine.AsQuotaController(engine)
	if quota != nil {
		quota.SetQuota(scriptengine.Quota{Timeout: r.timeout})
	}
	globalAccessor := scriptengine.AsGlobalAccessor(engine)
	if globalAccessor != nil {
		if err = globalAccessor.RegisterGlobal(contextGlobalName, cloneMap(input)); err != nil {
			return nil, fmt.Errorf("注入脚本上下文失败: %w", err)
		}
	}

	var result any
	loader := scriptengine.AsLoader(engine)
	syncExecutor := scriptengine.AsSyncExecutor(engine)
	if loader != nil && syncExecutor != nil {
		if err = loader.LoadString(ctx, "base_script", source); err != nil {
			return nil, fmt.Errorf("加载脚本失败: %w", err)
		}
		result, err = syncExecutor.ExecuteSync(ctx)
		if values, ok := result.([]any); ok && len(values) > 0 {
			result = values[len(values)-1]
		}
	} else {
		executor := scriptengine.AsExecutor(engine)
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
	value, err := globalAccessor.GetGlobal(contextGlobalName)
	if err != nil {
		return nil, fmt.Errorf("读取脚本上下文失败: %w", err)
	}
	return value, nil
}

// scriptLanguageType 将数据库语言标识转换为 go-scripts 引擎类型。
func scriptLanguageType(language string) (scriptengine.Type, error) {
	types := map[string]scriptengine.Type{
		JavaScriptLanguage: scriptengine.JavaScriptType,
		LuaLanguage:        scriptengine.LuaType,
		PythonLanguage:     scriptengine.GPythonType,
		GoLanguage:         scriptengine.YaegiType,
		WasmLanguage:       scriptengine.WazeroType,
		CELLanguage:        scriptengine.CELType,
		ExprLanguage:       scriptengine.ExprType,
		StarlarkLanguage:   scriptengine.StarlarkType,
		TclLanguage:        scriptengine.TclType,
	}
	typ, ok := types[language]
	if !ok {
		return scriptengine.UnknownType, fmt.Errorf("不支持的脚本语言: %s", language)
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

// cloneMap 复制脚本输入，避免脚本引擎改变调用方持有的 map。
func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	maps.Copy(output, input)
	return output
}
