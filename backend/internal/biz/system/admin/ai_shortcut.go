package biz

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	basebiz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AiShortcutCase 管理当前租户的 AI 快捷入口。
type AiShortcutCase struct {
	*biz.BaseCase
	*data.AiShortcutRepository
	tx         data.Transaction
	aiToolCase *basebiz.AiToolCase
	formMapper *mapper.CopierMapper[adminv1.AiShortcutForm, models.AiShortcut]
}

// NewAiShortcutCase 创建快捷入口管理业务实例。
func NewAiShortcutCase(baseCase *biz.BaseCase, repo *data.AiShortcutRepository, tx data.Transaction, aiToolCase *basebiz.AiToolCase) *AiShortcutCase {
	return &AiShortcutCase{
		BaseCase: baseCase, AiShortcutRepository: repo, tx: tx, aiToolCase: aiToolCase,
		formMapper: mapper.NewCopierMapper[adminv1.AiShortcutForm, models.AiShortcut](),
	}
}

// PageAiShortcut 分页查询当前租户的快捷入口列表。
func (c *AiShortcutCase) PageAiShortcut(ctx context.Context, req *adminv1.PageAiShortcutRequest) (*adminv1.PageAiShortcutResponse, error) {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiShortcut
	opts := []repository.QueryOption{
		repository.Where(query.TenantID.Eq(identity.TenantId)),
		repository.Order(query.Sort.Asc(), query.ID.Asc()),
	}
	if req.GetTitle() != "" {
		opts = append(opts, repository.Where(query.Title.Like("%"+req.GetTitle()+"%")))
	}
	if req.GetShortcutKey() != "" {
		opts = append(opts, repository.Where(query.ShortcutKey.Like("%"+req.GetShortcutKey()+"%")))
	}
	if req.SourceType != nil {
		opts = append(opts, repository.Where(query.SourceType.Eq(int32(req.GetSourceType()))))
	}
	if req.Terminal != nil {
		opts = append(opts, repository.Where(query.Terminal.Eq(int32(req.GetTerminal()))))
	}
	if req.GetToolName() != "" {
		opts = append(opts, repository.Where(query.ToolName.Like("%"+req.GetToolName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var records []*models.AiShortcut
	var total int64
	records, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*adminv1.AiShortcutForm, 0, len(records))
	for _, record := range records {
		form := c.formMapper.ToDTO(record)
		form.UpdatedAt = record.UpdatedAt.Format(time.DateTime)
		list = append(list, form)
	}
	return &adminv1.PageAiShortcutResponse{Shortcuts: list, Total: int32(total)}, nil
}

// GetAiShortcut 查询当前租户的入口配置。
func (c *AiShortcutCase) GetAiShortcut(ctx context.Context, id int64) (*adminv1.AiShortcutForm, error) {
	record, err := c.findShortcut(ctx, id)
	if err != nil {
		return nil, err
	}
	form := c.formMapper.ToDTO(record)
	form.UpdatedAt = record.UpdatedAt.Format(time.DateTime)
	if record.NodeConfig != "" {
		form.Graph = &adminv1.AiShortcutGraph{}
		if err = protojson.Unmarshal([]byte(record.NodeConfig), form.Graph); err != nil {
			return nil, errorsx.Internal("快捷入口流程配置无法读取").WithCause(err)
		}
	}
	return form, nil
}

// OptionAiShortcutTool 查询当前主体获准使用的 API 工具。
func (c *AiShortcutCase) OptionAiShortcutTool(ctx context.Context, terminal basev1.Terminal) (*adminv1.OptionAiShortcutToolResponse, error) {
	options, err := c.aiToolCase.ShortcutAPIOptions(ctx, terminal)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.AiShortcutToolOption, 0, len(options))
	for name, title := range options {
		result = append(result, &adminv1.AiShortcutToolOption{ToolName: name, Title: title})
	}
	slices.SortFunc(result, func(left, right *adminv1.AiShortcutToolOption) int {
		return strings.Compare(left.ToolName, right.ToolName)
	})
	return &adminv1.OptionAiShortcutToolResponse{Tools: result}, nil
}

// CreateAiShortcut 校验并保存当前租户的快捷入口。
func (c *AiShortcutCase) CreateAiShortcut(ctx context.Context, form *adminv1.AiShortcutForm) error {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var nodeConfig string
	nodeConfig, err = c.validateShortcut(ctx, form)
	if err != nil {
		return err
	}
	record := c.formMapper.ToEntity(form)
	record.ID, record.TenantID, record.Revision = 0, identity.TenantId, 1
	record.CreatedBy, record.UpdatedBy = identity.UserId, identity.UserId
	record.CreatedAt, record.UpdatedAt = time.Time{}, time.Time{}
	record.NodeConfig = nodeConfig
	if err = c.Create(ctx, record); err != nil {
		return shortcutWriteError(err)
	}
	return nil
}

// UpdateAiShortcut 按当前修订保存入口和完整流程图，启停由独立接口处理。
func (c *AiShortcutCase) UpdateAiShortcut(ctx context.Context, form *adminv1.AiShortcutForm) error {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var record *models.AiShortcut
	record, err = c.findShortcut(ctx, form.GetId())
	if err != nil {
		return err
	}
	if form.GetShortcutKey() != record.ShortcutKey {
		return errorsx.InvalidArgument("快捷入口标识创建后不可修改")
	}
	if form.GetRevision() != record.Revision {
		return errorsx.Conflict("快捷入口已修改，请刷新后重试")
	}
	var nodeConfig string
	nodeConfig, err = c.validateShortcut(ctx, form)
	if err != nil {
		return err
	}
	query := c.Query(ctx).AiShortcut
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).
		Where(query.ID.Eq(record.ID), query.TenantID.Eq(identity.TenantId), query.Revision.Eq(record.Revision)).
		UpdateSimple(
			query.Title.Value(form.GetTitle()),
			query.SourceType.Value(int32(form.GetSourceType())),
			query.Terminal.Value(int32(form.GetTerminal())),
			query.ToolName.Value(form.GetToolName()),
			query.Prompt.Value(form.GetPrompt()),
			query.GroupName.Value(form.GetGroupName()),
			query.Sort.Value(form.GetSort()),
			query.NodeConfig.Value(nodeConfig),
			query.Revision.Value(record.Revision+1),
			query.UpdatedBy.Value(identity.UserId),
			query.UpdatedAt.Value(time.Now()),
		)
	if err != nil {
		return shortcutWriteError(err)
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("快捷入口已修改，请刷新后重试")
	}
	return nil
}

// DeleteAiShortcut 原子删除当前租户指定的全部快捷入口。
func (c *AiShortcutCase) DeleteAiShortcut(ctx context.Context, rawIDs string) error {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	ids := make([]int64, 0)
	for value := range strings.SplitSeq(rawIDs, ",") {
		var id int64
		id, err = strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 || len(ids) == 100 {
			return errorsx.InvalidArgument("请选择一到一百条有效快捷入口")
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.Query(txCtx).AiShortcut
		conditions := []gen.Condition{query.TenantID.Eq(identity.TenantId), query.ID.In(ids...)}
		var records []*models.AiShortcut
		records, err = query.WithContext(txCtx).Where(conditions...).Clauses(clause.Locking{Strength: "UPDATE"}).Order(query.ID.Asc()).Find()
		if err != nil {
			return err
		}
		if len(records) != len(ids) {
			return errorsx.ResourceNotFound("快捷入口不存在")
		}
		var result gen.ResultInfo
		result, err = query.WithContext(txCtx).Where(conditions...).Delete()
		if err != nil {
			return err
		}
		if result.RowsAffected != int64(len(ids)) {
			return errorsx.Conflict("快捷入口已修改，请刷新后重试")
		}
		return nil
	})
}

// SetAiShortcutStatus 修改入口状态，启用时重新确认所有引用的工具可用。
func (c *AiShortcutCase) SetAiShortcutStatus(ctx context.Context, req *adminv1.SetAiShortcutStatusRequest) error {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var form *adminv1.AiShortcutForm
	form, err = c.GetAiShortcut(ctx, req.GetId())
	if err != nil {
		return err
	}
	if form.GetRevision() != req.GetRevision() {
		return errorsx.Conflict("快捷入口已修改，请刷新后重试")
	}
	if req.GetStatus() == commonv1.Status_STATUS_ENABLE {
		if _, err = c.validateShortcut(ctx, form); err != nil {
			return err
		}
	}
	query := c.Query(ctx).AiShortcut
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).
		Where(query.ID.Eq(req.GetId()), query.TenantID.Eq(identity.TenantId), query.Revision.Eq(form.GetRevision())).
		UpdateSimple(query.Status.Value(int32(req.GetStatus())), query.Revision.Value(form.GetRevision()+1), query.UpdatedBy.Value(identity.UserId), query.UpdatedAt.Value(time.Now()))
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return errorsx.Conflict("快捷入口已修改，请刷新后重试")
	}
	return nil
}

// findShortcut 根据当前认证租户读取入口，隐藏其他租户的记录。
func (c *AiShortcutCase) findShortcut(ctx context.Context, id int64) (*models.AiShortcut, error) {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiShortcut
	var record *models.AiShortcut
	record, err = c.Find(ctx, repository.Where(query.TenantID.Eq(identity.TenantId), query.ID.Eq(id)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("快捷入口不存在")
	}
	return record, err
}

// validateShortcut 校验来源、终端、图结构及当前用户可用的工具引用。
func (c *AiShortcutCase) validateShortcut(ctx context.Context, form *adminv1.AiShortcutForm) (string, error) {
	if strings.TrimSpace(form.GetTitle()) == "" || strings.TrimSpace(form.GetPrompt()) == "" {
		return "", errorsx.InvalidArgument("入口名称和提问内容不能为空")
	}
	if (form.GetSourceType() == adminv1.AiShortcutSourceType_AI_SHORTCUT_SOURCE_TYPE_API) != (form.GetToolName() != "") {
		return "", errorsx.InvalidArgument("API 来源需要选择工具，自定义入口使用流程配置工具")
	}
	var err error
	var graph *ai.ShortcutGraph
	var raw []byte
	if form.GetGraph() != nil {
		if form.GetTerminal() != basev1.Terminal_TERMINAL_ADMIN {
			return "", errorsx.InvalidArgument("应用端快捷入口仅支持直接发送提示词")
		}
		graph, err = ai.CompileShortcutGraph(form.GetGraph())
		if err != nil {
			return "", err
		}
		raw, err = (protojson.MarshalOptions{UseProtoNames: true}).Marshal(form.GetGraph())
		if err != nil {
			return "", errorsx.InvalidArgument("快捷入口流程配置无法保存").WithCause(err)
		}
		if len(raw) > 60000 {
			return "", errorsx.InvalidArgument("快捷入口流程配置过大")
		}
	}
	names := ai.ShortcutToolNames(form.GetToolName(), graph)
	if len(names) == 0 {
		return string(raw), nil
	}
	var options map[string]string
	options, err = c.aiToolCase.ShortcutAPIOptions(ctx, form.GetTerminal())
	if err != nil {
		return "", err
	}
	for _, name := range names {
		if _, exists := options[name]; !exists {
			return "", errorsx.InvalidArgument("快捷入口包含不可用的 API 工具")
		}
	}
	for _, node := range form.GetGraph().GetNodes() {
		if node.GetToolName() != "" {
			err = c.aiToolCase.ValidateShortcutToolArguments(ctx, form.GetTerminal(), node.GetToolName(), node.GetPayloadJson())
			if err != nil {
				return "", err
			}
		}
	}
	return string(raw), nil
}

// shortcutWriteError 将入口唯一约束冲突转换为业务错误。
func shortcutWriteError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errorsx.Conflict("同一租户和终端的快捷入口标识已存在")
	}
	return err
}
