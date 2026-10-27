package biz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	admini18n "github.com/liujitcn/kratos-admin/backend/internal/i18n"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	"github.com/liujitcn/kratos-admin/backend/pkg/scripting"
	pkgWorkflow "github.com/liujitcn/kratos-admin/backend/pkg/workflow"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen"
	"gorm.io/gen/field"
	gormdb "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// walkerStepLimit 单次推进调用的步数上限，防止回退边环路失控。
const walkerStepLimit = 1000

// 工作流站内信消息分类编码与来源标识。
const (
	workflowMessageCategory     = "WORKFLOW"
	workflowMessageActionTarget = "MESSAGE_INBOX"
	workflowMessageSource       = "workflow"
)

// workflowNotificationRecipient 描述一条工作流通知的收件人与审批定位目标。
type workflowNotificationRecipient struct {
	userID       int64
	taskID       int64
	instanceID   int64
	businessType string
}

// workflowTransactionMarker 标记工作流推进已处于外层事务，避免内部重新开启事务。
type workflowTransactionMarker struct{}

// WorkflowInstanceCase 工作流流程实例业务实例，承载引擎编排。
type WorkflowInstanceCase struct {
	*biz.BaseCase
	tx      data.Transaction
	catalog *i18n.I18n
	*data.WorkflowInstanceRepository
	definitionRepo *data.WorkflowDefinitionRepository
	taskRepo       *data.WorkflowTaskRepository
	logRepo        *data.WorkflowLogRepository
	joinRepo       *data.WorkflowJoinRepository
	delegationRepo *data.WorkflowDelegationRepository
	baseUserRepo   *data.BaseUserRepository
	eventRegistry  *pkgWorkflow.EventRegistry
	scriptRunRepo  *data.WorkflowScriptRunRepository
	scriptLogRepo  *data.BaseScriptLogRepository
	runtime        *scripting.Runtime
	mapper         *mapper.CopierMapper[adminv1.WorkflowInstance, models.WorkflowInstance]
	logMapper      *mapper.CopierMapper[adminv1.WorkflowLog, models.WorkflowLog]
	taskMapper     *mapper.CopierMapper[adminv1.WorkflowTask, models.WorkflowTask]
}

// NewWorkflowInstanceCase 创建工作流流程实例业务实例。
func NewWorkflowInstanceCase(
	baseCase *biz.BaseCase,
	catalog *i18n.I18n,
	tx data.Transaction,
	instanceRepo *data.WorkflowInstanceRepository,
	definitionRepo *data.WorkflowDefinitionRepository,
	taskRepo *data.WorkflowTaskRepository,
	logRepo *data.WorkflowLogRepository,
	joinRepo *data.WorkflowJoinRepository,
	delegationRepo *data.WorkflowDelegationRepository,
	baseUserRepo *data.BaseUserRepository,
	eventRegistry *pkgWorkflow.EventRegistry,
	scriptRunRepo *data.WorkflowScriptRunRepository,
	scriptLogRepo *data.BaseScriptLogRepository,
	runtime *scripting.Runtime,
) *WorkflowInstanceCase {
	caseValue := &WorkflowInstanceCase{
		BaseCase:                   baseCase,
		tx:                         tx,
		catalog:                    catalog,
		WorkflowInstanceRepository: instanceRepo,
		definitionRepo:             definitionRepo,
		taskRepo:                   taskRepo,
		logRepo:                    logRepo,
		joinRepo:                   joinRepo,
		delegationRepo:             delegationRepo,
		baseUserRepo:               baseUserRepo,
		eventRegistry:              eventRegistry,
		scriptRunRepo:              scriptRunRepo, scriptLogRepo: scriptLogRepo, runtime: runtime,
		mapper:     mapper.NewCopierMapper[adminv1.WorkflowInstance, models.WorkflowInstance](),
		logMapper:  mapper.NewCopierMapper[adminv1.WorkflowLog, models.WorkflowLog](),
		taskMapper: mapper.NewCopierMapper[adminv1.WorkflowTask, models.WorkflowTask](),
	}
	// 暴露公共引擎能力，宿主项目经 pkg/workflow.Submit 调用（对齐 notification.SetDefaultPublisher 模式）。
	pkgWorkflow.SetDefaultEngine(workflow.NewEngine(baseCase, instanceRepo, func(ctx context.Context, params pkgWorkflow.SubmitParams) (int64, error) {
		return caseValue.submit(ctx, params.TenantID, params.ApplicantUserID, params.Code, params.Version, params.FormData, params.BusinessType, params.BusinessID, params.OnCreated)
	}))
	return caseValue
}

// SubmitWorkflowInstance 提交审批申请：校验启用定义与节点图，创建实例后从开始节点推进。
func (c *WorkflowInstanceCase) SubmitWorkflowInstance(ctx context.Context, req *adminv1.SubmitWorkflowInstanceRequest) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	return c.submit(ctx, authInfo.TenantId, authInfo.UserId, req.GetCode(), req.GetVersion(), req.GetFormData(), req.GetBusinessType(), req.GetBusinessId(), nil)
}

// submit 持久化实例与启动恢复点，宿主关联成功后首次推进，中断由扫描核验后恢复。
func (c *WorkflowInstanceCase) submit(
	ctx context.Context, tenantID, applicantUserID int64,
	code string, version int32, formData, businessType string, businessID int64,
	onCreated func(context.Context, int64) error,
) (int64, error) {
	ctx = workflowParticipantContext(ctx)
	if onCreated != nil && !c.eventRegistry.HasBindingVerifier(businessType) {
		return 0, errorsx.InvalidArgument("宿主关联回写必须注册流程关联核验器")
	}
	// 宿主可以提交没有字段的表单，统一按空 JSON 对象参与脚本输入。
	if formData == "" {
		formData = "{}"
	}
	definitionQuery := c.definitionRepo.Query(ctx).WorkflowDefinition
	definitionOpts := make([]repository.QueryOption, 0, 4)
	definitionOpts = append(definitionOpts, repository.Where(definitionQuery.TenantID.Eq(tenantID)))
	definitionOpts = append(definitionOpts, repository.Where(definitionQuery.Code.Eq(code)))
	definitionOpts = append(definitionOpts, repository.Where(definitionQuery.Version.Eq(version)))
	definition, err := c.definitionRepo.Find(ctx, definitionOpts...)
	if err != nil {
		return 0, errorsx.ResourceNotFound("流程定义不存在")
	}
	if definition.Status != workflowStatusDefinitionEnabled {
		return 0, errorsx.ResourceNotFound("流程定义未启用")
	}
	var graph *workflow.Graph
	graph, err = workflow.Parse(definition.NodeConfig)
	if err != nil {
		return 0, errorsx.InvalidArgument("节点图配置无效: " + err.Error()).WithCause(err)
	}
	if err = workflow.Validate(graph); err != nil {
		return 0, errorsx.InvalidArgument("节点图配置无效: " + err.Error())
	}
	startNode := graph.StartNode()
	if startNode == nil {
		return 0, errorsx.InvalidArgument("节点图配置无效: 缺少开始节点")
	}

	instance := &models.WorkflowInstance{
		TenantID:        tenantID,
		DefinitionID:    definition.ID,
		ApplicantUserID: applicantUserID,
		FormData:        formData,
		NodeConfig:      definition.NodeConfig,
		Variables:       "{}",
		BusinessType:    businessType,
		BusinessID:      businessID,
		Status:          workflowStatusInstancePending,
		SubmitAt:        time.Now(),
	}
	var beforeRun *models.WorkflowScriptRun
	var startRun *models.WorkflowScriptRun
	beforeRun, err = c.beforeSubmitScript(ctx, instance, graph)
	if err != nil {
		return 0, err
	}
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, instance)
		if err != nil {
			return err
		}
		if beforeRun != nil {
			beforeRun.InstanceID = instance.ID
			if err = c.saveScriptRun(ctx, beforeRun); err != nil {
				return err
			}
		}
		startRun, err = c.enqueueWorkflowStart(ctx, instance, startNode.ID, onCreated != nil)
		if err != nil {
			return err
		}
		err = c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       tenantID,
			InstanceID:     instance.ID,
			Action:         workflowLogActionSubmit,
			OperatorUserID: applicantUserID,
		})
		return err
	})
	if err != nil {
		return 0, err
	}
	if onCreated != nil {
		err = onCreated(ctx, instance.ID)
		if err != nil {
			return 0, errorsx.Internal("保存业务单据流程关联失败").WithCause(errors.Join(err, c.abortUnlinkedSubmission(context.WithoutCancel(ctx), instance)))
		}
		if err = c.confirmWorkflowBinding(context.WithoutCancel(ctx), startRun); err != nil {
			// 宿主可能已经提交，禁止把未知结果当作未关联而终止，扫描将再次核验。
			log.Error(fmt.Sprintf("workflow binding confirmation failed: instance=%d err=%v", instance.ID, err))
			return instance.ID, nil
		}
	}
	if err = c.resumeWorkflowStart(ctx, startRun, true, time.Now()); err != nil {
		log.Error(fmt.Sprintf("workflow startup dispatch failed: instance=%d err=%v", instance.ID, err))
	}
	return instance.ID, nil
}

// abortUnlinkedSubmission 终结尚未成功绑定业务单据的申请，避免触发单据副作用。
func (c *WorkflowInstanceCase) abortUnlinkedSubmission(ctx context.Context, instance *models.WorkflowInstance) error {
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		current, err := c.lockWorkflowInstance(ctx, instance.TenantID, instance.ID)
		if err != nil {
			return err
		}
		if current.Status != workflowStatusInstancePending {
			return nil
		}
		current.Status = workflowStatusInstanceRejected
		now := time.Now()
		current.EndAt = &now
		if err = c.UpdateByID(ctx, current); err != nil {
			return err
		}
		if err = c.cancelNodeScripts(ctx, current); err != nil {
			return err
		}
		return c.logRepo.Create(ctx, &models.WorkflowLog{TenantID: current.TenantID, InstanceID: current.ID, Action: workflowLogActionReject, Comment: "业务单据关联失败，申请已终止"})
	})
}

// WithdrawWorkflowInstance 申请人撤回自己的进行中申请。
func (c *WorkflowInstanceCase) WithdrawWorkflowInstance(ctx context.Context, id int64) error {
	ctx = workflowParticipantContext(ctx)
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var instance *models.WorkflowInstance
	instance, err = c.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if instance.TenantID != authInfo.TenantId || instance.ApplicantUserID != authInfo.UserId {
		return errorsx.PermissionDenied("只能撤回自己的申请")
	}
	if instance.Status != workflowStatusInstancePending {
		return errorsx.StateConflict("撤回申请失败，申请不在审批中", "workflow_instance", instanceStatusName(instance.Status), "PENDING")
	}

	var assignees []int64
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err = c.lockWorkflowInstance(ctx, instance.TenantID, instance.ID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending {
			return errorsx.StateConflict("撤回申请失败，申请不在审批中", "workflow_instance", instanceStatusName(instance.Status), "PENDING")
		}
		startQuery := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
		var starts []*models.WorkflowScriptRun
		starts, err = c.scriptRunRepo.List(ctx, repository.Where(startQuery.TenantID.Eq(instance.TenantID)), repository.Where(startQuery.InstanceID.Eq(instance.ID)), repository.Where(startQuery.Stage.Eq("START")), repository.Where(startQuery.Status.In(scriptPending, scriptRunning, scriptFailed)))
		if err != nil {
			return err
		}
		for _, start := range starts {
			var input dto.WorkflowStartInput
			if err = json.Unmarshal([]byte(start.InputJSON), &input); err != nil {
				return err
			}
			// 防止慢速宿主关联提交晚于撤回，导致宿主已错过终态通知。
			if input.BindingRequired && !input.BindingConfirmed {
				return errorsx.Conflict("业务单据关联尚未确认，请稍后撤回")
			}
		}
		assignees, err = c.listPendingAssignees(ctx, instance.TenantID, instance.ID)
		if err != nil {
			return err
		}
		if err = c.cancelPendingTasks(ctx, instance.TenantID, instance.ID); err != nil {
			return err
		}
		if err = c.deleteJoins(ctx, instance.TenantID, instance.ID); err != nil {
			return err
		}
		instance.Status = workflowStatusInstanceWithdrawn
		now := time.Now()
		instance.EndAt = &now
		if err = c.UpdateByID(ctx, instance); err != nil {
			return err
		}
		if err = c.enqueueTerminalScript(ctx, instance); err != nil {
			return err
		}
		return c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       instance.TenantID,
			InstanceID:     instance.ID,
			Action:         workflowLogActionWithdraw,
			OperatorUserID: authInfo.UserId,
		})
	})
	if err != nil {
		return err
	}

	recipients := make([]workflowNotificationRecipient, 0, len(assignees))
	for _, userID := range assignees {
		recipients = append(recipients, workflowNotificationRecipient{userID: userID, instanceID: instance.ID, businessType: instance.BusinessType})
	}
	c.notifyManyAsync(ctx, instance.TenantID, recipients, c.workflowNotification(ctx, "system.workflow.notification.title.withdrawn", "system.workflow.notification.content.withdrawn"))
	c.fireBusinessEvent(ctx, instance, adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_WITHDRAWN)
	return nil
}

// PageWorkflowInstance 分页查询流程实例，合并流程定义与申请人信息。
func (c *WorkflowInstanceCase) PageWorkflowInstance(ctx context.Context, req *adminv1.PageWorkflowInstanceRequest) (*adminv1.PageWorkflowInstanceResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).WorkflowInstance
	opts := make([]repository.QueryOption, 0, 8)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		if authInfo.TenantCode != gorm.DefaultTenantCode && req.GetTenantId() != authInfo.TenantId {
			return nil, errorsx.PermissionDenied("不能查询其他租户的流程实例")
		}
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	} else if authInfo.TenantCode != gorm.DefaultTenantCode {
		opts = append(opts, repository.Where(query.TenantID.Eq(authInfo.TenantId)))
	}
	if code := req.GetCode(); code != "" {
		definitionQuery := c.definitionRepo.Query(ctx).WorkflowDefinition
		definitionOpts := make([]repository.QueryOption, 0, 3)
		definitionOpts = append(definitionOpts, repository.Select(definitionQuery.ID))
		definitionOpts = append(definitionOpts, repository.Where(definitionQuery.Code.Like("%"+code+"%")))
		if req.GetTenantId() > 0 {
			definitionOpts = append(definitionOpts, repository.Where(definitionQuery.TenantID.Eq(req.GetTenantId())))
		} else if authInfo.TenantCode != gorm.DefaultTenantCode {
			definitionOpts = append(definitionOpts, repository.Where(definitionQuery.TenantID.Eq(authInfo.TenantId)))
		}
		var definitions []*models.WorkflowDefinition
		definitions, err = c.definitionRepo.List(ctx, definitionOpts...)
		if err != nil {
			return nil, err
		}
		if len(definitions) == 0 {
			return &adminv1.PageWorkflowInstanceResponse{Instances: make([]*adminv1.WorkflowInstance, 0)}, nil
		}
		definitionIDs := make([]int64, 0, len(definitions))
		for _, definition := range definitions {
			definitionIDs = append(definitionIDs, definition.ID)
		}
		opts = append(opts, repository.Where(query.DefinitionID.In(definitionIDs...)))
	}
	if req.GetDefinitionId() > 0 {
		opts = append(opts, repository.Where(query.DefinitionID.Eq(req.GetDefinitionId())))
	}
	if req.GetApplicantUserId() > 0 {
		opts = append(opts, repository.Where(query.ApplicantUserID.Eq(req.GetApplicantUserId())))
	}
	if req.GetBusinessType() != "" {
		opts = append(opts, repository.Where(query.BusinessType.Eq(req.GetBusinessType())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.WorkflowInstance
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	var definitionMap map[int64]*models.WorkflowDefinition
	definitionMap, err = c.mapDefinitions(ctx, listDefinitionIDs(list))
	if err != nil {
		return nil, err
	}
	var userMap map[int64]string
	userMap, err = c.mapUserNames(ctx, listApplicantIDs(list))
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.WorkflowInstance, 0, len(list))
	for _, item := range list {
		dto := c.mapper.ToDTO(item)
		dto.NodeConfig = ""
		if definition := definitionMap[item.DefinitionID]; definition != nil {
			dto.DefinitionCode = definition.Code
			dto.DefinitionName = definition.Name
		}
		dto.ApplicantName = userMap[item.ApplicantUserID]
		resList = append(resList, dto)
	}
	return &adminv1.PageWorkflowInstanceResponse{Instances: resList, Total: int32(total)}, nil
}

// GetWorkflowInstance 查询流程实例详情，含审批日志与任务。
func (c *WorkflowInstanceCase) GetWorkflowInstance(ctx context.Context, id int64) (*adminv1.GetWorkflowInstanceResponse, error) {
	instance, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var logs []*models.WorkflowLog
	logs, err = c.listLogs(ctx, instance.TenantID, instance.ID)
	if err != nil {
		return nil, err
	}
	var tasks []*models.WorkflowTask
	tasks, err = c.listTasks(ctx, instance.TenantID, instance.ID)
	if err != nil {
		return nil, err
	}

	var userMap map[int64]string
	userMap, err = c.mapUserNames(ctx, collectWorkflowUserIDs(instance, logs, tasks))
	if err != nil {
		return nil, err
	}
	var definitionMap map[int64]*models.WorkflowDefinition
	definitionMap, err = c.mapDefinitions(ctx, []int64{instance.DefinitionID})
	if err != nil {
		return nil, err
	}

	dto := c.mapper.ToDTO(instance)
	dto.ApplicantName = userMap[instance.ApplicantUserID]
	if definition := definitionMap[instance.DefinitionID]; definition != nil {
		dto.DefinitionCode = definition.Code
		dto.DefinitionName = definition.Name
	}
	logDtos := make([]*adminv1.WorkflowLog, 0, len(logs))
	for _, item := range logs {
		logDto := c.logMapper.ToDTO(item)
		logDto.OperatorName = userMap[item.OperatorUserID]
		logDtos = append(logDtos, logDto)
	}
	taskDtos := make([]*adminv1.WorkflowTask, 0, len(tasks))
	for _, item := range tasks {
		taskDto := c.taskMapper.ToDTO(item)
		taskDto.AssigneeName = userMap[item.AssigneeUserID]
		taskDtos = append(taskDtos, taskDto)
	}
	var scriptRuns []*adminv1.WorkflowScriptRun
	scriptRuns, err = c.listScriptRuns(ctx, instance.TenantID, instance.ID)
	if err != nil {
		return nil, err
	}
	var graph *workflow.Graph
	graph, err = c.graphOfInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	graph.ScriptSnapshots = nil
	dto.NodeConfig, err = graph.Marshal()
	if err != nil {
		return nil, err
	}
	return &adminv1.GetWorkflowInstanceResponse{Instance: dto, Logs: logDtos, Tasks: taskDtos, ScriptRuns: scriptRuns}, nil
}

// AuditWorkflowTask 审批任务办理入口，按动作分派。
func (c *WorkflowInstanceCase) AuditWorkflowTask(ctx context.Context, req *adminv1.AuditWorkflowTaskRequest) error {
	ctx = workflowParticipantContext(ctx)
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var task *models.WorkflowTask
	task, err = c.taskRepo.FindByID(ctx, req.GetTaskId())
	if err != nil {
		return err
	}
	if task.TenantID != authInfo.TenantId || task.AssigneeUserID != authInfo.UserId {
		return errorsx.PermissionDenied("不是您的待审批任务")
	}
	if task.Status != workflowStatusTaskPending {
		return errorsx.StateConflict("任务已被处理", "workflow_task", taskStatusName(task.Status), "PENDING")
	}
	var instance *models.WorkflowInstance
	instance, err = c.FindByID(ctx, task.InstanceID)
	if err != nil {
		return err
	}
	if instance.Status != workflowStatusInstancePending {
		return errorsx.StateConflict("流程实例不在审批中", "workflow_instance", instanceStatusName(instance.Status), "PENDING")
	}

	var graph *workflow.Graph
	graph, err = c.graphOfInstance(ctx, instance)
	if err != nil {
		return err
	}
	node := graph.Nodes[task.NodeID]
	if node == nil || node.Type != workflow.NodeTypeTask {
		return errorsx.StateConflict("流程节点状态异常", "workflow_task", task.NodeID, workflow.NodeTypeTask)
	}

	switch req.GetAction() {
	case adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_APPROVE:
		return c.handleApprove(ctx, authInfo.UserId, task, graph, node, req.GetComment())
	case adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_REJECT:
		return c.handleReject(ctx, authInfo.UserId, task, instance, req.GetComment())
	case adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_FORWARD:
		return c.handleForward(ctx, authInfo.UserId, task, req.GetForwardTo())
	case adminv1.WorkflowAuditAction_WORKFLOW_AUDIT_ACTION_ADD_APPROVER:
		return c.handleAddApprover(ctx, authInfo.UserId, task, node, req.GetAdditionalApprover())
	default:
		return errorsx.InvalidArgument("审批动作无效")
	}
}

// handleApprove 锁定实例并核验当前办理人，或签立即推进，会签全员通过才推进。
func (c *WorkflowInstanceCase) handleApprove(
	ctx context.Context, userID int64,
	task *models.WorkflowTask, graph *workflow.Graph, node *workflow.Node, comment string,
) error {
	var err error
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		if err = c.lockPendingWorkflowTask(ctx, userID, task); err != nil {
			return err
		}
		if err = c.claimTask(ctx, task, workflowStatusTaskApproved, comment); err != nil {
			return err
		}
		err = c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       task.TenantID,
			InstanceID:     task.InstanceID,
			NodeID:         task.NodeID,
			TaskID:         task.ID,
			Action:         workflowLogActionApprove,
			OperatorUserID: userID,
			Comment:        comment,
		})
		if err != nil {
			return err
		}
		if len(node.OutEdges) != 1 {
			return c.failClosedTerminate(ctx, task.TenantID, task.InstanceID, "审批节点出边数量异常")
		}
		nextNodeID := node.OutEdges[0].To
		if node.IsAnyStrategy() {
			if err = c.cancelPendingTasksByNode(ctx, task.TenantID, task.InstanceID, task.NodeID); err != nil {
				return err
			}
			return c.advanceFromNode(ctx, task.TenantID, task.InstanceID, graph, nextNodeID, newWalkerState())
		}

		var pendingCount int64
		pendingCount, err = c.countPendingTasksByNode(ctx, task.TenantID, task.InstanceID, task.NodeID)
		if err != nil {
			return err
		}
		if pendingCount > 0 {
			return nil
		}
		return c.advanceFromNode(ctx, task.TenantID, task.InstanceID, graph, nextNodeID, newWalkerState())
	})
	return err
}

// withWorkflowTransaction 复用工作流事务，并仅在最外层提交成功后执行通知和业务回调。
func (c *WorkflowInstanceCase) withWorkflowTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, active := ctx.Value(workflowTransactionMarker{}).(*[]func(context.Context)); active {
		return fn(ctx)
	}
	var effects []func(context.Context)
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		return fn(context.WithValue(ctx, workflowTransactionMarker{}, &effects))
	})
	if err != nil {
		return err
	}
	for _, effect := range effects {
		effect(ctx)
	}
	return nil
}

// lockPendingWorkflowTask 锁定实例并重新核验任务状态和办理人，阻断转办或终结后的旧请求。
func (c *WorkflowInstanceCase) lockPendingWorkflowTask(ctx context.Context, userID int64, task *models.WorkflowTask) error {
	instance, err := c.lockWorkflowInstance(ctx, task.TenantID, task.InstanceID)
	if err != nil {
		return err
	}
	if instance.Status != workflowStatusInstancePending {
		return errorsx.StateConflict("流程实例不在审批中", "workflow_instance", instanceStatusName(instance.Status), "PENDING")
	}
	var current *models.WorkflowTask
	current, err = c.taskRepo.FindByID(ctx, task.ID)
	if err != nil {
		return err
	}
	if current.TenantID != task.TenantID || current.AssigneeUserID != userID || current.Status != workflowStatusTaskPending {
		return errorsx.Conflict("任务已被处理或转办")
	}
	return nil
}

// lockWorkflowInstance 锁定流程实例行，串行化审批推进与终态迁移。
func (c *WorkflowInstanceCase) lockWorkflowInstance(ctx context.Context, tenantID, instanceID int64) (*models.WorkflowInstance, error) {
	query := c.Query(ctx).WorkflowInstance
	return c.Find(ctx,
		repository.Where(query.TenantID.Eq(tenantID)),
		repository.Where(query.ID.Eq(instanceID)),
		repository.Clauses(clause.Locking{Strength: "UPDATE"}),
	)
}

// handleReject 在同一实例锁和事务内驳回任务、终结实例，提交后通知并回调业务。
func (c *WorkflowInstanceCase) handleReject(
	ctx context.Context, userID int64,
	task *models.WorkflowTask, instance *models.WorkflowInstance, comment string,
) error {
	var err error
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		if err = c.lockPendingWorkflowTask(ctx, userID, task); err != nil {
			return err
		}
		if err = c.claimTask(ctx, task, workflowStatusTaskRejected, comment); err != nil {
			return err
		}
		err = c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       task.TenantID,
			InstanceID:     task.InstanceID,
			NodeID:         task.NodeID,
			TaskID:         task.ID,
			Action:         workflowLogActionReject,
			OperatorUserID: userID,
			Comment:        comment,
		})
		if err != nil {
			return err
		}
		return c.terminateInstance(ctx, instance, workflowStatusInstanceRejected)
	})
	if err != nil {
		return err
	}

	c.notifyManyAsync(ctx, instance.TenantID, []workflowNotificationRecipient{{userID: instance.ApplicantUserID, instanceID: instance.ID, businessType: instance.BusinessType}}, c.workflowNotification(ctx, "system.workflow.notification.title.rejected", "system.workflow.notification.content.rejected"))
	c.fireBusinessEvent(ctx, instance, adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_REJECTED)
	return nil
}

// handleForward 锁定实例并核验当前办理人，在同一事务中转办任务和记录日志。
func (c *WorkflowInstanceCase) handleForward(
	ctx context.Context, userID int64, task *models.WorkflowTask, forwardTo int64,
) error {
	if forwardTo == 0 || forwardTo == userID {
		return errorsx.InvalidArgument("转办目标用户无效")
	}
	err := c.ensureUserActive(ctx, task.TenantID, forwardTo)
	if err != nil {
		return err
	}
	var businessType string
	businessType, err = c.workflowBusinessType(ctx, task.TenantID, task.InstanceID)
	if err != nil {
		return err
	}

	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		if err = c.lockPendingWorkflowTask(ctx, userID, task); err != nil {
			return err
		}
		taskQuery := c.taskRepo.Query(ctx).WorkflowTask
		var info gen.ResultInfo
		info, err = taskQuery.WithContext(ctx).
			Where(taskQuery.TenantID.Eq(task.TenantID), taskQuery.ID.Eq(task.ID), taskQuery.Status.Eq(workflowStatusTaskPending), taskQuery.AssigneeUserID.Eq(userID)).
			UpdateSimple(taskQuery.AssigneeUserID.Value(forwardTo))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return errorsx.Conflict("任务已被处理或转办")
		}
		return c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       task.TenantID,
			InstanceID:     task.InstanceID,
			NodeID:         task.NodeID,
			TaskID:         task.ID,
			Action:         workflowLogActionForward,
			OperatorUserID: userID,
			Comment:        "转办",
		})
	})
	if err != nil {
		return err
	}
	c.notifyManyAsync(ctx, task.TenantID, []workflowNotificationRecipient{{userID: forwardTo, taskID: task.ID, instanceID: task.InstanceID, businessType: businessType}}, c.workflowNotification(ctx, "system.workflow.notification.title.forwarded", "system.workflow.notification.content.pending"))
	return nil
}

// handleAddApprover 锁定进行中实例并核验原任务，在同一事务内创建加签任务和日志。
func (c *WorkflowInstanceCase) handleAddApprover(
	ctx context.Context, userID int64, task *models.WorkflowTask,
	node *workflow.Node, additionalApprover int64,
) error {
	if additionalApprover == 0 || additionalApprover == userID {
		return errorsx.InvalidArgument("加签用户无效")
	}
	err := c.ensureUserActive(ctx, task.TenantID, additionalApprover)
	if err != nil {
		return err
	}
	var businessType string
	businessType, err = c.workflowBusinessType(ctx, task.TenantID, task.InstanceID)
	if err != nil {
		return err
	}

	var addedTask *models.WorkflowTask
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		if err = c.lockPendingWorkflowTask(ctx, userID, task); err != nil {
			return err
		}
		addedTask = &models.WorkflowTask{
			TenantID:       task.TenantID,
			InstanceID:     task.InstanceID,
			NodeID:         task.NodeID,
			AssigneeUserID: additionalApprover,
			Status:         workflowStatusTaskPending,
		}
		err = c.taskRepo.Create(ctx, addedTask)
		if err != nil {
			return err
		}
		return c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       task.TenantID,
			InstanceID:     task.InstanceID,
			NodeID:         task.NodeID,
			TaskID:         task.ID,
			Action:         workflowLogActionForward,
			OperatorUserID: userID,
			Comment:        "加签：新增审批人",
		})
	})
	if err != nil {
		return err
	}
	c.notifyManyAsync(ctx, task.TenantID, []workflowNotificationRecipient{{userID: additionalApprover, taskID: addedTask.ID, instanceID: task.InstanceID, businessType: businessType}}, c.workflowNotification(ctx, "system.workflow.notification.title.added", "system.workflow.notification.content.pending"))
	return nil
}

// ===================== 图遍历 walker =====================

// walkerState 跟踪单次推进的步数和尚未展开完成的并行分支层级。
type walkerState struct {
	steps     int
	forkDepth int
}

// newWalkerState 创建步数守卫状态。
func newWalkerState() *walkerState {
	return &walkerState{}
}

// advanceFromNode 遍历节点图，等待并行分支全部展开后再检查通过终态。
func (c *WorkflowInstanceCase) advanceFromNode(
	ctx context.Context, tenantID, instanceID int64,
	graph *workflow.Graph, fromNodeID string, ws *walkerState,
) error {
	ws.steps++
	if ws.steps > walkerStepLimit {
		return c.failClosedTerminate(ctx, tenantID, instanceID, "推进步数超出上限")
	}

	node := graph.Nodes[fromNodeID]
	if node == nil {
		return c.failClosedTerminate(ctx, tenantID, instanceID, "流程节点丢失: "+fromNodeID)
	}

	var err error
	switch node.Type {
	case workflow.NodeTypeStart:
		if len(node.OutEdges) != 1 {
			return c.failClosedTerminate(ctx, tenantID, instanceID, "开始节点出边数量异常")
		}
		return c.advanceFromNode(ctx, tenantID, instanceID, graph, node.OutEdges[0].To, ws)

	case workflow.NodeTypeScript:
		var instance *models.WorkflowInstance
		instance, err = c.FindByID(ctx, instanceID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending {
			return nil
		}
		var ready bool
		_, ready, err = c.runNodeScript(ctx, instance, graph, node, "NODE", node.ScriptID, false)
		if err != nil || !ready {
			return err
		}
		return c.advanceFromNode(ctx, tenantID, instanceID, graph, node.OutEdges[0].To, ws)
	case workflow.NodeTypeTask:
		return c.enterTaskNode(ctx, tenantID, instanceID, graph, node, ws)

	case workflow.NodeTypeExclusiveGateway:
		var formData string
		formData, err = c.getFormData(ctx, tenantID, instanceID)
		if err != nil {
			return err
		}
		var nextNodeID, defaultNodeID string
		nextNodeID, defaultNodeID, err = selectExclusiveEdge(node, formData)
		if err != nil {
			return c.failClosedTerminate(ctx, tenantID, instanceID, "排他网关条件求值失败: "+err.Error())
		}
		if nextNodeID != "" {
			return c.advanceFromNode(ctx, tenantID, instanceID, graph, nextNodeID, ws)
		}
		if defaultNodeID != "" {
			return c.advanceFromNode(ctx, tenantID, instanceID, graph, defaultNodeID, ws)
		}
		return c.failClosedTerminate(ctx, tenantID, instanceID, "排他网关无命中条件且无默认边")

	case workflow.NodeTypeParallelFork:
		ws.forkDepth++
		for _, edge := range node.OutEdges {
			if err = c.advanceFromNode(ctx, tenantID, instanceID, graph, edge.To, ws); err != nil {
				ws.forkDepth--
				return err
			}
		}
		ws.forkDepth--
		if ws.forkDepth == 0 {
			return c.checkTermination(ctx, tenantID, instanceID)
		}
		return nil

	case workflow.NodeTypeParallelJoin:
		var released bool
		released, err = c.arriveJoin(ctx, tenantID, instanceID, node)
		if err != nil {
			return err
		}
		if !released {
			return nil
		}
		return c.advanceFromNode(ctx, tenantID, instanceID, graph, node.OutEdges[0].To, ws)

	case workflow.NodeTypeEnd:
		if ws.forkDepth > 0 {
			return nil
		}
		return c.checkTermination(ctx, tenantID, instanceID)

	default:
		return c.failClosedTerminate(ctx, tenantID, instanceID, "不支持的节点类型: "+node.Type)
	}
}

// enterTaskNode 在推进事务内创建节点待办，申请人命中或签审批人时自动通过。
func (c *WorkflowInstanceCase) enterTaskNode(
	ctx context.Context, tenantID, instanceID int64,
	graph *workflow.Graph, node *workflow.Node, ws *walkerState,
) error {
	instanceQuery := c.Query(ctx).WorkflowInstance
	instanceOpts := make([]repository.QueryOption, 0, 2)
	instanceOpts = append(instanceOpts, repository.Where(instanceQuery.TenantID.Eq(tenantID)))
	instanceOpts = append(instanceOpts, repository.Where(instanceQuery.ID.Eq(instanceID)))
	instance, err := c.Find(ctx, instanceOpts...)
	if err != nil {
		return err
	}
	if instance.Status != workflowStatusInstancePending {
		return nil
	}

	var approvers []int64
	if node.ApproverScriptID > 0 {
		var run *models.WorkflowScriptRun
		var ready bool
		run, ready, err = c.runNodeScript(ctx, instance, graph, node, "APPROVER", node.ApproverScriptID, false)
		if err != nil || !ready {
			return err
		}
		var output dto.WorkflowScriptOutput
		if err = json.Unmarshal([]byte(run.OutputJSON), &output); err != nil {
			return err
		}
		approvers = output.UserIDs
	} else {
		approvers, err = c.resolveApprovers(ctx, tenantID, instance.ApplicantUserID, node)
		if err != nil {
			return err
		}
	}
	others := make([]int64, 0, len(approvers))
	for _, approver := range approvers {
		if approver != instance.ApplicantUserID {
			others = append(others, approver)
		}
	}

	autoApprove := len(others) == 0
	if !autoApprove && node.IsAnyStrategy() {
		for _, approver := range approvers {
			if approver == instance.ApplicantUserID {
				autoApprove = true
				break
			}
		}
	}
	if autoApprove {
		// 审批人命中申请人或全部为申请人：清残留任务后自动通过并推进下一节点。
		if err = c.cancelPendingTasksByNode(ctx, tenantID, instanceID, node.ID); err != nil {
			return err
		}
		if err = c.logRepo.Create(ctx, &models.WorkflowLog{
			TenantID:       tenantID,
			InstanceID:     instanceID,
			NodeID:         node.ID,
			Action:         workflowLogActionApprove,
			OperatorUserID: instance.ApplicantUserID,
			Comment:        "审批人为申请人本人，自动通过",
		}); err != nil {
			return err
		}
		if len(node.OutEdges) != 1 {
			return c.failClosedTerminate(ctx, tenantID, instanceID, "审批节点出边数量异常")
		}
		return c.advanceFromNode(ctx, tenantID, instanceID, graph, node.OutEdges[0].To, ws)
	}

	var taskIDs map[int64]int64
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		taskIDs, err = c.createTaskNodeTasks(ctx, tenantID, instanceID, node.ID, others)
		return err
	})
	if err != nil {
		return err
	}
	recipients := make([]workflowNotificationRecipient, 0, len(others))
	for _, userID := range others {
		recipients = append(recipients, workflowNotificationRecipient{userID: userID, taskID: taskIDs[userID], instanceID: instanceID, businessType: instance.BusinessType})
	}
	c.notifyManyAsync(ctx, tenantID, recipients, c.workflowNotification(ctx, "system.workflow.notification.title.pending", "system.workflow.notification.content.pending"))
	return nil
}

// createTaskNodeTasks 清理节点旧待办并创建当前节点的审批任务。
func (c *WorkflowInstanceCase) createTaskNodeTasks(ctx context.Context, tenantID, instanceID int64, nodeID string, approvers []int64) (map[int64]int64, error) {
	err := c.cancelPendingTasksByNode(ctx, tenantID, instanceID, nodeID)
	if err != nil {
		return nil, err
	}
	taskIDs := make(map[int64]int64, len(approvers))
	for _, approver := range approvers {
		task := &models.WorkflowTask{
			TenantID:       tenantID,
			InstanceID:     instanceID,
			NodeID:         nodeID,
			AssigneeUserID: approver,
			Status:         workflowStatusTaskPending,
		}
		err = c.taskRepo.Create(ctx, task)
		if err != nil {
			return nil, err
		}
		taskIDs[approver] = task.ID
	}
	return taskIDs, nil
}

// selectExclusiveEdge 对排他网关出边条件求值，返回命中边与默认边的目标节点 ID。
func selectExclusiveEdge(node *workflow.Node, formData string) (hitTo, defaultTo string, err error) {
	for _, edge := range node.OutEdges {
		if edge.Condition == workflow.ConditionDefault {
			defaultTo = edge.To
			continue
		}
		result, evalErr := workflow.EvalCondition(edge.Condition, formData)
		if evalErr != nil {
			return "", "", evalErr
		}
		if result {
			return edge.To, "", nil
		}
	}
	return "", defaultTo, nil
}

// arriveJoin 并行汇聚到达计数：原子自增后满入度者负责删除计数并放行，未满者停止等待。
// 条件删除保证并发场景下仅一个分支放行，避免重复推进。
func (c *WorkflowInstanceCase) arriveJoin(ctx context.Context, tenantID, instanceID int64, node *workflow.Node) (bool, error) {
	joinQuery := c.joinRepo.Query(ctx).WorkflowJoin
	joinedWhere := []gen.Condition{
		joinQuery.TenantID.Eq(tenantID),
		joinQuery.InstanceID.Eq(instanceID),
		joinQuery.NodeID.Eq(node.ID),
	}
	info, err := joinQuery.WithContext(ctx).
		Where(joinedWhere...).
		UpdateSimple(joinQuery.ArrivedCount.Add(1))
	if err != nil {
		return false, err
	}
	if info.RowsAffected == 0 {
		if err := c.joinRepo.Create(ctx, &models.WorkflowJoin{
			TenantID:     tenantID,
			InstanceID:   instanceID,
			NodeID:       node.ID,
			ArrivedCount: 1,
		}); err != nil {
			if !errorsx.IsDuplicateKey(err) {
				return false, err
			}
			// 并发创建撞唯一键：退回自增路径，保证计数不丢。
			if _, err := joinQuery.WithContext(ctx).
				Where(joinedWhere...).
				UpdateSimple(joinQuery.ArrivedCount.Add(1)); err != nil {
				return false, err
			}
		}
	}

	delInfo, err := joinQuery.WithContext(ctx).
		Where(append(joinedWhere, joinQuery.ArrivedCount.Gte(int32(node.InDegree)))...).
		Delete()
	if err != nil {
		return false, err
	}
	return delInfo.RowsAffected > 0, nil
}

// checkTermination 事务内检查待办与汇聚并写入通过终态，提交后通知申请人和业务。
func (c *WorkflowInstanceCase) checkTermination(ctx context.Context, tenantID, instanceID int64) error {
	var instance *models.WorkflowInstance
	var err error
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err = c.checkTerminationInTransaction(ctx, tenantID, instanceID)
		return err
	})
	if err != nil {
		return err
	}
	if instance == nil {
		return nil
	}

	c.notifyManyAsync(ctx, tenantID, []workflowNotificationRecipient{{userID: instance.ApplicantUserID, instanceID: instance.ID, businessType: instance.BusinessType}}, c.workflowNotification(ctx, "system.workflow.notification.title.approved", "system.workflow.notification.content.approved"))
	c.fireBusinessEvent(ctx, instance, adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_APPROVED)
	return nil
}

// checkTerminationInTransaction 在已锁定实例行的事务中检查并写入通过终态。
func (c *WorkflowInstanceCase) checkTerminationInTransaction(ctx context.Context, tenantID, instanceID int64) (*models.WorkflowInstance, error) {
	instance, err := c.lockWorkflowInstance(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if instance.Status != workflowStatusInstancePending {
		return nil, nil
	}
	var pendingCount int64
	pendingCount, err = c.countPendingTasks(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		return nil, nil
	}
	var scriptCount int64
	scriptCount, err = c.pendingNodeScripts(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if scriptCount > 0 {
		return nil, nil
	}
	// 并行分支被网关绕行时汇聚永远不满，仍停留的 token 以未删的 join 行表达，不能提前判通过。
	joinQuery := c.joinRepo.Query(ctx).WorkflowJoin
	var joinCount int64
	joinCount, err = c.joinRepo.Count(ctx,
		repository.Where(joinQuery.TenantID.Eq(tenantID)),
		repository.Where(joinQuery.InstanceID.Eq(instanceID)),
	)
	if err != nil {
		return nil, err
	}
	if joinCount > 0 {
		return nil, nil
	}
	instance.Status = workflowStatusInstanceApproved
	now := time.Now()
	instance.EndAt = &now
	if err = c.UpdateByID(ctx, instance); err != nil {
		return nil, err
	}
	if err = c.enqueueTerminalScript(ctx, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// terminateInstance 复用推进事务终结实例，取消待办并清理汇聚计数。
func (c *WorkflowInstanceCase) terminateInstance(ctx context.Context, instance *models.WorkflowInstance, status int32) error {
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		return c.terminateInstanceInTransaction(ctx, instance.TenantID, instance.ID, status)
	})
}

// terminateInstanceInTransaction 在已锁定实例行的事务中写入终态，避免覆盖并发终结结果。
func (c *WorkflowInstanceCase) terminateInstanceInTransaction(ctx context.Context, tenantID, instanceID int64, status int32) error {
	instance, err := c.lockWorkflowInstance(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}
	if instance.Status != workflowStatusInstancePending {
		return nil
	}
	if err = c.cancelPendingTasks(ctx, tenantID, instanceID); err != nil {
		return err
	}
	if err = c.deleteJoins(ctx, tenantID, instanceID); err != nil {
		return err
	}
	instance.Status = status
	now := time.Now()
	instance.EndAt = &now
	if err = c.UpdateByID(ctx, instance); err != nil {
		return err
	}
	return c.enqueueTerminalScript(ctx, instance)
}

// failClosedTerminate 锁定进行中实例后按驳回终结，仅为实际发生的迁移安排提交后回调。
func (c *WorkflowInstanceCase) failClosedTerminate(ctx context.Context, tenantID, instanceID int64, reason string) error {
	log.Error(fmt.Sprintf("workflow fail-closed: instance=%d reason=%s", instanceID, reason))
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err := c.lockWorkflowInstance(ctx, tenantID, instanceID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending {
			return nil
		}
		if err = c.terminateInstance(ctx, instance, workflowStatusInstanceRejected); err != nil {
			return err
		}
		c.notifyManyAsync(ctx, tenantID, []workflowNotificationRecipient{{userID: instance.ApplicantUserID, instanceID: instance.ID, businessType: instance.BusinessType}}, c.workflowNotification(ctx, "system.workflow.notification.title.definition_rejected", "system.workflow.notification.content.definition_rejected"))
		c.fireBusinessEvent(ctx, instance, adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_REJECTED)
		return nil
	})
}

// ===================== 审批人解析 =====================

// resolveApprovers 解析审批节点审批人：按用户、角色、岗位展开为用户 ID，套用审批委托后去重保序。
func (c *WorkflowInstanceCase) resolveApprovers(ctx context.Context, tenantID, applicantUserID int64, node *workflow.Node) ([]int64, error) {
	var err error
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	seen := make(map[int64]struct{}, len(node.Approvers))
	approvers := make([]int64, 0, len(node.Approvers))

	appendUser := func(userID int64) {
		if userID == 0 {
			return
		}
		if _, dup := seen[userID]; dup {
			return
		}
		seen[userID] = struct{}{}
		approvers = append(approvers, userID)
	}
	for _, spec := range node.Approvers {
		switch spec.Type {
		case workflow.ApproverTypeUser:
			if err = c.ensureUserActive(ctx, tenantID, spec.ID); err != nil {
				return nil, err
			}
			appendUser(c.applyDelegation(ctx, tenantID, spec.ID))
		case workflow.ApproverTypeRole, workflow.ApproverTypePost:
			opts := make([]repository.QueryOption, 0, 3)
			opts = append(opts, repository.Where(userQuery.TenantID.Eq(tenantID)))
			opts = append(opts, repository.Select(userQuery.ID))
			opts = append(opts, repository.Where(userQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
			if spec.Type == workflow.ApproverTypeRole {
				opts = append(opts, repository.Where(userQuery.RoleID.Eq(spec.ID)))
			} else {
				opts = append(opts, repository.Where(userQuery.PostID.Eq(spec.ID)))
			}
			var users []*models.BaseUser
			users, err = c.baseUserRepo.List(ctx, opts...)
			if err != nil {
				return nil, err
			}
			for _, user := range users {
				appendUser(c.applyDelegation(ctx, tenantID, user.ID))
			}
		default:
			return nil, errorsx.InvalidArgument("审批人类型不支持: " + spec.Type)
		}
	}
	if len(approvers) == 0 {
		return nil, errorsx.InvalidArgument("审批节点未解析到有效审批人: " + node.ID)
	}
	for _, userID := range approvers {
		if err = c.ensureUserActive(ctx, tenantID, userID); err != nil {
			return nil, err
		}
	}
	return approvers, nil
}

// applyDelegation 返回用户设置审批委托时的被委托人，否则返回用户本身。
func (c *WorkflowInstanceCase) applyDelegation(ctx context.Context, tenantID, userID int64) int64 {
	delegationQuery := c.delegationRepo.Query(ctx).WorkflowDelegation
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Select(delegationQuery.DelegateUserID))
	opts = append(opts, repository.Where(delegationQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(delegationQuery.DelegatorUserID.Eq(userID)))
	opts = append(opts, repository.Where(delegationQuery.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	delegation, err := c.delegationRepo.Find(ctx, opts...)
	if err != nil {
		return userID
	}
	return delegation.DelegateUserID
}

// ensureUserActive 校验用户存在且启用。
func (c *WorkflowInstanceCase) ensureUserActive(ctx context.Context, tenantID, userID int64) error {
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Select(userQuery.ID, userQuery.Status))
	opts = append(opts, repository.Where(userQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(userQuery.ID.Eq(userID)))
	user, err := c.baseUserRepo.Find(ctx, opts...)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在")
	}
	if user.Status != _const.STATUS_STATUS_ENABLE {
		return errorsx.InvalidArgument("用户已禁用")
	}
	return nil
}

// ===================== 任务与查询辅助 =====================

// claimTask 仅占用仍属于原办理人的待办任务，防止转办后的旧请求继续推进。
func (c *WorkflowInstanceCase) claimTask(ctx context.Context, task *models.WorkflowTask, status int32, comment string) error {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	info, err := taskQuery.WithContext(ctx).
		Where(taskQuery.TenantID.Eq(task.TenantID), taskQuery.ID.Eq(task.ID), taskQuery.Status.Eq(workflowStatusTaskPending), taskQuery.AssigneeUserID.Eq(task.AssigneeUserID)).
		UpdateSimple(taskQuery.Status.Value(status), taskQuery.Comment.Value(comment))
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		return errorsx.Conflict("任务已被处理")
	}
	return nil
}

// cancelPendingTasks 取消实例全部待办任务。
func (c *WorkflowInstanceCase) cancelPendingTasks(ctx context.Context, tenantID, instanceID int64) error {
	return c.cancelPendingTasksByNode(ctx, tenantID, instanceID, "")
}

// cancelPendingTasksByNode 取消实例（或指定节点）的全部待办任务，防止回退推进后旧任务残留。
func (c *WorkflowInstanceCase) cancelPendingTasksByNode(ctx context.Context, tenantID, instanceID int64, nodeID string) error {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	stmt := taskQuery.WithContext(ctx).
		Where(taskQuery.TenantID.Eq(tenantID), taskQuery.InstanceID.Eq(instanceID), taskQuery.Status.Eq(workflowStatusTaskPending))
	if nodeID != "" {
		stmt = stmt.Where(taskQuery.NodeID.Eq(nodeID))
	}
	_, err := stmt.UpdateSimple(taskQuery.Status.Value(workflowStatusTaskCanceled))
	return err
}

// countPendingTasks 统计实例剩余待办任务数。
func (c *WorkflowInstanceCase) countPendingTasks(ctx context.Context, tenantID, instanceID int64) (int64, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	return c.taskRepo.Count(ctx,
		repository.Where(taskQuery.TenantID.Eq(tenantID)),
		repository.Where(taskQuery.InstanceID.Eq(instanceID)),
		repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)),
	)
}

// countPendingTasksByNode 统计节点剩余待办任务数。
func (c *WorkflowInstanceCase) countPendingTasksByNode(ctx context.Context, tenantID, instanceID int64, nodeID string) (int64, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	return c.taskRepo.Count(ctx,
		repository.Where(taskQuery.TenantID.Eq(tenantID)),
		repository.Where(taskQuery.InstanceID.Eq(instanceID)),
		repository.Where(taskQuery.NodeID.Eq(nodeID)),
		repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)),
	)
}

// deleteJoins 删除实例全部并行汇聚计数。
func (c *WorkflowInstanceCase) deleteJoins(ctx context.Context, tenantID, instanceID int64) error {
	joinQuery := c.joinRepo.Query(ctx).WorkflowJoin
	_, err := joinQuery.WithContext(ctx).
		Where(joinQuery.TenantID.Eq(tenantID), joinQuery.InstanceID.Eq(instanceID)).
		Delete()
	return err
}

// listPendingAssignees 查询实例待办任务的办理人去重列表。
func (c *WorkflowInstanceCase) listPendingAssignees(ctx context.Context, tenantID, instanceID int64) ([]int64, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	tasks, err := c.taskRepo.List(ctx,
		repository.Select(taskQuery.AssigneeUserID),
		repository.Where(taskQuery.TenantID.Eq(tenantID)),
		repository.Where(taskQuery.InstanceID.Eq(instanceID)),
		repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)),
	)
	if err != nil {
		return nil, err
	}
	assignees := make([]int64, 0, len(tasks))
	seen := make(map[int64]struct{}, len(tasks))
	for _, task := range tasks {
		if _, dup := seen[task.AssigneeUserID]; dup {
			continue
		}
		seen[task.AssigneeUserID] = struct{}{}
		assignees = append(assignees, task.AssigneeUserID)
	}
	return assignees, nil
}

// listLogs 查询实例审批日志，按时间正序。
func (c *WorkflowInstanceCase) listLogs(ctx context.Context, tenantID, instanceID int64) ([]*models.WorkflowLog, error) {
	logQuery := c.logRepo.Query(ctx).WorkflowLog
	return c.logRepo.List(ctx,
		repository.Where(logQuery.TenantID.Eq(tenantID)),
		repository.Where(logQuery.InstanceID.Eq(instanceID)),
		repository.Order(logQuery.CreatedAt.Asc()),
		repository.Order(logQuery.ID.Asc()),
	)
}

// listTasks 查询实例审批任务，按时间正序。
func (c *WorkflowInstanceCase) listTasks(ctx context.Context, tenantID, instanceID int64) ([]*models.WorkflowTask, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	return c.taskRepo.List(ctx,
		repository.Where(taskQuery.TenantID.Eq(tenantID)),
		repository.Where(taskQuery.InstanceID.Eq(instanceID)),
		repository.Order(taskQuery.CreatedAt.Asc()),
		repository.Order(taskQuery.ID.Asc()),
	)
}

// getFormData 查询实例申请表单数据。
func (c *WorkflowInstanceCase) getFormData(ctx context.Context, tenantID, instanceID int64) (string, error) {
	instanceQuery := c.Query(ctx).WorkflowInstance
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Select(instanceQuery.FormData, instanceQuery.Variables))
	opts = append(opts, repository.Where(instanceQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(instanceQuery.ID.Eq(instanceID)))
	instance, err := c.Find(ctx, opts...)
	if err != nil {
		return "", err
	}
	var form map[string]any
	form, err = scripting.ParseContext(instance.FormData)
	if err != nil {
		return "", err
	}
	var variables map[string]any
	variables, err = scripting.ParseContext(instance.Variables)
	if err != nil {
		return "", err
	}
	form["variables"] = variables
	return scripting.MarshalOutput(form)
}

// graphOfInstance 仅解析实例创建时固定的流程快照，定义编辑不影响在途流程。
func (c *WorkflowInstanceCase) graphOfInstance(ctx context.Context, instance *models.WorkflowInstance) (*workflow.Graph, error) {
	graph, err := workflow.Parse(instance.NodeConfig)
	if err != nil {
		return nil, errorsx.InvalidArgument("实例流程快照无效").WithCause(err)
	}
	return graph, nil
}

// mapDefinitions 按定义 ID 批量查询流程定义。
func (c *WorkflowInstanceCase) mapDefinitions(ctx context.Context, definitionIDs []int64) (map[int64]*models.WorkflowDefinition, error) {
	result := make(map[int64]*models.WorkflowDefinition, len(definitionIDs))
	if len(definitionIDs) == 0 {
		return result, nil
	}
	definitionQuery := c.definitionRepo.Query(ctx).WorkflowDefinition
	list, err := c.definitionRepo.List(ctx,
		repository.Select(definitionQuery.ID, definitionQuery.Code, definitionQuery.Name),
		repository.Where(definitionQuery.ID.In(definitionIDs...)),
	)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		result[item.ID] = item
	}
	return result, nil
}

// mapUserNames 按用户 ID 批量查询昵称映射，昵称缺失回退用户名。
func (c *WorkflowInstanceCase) mapUserNames(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	users, err := c.baseUserRepo.List(ctx,
		repository.Select(userQuery.ID, userQuery.NickName, userQuery.UserName),
		repository.Where(userQuery.ID.In(userIDs...)),
	)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user.NickName != "" {
			result[user.ID] = user.NickName
		} else {
			result[user.ID] = user.UserName
		}
	}
	return result, nil
}

// ===================== 超时催办 =====================

// 工作流超时催办阈值：催办提醒与升级提醒。
const (
	workflowTimeoutReminder   = 24 * time.Hour
	workflowTimeoutEscalation = 72 * time.Hour
)

// ScanWorkflowTimeouts 扫描全部待审批任务：超过催办阈值提醒办理人，超过升级阈值提醒办理人与申请人。
// 由定时任务在后台上下文跨租户调用；条件更新 remarked/escalated 标记保证多实例下每个待办各只执行一次。
// 升级仅追加通知不自动转办（部门暂无负责人字段，自动夺权风险高），与原设计的差异见工作流接入指南。
func (c *WorkflowInstanceCase) ScanWorkflowTimeouts(ctx context.Context, now time.Time) (int, int, error) {
	escalated, err := c.scanWorkflowEscalation(ctx, now)
	if err != nil {
		return 0, 0, err
	}
	reminded, err := c.scanWorkflowReminder(ctx, now)
	if err != nil {
		return 0, escalated, err
	}
	return reminded, escalated, nil
}

// scanWorkflowEscalation 升级批次：严重超时的待办提醒办理人与申请人各一次。
func (c *WorkflowInstanceCase) scanWorkflowEscalation(ctx context.Context, now time.Time) (int, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	escalatedAtEmpty := field.Or(taskQuery.EscalatedAt.IsNull(), taskQuery.EscalatedAt.Eq(time.Time{}))
	tasks, err := c.taskRepo.List(ctx,
		repository.Select(taskQuery.ID, taskQuery.TenantID, taskQuery.InstanceID, taskQuery.AssigneeUserID),
		repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)),
		repository.Where(escalatedAtEmpty),
		repository.Where(taskQuery.CreatedAt.Lte(now.Add(-workflowTimeoutEscalation))),
	)
	if err != nil {
		return 0, err
	}
	applicantMap, err := c.mapInstanceApplicants(ctx, collectTaskInstanceIDs(tasks))
	if err != nil {
		return 0, err
	}
	var businessTypeMap map[int64]string
	businessTypeMap, err = c.mapInstanceBusinessTypes(ctx, collectTaskInstanceIDs(tasks))
	if err != nil {
		return 0, err
	}

	escalated := 0
	for _, task := range tasks {
		if err = c.notifyMany(ctx, task.TenantID, workflowNotificationRecipient{userID: task.AssigneeUserID, taskID: task.ID, instanceID: task.InstanceID, businessType: businessTypeMap[task.InstanceID]}, c.workflowNotification(ctx, "system.workflow.notification.title.escalated_assignee", "system.workflow.notification.content.escalated_assignee"), fmt.Sprintf("workflow:timeout:%d:%d:escalated:assignee", task.TenantID, task.ID)); err != nil {
			log.Error(fmt.Sprintf("workflow escalation notification failed: task=%d err=%v", task.ID, err))
			continue
		}
		if applicant := applicantMap[task.InstanceID]; applicant > 0 {
			if err = c.notifyMany(ctx, task.TenantID, workflowNotificationRecipient{userID: applicant, instanceID: task.InstanceID, businessType: businessTypeMap[task.InstanceID]}, c.workflowNotification(ctx, "system.workflow.notification.title.escalated_applicant", "system.workflow.notification.content.escalated_applicant"), fmt.Sprintf("workflow:timeout:%d:%d:escalated:applicant", task.TenantID, task.ID)); err != nil {
				log.Error(fmt.Sprintf("workflow escalation notification failed: task=%d err=%v", task.ID, err))
				continue
			}
		}
		info, err := taskQuery.WithContext(ctx).
			Where(taskQuery.ID.Eq(task.ID), escalatedAtEmpty).
			UpdateSimple(taskQuery.EscalatedAt.Value(now))
		if err != nil {
			log.Error(fmt.Sprintf("workflow escalation mark failed: task=%d err=%v", task.ID, err))
			continue
		}
		if info.RowsAffected == 0 {
			continue
		}
		escalated++
	}
	return escalated, nil
}

// scanWorkflowReminder 催办批次：超时未处理的待办提醒办理人一次，已升级的不再重复催办。
func (c *WorkflowInstanceCase) scanWorkflowReminder(ctx context.Context, now time.Time) (int, error) {
	taskQuery := c.taskRepo.Query(ctx).WorkflowTask
	remindedAtEmpty := field.Or(taskQuery.RemindedAt.IsNull(), taskQuery.RemindedAt.Eq(time.Time{}))
	escalatedAtEmpty := field.Or(taskQuery.EscalatedAt.IsNull(), taskQuery.EscalatedAt.Eq(time.Time{}))
	tasks, err := c.taskRepo.List(ctx,
		repository.Select(taskQuery.ID, taskQuery.TenantID, taskQuery.InstanceID, taskQuery.AssigneeUserID),
		repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)),
		repository.Where(remindedAtEmpty),
		repository.Where(escalatedAtEmpty),
		repository.Where(taskQuery.CreatedAt.Lte(now.Add(-workflowTimeoutReminder))),
	)
	if err != nil {
		return 0, err
	}
	var businessTypeMap map[int64]string
	businessTypeMap, err = c.mapInstanceBusinessTypes(ctx, collectTaskInstanceIDs(tasks))
	if err != nil {
		return 0, err
	}

	reminded := 0
	for _, task := range tasks {
		if err = c.notifyMany(ctx, task.TenantID, workflowNotificationRecipient{userID: task.AssigneeUserID, taskID: task.ID, instanceID: task.InstanceID, businessType: businessTypeMap[task.InstanceID]}, c.workflowNotification(ctx, "system.workflow.notification.title.reminder", "system.workflow.notification.content.reminder"), fmt.Sprintf("workflow:timeout:%d:%d:reminder", task.TenantID, task.ID)); err != nil {
			log.Error(fmt.Sprintf("workflow reminder notification failed: task=%d err=%v", task.ID, err))
			continue
		}
		info, err := taskQuery.WithContext(ctx).
			Where(taskQuery.ID.Eq(task.ID), remindedAtEmpty, escalatedAtEmpty).
			UpdateSimple(taskQuery.RemindedAt.Value(now))
		if err != nil {
			log.Error(fmt.Sprintf("workflow reminder mark failed: task=%d err=%v", task.ID, err))
			continue
		}
		if info.RowsAffected == 0 {
			continue
		}
		reminded++
	}
	return reminded, nil
}

// notifyMany 同步发布超时通知，只有消息持久化成功后扫描才会写入催办标记。
func (c *WorkflowInstanceCase) notifyMany(ctx context.Context, tenantID int64, recipient workflowNotificationRecipient, message workflowNotificationMessage, idempotencyKey string) error {
	if recipient.userID == 0 {
		return nil
	}
	notifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.publishWorkflowNotification(notifyCtx, tenantID, recipient, message, idempotencyKey)
}

// mapInstanceApplicants 按实例 ID 批量查询申请人映射。
func (c *WorkflowInstanceCase) mapInstanceApplicants(ctx context.Context, instanceIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return result, nil
	}
	instanceQuery := c.Query(ctx).WorkflowInstance
	instances, err := c.List(ctx,
		repository.Select(instanceQuery.ID, instanceQuery.ApplicantUserID),
		repository.Where(instanceQuery.ID.In(instanceIDs...)),
	)
	if err != nil {
		return nil, err
	}
	for _, instance := range instances {
		result[instance.ID] = instance.ApplicantUserID
	}
	return result, nil
}

// mapInstanceBusinessTypes 按实例 ID 批量查询工作流业务类型。
func (c *WorkflowInstanceCase) mapInstanceBusinessTypes(ctx context.Context, instanceIDs []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return result, nil
	}
	query := c.Query(ctx).WorkflowInstance
	instances, err := c.List(ctx,
		repository.Select(query.ID, query.BusinessType),
		repository.Where(query.ID.In(instanceIDs...)),
	)
	if err != nil {
		return nil, err
	}
	for _, instance := range instances {
		result[instance.ID] = instance.BusinessType
	}
	return result, nil
}

// workflowBusinessType 查询实例业务类型，用于选择工作流消息来源。
func (c *WorkflowInstanceCase) workflowBusinessType(ctx context.Context, tenantID, instanceID int64) (string, error) {
	query := c.Query(ctx).WorkflowInstance
	instance, err := c.Find(ctx, repository.Where(query.TenantID.Eq(tenantID)), repository.Where(query.ID.Eq(instanceID)))
	if err != nil {
		return "", err
	}
	return instance.BusinessType, nil
}

// ===================== 通知与事件 =====================

// notifyManyAsync 在事务提交后使用独立身份上下文异步发送通知，避免复用已结束事务。
func (c *WorkflowInstanceCase) notifyManyAsync(ctx context.Context, tenantID int64, recipients []workflowNotificationRecipient, message workflowNotificationMessage) {
	if len(recipients) == 0 {
		return
	}
	afterWorkflowCommit(ctx, func(ctx context.Context) {
		// 仅复制认证身份，不能把请求上下文中的事务查询对象带入异步任务。
		notifyCtx := context.Background()
		authInfo, err := auth.FromContext(ctx)
		if err == nil {
			notifyCtx = engine.ContextWithAuthClaims(notifyCtx, authInfo.MakeAuthClaims())
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error(fmt.Sprintf("workflow notify panic: %v", r))
				}
			}()
			var cancel context.CancelFunc
			notifyCtx, cancel = context.WithTimeout(notifyCtx, 5*time.Second)
			defer cancel()
			for _, recipient := range recipients {
				if err = c.publishWorkflowNotification(notifyCtx, tenantID, recipient, message, ""); err != nil {
					log.Error(fmt.Sprintf("workflow notify failed: %v", err))
				}
			}
		}()
	})
}

// publishWorkflowNotification 发布一条绑定单个审批目标的管理端工作流通知。
func (c *WorkflowInstanceCase) publishWorkflowNotification(ctx context.Context, tenantID int64, recipient workflowNotificationRecipient, message workflowNotificationMessage, idempotencyKey string) error {
	if recipient.userID == 0 {
		return nil
	}
	source := workflowMessageSourceByBusinessType(recipient.businessType)
	if idempotencyKey == "" {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", recipient.taskID, recipient.instanceID, message.title)))
		idempotencyKey = fmt.Sprintf("workflow:notification:%d:%d:%x", tenantID, recipient.userID, digest)
	}
	_, err := notification.Publish(ctx, notification.Message{
		TenantID:       tenantID,
		CategoryCode:   workflowMessageCategory,
		Title:          message.title,
		Content:        message.content,
		ContentFormat:  basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT,
		Priority:       basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL,
		ActionType:     basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY,
		ActionTarget:   workflowMessageActionTarget,
		ActionParams:   workflowMessageActionParams(source, recipient.taskID, recipient.instanceID),
		Audiences:      []notification.Audience{{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER, ID: recipient.userID}},
		Source:         source,
		IdempotencyKey: idempotencyKey,
		SenderName:     message.senderName,
	})
	return err
}

// fireBusinessEvent 提交后触发业务终结回调，隔离业务 panic，回滚时不执行。
func (c *WorkflowInstanceCase) fireBusinessEvent(ctx context.Context, instance *models.WorkflowInstance, status adminv1.WorkflowInstanceStatus) {
	afterWorkflowCommit(ctx, func(ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(fmt.Sprintf("workflow business hook panic (%s): %v", instance.BusinessType, r))
			}
		}()
		c.eventRegistry.Fire(ctx, instance.TenantID, instance.ID, instance.BusinessID, instance.BusinessType, status)
	})
}

// ===================== DTO 装配辅助 =====================

// listDefinitionIDs 收集实例列表的定义 ID 去重结果。
func listDefinitionIDs(list []*models.WorkflowInstance) []int64 {
	seen := make(map[int64]struct{}, len(list))
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if _, exists := seen[item.DefinitionID]; exists {
			continue
		}
		seen[item.DefinitionID] = struct{}{}
		ids = append(ids, item.DefinitionID)
	}
	return ids
}

// listApplicantIDs 收集实例列表的申请人 ID 去重结果。
func listApplicantIDs(list []*models.WorkflowInstance) []int64 {
	seen := make(map[int64]struct{}, len(list))
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if _, exists := seen[item.ApplicantUserID]; exists {
			continue
		}
		seen[item.ApplicantUserID] = struct{}{}
		ids = append(ids, item.ApplicantUserID)
	}
	return ids
}

// collectWorkflowUserIDs 收集详情装配需要的全部用户 ID。
func collectWorkflowUserIDs(instance *models.WorkflowInstance, logs []*models.WorkflowLog, tasks []*models.WorkflowTask) []int64 {
	seen := map[int64]struct{}{instance.ApplicantUserID: {}}
	ids := []int64{instance.ApplicantUserID}
	for _, item := range logs {
		if _, exists := seen[item.OperatorUserID]; !exists {
			seen[item.OperatorUserID] = struct{}{}
			ids = append(ids, item.OperatorUserID)
		}
	}
	for _, item := range tasks {
		if _, exists := seen[item.AssigneeUserID]; !exists {
			seen[item.AssigneeUserID] = struct{}{}
			ids = append(ids, item.AssigneeUserID)
		}
	}
	return ids
}

// instanceStatusName 返回实例状态名称，用于冲突错误描述。
func instanceStatusName(status int32) string {
	switch status {
	case workflowStatusInstancePending:
		return "PENDING"
	case workflowStatusInstanceApproved:
		return "APPROVED"
	case workflowStatusInstanceRejected:
		return "REJECTED"
	case workflowStatusInstanceWithdrawn:
		return "WITHDRAWN"
	default:
		return "UNKNOWN"
	}
}

// taskStatusName 返回任务状态名称，用于冲突错误描述。
func taskStatusName(status int32) string {
	switch status {
	case workflowStatusTaskPending:
		return "PENDING"
	case workflowStatusTaskApproved:
		return "APPROVED"
	case workflowStatusTaskRejected:
		return "REJECTED"
	case workflowStatusTaskCanceled:
		return "CANCELED"
	default:
		return "UNKNOWN"
	}
}

// afterWorkflowCommit 将外部副作用延迟到最外层事务成功提交，非事务调用立即执行。
func afterWorkflowCommit(ctx context.Context, effect func(context.Context)) {
	if effects, active := ctx.Value(workflowTransactionMarker{}).(*[]func(context.Context)); active {
		*effects = append(*effects, effect)
		return
	}
	effect(ctx)
}

// workflowParticipantContext 保留租户和用户身份，由工作流显式校验申请人或办理人，替代创建人数据范围。
func workflowParticipantContext(ctx context.Context) context.Context {
	authInfo, err := auth.FromContext(ctx)
	if err != nil {
		return ctx
	}
	identity := *authInfo
	identity.DataScope = gorm.DataScopeAll
	return engine.ContextWithAuthClaims(ctx, identity.MakeAuthClaims())
}

// 工作流脚本执行、恢复和终态动作。
const (
	scriptPending     = "PENDING"
	scriptRunning     = "RUNNING"
	scriptSucceeded   = "SUCCEEDED"
	scriptFailed      = "FAILED"
	scriptCanceled    = "CANCELED"
	scriptMaxAttempts = 5
	scriptLease       = time.Minute
)

// RetryWorkflowScript 授权租户内参与者或管理人员重试失败执行，成功记录保持幂等。
func (c *WorkflowInstanceCase) RetryWorkflowScript(ctx context.Context, req *adminv1.RetryWorkflowScriptRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	ctx = workflowParticipantContext(ctx)
	var instance *models.WorkflowInstance
	instance, err = c.FindByID(ctx, req.GetInstanceId())
	if err != nil {
		return err
	}
	if instance.TenantID != authInfo.TenantId {
		return errorsx.PermissionDenied("不能重试其他租户的脚本")
	}
	if instance.ApplicantUserID != authInfo.UserId && authInfo.DataScope != gorm.DataScopeAll {
		query := c.taskRepo.Query(ctx).WorkflowTask
		var count int64
		count, err = c.taskRepo.Count(ctx, repository.Where(query.InstanceID.Eq(instance.ID)), repository.Where(query.TenantID.Eq(instance.TenantID)), repository.Where(query.AssigneeUserID.Eq(authInfo.UserId)))
		if err != nil {
			return err
		}
		if count == 0 {
			return errorsx.PermissionDenied("只有流程参与者或管理员可以重试脚本")
		}
	}
	var run *models.WorkflowScriptRun
	run, err = c.scriptRunRepo.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if run.InstanceID != instance.ID || run.TenantID != instance.TenantID {
		return errorsx.PermissionDenied("脚本执行不属于当前流程")
	}
	if run.Status == scriptSucceeded {
		return nil
	}
	if run.Status != scriptFailed {
		return errorsx.StateConflict("只有失败的脚本可以重试", "workflow_script_run", run.Status, scriptFailed)
	}
	return c.retryScriptRun(ctx, run, true, time.Now())
}

// ScanWorkflowScripts 按到期时间恢复待执行、失败或执行中断的脚本，每批最多处理一百条。
func (c *WorkflowInstanceCase) ScanWorkflowScripts(ctx context.Context, now time.Time) (int, error) {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	opts := []repository.QueryOption{
		repository.Where(query.InstanceID.Gt(0)),
		repository.Where(query.Status.In(scriptPending, scriptFailed, scriptRunning)),
		repository.Where(field.Or(query.Attempts.Lt(scriptMaxAttempts), query.Status.Eq(scriptRunning))),
		repository.Where(query.NextRetryAt.Lte(now)),
		repository.Order(query.ID.Asc()),
	}
	runs, _, err := c.scriptRunRepo.Page(ctx, 1, 100, opts...)
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, run := range runs {
		if run.Status == scriptRunning && run.LockedUntil.After(now) {
			continue
		}
		if err = c.retryScriptRun(workflowParticipantContext(ctx), run, false, now); err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}

// retryScriptRun 串行恢复计算节点，终态动作则使用持久化租约在事务外执行。
func (c *WorkflowInstanceCase) retryScriptRun(ctx context.Context, run *models.WorkflowScriptRun, manual bool, now time.Time) error {
	if run.Stage == "START" {
		return c.resumeWorkflowStart(ctx, run, manual, now)
	}
	if run.Stage != "NODE" && run.Stage != "APPROVER" {
		return c.executeTerminalScript(ctx, run, manual, now)
	}
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err := c.lockWorkflowInstance(ctx, run.TenantID, run.InstanceID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending {
			return c.cancelNodeScripts(ctx, instance)
		}
		run, err = c.scriptRunRepo.FindByID(ctx, run.ID)
		if err != nil {
			return err
		}
		if run.Status == scriptSucceeded || run.Status == scriptCanceled {
			return nil
		}
		if !manual && (run.Attempts >= scriptMaxAttempts || run.NextRetryAt.After(now)) {
			return nil
		}
		var graph *workflow.Graph
		graph, err = c.graphOfInstance(ctx, instance)
		if err != nil {
			return err
		}
		node := graph.Nodes[run.NodeID]
		if node == nil {
			return errorsx.InvalidArgument("脚本节点不存在")
		}
		var ready bool
		run, ready, err = c.runNodeScript(ctx, instance, graph, node, run.Stage, run.ScriptID, true)
		if err != nil || !ready {
			return err
		}
		if run.Stage == "APPROVER" {
			return c.enterTaskNode(ctx, instance.TenantID, instance.ID, graph, node, newWalkerState())
		}
		return c.advanceFromNode(ctx, instance.TenantID, instance.ID, graph, node.OutEdges[0].To, newWalkerState())
	})
}

// beforeSubmitScript 校验表单并执行提交钩子，拒绝或异常仅保留审计而不创建实例。
func (c *WorkflowInstanceCase) beforeSubmitScript(ctx context.Context, instance *models.WorkflowInstance, graph *workflow.Graph) (*models.WorkflowScriptRun, error) {
	var form map[string]any
	err := json.Unmarshal([]byte(instance.FormData), &form)
	if err != nil || form == nil || len(instance.FormData) > 65536 {
		return nil, errorsx.InvalidArgument("申请表单数据必须是JSON对象").WithCause(err)
	}
	if graph.Hooks.BeforeSubmit == 0 {
		return nil, nil
	}
	var run *models.WorkflowScriptRun
	run, err = c.newScriptRun(ctx, instance, graph, graph.StartNode().ID, "BEFORE_SUBMIT", graph.Hooks.BeforeSubmit, "submit:"+uuid.NewString())
	if err != nil {
		return nil, err
	}
	var output string
	output, err = c.evaluateWorkflowScript(ctx, instance, graph, run)
	scriptErr := err
	if err = c.recordScriptAttempt(ctx, run, output, scriptErr, graph.ScriptSnapshots[strconv.FormatInt(run.ScriptID, 10)].Language); err != nil {
		return nil, err
	}
	if scriptErr != nil {
		return nil, errorsx.InvalidArgument("提交校验失败: " + scriptErr.Error())
	}
	var result dto.WorkflowScriptOutput
	if err = json.Unmarshal([]byte(output), &result); err != nil {
		return nil, err
	}
	if err = c.mergeScriptVariables(ctx, instance, result.Variables, false); err != nil {
		return nil, err
	}
	return run, nil
}

// runNodeScript 在实例锁保护下执行纯计算，业务失败落库并暂停而不回滚审批。
func (c *WorkflowInstanceCase) runNodeScript(ctx context.Context, instance *models.WorkflowInstance, graph *workflow.Graph, node *workflow.Node, stage string, scriptID int64, retry bool) (*models.WorkflowScriptRun, bool, error) {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	key := stage + ":" + node.ID
	run, err := c.scriptRunRepo.Find(ctx, repository.Where(query.TenantID.Eq(instance.TenantID)), repository.Where(query.InstanceID.Eq(instance.ID)), repository.Where(query.ExecutionKey.Eq(key)))
	if err != nil && !errors.Is(err, gormdb.ErrRecordNotFound) {
		return nil, false, err
	}
	if errors.Is(err, gormdb.ErrRecordNotFound) {
		run, err = c.newScriptRun(ctx, instance, graph, node.ID, stage, scriptID, key)
		if err != nil {
			return nil, false, err
		}
	}
	if run.Status == scriptSucceeded {
		return run, true, nil
	}
	if run.Status == scriptCanceled || (!retry && run.Status == scriptFailed) {
		return run, false, nil
	}
	var output string
	output, err = c.evaluateWorkflowScript(ctx, instance, graph, run)
	scriptErr := err
	if scriptErr == nil {
		var result dto.WorkflowScriptOutput
		if err = json.Unmarshal([]byte(output), &result); err != nil {
			return nil, false, err
		}
		if err = c.mergeScriptVariables(ctx, instance, result.Variables, true); err != nil {
			return nil, false, err
		}
	}
	if err = c.recordScriptAttempt(ctx, run, output, scriptErr, graph.ScriptSnapshots[strconv.FormatInt(run.ScriptID, 10)].Language); err != nil {
		return nil, false, err
	}
	return run, scriptErr == nil, nil
}

// newScriptRun 创建持久化执行和不可变输入，脚本源码只从发布快照读取。
func (c *WorkflowInstanceCase) newScriptRun(ctx context.Context, instance *models.WorkflowInstance, graph *workflow.Graph, nodeID, stage string, scriptID int64, key string) (*models.WorkflowScriptRun, error) {
	snapshot, exists := graph.ScriptSnapshots[strconv.FormatInt(scriptID, 10)]
	if !exists || snapshot.ID != scriptID {
		return nil, errorsx.InvalidArgument("流程未发布脚本快照，请重新启用流程")
	}
	input, err := workflowScriptInput(instance, nodeID, graph.Nodes[nodeID].Type, fmt.Sprintf("workflow:%d:%d:%s", instance.TenantID, instance.ID, key))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	run := &models.WorkflowScriptRun{TenantID: instance.TenantID, InstanceID: instance.ID, NodeID: nodeID, Stage: stage, ScriptID: scriptID, ScriptName: snapshot.Name, Version: snapshot.Version, ExecutionKey: key, InputJSON: input, Status: scriptPending, NextRetryAt: now, LockedUntil: now}
	if err = c.scriptRunRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// evaluateWorkflowScript 在隔离运行时执行可信快照并校验各阶段输出。
func (c *WorkflowInstanceCase) evaluateWorkflowScript(ctx context.Context, instance *models.WorkflowInstance, graph *workflow.Graph, run *models.WorkflowScriptRun) (output string, err error) {
	started := time.Now()
	defer func() {
		run.DurationMs = time.Since(started).Milliseconds()
		if value := recover(); value != nil {
			err = fmt.Errorf("脚本执行异常: %v", value)
		}
	}()
	snapshot, exists := graph.ScriptSnapshots[strconv.FormatInt(run.ScriptID, 10)]
	if !exists || snapshot.Version != run.Version {
		return "", fmt.Errorf("脚本版本快照缺失")
	}
	if !workflow.ScriptLanguageSupported(snapshot.Language) {
		return "", fmt.Errorf("流程仅支持 JavaScript 脚本")
	}
	var input map[string]any
	input, err = scripting.ParseContext(run.InputJSON)
	if err != nil {
		return "", err
	}
	var value any
	value, err = c.runtime.ExecuteWorkflow(ctx, snapshot.Language, snapshot.Source, input)
	if err != nil {
		return "", err
	}
	output, err = scripting.MarshalOutput(value)
	if err != nil {
		return "", err
	}
	var result dto.WorkflowScriptOutput
	if err = json.Unmarshal([]byte(output), &result); err != nil {
		return output, fmt.Errorf("脚本必须返回JSON对象: %w", err)
	}
	if output == "null" {
		return output, fmt.Errorf("脚本必须返回JSON对象")
	}
	if run.Stage == "BEFORE_SUBMIT" {
		if result.Allowed == nil {
			return output, fmt.Errorf("提交校验必须返回allowed布尔值")
		}
		if !*result.Allowed {
			if result.Message == "" {
				result.Message = "申请未通过脚本校验"
			}
			return output, fmt.Errorf("%s", result.Message)
		}
	}
	if run.Stage == "NODE" && result.Variables == nil {
		return output, fmt.Errorf("脚本节点必须返回variables对象")
	}
	if run.Stage == "APPROVER" {
		if len(result.UserIDs) == 0 || len(result.UserIDs) > 100 {
			return output, fmt.Errorf("审批人脚本必须返回1至100个用户ID")
		}
		node := &workflow.Node{ID: run.NodeID, Approvers: make([]workflow.Approver, 0, len(result.UserIDs))}
		for _, id := range result.UserIDs {
			if id <= 0 {
				return output, fmt.Errorf("审批人ID必须为正整数")
			}
			node.Approvers = append(node.Approvers, workflow.Approver{Type: workflow.ApproverTypeUser, ID: id})
		}
		result.UserIDs, err = c.resolveApprovers(ctx, instance.TenantID, instance.ApplicantUserID, node)
		if err != nil {
			return output, err
		}
		for _, id := range result.UserIDs {
			if err = c.ensureUserActive(ctx, instance.TenantID, id); err != nil {
				return output, err
			}
		}
		output, err = scripting.MarshalOutput(result)
		if err != nil {
			return output, err
		}
	}
	if result.Variables != nil {
		var merged map[string]any
		merged, err = scripting.ParseContext(instance.Variables)
		if err != nil {
			return output, err
		}
		for key, value := range result.Variables {
			merged[key] = value
		}
		var variables string
		variables, err = scripting.MarshalOutput(merged)
		if err != nil {
			return output, err
		}
		next := *instance
		next.Variables = variables
		next.ID = math.MaxInt64
		// 节点 ID 至多 100 字符，每字符最坏转义为 6 字节；执行键额外包含阶段与实例编号。
		if _, err = workflowScriptInput(&next, strings.Repeat("<", 100), workflow.NodeTypeScript, strings.Repeat("<", 200)); err != nil {
			return output, err
		}
	}
	if len(result.Actions) > 20 {
		return output, fmt.Errorf("单次最多执行20个业务动作")
	}
	if len(result.Actions) > 0 && (run.Stage == "NODE" || run.Stage == "APPROVER" || run.Stage == "BEFORE_SUBMIT") {
		return output, fmt.Errorf("业务动作只允许在流程终结后执行")
	}
	for _, action := range result.Actions {
		if action.Type == "" || action.Payload == nil {
			return output, fmt.Errorf("业务动作必须包含type和payload对象")
		}
	}
	return output, nil
}

// recordScriptAttempt 更新执行状态并同步写入脚本管理的统一审计日志。
func (c *WorkflowInstanceCase) recordScriptAttempt(ctx context.Context, run *models.WorkflowScriptRun, output string, executionErr error, language string) error {
	run.Attempts++
	run.OutputJSON = output
	run.Status = scriptSucceeded
	run.LastError = ""
	if executionErr != nil {
		run.Status = scriptFailed
		run.LastError = executionErr.Error()
		run.NextRetryAt = time.Now().Add(time.Minute * time.Duration(1<<min(run.Attempts-1, 8)))
	}
	if err := c.saveScriptRun(ctx, run); err != nil {
		return err
	}
	return c.auditScriptRun(ctx, run, language)
}

// auditScriptRun 将工作流执行关联到共享脚本日志，可通过钩子点追踪实例和执行记录。
func (c *WorkflowInstanceCase) auditScriptRun(ctx context.Context, run *models.WorkflowScriptRun, language string) error {
	success := int32(0)
	if run.Status == scriptSucceeded {
		success = 1
	}
	return c.scriptLogRepo.Create(ctx, &models.BaseScriptLog{ScriptID: run.ScriptID, ScriptName: run.ScriptName, Language: language, TriggerType: "WORKFLOW", HookPoint: fmt.Sprintf("workflow/%d/%d/%s", run.InstanceID, run.ID, run.Stage), Version: run.Version, Success: success, DurationMs: run.DurationMs, Error: run.LastError, TenantID: run.TenantID})
}

// saveScriptRun 显式保存包括零值在内的执行状态，避免成功后遗留旧错误或动作进度。
func (c *WorkflowInstanceCase) saveScriptRun(ctx context.Context, run *models.WorkflowScriptRun) error {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	return c.scriptRunRepo.Update(ctx, run, repository.Where(query.ID.Eq(run.ID)), repository.Where(query.TenantID.Eq(run.TenantID)), repository.Select(query.InstanceID, query.Status, query.Attempts, query.InputJSON, query.OutputJSON, query.LastError, query.DurationMs, query.NextRetryAt, query.LockedUntil, query.EffectsCompleted))
}

// saveClaimedScriptRun 仅允许当前租约持有者写入进度，阻止过期执行覆盖新尝试的结果。
func (c *WorkflowInstanceCase) saveClaimedScriptRun(ctx context.Context, run *models.WorkflowScriptRun) error {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	info, err := query.WithContext(ctx).Where(query.ID.Eq(run.ID), query.TenantID.Eq(run.TenantID), query.Attempts.Eq(run.Attempts), query.Status.Eq(scriptRunning)).Select(query.Status, query.OutputJSON, query.LastError, query.DurationMs, query.NextRetryAt, query.EffectsCompleted).Updates(run)
	if err != nil {
		return err
	}
	if info.RowsAffected == 0 {
		return errorsx.StateConflict("执行租约已被其他任务接管", "workflow_script_run", scriptRunning, "CURRENT_LEASE")
	}
	return nil
}

// mergeScriptVariables 合并当前持久化变量，保留并行分支已经写入的其他键。
func (c *WorkflowInstanceCase) mergeScriptVariables(ctx context.Context, instance *models.WorkflowInstance, variables map[string]any, persist bool) error {
	if variables == nil {
		return nil
	}
	var err error
	if persist {
		var current *models.WorkflowInstance
		current, err = c.FindByID(ctx, instance.ID)
		if err != nil {
			return err
		}
		instance.Variables = current.Variables
	}
	var merged map[string]any
	merged, err = scripting.ParseContext(instance.Variables)
	if err != nil {
		return err
	}
	for key, value := range variables {
		merged[key] = value
	}
	instance.Variables, err = scripting.MarshalOutput(merged)
	if err != nil {
		return err
	}
	if !persist {
		return nil
	}
	query := c.Query(ctx).WorkflowInstance
	return c.Update(ctx, &models.WorkflowInstance{Variables: instance.Variables}, repository.Where(query.ID.Eq(instance.ID)), repository.Where(query.TenantID.Eq(instance.TenantID)), repository.Select(query.Variables))
}

// enqueueTerminalScript 在审批终态事务内登记后续处理，提交后尝试执行，崩溃后由扫描恢复。
func (c *WorkflowInstanceCase) enqueueTerminalScript(ctx context.Context, instance *models.WorkflowInstance) error {
	err := c.cancelNodeScripts(ctx, instance)
	if err != nil {
		return err
	}
	var graph *workflow.Graph
	graph, err = c.graphOfInstance(ctx, instance)
	if err != nil {
		return err
	}
	var stage string
	var scriptID int64
	switch instance.Status {
	case workflowStatusInstanceApproved:
		stage = "APPROVED"
		scriptID = graph.Hooks.Approved
	case workflowStatusInstanceRejected:
		stage = "REJECTED"
		scriptID = graph.Hooks.Rejected
	case workflowStatusInstanceWithdrawn:
		stage = "WITHDRAWN"
		scriptID = graph.Hooks.Withdrawn
	default:
		return nil
	}
	if scriptID == 0 {
		return nil
	}
	var run *models.WorkflowScriptRun
	var endID string
	for _, node := range graph.Nodes {
		if node.Type == workflow.NodeTypeEnd {
			endID = node.ID
			break
		}
	}
	run, err = c.newScriptRun(ctx, instance, graph, endID, stage, scriptID, stage+":"+endID)
	if err != nil {
		return err
	}
	afterWorkflowCommit(ctx, func(ctx context.Context) {
		err = c.executeTerminalScript(context.WithoutCancel(ctx), run, false, time.Now())
		if err != nil {
			log.Error(fmt.Sprintf("workflow script dispatch failed: run=%d err=%v", run.ID, err))
		}
	})
	return nil
}

// cancelNodeScripts 终结实例时取消尚未完成的计算，阻止重试重新创建审批任务。
func (c *WorkflowInstanceCase) cancelNodeScripts(ctx context.Context, instance *models.WorkflowInstance) error {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	return c.scriptRunRepo.Update(ctx, &models.WorkflowScriptRun{Status: scriptCanceled}, repository.Where(query.TenantID.Eq(instance.TenantID)), repository.Where(query.InstanceID.Eq(instance.ID)), repository.Where(query.Stage.In("START", "NODE", "APPROVER")), repository.Where(query.Status.In(scriptPending, scriptRunning, scriptFailed)))
}

// pendingNodeScripts 统计阻塞中的脚本，避免无人工待办时提前通过。
func (c *WorkflowInstanceCase) pendingNodeScripts(ctx context.Context, tenantID, instanceID int64) (int64, error) {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	return c.scriptRunRepo.Count(ctx, repository.Where(query.TenantID.Eq(tenantID)), repository.Where(query.InstanceID.Eq(instanceID)), repository.Where(query.Stage.In("NODE", "APPROVER")), repository.Where(query.Status.In(scriptPending, scriptRunning, scriptFailed)))
}

// executeTerminalScript 原子领取持久化租约，在事务外运行脚本与业务动作并保存进度。
func (c *WorkflowInstanceCase) executeTerminalScript(ctx context.Context, run *models.WorkflowScriptRun, manual bool, now time.Time) error {
	var instance *models.WorkflowInstance
	var claimed bool
	var err error
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err = c.lockWorkflowInstance(ctx, run.TenantID, run.InstanceID)
		if err != nil {
			return err
		}
		run, err = c.scriptRunRepo.FindByID(ctx, run.ID)
		if err != nil {
			return err
		}
		if run.Status == scriptSucceeded || run.Status == scriptCanceled {
			return nil
		}
		if run.Status == scriptRunning && run.LockedUntil.After(now) {
			return nil
		}
		if !manual && run.Attempts >= scriptMaxAttempts {
			if run.Status == scriptRunning {
				run.Status = scriptFailed
				run.LastError = "执行租约已过期，自动重试次数已耗尽，请人工重试"
				return c.saveScriptRun(ctx, run)
			}
			return nil
		}
		if !manual && run.NextRetryAt.After(now) {
			return nil
		}
		if instance.Status == workflowStatusInstancePending {
			return errorsx.InvalidArgument("流程尚未终结")
		}
		run.Status = scriptRunning
		run.LockedUntil = now.Add(scriptLease)
		run.NextRetryAt = run.LockedUntil
		run.Attempts++
		claimed = true
		return c.saveScriptRun(ctx, run)
	})
	if err != nil || !claimed {
		return err
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	started := time.Now()
	var graph *workflow.Graph
	graph, err = c.graphOfInstance(ctx, instance)
	if err == nil && run.OutputJSON == "" {
		run.OutputJSON, err = c.evaluateWorkflowScript(ctx, instance, graph, run)
		// 错误输出不能作为重试缓存；动作执行前固定成功输出，避免重试重新生成动作。
		if err != nil {
			run.OutputJSON = ""
		} else {
			err = c.saveClaimedScriptRun(ctx, run)
		}
	}
	if err == nil {
		err = c.executeScriptActions(ctx, instance, run)
	}
	executionErr := err
	run.DurationMs = time.Since(started).Milliseconds()
	run.Status = scriptSucceeded
	run.LastError = ""
	if executionErr != nil {
		run.Status = scriptFailed
		run.LastError = executionErr.Error()
		run.NextRetryAt = time.Now().Add(time.Minute * time.Duration(1<<min(run.Attempts-1, 8)))
	}
	// 请求或动作超时后仍需释放租约并记录失败，使用独立持久化上下文。
	persistCtx := context.WithoutCancel(ctx)
	return c.withWorkflowTransaction(persistCtx, func(ctx context.Context) error {
		if err = c.saveClaimedScriptRun(ctx, run); err != nil {
			return err
		}
		language := ""
		if graph != nil {
			language = graph.ScriptSnapshots[strconv.FormatInt(run.ScriptID, 10)].Language
		}
		return c.auditScriptRun(ctx, run, language)
	})
}

// executeScriptActions 依次执行白名单业务动作，稳定幂等键和落库进度使失败可恢复。
func (c *WorkflowInstanceCase) executeScriptActions(ctx context.Context, instance *models.WorkflowInstance, run *models.WorkflowScriptRun) error {
	var output dto.WorkflowScriptOutput
	err := json.Unmarshal([]byte(run.OutputJSON), &output)
	if err != nil {
		return err
	}
	for index := int(run.EffectsCompleted); index < len(output.Actions); index++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		action := output.Actions[index]
		actionInput := pkgWorkflow.ActionContext{TenantID: instance.TenantID, InstanceID: instance.ID, BusinessID: instance.BusinessID, BusinessType: instance.BusinessType, Status: instance.Status, IdempotencyKey: fmt.Sprintf("workflow:%d:%d:%s:action:%d", run.TenantID, run.InstanceID, run.ExecutionKey, index), Payload: action.Payload}
		if action.Type == pkgWorkflow.ActionNotify {
			err = c.sendScriptNotification(ctx, actionInput)
		} else {
			err = c.eventRegistry.ExecuteAction(ctx, action.Type, actionInput)
		}
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		run.EffectsCompleted = int32(index + 1)
		if err = c.saveClaimedScriptRun(ctx, run); err != nil {
			return err
		}
	}
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		var current *models.WorkflowInstance
		current, err = c.lockWorkflowInstance(ctx, instance.TenantID, instance.ID)
		if err != nil {
			return err
		}
		return c.mergeScriptVariables(ctx, current, output.Variables, true)
	})
}

// sendScriptNotification 通过现有站内信模块执行幂等通知，受众限定为当前租户启用用户。
func (c *WorkflowInstanceCase) sendScriptNotification(ctx context.Context, input pkgWorkflow.ActionContext) error {
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return err
	}
	var message dto.WorkflowScriptNotification
	if err = json.Unmarshal(payload, &message); err != nil {
		return errorsx.InvalidArgument("通知动作参数无效").WithCause(err)
	}
	if message.Title == "" || utf8.RuneCountInString(message.Title) > 200 || message.Content == "" || len(message.Content) > 65536 || len(message.UserIDs) == 0 || len(message.UserIDs) > 100 {
		return errorsx.InvalidArgument("通知动作需要标题、内容和1至100个本租户用户")
	}
	audiences := make([]notification.Audience, 0, len(message.UserIDs))
	seen := make(map[int64]bool, len(message.UserIDs))
	for _, userID := range message.UserIDs {
		if seen[userID] {
			continue
		}
		if err = c.ensureUserActive(ctx, input.TenantID, userID); err != nil {
			return err
		}
		seen[userID] = true
		audiences = append(audiences, notification.Audience{Type: basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER, ID: userID})
	}
	key := fmt.Sprintf("workflow:%x", sha256.Sum256([]byte(input.IdempotencyKey)))
	_, err = notification.Publish(ctx, notification.Message{TenantID: input.TenantID, CategoryCode: workflowMessageCategory, Title: message.Title, Content: message.Content, ContentFormat: basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT, Priority: basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL, Audiences: audiences, Source: workflowMessageSourceByBusinessType(input.BusinessType), IdempotencyKey: key, SenderName: c.catalog.Localize(biz.LocaleFromContext(ctx), "zh-CN", "system.notification.sender.system", nil, "系统")})
	return err
}

// listScriptRuns 按实例装配可审计执行记录，避免向管理端暴露发布脚本源码。
func (c *WorkflowInstanceCase) listScriptRuns(ctx context.Context, tenantID, instanceID int64) ([]*adminv1.WorkflowScriptRun, error) {
	query := c.scriptRunRepo.Query(ctx).WorkflowScriptRun
	runs, err := c.scriptRunRepo.List(ctx, repository.Where(query.TenantID.Eq(tenantID)), repository.Where(query.InstanceID.Eq(instanceID)), repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.WorkflowScriptRun, 0, len(runs))
	for _, run := range runs {
		nextRetry := ""
		if run.Status != scriptSucceeded && run.Status != scriptCanceled && run.Attempts < scriptMaxAttempts && !run.NextRetryAt.IsZero() {
			nextRetry = run.NextRetryAt.Format(time.DateTime)
		}
		result = append(result, &adminv1.WorkflowScriptRun{Id: run.ID, InstanceId: run.InstanceID, NodeId: run.NodeID, Stage: run.Stage, ScriptId: run.ScriptID, ScriptName: run.ScriptName, Version: run.Version, Status: run.Status, Attempts: run.Attempts, InputJson: run.InputJSON, OutputJson: run.OutputJSON, LastError: run.LastError, DurationMs: run.DurationMs, NextRetryAt: nextRetry, CreatedAt: run.CreatedAt.Format(time.DateTime), UpdatedAt: run.UpdatedAt.Format(time.DateTime)})
	}
	return result, nil
}

// workflowScriptInput 构造独立JSON输入，脚本无法修改引擎持有的表单与变量对象。
func workflowScriptInput(instance *models.WorkflowInstance, nodeID, nodeType, key string) (string, error) {
	form, err := scripting.ParseContext(instance.FormData)
	if err != nil {
		return "", err
	}
	var variables map[string]any
	variables, err = scripting.ParseContext(instance.Variables)
	if err != nil {
		return "", err
	}
	return scripting.MarshalOutput(map[string]any{"form": form, "variables": variables, "workflow": map[string]any{"instance_id": instance.ID, "tenant_id": instance.TenantID, "applicant_user_id": instance.ApplicantUserID, "business_type": instance.BusinessType, "business_id": instance.BusinessID, "status": instance.Status}, "node": map[string]any{"id": nodeID, "type": nodeType}, "idempotency_key": key})
}

// 工作流启动关联确认与恢复。
func (c *WorkflowInstanceCase) enqueueWorkflowStart(ctx context.Context, instance *models.WorkflowInstance, nodeID string, bindingRequired bool) (*models.WorkflowScriptRun, error) {
	input, err := json.Marshal(dto.WorkflowStartInput{BindingRequired: bindingRequired})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	run := &models.WorkflowScriptRun{TenantID: instance.TenantID, InstanceID: instance.ID, NodeID: nodeID, Stage: "START", ExecutionKey: "START", ScriptName: "流程启动", Status: scriptPending, InputJSON: string(input), NextRetryAt: now, LockedUntil: now}
	if bindingRequired {
		run.Status = scriptRunning
		run.LockedUntil = now.Add(scriptLease)
		run.NextRetryAt = run.LockedUntil
	}
	if err = c.scriptRunRepo.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// confirmWorkflowBinding 持久化宿主成功回写结果，再允许首次推进或扫描恢复。
func (c *WorkflowInstanceCase) confirmWorkflowBinding(ctx context.Context, run *models.WorkflowScriptRun) error {
	return c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		instance, err := c.lockWorkflowInstance(ctx, run.TenantID, run.InstanceID)
		if err != nil {
			return err
		}
		run, err = c.scriptRunRepo.FindByID(ctx, run.ID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending || run.Status == scriptSucceeded || run.Status == scriptCanceled {
			return nil
		}
		var input dto.WorkflowStartInput
		if err = json.Unmarshal([]byte(run.InputJSON), &input); err != nil {
			return err
		}
		input.BindingConfirmed = true
		var raw []byte
		raw, err = json.Marshal(input)
		if err != nil {
			return err
		}
		run.InputJSON = string(raw)
		run.Status = scriptPending
		run.NextRetryAt = time.Now()
		return c.saveScriptRun(ctx, run)
	})
}

// resumeWorkflowStart 在实例锁内幂等消费启动记录，关联不确定时通过宿主持久化状态核验。
func (c *WorkflowInstanceCase) resumeWorkflowStart(ctx context.Context, run *models.WorkflowScriptRun, manual bool, now time.Time) error {
	var err error
	err = c.withWorkflowTransaction(ctx, func(ctx context.Context) error {
		var instance *models.WorkflowInstance
		instance, err = c.lockWorkflowInstance(ctx, run.TenantID, run.InstanceID)
		if err != nil {
			return err
		}
		run, err = c.scriptRunRepo.FindByID(ctx, run.ID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending {
			return c.cancelNodeScripts(ctx, instance)
		}
		if run.Status == scriptSucceeded || run.Status == scriptCanceled || (!manual && (run.Attempts >= scriptMaxAttempts || run.NextRetryAt.After(now))) {
			return nil
		}
		var input dto.WorkflowStartInput
		if err = json.Unmarshal([]byte(run.InputJSON), &input); err != nil {
			return err
		}
		if input.BindingRequired && !input.BindingConfirmed {
			verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			input.BindingConfirmed, err = c.eventRegistry.VerifyBinding(verifyCtx, instance.BusinessType, instance.TenantID, instance.ID, instance.BusinessID)
			if err != nil {
				return err
			}
			if !input.BindingConfirmed {
				return fmt.Errorf("业务单据关联尚未确认，等待核验后重试")
			}
		}
		var graph *workflow.Graph
		graph, err = c.graphOfInstance(ctx, instance)
		if err != nil {
			return err
		}
		var raw []byte
		raw, err = json.Marshal(input)
		if err != nil {
			return err
		}
		run.InputJSON = string(raw)
		run.Status = scriptSucceeded
		run.LastError = ""
		run.Attempts++
		// 成功标记和首批待办同事务提交，终态回调也只能在该事务提交后执行。
		if err = c.saveScriptRun(ctx, run); err != nil {
			return err
		}
		return c.advanceFromNode(ctx, instance.TenantID, instance.ID, graph, graph.StartNode().ID, newWalkerState())
	})
	if err == nil {
		return nil
	}
	executionErr := err
	// 推进事务回滚后保留失败恢复点；基础设施恢复或人工重试仍消费同一启动记录。
	return c.withWorkflowTransaction(context.WithoutCancel(ctx), func(ctx context.Context) error {
		var instance *models.WorkflowInstance
		instance, err = c.lockWorkflowInstance(ctx, run.TenantID, run.InstanceID)
		if err != nil {
			return err
		}
		run, err = c.scriptRunRepo.FindByID(ctx, run.ID)
		if err != nil {
			return err
		}
		if instance.Status != workflowStatusInstancePending || run.Status == scriptSucceeded || run.Status == scriptCanceled {
			return nil
		}
		run.Status = scriptFailed
		run.LastError = executionErr.Error()
		run.Attempts++
		run.NextRetryAt = time.Now().Add(time.Minute * time.Duration(1<<min(run.Attempts-1, 8)))
		return c.saveScriptRun(ctx, run)
	})
}

// workflowNotificationMessage 保存工作流站内信的持久化国际化标记。
type workflowNotificationMessage struct {
	title      string
	content    string
	senderName string
}

// workflowMessageAction 描述管理端审批深链参数。
type workflowMessageAction struct {
	Admin workflowMessageAdminAction `json:"admin"`
}

// workflowMessageAdminAction 描述管理端审批页面定位信息。
type workflowMessageAdminAction struct {
	Route      string `json:"route"`
	Source     string `json:"source"`
	TaskID     string `json:"task_id,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	ListType   string `json:"list_type"`
}

// workflowMessageSourceByBusinessType 将工作流业务类型转换为消息来源。
func workflowMessageSourceByBusinessType(businessType string) string {
	switch businessType {
	case baseTenantProjectCreateBusinessType:
		return "project"
	case baseTenantCreateBusinessType:
		return "tenant"
	default:
		return workflowMessageSource
	}
}

// workflowMessageActionParams 生成管理端审批消息的动作参数。
func workflowMessageActionParams(source string, taskID, instanceID int64) string {
	adminAction := workflowMessageAdminAction{Route: "WorkflowApproval", Source: source}
	if taskID > 0 {
		adminAction.TaskID = strconv.FormatInt(taskID, 10)
		adminAction.ListType = "PENDING"
	} else if instanceID > 0 {
		adminAction.InstanceID = strconv.FormatInt(instanceID, 10)
		adminAction.ListType = "SUBMITTED"
	}
	data, err := json.Marshal(workflowMessageAction{Admin: adminAction})
	if err != nil {
		return "{}"
	}
	return string(data)
}

// workflowNotification 生成工作流站内信的稳定国际化标记，由收件端按当前语言渲染。
func (c *WorkflowInstanceCase) workflowNotification(_ context.Context, titleKey, contentKey string) workflowNotificationMessage {
	return workflowNotificationMessage{
		title:      admini18n.EncodeMessage(titleKey, nil),
		content:    admini18n.EncodeMessage(contentKey, nil),
		senderName: admini18n.EncodeMessage("system.notification.sender.system", nil),
	}
}
