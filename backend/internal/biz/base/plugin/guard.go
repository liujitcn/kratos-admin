package plugin

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

// AccessGuard 将请求工具池的逐次授权委托给持久化插件策略。
type AccessGuard struct {
	Configs func(context.Context, string, []string) (map[string]agent.ToolConfig, error)
	Prepare func(context.Context, string, string, string) (string, bool, error)
}

// ToolConfigs 重新读取工具池当前允许范围。
func (g *AccessGuard) ToolConfigs(ctx context.Context, terminal string, names []string) (map[string]agent.ToolConfig, error) {
	return g.Configs(ctx, terminal, names)
}

// PrepareToolCall 在每次标准、增强或恢复调用前重新校验。
func (g *AccessGuard) PrepareToolCall(ctx context.Context, terminal, name, arguments string) (string, bool, error) {
	return g.Prepare(ctx, terminal, name, arguments)
}
