package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-admin/backend/pkg/projectaccess"
	"github.com/liujitcn/kratos-admin/backend/pkg/workflow"
	"gorm.io/gen/field"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/resource/i18n"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
	gormdb "gorm.io/gorm"
)

// BaseTenantProjectCase 项目业务实例。
type BaseTenantProjectCase struct {
	lifecycle *projectaccess.Lifecycle
	grants    *BaseTenantProjectGrantCase
	*biz.BaseCase
	tx data.Transaction
	*data.BaseTenantProjectRepository
	catalog       *i18n.I18n
	eventRegistry *workflow.EventRegistry
	formMapper    *mapper.CopierMapper[adminv1.BaseTenantProjectForm, models.BaseTenantProject]
	mapper        *mapper.CopierMapper[adminv1.BaseTenantProject, models.BaseTenantProject]
}

// NewBaseTenantProjectCase 创建项目业务实例。
func NewBaseTenantProjectCase(
	lifecycle *projectaccess.Lifecycle,
	baseCase *biz.BaseCase,
	grants *BaseTenantProjectGrantCase,
	tx data.Transaction,
	baseTenantProjectRepo *data.BaseTenantProjectRepository,
	catalog *i18n.I18n,
	eventRegistry *workflow.EventRegistry,
) *BaseTenantProjectCase {
	caseValue := &BaseTenantProjectCase{
		lifecycle:                   lifecycle,
		grants:                      grants,
		BaseCase:                    baseCase,
		tx:                          tx,
		BaseTenantProjectRepository: baseTenantProjectRepo,
		catalog:                     catalog,
		eventRegistry:               eventRegistry,
		formMapper:                  mapper.NewCopierMapper[adminv1.BaseTenantProjectForm, models.BaseTenantProject](),
		mapper:                      mapper.NewCopierMapper[adminv1.BaseTenantProject, models.BaseTenantProject](),
	}
	eventRegistry.Register(baseTenantProjectCreateBusinessType, caseValue.handleTenantProjectWorkflowEvent)
	eventRegistry.RegisterAction(baseTenantProjectCreateBusinessType, baseTenantProjectCreateAction, caseValue.handleTenantProjectWorkflowAction)
	return caseValue
}

// OptionBaseTenantProject 查询项目选项。
func (c *BaseTenantProjectCase) OptionBaseTenantProject(ctx context.Context, req *adminv1.OptionBaseTenantProjectRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseTenantProject
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: item.Status != _const.STATUS_STATUS_ENABLE,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// TreeBaseTenantProject 查询当前账号可用的租户项目树。
func (c *BaseTenantProjectCase) TreeBaseTenantProject(ctx context.Context, req *adminv1.TreeBaseTenantProjectRequest) (*adminv1.TreeBaseTenantProjectResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseTenantProject
	var opts []repository.QueryOption
	opts, err = c.projectOptions(ctx)
	if err != nil {
		return nil, err
	}
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)))
	if req.GetKeyword() != "" {
		opts = append(opts, repository.Where(field.Or(query.Code.Like("%"+req.GetKeyword()+"%"), query.Name.Like("%"+req.GetKeyword()+"%"))))
	}
	var rows []*models.BaseTenantProject
	rows, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := &adminv1.TreeBaseTenantProjectResponse{List: make([]*adminv1.TreeBaseTenantProjectResponse_Option, 0, len(rows))}
	tenants := make(map[int64]*adminv1.TreeBaseTenantProjectResponse_Option)
	tenantNames := make(map[int64]string)
	if authInfo.TenantCode == gorm.DefaultTenantCode {
		tenantIDs := make([]int64, 0, len(rows))
		tenantsSeen := make(map[int64]struct{}, len(rows))
		for _, row := range rows {
			if _, exists := tenantsSeen[row.TenantID]; exists {
				continue
			}
			tenantsSeen[row.TenantID] = struct{}{}
			tenantIDs = append(tenantIDs, row.TenantID)
		}
		if len(tenantIDs) > 0 {
			tenantQuery := c.Query(ctx).BaseTenant
			var tenantRows []*models.BaseTenant
			tenantRows, err = tenantQuery.WithContext(ctx).Where(tenantQuery.ID.In(tenantIDs...)).Find()
			if err != nil {
				return nil, err
			}
			for _, tenant := range tenantRows {
				tenantNames[tenant.ID] = tenant.Name
			}
		}
	}
	for _, row := range rows {
		option := &adminv1.TreeBaseTenantProjectResponse_Option{Value: fmt.Sprintf("project:%d:%d", row.TenantID, row.ID), Label: row.Name, Type: "project", TenantId: row.TenantID, ProjectId: row.ID}
		if authInfo.TenantCode != gorm.DefaultTenantCode {
			result.List = append(result.List, option)
			continue
		}
		tenant, exists := tenants[row.TenantID]
		if !exists {
			tenantName, exists := tenantNames[row.TenantID]
			if !exists || tenantName == "" {
				tenantName = localizedTenantFallbackName(c.catalog, biz.LocaleFromContext(ctx), row.TenantID)
			}
			tenant = &adminv1.TreeBaseTenantProjectResponse_Option{Value: fmt.Sprintf("tenant:%d", row.TenantID), Label: tenantName, Type: "tenant", TenantId: row.TenantID, Children: make([]*adminv1.TreeBaseTenantProjectResponse_Option, 0)}
			tenants[row.TenantID] = tenant
			result.List = append(result.List, tenant)
		}
		tenant.Children = append(tenant.Children, option)
	}
	return result, nil
}

// PageBaseTenantProject 分页查询项目。
func (c *BaseTenantProjectCase) PageBaseTenantProject(ctx context.Context, req *adminv1.PageBaseTenantProjectRequest) (*adminv1.PageBaseTenantProjectResponse, error) {
	query := c.Query(ctx).BaseTenantProject
	opts, err := c.projectOptions(ctx)
	if err != nil {
		return nil, err
	}
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
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
	var list []*models.BaseTenantProject
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseTenantProject, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &adminv1.PageBaseTenantProjectResponse{BaseTenantProjects: resList, Total: int32(total)}, nil
}

// GetBaseTenantProject 获取项目。
func (c *BaseTenantProjectCase) GetBaseTenantProject(ctx context.Context, id int64) (*adminv1.BaseTenantProjectForm, error) {
	baseTenantProject, err := c.findProject(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseTenantProject), nil
}

// CreateBaseTenantProject 创建项目。
func (c *BaseTenantProjectCase) CreateBaseTenantProject(ctx context.Context, req *adminv1.BaseTenantProjectForm) error {
	_, err := c.submitTenantProjectApproval(ctx, req)
	return err
}

const (
	baseTenantProjectCreateWorkflowCode = "BASE_TENANT_PROJECT_CREATE"
	baseTenantProjectCreateBusinessType = "BASE_TENANT_PROJECT_CREATE"
	baseTenantProjectCreateAction       = "base.tenant.project.create.complete"
)

// submitTenantProjectApproval 创建待审批项目占位记录并提交默认审批流程。
func (c *BaseTenantProjectCase) submitTenantProjectApproval(ctx context.Context, req *adminv1.BaseTenantProjectForm) (int64, error) {
	err := c.requireProjectManager(ctx)
	if err != nil {
		return 0, err
	}
	if req.GetId() != 0 {
		return 0, errorsx.InvalidArgument("新增项目不能指定ID")
	}
	tenantID, err := c.resolveTenantID(ctx, req.GetTenantId())
	if err != nil {
		return 0, err
	}
	project, err := c.createProjectPlaceholder(ctx, req, tenantID)
	if err != nil {
		return 0, err
	}
	formDataValue := *req
	formDataValue.Id = 0
	formDataValue.TenantId = project.TenantID
	formData, err := json.Marshal(&formDataValue)
	if err != nil {
		return 0, err
	}
	instanceID, err := workflow.Submit(ctx, workflow.SubmitParams{
		Code:         baseTenantProjectCreateWorkflowCode,
		Version:      1,
		FormData:     string(formData),
		BusinessType: baseTenantProjectCreateBusinessType,
		BusinessID:   project.ID,
	})
	if err != nil {
		_ = c.removeProjectPlaceholder(context.WithoutCancel(ctx), project.ID)
		return 0, err
	}
	return instanceID, nil
}

// createProjectPlaceholder 创建禁用状态的项目占位记录，审批通过后才启用项目。
func (c *BaseTenantProjectCase) createProjectPlaceholder(ctx context.Context, req *adminv1.BaseTenantProjectForm, tenantID int64) (*models.BaseTenantProject, error) {
	project := c.formMapper.ToEntity(req)
	project.ID = 0
	project.TenantID = tenantID
	project.Status = _const.STATUS_STATUS_DISABLE
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		if err := c.Create(ctx, project); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一租户的项目编号重复", "base_tenant_project", "code", "unique_base_tenant_project").WithCause(err)
			}
			return err
		}
		return nil
	})
	return project, err
}

// completeProjectCreation 审批通过后按申请状态启用项目，重复执行保持幂等。
func (c *BaseTenantProjectCase) completeProjectCreation(ctx context.Context, projectID, tenantID int64, requestedStatus int32) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).BaseTenantProject
		project, err := c.Find(ctx, repository.Where(query.ID.Eq(projectID)), repository.Where(query.TenantID.Eq(tenantID)))
		if err != nil {
			return err
		}
		if project.Status == _const.STATUS_STATUS_ENABLE {
			return nil
		}
		if requestedStatus != _const.STATUS_STATUS_DISABLE {
			requestedStatus = _const.STATUS_STATUS_ENABLE
		}
		return c.UpdateByID(ctx, &models.BaseTenantProject{ID: projectID, Status: requestedStatus})
	})
}

// removeProjectPlaceholder 清理驳回或撤回的待审批项目占位记录。
func (c *BaseTenantProjectCase) removeProjectPlaceholder(ctx context.Context, projectID int64) error {
	project, err := c.BaseTenantProjectRepository.FindByID(ctx, projectID)
	if errors.Is(err, gormdb.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if project.Status != _const.STATUS_STATUS_DISABLE {
		return nil
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error { return c.DeleteByIDs(ctx, []int64{projectID}) })
}

// handleTenantProjectWorkflowAction 执行审批通过后的项目启用动作。
func (c *BaseTenantProjectCase) handleTenantProjectWorkflowAction(ctx context.Context, input workflow.ActionContext) error {
	defaultTenantID, err := findDefaultTenantID(c.Query)
	if err != nil {
		return errorsx.Internal("校验项目审批来源失败").WithCause(err)
	}
	if input.TenantID != defaultTenantID {
		return errorsx.PermissionDenied("只有默认租户可以执行新增项目审批动作")
	}
	var bound bool
	bound, err = workflowInstanceBindingExists(c.Query, input.TenantID, input.InstanceID, input.BusinessID, baseTenantProjectCreateBusinessType, input.Status)
	if err != nil {
		return errorsx.Internal("校验项目审批关联失败").WithCause(err)
	}
	if !bound {
		return errorsx.PermissionDenied("新增项目审批业务关联无效")
	}
	form, ok := input.Payload["form"].(map[string]any)
	if !ok {
		return errorsx.InvalidArgument("新增项目审批表单无效")
	}
	formTenantID, exists := form["tenant_id"].(float64)
	if !exists || formTenantID <= 0 || formTenantID != float64(int64(formTenantID)) {
		return errorsx.InvalidArgument("新增项目审批所属租户无效")
	}
	status := int32(_const.STATUS_STATUS_ENABLE)
	if value, exists := form["status"].(float64); exists {
		status = int32(value)
	}
	return c.completeProjectCreation(ctx, input.BusinessID, int64(formTenantID), status)
}

// handleTenantProjectWorkflowEvent 清理被驳回或撤回的项目占位记录。
func (c *BaseTenantProjectCase) handleTenantProjectWorkflowEvent(_ context.Context, tenantID, instanceID, businessID int64, status adminv1.WorkflowInstanceStatus) {
	defaultTenantID, err := findDefaultTenantID(c.Query)
	if err != nil {
		log.Error(fmt.Sprintf("校验项目审批来源失败: %v", err))
		return
	}
	if tenantID != defaultTenantID {
		log.Error(fmt.Sprintf("拒绝非默认租户清理项目审批占位记录: tenant_id=%d business_id=%d", tenantID, businessID))
		return
	}
	var bound bool
	bound, err = workflowInstanceBindingExists(c.Query, tenantID, instanceID, businessID, baseTenantProjectCreateBusinessType, int32(status))
	if err != nil {
		log.Error(fmt.Sprintf("校验项目审批关联失败: %v", err))
		return
	}
	if !bound {
		log.Error(fmt.Sprintf("拒绝清理未绑定流程实例的项目审批占位记录: instance_id=%d business_id=%d", instanceID, businessID))
		return
	}
	if status == adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_REJECTED || status == adminv1.WorkflowInstanceStatus_WORKFLOW_INSTANCE_STATUS_WITHDRAWN {
		err = c.removeProjectPlaceholder(context.Background(), businessID)
		if err != nil {
			log.Error(fmt.Sprintf("清理项目审批占位记录失败: %v", err))
		}
	}
}

// UpdateBaseTenantProject 更新项目。
func (c *BaseTenantProjectCase) UpdateBaseTenantProject(ctx context.Context, req *adminv1.BaseTenantProjectForm) error {
	var err error
	err = c.requireProjectManager(ctx)
	if err != nil {
		return err
	}
	var oldBaseTenantProject *models.BaseTenantProject
	oldBaseTenantProject, err = c.findProject(ctx, req.GetId())
	if err != nil {
		return err
	}
	baseTenantProject := c.formMapper.ToEntity(req)
	baseTenantProject.TenantID = oldBaseTenantProject.TenantID
	baseTenantProject.ID = oldBaseTenantProject.ID
	if baseTenantProject.Status == 0 {
		baseTenantProject.Status = oldBaseTenantProject.Status
	}
	apply := func(ctx context.Context) error {
		return c.tx.Transaction(ctx, func(ctx context.Context) error {
			err = c.UpdateByID(ctx, baseTenantProject)
			if err != nil {
				if errorsx.IsDuplicateKey(err) {
					return errorsx.UniqueConflict("同一租户的项目编号重复", "base_tenant_project", "code", "unique_base_tenant_project").WithCause(err)
				}
				return err
			}
			return nil
		})
	}
	if baseTenantProject.Status == _const.STATUS_STATUS_DISABLE && oldBaseTenantProject.Status != baseTenantProject.Status {
		return c.lifecycle.Change(ctx, []projectaccess.ProjectKey{{TenantID: oldBaseTenantProject.TenantID, ProjectID: oldBaseTenantProject.ID}}, apply)
	}
	return apply(ctx)
}

// DeleteBaseTenantProject 删除项目。
func (c *BaseTenantProjectCase) DeleteBaseTenantProject(ctx context.Context, id string) error {
	var err error
	err = c.requireProjectManager(ctx)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(id)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("请选择项目")
	}
	for _, projectID := range ids {
		if projectID <= 0 {
			return errorsx.InvalidArgument("项目ID必须为正数")
		}
	}
	var opts []repository.QueryOption
	opts, err = c.projectOptions(ctx)
	if err != nil {
		return err
	}
	var projects []*models.BaseTenantProject
	opts = append(opts, repository.Where(c.Query(ctx).BaseTenantProject.ID.In(ids...)))
	projects, err = c.List(ctx, opts...)
	if err != nil {
		return err
	}
	projectMap := make(map[int64]*models.BaseTenantProject, len(projects))
	for _, item := range projects {
		projectMap[item.ID] = item
	}
	for _, projectID := range ids {
		if _, exists := projectMap[projectID]; !exists {
			return errorsx.ResourceNotFound("删除项目失败，项目不存在")
		}
	}

	keys := make([]projectaccess.ProjectKey, 0, len(projects))
	for _, project := range projects {
		keys = append(keys, projectaccess.ProjectKey{TenantID: project.TenantID, ProjectID: project.ID})
	}
	return c.lifecycle.Change(ctx, keys, func(ctx context.Context) error {
		return c.tx.Transaction(ctx, func(ctx context.Context) error { return c.DeleteByIDs(ctx, ids) })
	})
}

// SetBaseTenantProjectStatus 设置项目状态。
func (c *BaseTenantProjectCase) SetBaseTenantProjectStatus(ctx context.Context, req *adminv1.SetBaseTenantProjectStatusRequest) error {
	var err error
	err = c.requireProjectManager(ctx)
	if err != nil {
		return err
	}
	var baseTenantProject *models.BaseTenantProject
	baseTenantProject, err = c.findProject(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetStatus() != _const.STATUS_STATUS_ENABLE && req.GetStatus() != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("项目状态无效")
	}
	if baseTenantProject.Status == req.GetStatus() {
		return nil
	}
	apply := func(ctx context.Context) error {
		return c.tx.Transaction(ctx, func(ctx context.Context) error {
			baseTenantProject.Status = req.GetStatus()
			return c.UpdateByID(ctx, baseTenantProject)
		})
	}
	if req.GetStatus() == _const.STATUS_STATUS_DISABLE {
		return c.lifecycle.Change(ctx, []projectaccess.ProjectKey{{TenantID: baseTenantProject.TenantID, ProjectID: baseTenantProject.ID}}, apply)
	}
	return apply(ctx)
}

// requireProjectManager 校验当前账号是否为项目目录维护者。
func (c *BaseTenantProjectCase) requireProjectManager(ctx context.Context) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode {
		return errorsx.PermissionDenied("只有默认租户可以维护项目目录")
	}
	return nil
}

// resolveTenantID 解析项目创建时的所属租户并校验租户范围。
func (c *BaseTenantProjectCase) resolveTenantID(ctx context.Context, tenantID int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if tenantID == 0 {
		tenantID = authInfo.TenantId
	} else if authInfo.TenantCode != gorm.DefaultTenantCode && tenantID != authInfo.TenantId {
		return 0, errorsx.PermissionDenied("不能创建其他租户的项目")
	}
	query := c.Query(ctx).BaseTenant
	var count int64
	count, err = query.WithContext(ctx).Where(query.ID.Eq(tenantID), query.Status.Eq(_const.STATUS_STATUS_ENABLE)).Count()
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, errorsx.InvalidArgument("所属租户不存在或已停用")
	}
	return tenantID, nil
}

// findProject 按项目主键查询并强制应用当前用户的有效项目范围。
func (c *BaseTenantProjectCase) findProject(ctx context.Context, id int64) (*models.BaseTenantProject, error) {
	opts, err := c.projectOptions(ctx)
	if err != nil {
		return nil, err
	}
	opts = append(opts, repository.Where(c.Query(ctx).BaseTenantProject.ID.Eq(id)))
	return c.Find(ctx, opts...)
}

// projectOptions 生成项目基础表的租户与主键配对范围，空授权不能省略过滤。
func (c *BaseTenantProjectCase) projectOptions(ctx context.Context) ([]repository.QueryOption, error) {
	scopes, err := c.grants.EffectiveProjects(ctx)
	if err != nil {
		return nil, err
	}
	table := c.Query(ctx).BaseTenantProject
	predicates := make([]field.Expr, 0, len(scopes))
	for tenantID, ids := range scopes {
		if slices.Equal(ids, []int64{0}) {
			predicates = append(predicates, table.TenantID.Eq(tenantID))
		} else if len(ids) > 0 {
			predicates = append(predicates, field.And(table.TenantID.Eq(tenantID), table.ID.In(ids...)))
		}
	}
	if len(predicates) == 0 {
		return []repository.QueryOption{repository.Where(table.ID.Eq(0))}, nil
	}
	return []repository.QueryOption{repository.Where(field.Or(predicates...))}, nil
}

// localizedTenantFallbackName 为失效租户记录生成带编号的本地化标签。
func localizedTenantFallbackName(catalog *i18n.I18n, locale string, tenantID int64) string {
	fallback := fmt.Sprintf("Tenant %d", tenantID)
	if catalog == nil {
		return fallback
	}
	return catalog.Localize(locale, "zh-CN", "system.base.tenant.value.missing_name", map[string]any{"ID": tenantID}, fallback)
}
