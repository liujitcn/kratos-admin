package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-core/biz"
)

// AiToolCase 管理 AI 助手工具能力。
type AiToolCase struct {
	*biz.BaseCase
	aiRuntime *ai.Runtime
}

// NewAiToolCase 创建 AI 助手工具业务实例。
func NewAiToolCase(baseCase *biz.BaseCase, aiRuntime *ai.Runtime) *AiToolCase {
	return &AiToolCase{BaseCase: baseCase, aiRuntime: aiRuntime}
}

// ListAiShortcut 查询当前终端可用的 AI 助手快捷入口。
func (c *AiToolCase) ListAiShortcut(ctx context.Context, req *basev1.ListAiShortcutRequest) (*basev1.ListAiShortcutResponse, error) {
	terminal := ai.NormalizeTerminal(req.GetTerminal())
	terminalName := ai.NormalizeTerminalString(terminal)
	if c == nil || c.aiRuntime == nil {
		return &basev1.ListAiShortcutResponse{}, nil
	}
	enabledTools := c.aiRuntime.EnabledToolNames(ctx, terminalName)
	return &basev1.ListAiShortcutResponse{Shortcuts: c.aiRuntime.FixedFlowShortcuts(terminal, enabledTools)}, nil
}
