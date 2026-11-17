package ai

import (
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// ResponseBlockRule 描述一条“工具执行成功后注入结构化引导块”的规则。
// 业务模块在初始化阶段注册自己的规则，避免在通用链路里硬编码业务工具名。
type ResponseBlockRule struct {
	// ToolName 是触发引导块的工具名，与 base_api 注册的工具名一致。
	ToolName string
	// Build 在工具执行成功时构建引导块；返回空切片表示本次不注入。
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
	log.Info("BI 引导块诊断", "rules", len(rules), "tools", len(tools))
	blocks := make([]map[string]any, 0)
	for _, tool := range tools {
		log.Info("BI 引导块诊断: tool", "name", tool.Name, "status", tool.Status, "output_len", len(tool.Output))
		if tool.Status != "success" {
			continue
		}
		for _, rule := range rules {
			if rule.ToolName != tool.Name {
				continue
			}
			if values := rule.Build(tool); len(values) > 0 {
				log.Info("BI 引导块诊断: 注入成功")
				blocks = append(blocks, values...)
			}
		}
	}
	if len(blocks) == 0 {
		log.Info("BI 引导块诊断: 未注入")
		return nil
	}
	return blocks
}
