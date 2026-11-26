package ai

import (
	"sync"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// ResponseBlockRule 描述一条按工具结果注入结构化引导块的规则。
// 业务模块在初始化阶段注册自己的规则，避免在通用链路里硬编码业务工具名。
type ResponseBlockRule struct {
	// ToolName 是触发引导块的工具名，与 base_api 注册的工具名一致。
	ToolName string
	// IncludeErrors 允许按稳定失败原因构建展示块。
	IncludeErrors bool
	// Build 根据工具结果构建引导块；返回空切片表示本次不注入。
	Build func(tool agent.ToolUsage) []map[string]any
}

var (
	responseBlockRulesMu sync.RWMutex
	responseBlockRules   []ResponseBlockRule
)

// RegisterResponseBlockRule 注册响应引导块规则；同一工具可以组合多个规则。
func RegisterResponseBlockRule(rule ResponseBlockRule) {
	if rule.ToolName == "" || rule.Build == nil {
		return
	}
	responseBlockRulesMu.Lock()
	defer responseBlockRulesMu.Unlock()
	responseBlockRules = append(responseBlockRules, rule)
}

// buildResponseBlocks 按工具执行顺序和规则注册顺序合并本轮结构化编排块。
func buildResponseBlocks(tools []agent.ToolUsage) []map[string]any {
	responseBlockRulesMu.RLock()
	rules := append([]ResponseBlockRule(nil), responseBlockRules...)
	responseBlockRulesMu.RUnlock()
	var blocks []map[string]any
	for _, tool := range tools {
		for _, rule := range rules {
			if rule.ToolName != tool.Name || (tool.Status != "success" && !(tool.Status == "error" && rule.IncludeErrors)) {
				continue
			}
			blocks = append(blocks, rule.Build(tool)...)
		}
	}
	return blocks
}
