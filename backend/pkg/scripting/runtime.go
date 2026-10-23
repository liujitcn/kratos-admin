package scripting

import (
	"maps"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/wire"
	scriptengine "github.com/liujitcn/go-scripts"
	_ "github.com/liujitcn/go-scripts/javascript"
)

// ProviderSet 提供脚本运行时的依赖注入集合。
var ProviderSet = wire.NewSet(NewRuntime)

const (
	// JavaScriptLanguage 是脚本表中 JavaScript 的稳定语言标识。
	JavaScriptLanguage = "JAVASCRIPT"
	// DefaultTimeout 是单次脚本执行的最大耗时。
	DefaultTimeout = 5 * time.Second
	// MaxContextBytes 是单次脚本上下文允许的最大 JSON 大小。
	MaxContextBytes = 200 * 1024
)

// Runtime 负责创建隔离的 JavaScript 脚本执行实例。
type Runtime struct {
	timeout time.Duration
}

// NewRuntime 创建脚本执行运行时。
func NewRuntime() *Runtime {
	return &Runtime{timeout: DefaultTimeout}
}

// Execute 执行一段 JavaScript 源码并返回脚本输出。
// 脚本可以通过全局对象 ctx 读取和修改上下文，返回值为空时使用 ctx 作为输出。
func (r *Runtime) Execute(ctx context.Context, source string, input map[string]any) (any, error) {
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

	engine, err := scriptengine.NewScriptEngine(scriptengine.JavaScriptType)
	if err != nil {
		return nil, fmt.Errorf("创建 JavaScript 引擎失败: %w", err)
	}
	defer engine.Close()
	if err = engine.Init(ctx); err != nil {
		return nil, fmt.Errorf("初始化 JavaScript 引擎失败: %w", err)
	}

	quota := scriptengine.AsQuotaController(engine)
	if quota != nil {
		quota.SetQuota(scriptengine.Quota{Timeout: r.timeout})
	}
	if err = engine.RegisterGlobal("ctx", cloneMap(input)); err != nil {
		return nil, fmt.Errorf("注入脚本上下文失败: %w", err)
	}

	var result any
	loader := scriptengine.AsLoader(engine)
	syncExecutor := scriptengine.AsSyncExecutor(engine)
	if loader != nil && syncExecutor != nil {
		if err = loader.LoadString(ctx, "base_script.js", source); err != nil {
			return nil, fmt.Errorf("加载脚本失败: %w", err)
		}
		var results any
		results, err = syncExecutor.ExecuteSync(ctx)
		if err != nil {
			return nil, fmt.Errorf("执行脚本失败: %w", err)
		}
		if values, ok := results.([]any); ok && len(values) > 0 {
			result = values[len(values)-1]
		}
	} else {
		result, err = engine.ExecuteString(ctx, "base_script.js", source)
		if err != nil {
			return nil, fmt.Errorf("执行脚本失败: %w", err)
		}
	}
	if result != nil {
		return result, nil
	}
	value, err := engine.GetGlobal("ctx")
	if err != nil {
		return nil, fmt.Errorf("读取脚本上下文失败: %w", err)
	}
	return value, nil
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
