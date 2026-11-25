package biz

import (
	"context"
	"slices"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/errorsx"
)

// runManagedShortcut 按最新配置以及当前会话已授权的工具池运行流程。
func (c *AiMessageCase) runManagedShortcut(ctx context.Context, session *models.AiSession, input ai.RuntimeInput, action *basev1.AiAction, latest string) (*ai.Response, error) {
	record, err := c.aiToolCase.findManagedShortcut(ctx, session, action)
	if err != nil {
		return nil, err
	}
	var graph *ai.ShortcutGraph
	graph, err = managedShortcutGraph(record.NodeConfig)
	if err != nil {
		return nil, err
	}
	ctx, err = c.bindAiSessionTools(ctx, session)
	if err != nil {
		return nil, err
	}
	required := ai.ShortcutToolNames(record.ToolName, graph)
	if len(required) > 0 {
		var allowed map[string]string
		allowed, err = c.aiToolCase.ShortcutAPIOptions(ctx, basev1.Terminal(session.Terminal))
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(required, func(name string) bool {
			_, exists := allowed[name]
			return !exists
		}) {
			return nil, errorsx.PermissionDenied("快捷入口依赖的工具在当前会话中不可用")
		}
	}
	input.Content = record.Prompt
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return c.aiRuntime.RunShortcutGraph(ctx, graph, input, action, latest)
}
