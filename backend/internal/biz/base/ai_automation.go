package biz

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/automation"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AiAutomationCase 管理本人计划、持久化执行及现有 AI 执行链的调度。
type AiAutomationCase struct {
	*biz.BaseCase
	*data.AiAutomationRepository
	tx            data.Transaction
	runRepo       *data.AiAutomationRunRepository
	userRepo      *data.BaseUserRepository
	aiSessionCase *AiSessionCase
	aiMessageCase *AiMessageCase
	loginCase     *LoginCase
	pluginCase    *AiPluginCase
	workers       chan struct{}
}

// NewAiAutomationCase 创建自动化业务并复用现有会话、身份和工具运行时。
func NewAiAutomationCase(baseCase *biz.BaseCase, tx data.Transaction, automationRepo *data.AiAutomationRepository, runRepo *data.AiAutomationRunRepository, userRepo *data.BaseUserRepository, sessionCase *AiSessionCase, messageCase *AiMessageCase, loginCase *LoginCase, pluginCase *AiPluginCase) *AiAutomationCase {
	c := &AiAutomationCase{BaseCase: baseCase, AiAutomationRepository: automationRepo, tx: tx, runRepo: runRepo, userRepo: userRepo, aiSessionCase: sessionCase, aiMessageCase: messageCase, loginCase: loginCase, pluginCase: pluginCase, workers: make(chan struct{}, 4)}
	messageCase.automationCase = c
	return c
}

// ListAiAutomation 只查询当前租户当前用户的计划，并汇总待确认数量。
func (c *AiAutomationCase) ListAiAutomation(ctx context.Context, req *basev1.ListAiAutomationRequest) (*basev1.ListAiAutomationResponse, error) {
	auth, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiAutomation
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(auth.TenantId)), repository.Where(query.OwnerUserID.Eq(auth.UserId)), repository.Order(query.UpdatedAt.Desc(), query.ID.Desc())}
	if req.Status != "" && req.Status != "attention" {
		opts = append(opts, repository.Where(query.Status.Eq(req.Status)))
	} else {
		opts = append(opts, repository.Where(query.Status.Neq("archived")))
	}
	var rows []*models.AiAutomation
	rows, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := &basev1.ListAiAutomationResponse{Automations: make([]*basev1.AiAutomation, 0, len(rows))}
	for _, row := range rows {
		if req.Keyword != "" && !strings.Contains(strings.ToLower(row.Name+" "+row.Prompt), strings.ToLower(req.Keyword)) {
			continue
		}
		var item *basev1.AiAutomation
		item, err = c.automationDTO(ctx, row)
		if err != nil {
			return nil, err
		}
		if req.Status == "attention" && row.Problem == "" && (item.LastRun == nil || item.LastRun.Status != "waiting_approval") {
			continue
		}
		result.Automations = append(result.Automations, item)
	}
	runQuery := c.runRepo.Query(ctx).AiAutomationRun
	var count int64
	count, err = runQuery.WithContext(ctx).Where(runQuery.TenantID.Eq(auth.TenantId), runQuery.OwnerUserID.Eq(auth.UserId), runQuery.Status.Eq("waiting_approval")).Count()
	result.PendingCount = int32(count)
	return result, err
}

// GetAiAutomation 按本人归属查询计划详情。
func (c *AiAutomationCase) GetAiAutomation(ctx context.Context, req *basev1.GetAiAutomationRequest) (*basev1.AiAutomation, error) {
	row, err := c.ownedAutomation(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return c.automationDTO(ctx, row)
}

// PreviewAiAutomationSchedule 使用与后台派发相同的算法预览未来三次时间。
func (c *AiAutomationCase) PreviewAiAutomationSchedule(ctx context.Context, req *basev1.PreviewAiAutomationScheduleRequest) (*basev1.PreviewAiAutomationScheduleResponse, error) {
	times, err := automation.Preview(req.Schedule, time.Now(), 3)
	if err != nil {
		return nil, errorsx.InvalidArgument("计划时间不合法").WithCause(err)
	}
	result := &basev1.PreviewAiAutomationScheduleResponse{Timezone: req.Schedule.Timezone}
	for _, at := range times {
		result.Times = append(result.Times, timestamppb.New(at))
	}
	return result, nil
}

// CreateAiAutomation 按当前用户权限和管理工具配置创建启用计划。
func (c *AiAutomationCase) CreateAiAutomation(ctx context.Context, req *basev1.CreateAiAutomationRequest) (*basev1.AiAutomation, error) {
	auth, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var row *models.AiAutomation
	row, err = automationModel(req.Automation)
	if err != nil {
		return nil, err
	}
	if err = c.validateAutomationDependencies(ctx, req.Automation, "admin"); err != nil {
		return nil, err
	}
	var times []time.Time
	times, err = automation.Preview(req.Automation.Schedule, time.Now(), 1)
	if err != nil {
		return nil, errorsx.InvalidArgument("计划时间不合法").WithCause(err)
	}
	now := time.Now()
	row.TenantID = auth.TenantId
	row.OwnerUserID = auth.UserId
	row.Terminal = "admin"
	row.Status = "active"
	row.Revision = 1
	row.NextFireAt = times[0].UnixMilli()
	row.CreatedAt = now
	row.UpdatedAt = now
	row.CreatedBy = auth.UserId
	row.UpdatedBy = auth.UserId
	if err = c.Create(ctx, row); err != nil {
		return nil, err
	}
	return c.automationDTO(ctx, row)
}

// UpdateAiAutomation 按修订更新；执行中拒绝编辑，排队和等待审批的旧配置作废。
func (c *AiAutomationCase) UpdateAiAutomation(ctx context.Context, req *basev1.UpdateAiAutomationRequest) (*basev1.AiAutomation, error) {
	row, err := c.ownedAutomation(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	var updated *models.AiAutomation
	updated, err = automationModel(req.Automation)
	if err != nil {
		return nil, err
	}
	if err = c.validateAutomationDependencies(ctx, req.Automation, row.Terminal); err != nil {
		return nil, err
	}
	var times []time.Time
	times, err = automation.Preview(req.Automation.Schedule, time.Now(), 1)
	if err != nil {
		return nil, errorsx.InvalidArgument("计划时间不合法").WithCause(err)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var locked *models.AiAutomation
		locked, err = c.lockAutomation(txCtx, row.ID)
		if err != nil {
			return err
		}
		if locked.Revision != req.Revision || locked.Status == "archived" {
			return errorsx.Conflict("计划已变化，请刷新后重试")
		}
		if err = c.invalidateAutomationRun(txCtx, locked, true); err != nil {
			return err
		}
		query := c.Query(txCtx).AiAutomation
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.Name.Value(updated.Name), query.Prompt.Value(updated.Prompt), query.Schedule.Value(updated.Schedule), query.ProviderID.Value(updated.ProviderID), query.ModelName.Value(updated.ModelName), query.KnowledgeBaseIds.Value(updated.KnowledgeBaseIds), query.Plugins.Value(updated.Plugins), query.Revision.Value(locked.Revision+1), query.NextFireAt.Value(times[0].UnixMilli()), query.Problem.Value(""), query.UpdatedBy.Value(row.OwnerUserID), query.UpdatedAt.Value(time.Now()))
		return err
	})
	if err != nil {
		return nil, err
	}
	return c.GetAiAutomation(ctx, &basev1.GetAiAutomationRequest{Id: req.Id})
}

// SetAiAutomationStatus 暂停、恢复或归档，恢复须无活动执行且仅安排未来时隙。
func (c *AiAutomationCase) SetAiAutomationStatus(ctx context.Context, req *basev1.SetAiAutomationStatusRequest) (*basev1.AiAutomation, error) {
	if req.Status != "active" && req.Status != "paused" && req.Status != "archived" {
		return nil, errorsx.InvalidArgument("计划状态不合法")
	}
	row, err := c.ownedAutomation(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	next := int64(0)
	if req.Status == "active" {
		var form *basev1.AiAutomationForm
		form, err = automationConfiguration(row)
		if err != nil {
			return nil, err
		}
		if err = c.validateAutomationDependencies(ctx, form, row.Terminal); err != nil {
			return nil, err
		}
		var times []time.Time
		times, err = automation.Preview(form.Schedule, time.Now(), 1)
		if err != nil {
			return nil, errorsx.InvalidArgument("计划时间不合法").WithCause(err)
		}
		next = times[0].UnixMilli()
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var locked *models.AiAutomation
		locked, err = c.lockAutomation(txCtx, row.ID)
		if err != nil {
			return err
		}
		if locked.Revision != req.Revision || locked.Status == "archived" {
			return errorsx.Conflict("计划已变化，请刷新后重试")
		}
		if req.Status == "active" && locked.ActiveRunID != 0 {
			return errorsx.Conflict("此计划仍有执行或待确认事项，请结束后再恢复")
		}
		if req.Status != "active" {
			if err = c.invalidateAutomationRun(txCtx, locked, req.Status == "archived"); err != nil {
				return err
			}
		}
		query := c.Query(txCtx).AiAutomation
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(row.ID)).UpdateSimple(query.Status.Value(req.Status), query.Revision.Value(locked.Revision+1), query.NextFireAt.Value(next), query.Problem.Value(""), query.UpdatedBy.Value(row.OwnerUserID), query.UpdatedAt.Value(time.Now()))
		return err
	})
	if err != nil {
		return nil, err
	}
	return c.GetAiAutomation(ctx, &basev1.GetAiAutomationRequest{Id: req.Id})
}

// RunAiAutomation 按本人计划的工具能力配置幂等入队，不改变计划启用状态。
func (c *AiAutomationCase) RunAiAutomation(ctx context.Context, req *basev1.RunAiAutomationRequest) (*basev1.AiAutomationRun, error) {
	row, err := c.ownedAutomation(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if len(req.RequestKey) == 0 || len(req.RequestKey) > 64 {
		return nil, errorsx.InvalidArgument("请求幂等键不合法")
	}
	var form *basev1.AiAutomationForm
	form, err = automationConfiguration(row)
	if err != nil {
		return nil, err
	}
	if err = c.validateAutomationDependencies(ctx, form, row.Terminal); err != nil {
		return nil, err
	}
	var run *models.AiAutomationRun
	run, err = c.enqueueAutomation(ctx, row.ID, req.Revision, "manual:"+req.RequestKey, row.Terminal, 0, time.Now())
	if err != nil {
		return nil, err
	}
	return c.automationRunDTO(ctx, run)
}

// ListAiAutomationRun 查询本人执行记录，保留已删除结果会话的审计。
func (c *AiAutomationCase) ListAiAutomationRun(ctx context.Context, req *basev1.ListAiAutomationRunRequest) (*basev1.ListAiAutomationRunResponse, error) {
	auth, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(auth.TenantId)), repository.Where(query.OwnerUserID.Eq(auth.UserId)), repository.Order(query.ID.Desc()), repository.Limit(int(min(max(req.Limit, 1), 100)))}
	if req.Limit == 0 {
		opts[len(opts)-1] = repository.Limit(50)
	}
	if req.AutomationId != "" {
		var id int64
		id, err = strconv.ParseInt(req.AutomationId, 10, 64)
		if err != nil || id <= 0 {
			return nil, errorsx.InvalidArgument("任务编号不合法")
		}
		opts = append(opts, repository.Where(query.AutomationID.Eq(id)))
	}
	var rows []*models.AiAutomationRun
	rows, err = c.runRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := &basev1.ListAiAutomationRunResponse{Runs: make([]*basev1.AiAutomationRun, 0, len(rows))}
	for _, row := range rows {
		var run *basev1.AiAutomationRun
		run, err = c.automationRunDTO(ctx, row)
		if err != nil {
			return nil, err
		}
		result.Runs = append(result.Runs, run)
	}
	return result, nil
}

// GetAiAutomationRun 查询本人单次执行详情。
func (c *AiAutomationCase) GetAiAutomationRun(ctx context.Context, req *basev1.GetAiAutomationRunRequest) (*basev1.AiAutomationRun, error) {
	row, err := c.ownedAutomationRun(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return c.automationRunDTO(ctx, row)
}

// StopAiAutomationRun 只停止本人本次执行，已发出的远端操作等待明确结果。
func (c *AiAutomationCase) StopAiAutomationRun(ctx context.Context, req *basev1.StopAiAutomationRunRequest) (*basev1.AiAutomationRun, error) {
	run, err := c.ownedAutomationRun(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		plan, err := c.lockAutomation(txCtx, run.AutomationID)
		if err != nil {
			return err
		}
		var current *models.AiAutomationRun
		current, err = c.runRepo.FindByID(txCtx, run.ID)
		if err != nil {
			return err
		}
		if current.Status == "queued" || current.Status == "waiting_approval" {
			return c.cancelIdleAutomationRun(txCtx, plan, current)
		}
		if current.Status != "running" {
			return nil
		}
		query := c.runRepo.Query(txCtx).AiAutomationRun
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(run.ID), query.Status.Eq("running")).UpdateSimple(query.CancelRequested.Value(1), query.UpdatedAt.Value(time.Now()))
		return err
	})
	if err != nil {
		return nil, err
	}
	return c.GetAiAutomationRun(ctx, &basev1.GetAiAutomationRunRequest{Id: req.Id})
}

// ContinueAiAutomationRun 从终态结果创建普通讨论，保留原结果引用并移除旧许可和动作。
func (c *AiAutomationCase) ContinueAiAutomationRun(ctx context.Context, req *basev1.ContinueAiAutomationRunRequest) (*basev1.CreateAiSessionResponse, error) {
	run, err := c.ownedAutomationRun(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if run.Status == "queued" || run.Status == "running" || run.Status == "waiting_approval" {
		return nil, errorsx.Conflict("请等待本次执行结束后继续讨论")
	}
	if run.SessionID <= 0 {
		return nil, errorsx.ResourceNotFound("本次执行没有结果会话")
	}
	source, err := c.aiSessionCase.FindCurrentUserSessionByRawID(ctx, strconv.FormatInt(run.SessionID, 10))
	if err != nil {
		return nil, err
	}
	terminal := ai.NormalizeTerminalEnum(source.Terminal)
	var session *basev1.AiSession
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var err error
		session, err = c.aiSessionCase.CreateAiSession(txCtx, &basev1.CreateAiSessionRequest{Terminal: terminal, Title: "讨论 · " + source.Title, Plugins: make([]*basev1.AiPluginSelection, 0)})
		if err != nil {
			return err
		}
		var id int64
		id, err = strconv.ParseInt(session.Id, 10, 64)
		if err != nil {
			return err
		}
		query := c.aiSessionCase.Query(txCtx).AiSession
		_, err = query.WithContext(txCtx).Where(query.ID.Eq(id)).UpdateSimple(query.ResultReferenceID.Value(source.ID))
		if err != nil {
			return err
		}
		messages := c.aiMessageCase.aiMessageRepo.Query(txCtx).AiMessage
		var previous []*models.AiMessage
		previous, err = c.aiMessageCase.aiMessageRepo.List(txCtx, repository.Where(messages.SessionID.Eq(source.ID)), repository.Where(messages.TenantID.Eq(source.TenantID)), repository.Where(messages.UserID.Eq(source.UserID)), repository.Where(messages.Status.Eq(int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS))), repository.Order(messages.ID.Asc()))
		if err != nil {
			return err
		}
		now := time.Now()
		for _, message := range previous {
			output := ai.ParseOutputContent(message.OutputContent)
			if output.Content == "" {
				continue
			}
			model := &models.AiMessage{TenantID: source.TenantID, SessionID: id, UserID: source.UserID, InputContent: ai.MarshalInputContentPayload(ai.InputContentPayload{Kind: ai.KindText, Content: "继续讨论自动化结果"}), OutputContent: ai.MarshalReplyContent(&ai.Response{Content: output.Content}), Attachments: "[]", Tools: "[]", Token: "{}", Status: int32(basev1.AiMessageStatus_AI_MESSAGE_STATUS_SUCCESS), CreatedAt: now, UpdatedAt: now}
			if err = c.aiMessageCase.aiMessageRepo.Create(txCtx, model); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &basev1.CreateAiSessionResponse{Session: session}, nil
}

// ownedAutomation 统一计划编号与本人归属边界。
func (c *AiAutomationCase) ownedAutomation(ctx context.Context, rawID string) (*models.AiAutomation, error) {
	auth, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	id, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return nil, errorsx.InvalidArgument("任务编号不合法")
	}
	query := c.Query(ctx).AiAutomation
	var row *models.AiAutomation
	row, err = c.Find(ctx, repository.Where(query.ID.Eq(id)), repository.Where(query.TenantID.Eq(auth.TenantId)), repository.Where(query.OwnerUserID.Eq(auth.UserId)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("自动化计划不存在")
	}
	return row, err
}

// ownedAutomationRun 统一执行编号与本人归属边界。
func (c *AiAutomationCase) ownedAutomationRun(ctx context.Context, rawID string) (*models.AiAutomationRun, error) {
	auth, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	id, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return nil, errorsx.InvalidArgument("执行编号不合法")
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	var row *models.AiAutomationRun
	row, err = c.runRepo.Find(ctx, repository.Where(query.ID.Eq(id)), repository.Where(query.TenantID.Eq(auth.TenantId)), repository.Where(query.OwnerUserID.Eq(auth.UserId)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("执行记录不存在")
	}
	return row, err
}

// lockAutomation 锁定计划行，统一调度、手动入队和配置修改的竞争顺序。
func (c *AiAutomationCase) lockAutomation(ctx context.Context, id int64) (*models.AiAutomation, error) {
	query := c.Query(ctx).AiAutomation
	return query.WithContext(ctx).Where(query.ID.Eq(id)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
}

// automationDTO 返回配置和最近一次记录，不携带外部凭据。
func (c *AiAutomationCase) automationDTO(ctx context.Context, row *models.AiAutomation) (*basev1.AiAutomation, error) {
	form, err := automationConfiguration(row)
	if err != nil {
		return nil, err
	}
	item := &basev1.AiAutomation{Id: strconv.FormatInt(row.ID, 10), Configuration: form, Status: row.Status, Revision: row.Revision, Problem: row.Problem, CreatedAt: timestamppb.New(row.CreatedAt), UpdatedAt: timestamppb.New(row.UpdatedAt)}
	if row.NextFireAt > 0 {
		item.NextFireAt = timestamppb.New(time.UnixMilli(row.NextFireAt))
	}
	if row.ActiveRunID > 0 {
		item.ActiveRunId = strconv.FormatInt(row.ActiveRunID, 10)
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	var run *models.AiAutomationRun
	run, err = c.runRepo.Find(ctx, repository.Where(query.AutomationID.Eq(row.ID)), repository.Order(query.ID.Desc()), repository.Limit(1))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return item, nil
	}
	if err != nil {
		return nil, err
	}
	item.LastRun, err = c.automationRunDTO(ctx, run)
	return item, err
}

// automationRunDTO 返回冻结配置及结果会话可用性。
func (c *AiAutomationCase) automationRunDTO(ctx context.Context, row *models.AiAutomationRun) (*basev1.AiAutomationRun, error) {
	form := &basev1.AiAutomationForm{}
	err := json.Unmarshal([]byte(row.Configuration), form)
	if err != nil {
		return nil, errorsx.Internal("执行配置损坏").WithCause(err)
	}
	item := &basev1.AiAutomationRun{Id: strconv.FormatInt(row.ID, 10), AutomationId: strconv.FormatInt(row.AutomationID, 10), Revision: row.Revision, Configuration: form, Terminal: row.Terminal, Trigger: row.Trigger, Status: row.Status, Summary: row.Summary, ErrorCode: row.ErrorCode, ResultUnknown: row.ResultUnknown != 0, CancelRequested: row.CancelRequested != 0, CreatedAt: timestamppb.New(row.CreatedAt)}
	if row.ScheduledFor > 0 {
		item.ScheduledFor = timestamppb.New(time.UnixMilli(row.ScheduledFor))
	}
	if row.StartedAt > 0 {
		item.StartedAt = timestamppb.New(time.UnixMilli(row.StartedAt))
	}
	if row.FinishedAt > 0 {
		item.FinishedAt = timestamppb.New(time.UnixMilli(row.FinishedAt))
	}
	if row.SessionID > 0 {
		item.SessionId = strconv.FormatInt(row.SessionID, 10)
		sessionQuery := c.aiSessionCase.Query(ctx).AiSession
		var count int64
		count, err = c.aiSessionCase.Count(ctx, repository.Where(sessionQuery.ID.Eq(row.SessionID)), repository.Where(sessionQuery.TenantID.Eq(row.TenantID)), repository.Where(sessionQuery.UserID.Eq(row.OwnerUserID)))
		if err != nil {
			return nil, err
		}
		item.SessionDeleted = count == 0
	}
	return item, nil
}

// automationConfiguration 还原持久化依赖，明确空插件集合不退回内置工具。
func automationConfiguration(row *models.AiAutomation) (*basev1.AiAutomationForm, error) {
	form := &basev1.AiAutomationForm{Name: row.Name, Prompt: row.Prompt, Schedule: &basev1.AiAutomationSchedule{}, ProviderId: row.ProviderID, ModelName: row.ModelName, Plugins: make([]*basev1.AiPluginSelection, 0)}
	err := json.Unmarshal([]byte(row.Schedule), form.Schedule)
	if err != nil {
		return nil, errorsx.Internal("计划时间配置损坏").WithCause(err)
	}
	if err = json.Unmarshal([]byte(row.KnowledgeBaseIds), &form.KnowledgeBaseIds); err != nil {
		return nil, errorsx.Internal("计划知识库配置损坏").WithCause(err)
	}
	if err = json.Unmarshal([]byte(row.Plugins), &form.Plugins); err != nil {
		return nil, errorsx.Internal("计划插件配置损坏").WithCause(err)
	}
	if form.Plugins == nil {
		form.Plugins = make([]*basev1.AiPluginSelection, 0)
	}
	return form, nil
}

// automationModel 校验并编码计划配置，凭据仅保留连接引用。
func automationModel(form *basev1.AiAutomationForm) (*models.AiAutomation, error) {
	if form.Name == "" || len([]rune(form.Name)) > 100 || form.Prompt == "" || len([]rune(form.Prompt)) > 16000 || form.ProviderId <= 0 || form.ModelName == "" {
		return nil, errorsx.InvalidArgument("计划名称、指令和固定模型不能为空")
	}
	schedule, err := json.Marshal(form.Schedule)
	if err != nil {
		return nil, err
	}
	var knowledge []byte
	knowledge, err = json.Marshal(form.KnowledgeBaseIds)
	if err != nil {
		return nil, err
	}
	var plugins []byte
	plugins, err = json.Marshal(append([]*basev1.AiPluginSelection{}, form.Plugins...))
	if err != nil {
		return nil, err
	}
	return &models.AiAutomation{Name: form.Name, Prompt: form.Prompt, Schedule: string(schedule), ProviderID: form.ProviderId, ModelName: form.ModelName, KnowledgeBaseIds: string(knowledge), Plugins: string(plugins)}, nil
}

// claimAutomationApproval 与消息审批共用事务，消费前锁定任务并重新核实 owner 和配置。
func (c *AiAutomationCase) claimAutomationApproval(ctx context.Context, session *models.AiSession) (*models.AiAutomationRun, error) {
	plan, err := c.lockAutomation(ctx, session.AutomationID)
	if err != nil {
		return nil, err
	}
	var run *models.AiAutomationRun
	run, err = c.runRepo.FindByID(ctx, session.AutomationRunID)
	if err != nil {
		return nil, err
	}
	if run.TenantID != session.TenantID || run.OwnerUserID != session.UserID || run.SessionID != session.ID || run.AutomationID != plan.ID || plan.ActiveRunID != run.ID || plan.Revision != run.Revision || run.Status != "waiting_approval" || run.CancelRequested != 0 || plan.Status == "archived" {
		return nil, errorsx.Conflict("本次自动化确认已失效")
	}
	var ownerCtx context.Context
	ownerCtx, err = c.automationOwnerContext(ctx, run)
	if err != nil {
		return nil, err
	}
	form := &basev1.AiAutomationForm{}
	if err = json.Unmarshal([]byte(run.Configuration), form); err != nil {
		return nil, err
	}
	if err = c.validateAutomationDependencies(ownerCtx, form, run.Terminal); err != nil {
		return nil, err
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	now := time.Now()
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.Status.Eq("waiting_approval"), query.LeaseVersion.Eq(run.LeaseVersion), query.CancelRequested.Eq(0)).UpdateSimple(query.Status.Value("running"), query.LeaseVersion.Value(run.LeaseVersion+1), query.LeaseUntil.Value(now.Add(automationLeaseLifetime).UnixMilli()), query.UpdatedAt.Value(now))
	if err != nil {
		return nil, err
	}
	if result.RowsAffected != 1 {
		return nil, errorsx.Conflict("本次自动化已被处理")
	}
	run.Status = "running"
	run.LeaseVersion++
	run.LeaseUntil = now.Add(automationLeaseLifetime).UnixMilli()
	return run, nil
}

// bindAutomationApproval 为已领取的确认续租，并在真实工具边界重新核实取消标记。
func (c *AiAutomationCase) bindAutomationApproval(ctx context.Context, run *models.AiAutomationRun) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	ctx = automation.ContextWithExecution(ctx, automation.Execution{RunID: run.ID, LeaseVersion: run.LeaseVersion})
	ctx = agent.ContextWithToolExecutionObserver(ctx, c.automationToolObserver(run))
	go c.heartbeatAutomation(ctx, run, cancel, ctx.Done())
	return ctx, cancel
}

// lockAutomationExecution 在同一结果事务内锁定任务和运行，拒绝旧 worker 回写。
func (c *AiAutomationCase) lockAutomationExecution(ctx context.Context, execution automation.Execution, pending bool) error {
	run, err := c.runRepo.FindByID(ctx, execution.RunID)
	if err != nil {
		return err
	}
	var plan *models.AiAutomation
	plan, err = c.lockAutomation(ctx, run.AutomationID)
	if err != nil {
		return err
	}
	query := c.runRepo.Query(ctx).AiAutomationRun
	run, err = query.WithContext(ctx).Where(query.ID.Eq(execution.RunID)).Clauses(clause.Locking{Strength: "UPDATE"}).First()
	if err != nil {
		return err
	}
	if run.LeaseVersion != execution.LeaseVersion || run.Status != "running" || run.LeaseUntil <= time.Now().UnixMilli() || plan.ActiveRunID != run.ID {
		return errorsx.Conflict("自动化租约已失效，不能回写结果")
	}
	if pending && (run.CancelRequested != 0 || plan.Revision != run.Revision || plan.Status == "archived") {
		return errorsx.Conflict("自动化已停止，不能创建新的确认")
	}
	return nil
}
