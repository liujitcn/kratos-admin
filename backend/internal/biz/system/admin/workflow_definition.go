package biz

import (
	"context"
	"strconv"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/scripting"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm/clause"
)

// WorkflowDefinitionCase 工作流流程定义业务实例。
type WorkflowDefinitionCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.WorkflowDefinitionRepository
	baseScriptCase *BaseScriptCase
	scriptRepo     *data.BaseScriptRepository
	instanceRepo   *data.WorkflowInstanceRepository
	formMapper     *mapper.CopierMapper[adminv1.WorkflowDefinitionForm, models.WorkflowDefinition]
	mapper         *mapper.CopierMapper[adminv1.WorkflowDefinition, models.WorkflowDefinition]
}

// NewWorkflowDefinitionCase 创建工作流流程定义业务实例。
func NewWorkflowDefinitionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	definitionRepo *data.WorkflowDefinitionRepository,
	instanceRepo *data.WorkflowInstanceRepository,
	scriptRepo *data.BaseScriptRepository,
	baseScriptCase *BaseScriptCase,
) *WorkflowDefinitionCase {
	return &WorkflowDefinitionCase{
		BaseCase:                     baseCase,
		tx:                           tx,
		WorkflowDefinitionRepository: definitionRepo,
		instanceRepo:                 instanceRepo,
		scriptRepo:                   scriptRepo,
		baseScriptCase:               baseScriptCase,
		formMapper:                   mapper.NewCopierMapper[adminv1.WorkflowDefinitionForm, models.WorkflowDefinition](),
		mapper:                       mapper.NewCopierMapper[adminv1.WorkflowDefinition, models.WorkflowDefinition](),
	}
}

// PageWorkflowScript 查询系统共享库中启用且支持受限流程执行的脚本，不接纳客户端状态或语言过滤。
func (c *WorkflowDefinitionCase) PageWorkflowScript(ctx context.Context, req *adminv1.PageBaseScriptRequest) (*adminv1.PageBaseScriptResponse, error) {
	ctx = workflowParticipantContext(ctx)
	query := c.scriptRepo.Query(ctx).BaseScript
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)),
		repository.Where(query.Language.Eq(scripting.JavaScriptLanguage)),
		repository.Order(query.ID.Asc()),
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	list, total, err := c.scriptRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseScript, 0, len(list))
	for _, item := range list {
		result = append(result, toBaseScript(item))
	}
	return &adminv1.PageBaseScriptResponse{BaseScripts: result, Total: int32(total)}, nil
}

// TestWorkflowScript 仅试运行已保存且启用的受限流程脚本，复用统一运行时与脚本审计。
func (c *WorkflowDefinitionCase) TestWorkflowScript(ctx context.Context, req *adminv1.TestWorkflowScriptRequest) (*adminv1.TestRunBaseScriptResponse, error) {
	ctx = workflowParticipantContext(ctx)
	_, err := scripting.ParseContext(req.GetContextJson())
	if err != nil {
		return nil, errorsx.InvalidArgument(err.Error())
	}
	var response *adminv1.TestRunBaseScriptResponse
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.scriptRepo.Query(ctx).BaseScript
		var script *models.BaseScript
		script, err = c.scriptRepo.Find(ctx, repository.Where(query.ID.Eq(req.GetScriptId())), repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)), repository.Clauses(clause.Locking{Strength: "SHARE"}))
		if err != nil {
			return err
		}
		if !workflow.ScriptLanguageSupported(script.Language) {
			return errorsx.InvalidArgument("流程仅支持 JavaScript 脚本")
		}
		// 共享锁覆盖保存脚本的再次读取与执行，防止语言或源码并发变化绕过检查。
		response, err = c.baseScriptCase.testRunBaseScript(ctx, &adminv1.TestRunBaseScriptRequest{Id: script.ID, ContextJson: req.GetContextJson()}, true)
		return err
	})
	return response, err
}

// OptionWorkflowDefinition 查询启用流程定义下拉选择。
func (c *WorkflowDefinitionCase) OptionWorkflowDefinition(ctx context.Context, req *adminv1.OptionWorkflowDefinitionRequest) (*commonv1.SelectOptionResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authInfo.TenantId
	if req.GetTenantId() > 0 {
		if authInfo.TenantCode != gorm.DefaultTenantCode && req.GetTenantId() != authInfo.TenantId {
			return nil, errorsx.PermissionDenied("不能查询其他租户的流程定义")
		}
		tenantID = req.GetTenantId()
	}

	query := c.Query(ctx).WorkflowDefinition
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.Status.Eq(workflowStatusDefinitionEnabled)))
	opts = append(opts, repository.Order(query.Code.Asc()))
	opts = append(opts, repository.Order(query.Version.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label: item.Name + " v" + strconv.FormatInt(int64(item.Version), 10),
			Value: item.ID,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// GetWorkflowApplyForm 查询启用流程定义的申请表单。
func (c *WorkflowDefinitionCase) GetWorkflowApplyForm(ctx context.Context, req *adminv1.GetWorkflowApplyFormRequest) (*adminv1.GetWorkflowApplyFormResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := c.findEnabledByCodeVersion(ctx, authInfo.TenantId, req.GetCode(), req.GetVersion())
	if err != nil {
		return nil, err
	}
	return &adminv1.GetWorkflowApplyFormResponse{
		DefinitionId:   definition.ID,
		DefinitionName: definition.Name,
		FormSchema:     definition.FormSchema,
	}, nil
}

// PageWorkflowDefinition 分页查询流程定义，隐藏仅用于服务端执行的脚本源码快照。
func (c *WorkflowDefinitionCase) PageWorkflowDefinition(ctx context.Context, req *adminv1.PageWorkflowDefinitionRequest) (*adminv1.PageWorkflowDefinitionResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).WorkflowDefinition
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		if authInfo.TenantCode != gorm.DefaultTenantCode && req.GetTenantId() != authInfo.TenantId {
			return nil, errorsx.PermissionDenied("不能查询其他租户的流程定义")
		}
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	} else if authInfo.TenantCode != gorm.DefaultTenantCode {
		opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.WorkflowDefinition, 0, len(list))
	for _, item := range list {
		dto := c.mapper.ToDTO(item)
		dto.NodeConfig, err = draftNodeConfig(item.NodeConfig)
		if err != nil {
			return nil, err
		}
		resList = append(resList, dto)
	}
	return &adminv1.PageWorkflowDefinitionResponse{Definitions: resList, Total: int32(total)}, nil
}

// GetWorkflowDefinition 获取流程定义，隐藏发布时保存的服务端脚本源码快照。
func (c *WorkflowDefinitionCase) GetWorkflowDefinition(ctx context.Context, id int64) (*adminv1.WorkflowDefinitionForm, error) {
	definition, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := c.formMapper.ToDTO(definition)
	dto.NodeConfig, err = draftNodeConfig(definition.NodeConfig)
	if err != nil {
		return nil, err
	}
	return dto, nil
}

// CreateWorkflowDefinition 创建流程定义，强制落草稿状态并校验节点图配置。
func (c *WorkflowDefinitionCase) CreateWorkflowDefinition(ctx context.Context, req *adminv1.WorkflowDefinitionForm) error {
	err := validateNodeConfig(req.GetNodeConfig())
	if err != nil {
		return err
	}
	definition := c.formMapper.ToEntity(req)
	definition.NodeConfig, err = draftNodeConfig(req.GetNodeConfig())
	if err != nil {
		return err
	}
	var tenantID int64
	tenantID, err = c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return err
	}
	definition.TenantID = tenantID
	definition.Status = workflowStatusDefinitionDraft
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, definition)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的流程编码与版本重复", "workflow_definition", "code", "unique_workflow_definition").WithCause(err)
			}
			return err
		}
		return nil
	})
}

// UpdateWorkflowDefinition 更新流程定义，仅草稿状态可编辑且不可修改编码与版本。
func (c *WorkflowDefinitionCase) UpdateWorkflowDefinition(ctx context.Context, req *adminv1.WorkflowDefinitionForm) error {
	err := validateNodeConfig(req.GetNodeConfig())
	if err != nil {
		return err
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).WorkflowDefinition
		var oldDefinition *models.WorkflowDefinition
		oldDefinition, err = c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if oldDefinition.Status != workflowStatusDefinitionDraft {
			return errorsx.StateConflict("更新流程定义失败，仅草稿状态可编辑", "workflow_definition", definitionStatusName(oldDefinition.Status), "DRAFT")
		}
		if oldDefinition.Code != req.GetCode() || oldDefinition.Version != req.GetVersion() {
			return errorsx.InvalidArgument("流程编码与版本不可修改，请创建新版本")
		}
		definition := c.formMapper.ToEntity(req)
		definition.NodeConfig, err = draftNodeConfig(req.GetNodeConfig())
		if err != nil {
			return err
		}
		definition.TenantID, definition.ID, definition.Status = oldDefinition.TenantID, oldDefinition.ID, oldDefinition.Status
		err = c.UpdateByID(ctx, definition)
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一租户的流程编码与版本重复", "workflow_definition", "code", "unique_workflow_definition").WithCause(err)
		}
		return err
	})
}

// DeleteWorkflowDefinition 删除流程定义，仅草稿且无实例引用时可删除。
func (c *WorkflowDefinitionCase) DeleteWorkflowDefinition(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	definitions, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	definitionMap := make(map[int64]*models.WorkflowDefinition, len(definitions))
	for _, item := range definitions {
		definitionMap[item.ID] = item
	}
	for _, definitionID := range ids {
		item, exists := definitionMap[definitionID]
		if !exists {
			return errorsx.ResourceNotFound("删除流程定义失败，流程定义不存在")
		}
		if item.Status != workflowStatusDefinitionDraft {
			return errorsx.StateConflict("删除流程定义失败，仅草稿状态可删除", "workflow_definition", definitionStatusName(item.Status), "DRAFT")
		}
	}

	query := c.instanceRepo.Query(ctx).WorkflowInstance
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.DefinitionID.In(ids...)))
	count, err := c.instanceRepo.Count(ctx, opts...)
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.HasChildrenConflict("删除流程定义失败，仍有流程实例引用该定义", "workflow_definition", "workflow_instance")
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.DeleteByIDs(ctx, ids)
	})
}

// SetWorkflowDefinitionStatus 设置流程定义状态，启用前校验节点图配置。
func (c *WorkflowDefinitionCase) SetWorkflowDefinitionStatus(ctx context.Context, req *adminv1.SetWorkflowDefinitionStatusRequest) error {
	status := int32(req.GetStatus())
	if status != workflowStatusDefinitionDraft && status != workflowStatusDefinitionEnabled && status != workflowStatusDefinitionDisabled {
		return errorsx.InvalidArgument("流程定义状态无效")
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).WorkflowDefinition
		definition, err := c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())), repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		if err != nil {
			return err
		}
		if definition.Status == status {
			return nil
		}
		if status == workflowStatusDefinitionEnabled {
			var graph *workflow.Graph
			graph, err = workflow.Parse(definition.NodeConfig)
			if err != nil {
				return errorsx.InvalidArgument("节点图配置无效").WithCause(err)
			}
			err = workflow.Validate(graph)
			if err != nil {
				return errorsx.InvalidArgument("节点图配置无效").WithCause(err)
			}
			graph.ScriptSnapshots = make(map[string]workflow.ScriptSnapshot)
			ids := graph.ScriptIDs()
			if len(ids) > 0 {
				// 脚本库为系统级共享能力；发布仅引用启用版本，流程本身继续使用仓储租户作用域。
				scriptCtx := workflowParticipantContext(ctx)
				scriptQuery := c.scriptRepo.Query(scriptCtx).BaseScript
				var scripts []*models.BaseScript
				scripts, err = c.scriptRepo.List(scriptCtx, repository.Where(scriptQuery.ID.In(ids...)), repository.Where(scriptQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)), repository.Clauses(clause.Locking{Strength: "SHARE"}))
				if err != nil {
					return err
				}
				if len(scripts) != len(ids) {
					return errorsx.InvalidArgument("引用脚本不存在或未启用")
				}
				for _, script := range scripts {
					if !workflow.ScriptLanguageSupported(script.Language) {
						return errorsx.InvalidArgument("流程仅支持 JavaScript 脚本")
					}
					graph.ScriptSnapshots[strconv.FormatInt(script.ID, 10)] = workflow.ScriptSnapshot{ID: script.ID, Name: script.Name, Language: script.Language, Version: script.Version, Source: script.Source}
				}
			}
			definition.NodeConfig, err = graph.Marshal()
			if err != nil {
				return err
			}
			if len(definition.NodeConfig) > 1<<20 {
				return errorsx.InvalidArgument("发布节点图配置不能超过 1MiB")
			}
		}
		definition.Status = status
		return c.UpdateByID(ctx, definition)
	})
}

// findEnabledByCodeVersion 按编码与版本查询租户内启用流程定义。
func (c *WorkflowDefinitionCase) findEnabledByCodeVersion(ctx context.Context, tenantID int64, code string, version int32) (*models.WorkflowDefinition, error) {
	query := c.Query(ctx).WorkflowDefinition
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(query.Code.Eq(code)))
	opts = append(opts, repository.Where(query.Version.Eq(version)))
	definition, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if definition.Status != workflowStatusDefinitionEnabled {
		return nil, errorsx.ResourceNotFound("流程定义未启用")
	}
	return definition, nil
}

// resolveTenantID 解析流程定义创建时的所属租户并校验租户范围。
func (c *WorkflowDefinitionCase) resolveTenantID(ctx context.Context, tenantID int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if tenantID == 0 {
		return authInfo.TenantId, nil
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode && tenantID != authInfo.TenantId {
		return 0, errorsx.PermissionDenied("不能操作其他租户的流程定义")
	}
	return tenantID, nil
}

// validateNodeConfig 解析并校验节点图配置。
func validateNodeConfig(nodeConfig string) error {
	graph, err := workflow.Parse(nodeConfig)
	if err != nil {
		return errorsx.InvalidArgument("节点图配置无效: " + err.Error()).WithCause(err)
	}
	if err := workflow.Validate(graph); err != nil {
		return errorsx.InvalidArgument("节点图配置无效: " + err.Error())
	}
	return nil
}

// definitionStatusName 返回流程定义状态名称，用于冲突错误描述。
func definitionStatusName(status int32) string {
	switch status {
	case workflowStatusDefinitionDraft:
		return "DRAFT"
	case workflowStatusDefinitionEnabled:
		return "ENABLED"
	case workflowStatusDefinitionDisabled:
		return "DISABLED"
	default:
		return "UNKNOWN"
	}
}

// 工作流状态常量，DB 存 int32。
const (
	workflowStatusDefinitionDraft    = int32(adminv1.WorkflowDefinitionStatus_WORKFLOW_DEFINITION_STATUS_DRAFT)
	workflowStatusDefinitionEnabled  = int32(adminv1.WorkflowDefinitionStatus_WORKFLOW_DEFINITION_STATUS_ENABLED)
	workflowStatusDefinitionDisabled = int32(adminv1.WorkflowDefinitionStatus_WORKFLOW_DEFINITION_STATUS_DISABLED)

	workflowStatusInstancePending   = int32(adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_PENDING)
	workflowStatusInstanceApproved  = int32(adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_APPROVED)
	workflowStatusInstanceRejected  = int32(adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_REJECTED)
	workflowStatusInstanceWithdrawn = int32(adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_WITHDRAWN)

	workflowStatusTaskPending  = int32(adminv1.WorkflowTaskStatus_WORKFLOW_TASK_STATUS_PENDING)
	workflowStatusTaskApproved = int32(adminv1.WorkflowTaskStatus_WORKFLOW_TASK_STATUS_APPROVED)
	workflowStatusTaskRejected = int32(adminv1.WorkflowTaskStatus_WORKFLOW_TASK_STATUS_REJECTED)
	workflowStatusTaskCanceled = int32(adminv1.WorkflowTaskStatus_WORKFLOW_TASK_STATUS_CANCELED)

	workflowLogActionSubmit   = int32(adminv1.WorkflowLogAction_WORKFLOW_LOG_ACTION_SUBMIT)
	workflowLogActionApprove  = int32(adminv1.WorkflowLogAction_WORKFLOW_LOG_ACTION_APPROVE)
	workflowLogActionReject   = int32(adminv1.WorkflowLogAction_WORKFLOW_LOG_ACTION_REJECT)
	workflowLogActionForward  = int32(adminv1.WorkflowLogAction_WORKFLOW_LOG_ACTION_FORWARD)
	workflowLogActionWithdraw = int32(adminv1.WorkflowLogAction_WORKFLOW_LOG_ACTION_WITHDRAW)
)

// draftNodeConfig 移除客户端提供的脚本源码快照，保留引用和画布配置。
func draftNodeConfig(nodeConfig string) (string, error) {
	graph, err := workflow.Parse(nodeConfig)
	if err != nil {
		return "", errorsx.InvalidArgument("节点图配置无效").WithCause(err)
	}
	graph.ScriptSnapshots = nil
	return graph.Marshal()
}
