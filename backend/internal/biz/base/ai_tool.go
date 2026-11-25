package biz

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	admindata "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

// AiToolCase 管理 AI 助手工具能力与对话可用快捷入口。
type AiToolCase struct {
	*biz.BaseCase
	aiRuntime      *ai.Runtime
	aiShortcutRepo *data.AiShortcutRepository
	mcpCase        *McpCase
}

// NewAiToolCase 创建 AI 助手工具业务实例。
func NewAiToolCase(baseCase *biz.BaseCase, aiRuntime *ai.Runtime, aiShortcutRepo *data.AiShortcutRepository, mcpCase *McpCase) *AiToolCase {
	return &AiToolCase{BaseCase: baseCase, aiRuntime: aiRuntime, aiShortcutRepo: aiShortcutRepo, mcpCase: mcpCase}
}

// ListAiShortcut 汇总当前租户、终端与身份可使用的快捷入口。
func (c *AiToolCase) ListAiShortcut(ctx context.Context, req *basev1.ListAiShortcutRequest) (*basev1.ListAiShortcutResponse, error) {
	terminal := ai.NormalizeTerminal(req.GetTerminal())
	terminalName := ai.NormalizeTerminalString(terminal)
	enabledTools := c.aiRuntime.EnabledToolNames(ctx, terminalName)
	shortcuts := c.aiRuntime.FixedFlowShortcuts(terminal, enabledTools)
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.aiShortcutRepo.Query(ctx).AiShortcut
	var records []*models.AiShortcut
	records, err = c.aiShortcutRepo.List(ctx,
		repository.Where(query.TenantID.Eq(identity.TenantId), query.Terminal.Eq(terminal), query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))),
		repository.Order(query.Sort.Asc(), query.ID.Asc()),
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &basev1.ListAiShortcutResponse{Shortcuts: shortcuts}, nil
	}
	var options map[string]string
	if slices.ContainsFunc(records, func(record *models.AiShortcut) bool {
		return record.ToolName != "" || record.NodeConfig != ""
	}) {
		options, err = c.shortcutAPIOptions(ctx, enabledTools)
		if err != nil {
			return nil, err
		}
	}
	for _, record := range records {
		var graph *ai.ShortcutGraph
		graph, err = managedShortcutGraph(record.NodeConfig)
		if err != nil {
			log.Error("读取快捷入口流程失败", "shortcut_id", record.ID, "error", err)
			continue
		}
		if graph != nil && terminal == ai.TerminalApp {
			continue
		}
		required := ai.ShortcutToolNames(record.ToolName, graph)
		if slices.ContainsFunc(required, func(name string) bool {
			_, allowed := options[name]
			return !allowed
		}) {
			continue
		}
		shortcut := &basev1.AiShortcut{
			Key: "managed:" + record.ShortcutKey, Title: record.Title, Prompt: record.Prompt,
			Sort: record.Sort, Group: record.GroupName, RequiredTools: required,
		}
		if graph != nil {
			shortcut.Action = &basev1.AiShortcutAction{
				Flow: fmt.Sprintf("%s%d/%d", ai.ManagedShortcutFlowPrefix, record.ID, record.Revision),
				Step: "start", Type: "shortcut_start",
			}
		}
		shortcuts = append(shortcuts, shortcut)
	}
	return &basev1.ListAiShortcutResponse{Shortcuts: shortcuts}, nil
}

// ShortcutAPIOptions 返回当前终端和用户可使用的已登记只读 API 工具。
func (c *AiToolCase) ShortcutAPIOptions(ctx context.Context, terminal basev1.Terminal) (map[string]string, error) {
	enabled := c.aiRuntime.EnabledToolNames(ctx, ai.NormalizeTerminalString(int32(terminal)))
	return c.shortcutAPIOptions(ctx, enabled)
}

// ValidateShortcutToolArguments 校验只读工具请求的契约与当前授权。
func (c *AiToolCase) ValidateShortcutToolArguments(ctx context.Context, terminal basev1.Terminal, name, arguments string) error {
	if arguments == "" {
		arguments = "{}"
	}
	_, requiresApproval, err := c.mcpCase.PrepareToolCall(ctx, ai.NormalizeTerminalString(int32(terminal)), name, arguments)
	if err != nil {
		return err
	}
	if requiresApproval {
		return errorsx.InvalidArgument("快捷入口流程仅支持只读 API 工具")
	}
	return nil
}

// shortcutAPIOptions 从 API 快照筛选由运行时授权且所有映射均为只读的工具。
func (c *AiToolCase) shortcutAPIOptions(ctx context.Context, enabled map[string]bool) (map[string]string, error) {
	records, err := admindata.LoadBaseAPICache(ctx, c.mcpCase.baseAPIRepo, c.Cache)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(enabled))
	for name := range enabled {
		items := records[name]
		if len(items) == 0 || slices.ContainsFunc(items, func(item *models.BaseAPI) bool {
			return !agentToolReadOnly(item) || agentToolBlocked(item) || item.AgentStatus != int32(commonv1.Status_STATUS_ENABLE)
		}) {
			continue
		}
		result[name] = items[0].Desc
	}
	return result, nil
}

// findManagedShortcut 按当前租户、会话终端及动作修订号加载启用的流程。
func (c *AiToolCase) findManagedShortcut(ctx context.Context, session *models.AiSession, action *basev1.AiAction) (*models.AiShortcut, error) {
	id, revision, ok := ai.ParseManagedShortcutFlow(action.GetFlow())
	if !ok || session.Terminal != ai.TerminalAdmin {
		return nil, aiExpiredActionError("", "")
	}
	query := c.aiShortcutRepo.Query(ctx).AiShortcut
	record, err := c.aiShortcutRepo.Find(ctx, repository.Where(
		query.ID.Eq(id), query.TenantID.Eq(session.TenantID), query.Terminal.Eq(session.Terminal),
		query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE)), query.Revision.Eq(revision),
	))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, aiExpiredActionError("", "")
	}
	if err != nil {
		return nil, err
	}
	if record.NodeConfig == "" {
		return nil, aiExpiredActionError("", "")
	}
	return record, nil
}

// managedShortcutGraph 读取持久化图配置并用当前运行时规则校验。
func managedShortcutGraph(raw string) (*ai.ShortcutGraph, error) {
	if raw == "" {
		return nil, nil
	}
	graph := &adminv1.AiShortcutGraph{}
	err := protojson.Unmarshal([]byte(raw), graph)
	if err != nil {
		return nil, err
	}
	return ai.CompileShortcutGraph(graph)
}
