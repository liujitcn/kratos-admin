package biz

import (
	"context"
	"sort"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/workflow"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// WorkflowTaskCase 工作流审批任务业务实例。
type WorkflowTaskCase struct {
	*biz.BaseCase
	*data.WorkflowTaskRepository
	instanceCase   *WorkflowInstanceCase
	instanceRepo   *data.WorkflowInstanceRepository
	definitionRepo *data.WorkflowDefinitionRepository
	baseUserRepo   *data.BaseUserRepository
	taskMapper     *mapper.CopierMapper[adminv1.WorkflowTask, models.WorkflowTask]
}

// NewWorkflowTaskCase 创建工作流审批任务业务实例。
func NewWorkflowTaskCase(
	baseCase *biz.BaseCase,
	taskRepo *data.WorkflowTaskRepository,
	instanceCase *WorkflowInstanceCase,
	instanceRepo *data.WorkflowInstanceRepository,
	definitionRepo *data.WorkflowDefinitionRepository,
	baseUserRepo *data.BaseUserRepository,
) *WorkflowTaskCase {
	return &WorkflowTaskCase{
		BaseCase:               baseCase,
		WorkflowTaskRepository: taskRepo,
		instanceCase:           instanceCase,
		instanceRepo:           instanceRepo,
		definitionRepo:         definitionRepo,
		baseUserRepo:           baseUserRepo,
		taskMapper:             mapper.NewCopierMapper[adminv1.WorkflowTask, models.WorkflowTask](),
	}
}

// PageMyWorkflowTask 按当前租户内的办理人或申请人查询任务，不受记录创建人范围限制。
func (c *WorkflowTaskCase) PageMyWorkflowTask(ctx context.Context, req *adminv1.PageMyWorkflowTaskRequest) (*adminv1.PageMyWorkflowTaskResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	ctx = workflowParticipantContext(ctx)

	var items []*adminv1.MyWorkflowTaskItem
	var total int64
	switch req.GetListType() {
	case adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_UNSPECIFIED:
		items, total, err = c.pageMyAll(ctx, authInfo.TenantId, authInfo.UserId, req)
	case adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_PENDING:
		items, total, err = c.pageMyPending(ctx, authInfo.TenantId, authInfo.UserId, req)
	case adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_DONE:
		items, total, err = c.pageMyDone(ctx, authInfo.TenantId, authInfo.UserId, req)
	case adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_SUBMITTED:
		items, total, err = c.pageMySubmitted(ctx, authInfo.TenantId, authInfo.UserId, req)
	default:
		return nil, errorsx.InvalidArgument("列表类型无效")
	}
	if err != nil {
		return nil, err
	}
	return &adminv1.PageMyWorkflowTaskResponse{Items: items, Total: int32(total)}, nil
}

// pageMyAll 分页查询当前用户全部审批视角，并按最新活动时间合并排序。
func (c *WorkflowTaskCase) pageMyAll(ctx context.Context, tenantID, userID int64, req *adminv1.PageMyWorkflowTaskRequest) ([]*adminv1.MyWorkflowTaskItem, int64, error) {
	pageNum := req.GetPageNum()
	if pageNum < 1 {
		pageNum = 1
	}
	pageSize := req.GetPageSize()
	if pageSize < 1 {
		pageSize = 10
	}
	limit := pageNum * pageSize
	pageReq := *req
	pageReq.PageNum = 1
	pageReq.PageSize = limit

	pendingReq := pageReq
	pendingReq.ListType = adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_PENDING
	pendingItems, pendingTotal, err := c.pageMyPending(ctx, tenantID, userID, &pendingReq)
	if err != nil {
		return nil, 0, err
	}
	doneReq := pageReq
	doneReq.ListType = adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_DONE
	var doneItems []*adminv1.MyWorkflowTaskItem
	var doneTotal int64
	doneItems, doneTotal, err = c.pageMyDone(ctx, tenantID, userID, &doneReq)
	if err != nil {
		return nil, 0, err
	}
	submittedReq := pageReq
	submittedReq.ListType = adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_SUBMITTED
	var submittedItems []*adminv1.MyWorkflowTaskItem
	var submittedTotal int64
	submittedItems, submittedTotal, err = c.pageMySubmitted(ctx, tenantID, userID, &submittedReq)
	if err != nil {
		return nil, 0, err
	}
	items := append(append(pendingItems, doneItems...), submittedItems...)
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].TaskCreatedAt
		if left == "" {
			left = items[i].SubmitAt
		}
		right := items[j].TaskCreatedAt
		if right == "" {
			right = items[j].SubmitAt
		}
		return left > right
	})
	offset := (pageNum - 1) * pageSize
	total := pendingTotal + doneTotal + submittedTotal
	if offset >= int64(len(items)) {
		return make([]*adminv1.MyWorkflowTaskItem, 0), total, nil
	}
	end := offset + pageSize
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	return items[offset:end], total, nil
}

// pageMyPending 分页查询待我审批的任务。
func (c *WorkflowTaskCase) pageMyPending(ctx context.Context, tenantID, userID int64, req *adminv1.PageMyWorkflowTaskRequest) ([]*adminv1.MyWorkflowTaskItem, int64, error) {
	taskQuery := c.Query(ctx).WorkflowTask
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(taskQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(taskQuery.AssigneeUserID.Eq(userID)))
	opts = append(opts, repository.Where(taskQuery.Status.Eq(workflowStatusTaskPending)))
	var err error
	if keyword := req.GetKeyword(); keyword != "" {
		var instanceIDs []int64
		instanceIDs, err = c.instanceIDsByKeyword(ctx, tenantID, keyword)
		if err != nil {
			return nil, 0, err
		}
		if len(instanceIDs) == 0 {
			return make([]*adminv1.MyWorkflowTaskItem, 0), 0, nil
		}
		opts = append(opts, repository.Where(taskQuery.InstanceID.In(instanceIDs...)))
	}
	opts = append(opts, repository.Order(taskQuery.CreatedAt.Desc()))
	var tasks []*models.WorkflowTask
	var total int64
	tasks, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, 0, err
	}
	var items []*adminv1.MyWorkflowTaskItem
	items, err = c.assembleTaskItems(ctx, tasks, adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_PENDING)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// pageMyDone 分页查询我已审批的任务，按任务办结时间倒序。
func (c *WorkflowTaskCase) pageMyDone(ctx context.Context, tenantID, userID int64, req *adminv1.PageMyWorkflowTaskRequest) ([]*adminv1.MyWorkflowTaskItem, int64, error) {
	taskQuery := c.Query(ctx).WorkflowTask
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(taskQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(taskQuery.AssigneeUserID.Eq(userID)))
	opts = append(opts, repository.Where(taskQuery.Status.In(workflowStatusTaskApproved, workflowStatusTaskRejected)))
	var err error
	if keyword := req.GetKeyword(); keyword != "" {
		var instanceIDs []int64
		instanceIDs, err = c.instanceIDsByKeyword(ctx, tenantID, keyword)
		if err != nil {
			return nil, 0, err
		}
		if len(instanceIDs) == 0 {
			return make([]*adminv1.MyWorkflowTaskItem, 0), 0, nil
		}
		opts = append(opts, repository.Where(taskQuery.InstanceID.In(instanceIDs...)))
	}
	opts = append(opts, repository.Order(taskQuery.UpdatedAt.Desc()))
	var tasks []*models.WorkflowTask
	var total int64
	tasks, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, 0, err
	}
	var items []*adminv1.MyWorkflowTaskItem
	items, err = c.assembleTaskItems(ctx, tasks, adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_DONE)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// pageMySubmitted 分页查询我发起的申请。
func (c *WorkflowTaskCase) pageMySubmitted(ctx context.Context, tenantID, userID int64, req *adminv1.PageMyWorkflowTaskRequest) ([]*adminv1.MyWorkflowTaskItem, int64, error) {
	instanceQuery := c.instanceRepo.Query(ctx).WorkflowInstance
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(instanceQuery.TenantID.Eq(tenantID)))
	opts = append(opts, repository.Where(instanceQuery.ApplicantUserID.Eq(userID)))
	var err error
	if keyword := req.GetKeyword(); keyword != "" {
		var instanceIDs []int64
		instanceIDs, err = c.instanceIDsByKeyword(ctx, tenantID, keyword)
		if err != nil {
			return nil, 0, err
		}
		if len(instanceIDs) == 0 {
			return make([]*adminv1.MyWorkflowTaskItem, 0), 0, nil
		}
		opts = append(opts, repository.Where(instanceQuery.ID.In(instanceIDs...)))
	}
	opts = append(opts, repository.Order(instanceQuery.CreatedAt.Desc()))
	var instances []*models.WorkflowInstance
	var total int64
	instances, total, err = c.instanceRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, 0, err
	}
	var items []*adminv1.MyWorkflowTaskItem
	items, err = c.assembleSubmittedItems(ctx, instances)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// instanceIDsByKeyword 按流程名称或编码关键词查询租户内的流程实例 ID。
func (c *WorkflowTaskCase) instanceIDsByKeyword(ctx context.Context, tenantID int64, keyword string) ([]int64, error) {
	definitionQuery := c.definitionRepo.Query(ctx).WorkflowDefinition
	pattern := "%" + keyword + "%"
	definitions, err := c.definitionRepo.List(ctx,
		repository.Select(definitionQuery.ID),
		repository.Where(definitionQuery.TenantID.Eq(tenantID)),
		repository.Where(field.Or(definitionQuery.Code.Like(pattern), definitionQuery.Name.Like(pattern))),
	)
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 {
		return []int64{}, nil
	}
	definitionIDs := make([]int64, 0, len(definitions))
	for _, definition := range definitions {
		definitionIDs = append(definitionIDs, definition.ID)
	}

	instanceQuery := c.instanceRepo.Query(ctx).WorkflowInstance
	var instances []*models.WorkflowInstance
	instances, err = c.instanceRepo.List(ctx,
		repository.Select(instanceQuery.ID),
		repository.Where(instanceQuery.TenantID.Eq(tenantID)),
		repository.Where(instanceQuery.DefinitionID.In(definitionIDs...)),
	)
	if err != nil {
		return nil, err
	}
	instanceIDs := make([]int64, 0, len(instances))
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.ID)
	}
	return instanceIDs, nil
}

// GetWorkflowTask 按租户和办理人授权后查询完整任务详情，不受申请人的创建人范围限制。
func (c *WorkflowTaskCase) GetWorkflowTask(ctx context.Context, id int64) (*adminv1.GetWorkflowTaskResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	ctx = workflowParticipantContext(ctx)

	var task *models.WorkflowTask
	task, err = c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.TenantID != authInfo.TenantId || task.AssigneeUserID != authInfo.UserId {
		return nil, errorsx.PermissionDenied("不是您的审批任务")
	}

	// 复用实例详情装配：已合并流程定义、申请人与日志操作人昵称。
	var detail *adminv1.GetWorkflowInstanceResponse
	detail, err = c.instanceCase.GetWorkflowInstance(ctx, task.InstanceID)
	if err != nil {
		return nil, err
	}
	taskDto := c.taskMapper.ToDTO(task)
	taskDto.AssigneeName = detail.Instance.ApplicantName
	if task.AssigneeUserID != detail.Instance.ApplicantUserId {
		taskDto.AssigneeName = c.nickNameOf(ctx, task.AssigneeUserID)
	}
	return &adminv1.GetWorkflowTaskResponse{Task: taskDto, Instance: detail.Instance, Logs: detail.Logs, ScriptRuns: detail.ScriptRuns}, nil
}

// AuditWorkflowTask 审批任务办理，委托引擎实例执行。
func (c *WorkflowTaskCase) AuditWorkflowTask(ctx context.Context, req *adminv1.AuditWorkflowTaskRequest) error {
	return c.instanceCase.AuditWorkflowTask(ctx, req)
}

// assembleTaskItems 装配任务列表项，合并实例与流程定义摘要。
func (c *WorkflowTaskCase) assembleTaskItems(ctx context.Context, tasks []*models.WorkflowTask, listType adminv1.WorkflowListType) ([]*adminv1.MyWorkflowTaskItem, error) {
	if len(tasks) == 0 {
		return make([]*adminv1.MyWorkflowTaskItem, 0), nil
	}

	instanceQuery := c.instanceRepo.Query(ctx).WorkflowInstance
	instances, err := c.instanceRepo.List(ctx, repository.Where(instanceQuery.ID.In(collectTaskInstanceIDs(tasks)...)))
	if err != nil {
		return nil, err
	}
	instanceMap := make(map[int64]*models.WorkflowInstance, len(instances))
	definitionIDs := make([]int64, 0, len(instances))
	applicantIDs := make([]int64, 0, len(instances))
	for _, instance := range instances {
		instanceMap[instance.ID] = instance
		definitionIDs = append(definitionIDs, instance.DefinitionID)
		applicantIDs = append(applicantIDs, instance.ApplicantUserID)
	}
	definitionMap, err := c.mapDefinitionSummaries(ctx, dedupeInt64(definitionIDs))
	if err != nil {
		return nil, err
	}
	userMap, err := c.mapNickNames(ctx, dedupeInt64(applicantIDs))
	if err != nil {
		return nil, err
	}

	items := make([]*adminv1.MyWorkflowTaskItem, 0, len(tasks))
	for _, task := range tasks {
		item := &adminv1.MyWorkflowTaskItem{
			TaskId:        task.ID,
			InstanceId:    task.InstanceID,
			TaskStatus:    adminv1.WorkflowTaskStatus(task.Status),
			ListType:      listType,
			TaskCreatedAt: workflow.FormatTime(task.CreatedAt),
		}
		if instance := instanceMap[task.InstanceID]; instance != nil {
			item.ApplicantUserId = instance.ApplicantUserID
			item.ApplicantName = userMap[instance.ApplicantUserID]
			item.BusinessType = instance.BusinessType
			item.InstanceStatus = adminv1.WorkflowInstanceStatus(instance.Status)
			item.SubmitAt = workflow.FormatTime(instance.SubmitAt)
			if definition := definitionMap[instance.DefinitionID]; definition != nil {
				item.DefinitionCode = definition.Code
				item.DefinitionName = definition.Name
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// assembleSubmittedItems 装配我发起的列表项。
func (c *WorkflowTaskCase) assembleSubmittedItems(ctx context.Context, instances []*models.WorkflowInstance) ([]*adminv1.MyWorkflowTaskItem, error) {
	if len(instances) == 0 {
		return make([]*adminv1.MyWorkflowTaskItem, 0), nil
	}

	definitionIDs := make([]int64, 0, len(instances))
	applicantIDs := make([]int64, 0, len(instances))
	for _, instance := range instances {
		definitionIDs = append(definitionIDs, instance.DefinitionID)
		applicantIDs = append(applicantIDs, instance.ApplicantUserID)
	}
	definitionMap, err := c.mapDefinitionSummaries(ctx, dedupeInt64(definitionIDs))
	if err != nil {
		return nil, err
	}
	userMap, err := c.mapNickNames(ctx, dedupeInt64(applicantIDs))
	if err != nil {
		return nil, err
	}

	items := make([]*adminv1.MyWorkflowTaskItem, 0, len(instances))
	for _, instance := range instances {
		item := &adminv1.MyWorkflowTaskItem{
			InstanceId:      instance.ID,
			ApplicantUserId: instance.ApplicantUserID,
			ApplicantName:   userMap[instance.ApplicantUserID],
			BusinessType:    instance.BusinessType,
			InstanceStatus:  adminv1.WorkflowInstanceStatus(instance.Status),
			ListType:        adminv1.WorkflowListType_WORKFLOW_LIST_TYPE_SUBMITTED,
			SubmitAt:        workflow.FormatTime(instance.SubmitAt),
		}
		if definition := definitionMap[instance.DefinitionID]; definition != nil {
			item.DefinitionCode = definition.Code
			item.DefinitionName = definition.Name
		}
		items = append(items, item)
	}
	return items, nil
}

// nickNameOf 查询单个用户昵称，缺失回退用户名，查询失败返回空串。
func (c *WorkflowTaskCase) nickNameOf(ctx context.Context, userID int64) string {
	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Select(userQuery.ID, userQuery.NickName, userQuery.UserName))
	opts = append(opts, repository.Where(userQuery.ID.Eq(userID)))
	user, err := c.baseUserRepo.Find(ctx, opts...)
	if err != nil {
		return ""
	}
	if user.NickName != "" {
		return user.NickName
	}
	return user.UserName
}

// mapDefinitionSummaries 按定义 ID 批量查询流程定义摘要。
func (c *WorkflowTaskCase) mapDefinitionSummaries(ctx context.Context, definitionIDs []int64) (map[int64]*models.WorkflowDefinition, error) {
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

// mapNickNames 按用户 ID 批量查询昵称映射，昵称缺失回退用户名。
func (c *WorkflowTaskCase) mapNickNames(ctx context.Context, userIDs []int64) (map[int64]string, error) {
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

// collectTaskInstanceIDs 收集任务列表的实例 ID。
func collectTaskInstanceIDs(tasks []*models.WorkflowTask) []int64 {
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.InstanceID)
	}
	return ids
}

// dedupeInt64 去重 int64 列表并保序。
func dedupeInt64(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
