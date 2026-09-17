package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/projectauth"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	databasegorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	grantViewPermission   = "base/project/grant"
	grantCreatePermission = "base:tenant:project:grant:create"
	grantUpdatePermission = "base:tenant:project:grant:update"
	grantDeletePermission = "base:tenant:project:grant:delete"
)

// BaseTenantProjectGrantCase 管理岗位、角色、部门和用户在目标租户下的项目授权。
type BaseTenantProjectGrantCase struct {
	*biz.BaseCase
	*data.BaseTenantProjectGrantRepository
	tx data.Transaction
}

// NewBaseTenantProjectGrantCase 创建项目授权业务。
func NewBaseTenantProjectGrantCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	grantRepo *data.BaseTenantProjectGrantRepository,
) *BaseTenantProjectGrantCase {
	return &BaseTenantProjectGrantCase{
		BaseCase:                         baseCase,
		BaseTenantProjectGrantRepository: grantRepo,
		tx:                               tx,
	}
}

// PageBaseTenantProjectGrant 分页查询项目授权，并补充租户、主体和项目名称。
func (c *BaseTenantProjectGrantCase) PageBaseTenantProjectGrant(ctx context.Context, req *adminv1.PageBaseTenantProjectGrantRequest) (*adminv1.PageBaseTenantProjectGrantResponse, error) {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	err = c.requireGrantPermission(ctx, grantViewPermission)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseTenantProjectGrant
	opts := make([]repository.QueryOption, 0, 5)
	if identity.TenantCode != databasegorm.DefaultTenantCode {
		if req.GetTenantId() > 0 && req.GetTenantId() != identity.TenantId {
			return nil, errorsx.PermissionDenied("不能查询其他租户的项目授权")
		}
		opts = append(opts, repository.Where(query.TenantID.Eq(identity.TenantId)))
	} else if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.SubjectType != nil {
		opts = append(opts, repository.Where(query.SubjectType.Eq(int32(req.GetSubjectType()))))
	}
	if req.SubjectId != nil {
		opts = append(opts, repository.Where(query.SubjectID.Eq(req.GetSubjectId())))
	}
	opts = append(opts, repository.Order(query.TenantID.Asc()), repository.Order(query.SubjectType.Asc()), repository.Order(query.SubjectID.Asc()))
	rows, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	return c.mapGrantPage(ctx, rows, total)
}

// GetBaseTenantProjectGrant 查询项目授权详情。
func (c *BaseTenantProjectGrantCase) GetBaseTenantProjectGrant(ctx context.Context, req *adminv1.GetBaseTenantProjectGrantRequest) (*adminv1.BaseTenantProjectGrant, error) {
	err := c.requireTarget(ctx, req.GetTenantId(), req.GetSubjectType(), req.GetSubjectId(), grantViewPermission)
	if err != nil {
		return nil, err
	}
	row, err := c.findGrant(ctx, req.GetTenantId(), req.GetSubjectType(), req.GetSubjectId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("项目授权不存在")
		}
		return nil, err
	}
	result, err := c.mapGrantPage(ctx, []*models.BaseTenantProjectGrant{row}, 1)
	if err != nil {
		return nil, err
	}
	return result.Grants[0], nil
}

// CreateBaseTenantProjectGrant 新增项目授权，同一联合主键只能存在一条记录。
func (c *BaseTenantProjectGrantCase) CreateBaseTenantProjectGrant(ctx context.Context, grant *adminv1.BaseTenantProjectGrant) error {
	ids, err := projectauth.Normalize(grant.GetProjectId())
	if err != nil {
		return err
	}
	err = c.requireTarget(ctx, grant.GetTenantId(), grant.GetSubjectType(), grant.GetSubjectId(), grantCreatePermission)
	if err != nil {
		return err
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		tenant := c.Query(txCtx).BaseTenant
		if _, err = tenant.WithContext(txCtx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(tenant.ID.Eq(grant.GetTenantId())).First(); err != nil {
			return err
		}
		if err = c.validateProjectSelection(txCtx, grant.GetTenantId(), ids); err != nil {
			return err
		}
		_, findErr := c.findGrant(txCtx, grant.GetTenantId(), grant.GetSubjectType(), grant.GetSubjectId())
		if findErr == nil {
			return errorsx.UniqueConflict("该授权主体已存在项目授权", "base_tenant_project_grant", "tenant_id,subject_type,subject_id", "PRIMARY").WithCause(gorm.ErrDuplicatedKey)
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		value, marshalErr := json.Marshal(ids)
		if marshalErr != nil {
			return marshalErr
		}
		table := c.Query(txCtx).BaseTenantProjectGrant
		return table.WithContext(txCtx).Create(&models.BaseTenantProjectGrant{
			TenantID:    grant.GetTenantId(),
			SubjectType: int32(grant.GetSubjectType()),
			SubjectID:   grant.GetSubjectId(),
			ProjectID:   string(value),
		})
	})
}

// UpdateBaseTenantProjectGrant 更新项目范围，租户、授权类型和授权主体联合键保持不变。
func (c *BaseTenantProjectGrantCase) UpdateBaseTenantProjectGrant(ctx context.Context, grant *adminv1.BaseTenantProjectGrant) error {
	ids, err := projectauth.Normalize(grant.GetProjectId())
	if err != nil {
		return err
	}
	err = c.requireTarget(ctx, grant.GetTenantId(), grant.GetSubjectType(), grant.GetSubjectId(), grantUpdatePermission)
	if err != nil {
		return err
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		tenant := c.Query(txCtx).BaseTenant
		if _, err = tenant.WithContext(txCtx).Clauses(clause.Locking{Strength: "UPDATE"}).Where(tenant.ID.Eq(grant.GetTenantId())).First(); err != nil {
			return err
		}
		if err = c.validateProjectSelection(txCtx, grant.GetTenantId(), ids); err != nil {
			return err
		}
		if _, err = c.findGrant(txCtx, grant.GetTenantId(), grant.GetSubjectType(), grant.GetSubjectId()); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorsx.ResourceNotFound("项目授权不存在")
			}
			return err
		}
		value, marshalErr := json.Marshal(ids)
		if marshalErr != nil {
			return marshalErr
		}
		table := c.Query(txCtx).BaseTenantProjectGrant
		_, err = table.WithContext(txCtx).Where(
			table.TenantID.Eq(grant.GetTenantId()),
			table.SubjectType.Eq(int32(grant.GetSubjectType())),
			table.SubjectID.Eq(grant.GetSubjectId()),
		).UpdateSimple(table.ProjectID.Value(string(value)))
		return err
	})
}

// DeleteBaseTenantProjectGrant 删除一条项目授权记录。
func (c *BaseTenantProjectGrantCase) DeleteBaseTenantProjectGrant(ctx context.Context, req *adminv1.DeleteBaseTenantProjectGrantRequest) error {
	err := c.requireTarget(ctx, req.GetTenantId(), req.GetSubjectType(), req.GetSubjectId(), grantDeletePermission)
	if err != nil {
		return err
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		table := c.Query(txCtx).BaseTenantProjectGrant
		result, deleteErr := table.WithContext(txCtx).Where(
			table.TenantID.Eq(req.GetTenantId()),
			table.SubjectType.Eq(int32(req.GetSubjectType())),
			table.SubjectID.Eq(req.GetSubjectId()),
		).Delete()
		if deleteErr != nil {
			return deleteErr
		}
		if result.RowsAffected == 0 {
			return errorsx.ResourceNotFound("项目授权不存在")
		}
		return nil
	})
}

// EffectiveProjects 从当前用户的岗位、角色、直属部门和用户授权实时合并有效范围。
func (c *BaseTenantProjectGrantCase) EffectiveProjects(ctx context.Context) (map[int64][]int64, error) {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx)
	var user *models.BaseUser
	user, err = query.BaseUser.WithContext(ctx).
		Select(query.BaseUser.ID, query.BaseUser.TenantID, query.BaseUser.RoleID, query.BaseUser.DeptID, query.BaseUser.PostID, query.BaseUser.Status).
		Where(query.BaseUser.ID.Eq(identity.UserId)).First()
	if err != nil {
		return nil, err
	}
	if user.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("用户已停用")
	}
	var role *models.BaseRole
	role, err = query.BaseRole.WithContext(ctx).Where(query.BaseRole.ID.Eq(user.RoleID)).First()
	if err != nil {
		return nil, err
	}
	if role.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("角色已停用")
	}
	var dept *models.BaseDept
	dept, err = query.BaseDept.WithContext(ctx).Where(query.BaseDept.ID.Eq(user.DeptID)).First()
	if err != nil {
		return nil, err
	}
	if dept.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("部门已停用")
	}
	if role.TenantID != user.TenantID || dept.TenantID != user.TenantID {
		return nil, errorsx.PermissionDenied("用户组织关系与所属租户不一致")
	}
	// 系统维护的超级管理员用于平台初始化授权，普通默认租户账号仍按四维授权处理。
	if role.Code == _const.BASE_ROLE_CODE_SUPER {
		var tenants []*models.BaseTenant
		tenants, err = query.BaseTenant.WithContext(ctx).Where(query.BaseTenant.Status.Eq(_const.STATUS_STATUS_ENABLE)).Find()
		if err != nil {
			return nil, err
		}
		result := make(map[int64][]int64, len(tenants))
		for _, tenant := range tenants {
			result[tenant.ID] = []int64{0}
		}
		return result, nil
	}
	table := query.BaseTenantProjectGrant
	predicates := []field.Expr{field.And(table.SubjectType.Eq(4), table.SubjectID.Eq(user.ID))}
	if user.PostID > 0 {
		var post *models.BasePost
		post, err = query.BasePost.WithContext(ctx).Where(query.BasePost.ID.Eq(user.PostID)).First()
		if err != nil {
			return nil, err
		}
		if post.TenantID != user.TenantID {
			return nil, errorsx.PermissionDenied("岗位与用户所属租户不一致")
		}
		if post.Status == _const.STATUS_STATUS_ENABLE {
			predicates = append(predicates, field.And(table.SubjectType.Eq(1), table.SubjectID.Eq(user.PostID)))
		}
	}
	if user.RoleID > 0 {
		predicates = append(predicates, field.And(table.SubjectType.Eq(2), table.SubjectID.Eq(user.RoleID)))
	}
	if user.DeptID > 0 {
		predicates = append(predicates, field.And(table.SubjectType.Eq(3), table.SubjectID.Eq(user.DeptID)))
	}
	var rows []*models.BaseTenantProjectGrant
	rows, err = table.WithContext(ctx).Where(field.Or(predicates...)).Find()
	if err != nil {
		return nil, err
	}
	result := make(map[int64][]int64)
	for _, row := range rows {
		if identity.TenantCode != databasegorm.DefaultTenantCode && row.TenantID != user.TenantID {
			continue
		}
		var ids []int64
		ids, err = projectauth.Decode(row.ProjectID)
		if err != nil {
			return nil, err
		}
		result[row.TenantID], err = projectauth.Merge(result[row.TenantID], ids)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// findGrant 按联合主键查询项目授权。
func (c *BaseTenantProjectGrantCase) findGrant(ctx context.Context, tenantID int64, subjectType adminv1.BaseTenantProjectGrantSubjectType, subjectID int64) (*models.BaseTenantProjectGrant, error) {
	table := c.Query(ctx).BaseTenantProjectGrant
	row, err := table.WithContext(ctx).Where(
		table.TenantID.Eq(tenantID),
		table.SubjectType.Eq(int32(subjectType)),
		table.SubjectID.Eq(subjectID),
	).First()
	return row, err
}

// validateProjectSelection 校验当前操作者可授予的项目范围和目标租户项目归属。
func (c *BaseTenantProjectGrantCase) validateProjectSelection(ctx context.Context, tenantID int64, ids []int64) error {
	scopes, err := c.EffectiveProjects(ctx)
	if err != nil {
		return err
	}
	allowed := scopes[tenantID]
	if !slices.Equal(allowed, []int64{0}) {
		for _, id := range ids {
			if !projectauth.Allows(allowed, id) {
				return errorsx.PermissionDenied("不能授予超出自身范围的项目权限")
			}
		}
	}
	if len(ids) == 0 || ids[0] == 0 {
		return nil
	}
	query := c.Query(ctx).BaseTenantProject
	count, err := query.WithContext(ctx).Where(query.TenantID.Eq(tenantID), query.ID.In(ids...)).Count()
	if err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errorsx.InvalidArgument("指定项目不存在或不属于目标租户")
	}
	return nil
}

// requireTarget 校验主体类型、归属、目标租户和操作权限。
func (c *BaseTenantProjectGrantCase) requireTarget(ctx context.Context, tenantID int64, subjectType adminv1.BaseTenantProjectGrantSubjectType, subjectID int64, permission string) error {
	if tenantID <= 0 || subjectID <= 0 {
		return errorsx.InvalidArgument("租户和授权主体ID必须为正数")
	}
	if err := c.requireGrantPermission(ctx, permission); err != nil {
		return err
	}
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if identity.TenantCode != databasegorm.DefaultTenantCode && tenantID != identity.TenantId {
		return errorsx.PermissionDenied("不能管理其他租户的项目授权")
	}
	query := c.Query(ctx)
	var ownerID int64
	switch subjectType {
	case adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_POST:
		var row *models.BasePost
		row, err = query.BasePost.WithContext(ctx).Where(query.BasePost.ID.Eq(subjectID)).First()
		if err != nil {
			return err
		}
		ownerID = row.TenantID
	case adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_ROLE:
		var row *models.BaseRole
		row, err = query.BaseRole.WithContext(ctx).Where(query.BaseRole.ID.Eq(subjectID)).First()
		if err != nil {
			return err
		}
		ownerID = row.TenantID
	case adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_DEPT:
		var row *models.BaseDept
		row, err = query.BaseDept.WithContext(ctx).Where(query.BaseDept.ID.Eq(subjectID)).First()
		if err != nil {
			return err
		}
		ownerID = row.TenantID
	case adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_USER:
		var row *models.BaseUser
		row, err = query.BaseUser.WithContext(ctx).
			Select(query.BaseUser.ID, query.BaseUser.TenantID).
			Where(query.BaseUser.ID.Eq(subjectID)).First()
		if err != nil {
			return err
		}
		ownerID = row.TenantID
	default:
		return errorsx.InvalidArgument("授权主体类型无效")
	}
	var owner *models.BaseTenant
	owner, err = query.BaseTenant.WithContext(ctx).Where(query.BaseTenant.ID.Eq(ownerID)).First()
	if err != nil {
		return err
	}
	if owner.Code != databasegorm.DefaultTenantCode && ownerID != tenantID {
		return errorsx.PermissionDenied("普通租户的授权主体不能获得其他租户项目权限")
	}
	_, err = query.BaseTenant.WithContext(ctx).Where(query.BaseTenant.ID.Eq(tenantID)).First()
	return err
}

// requireGrantPermission 校验当前角色持有统一项目授权入口及对应操作按钮。
func (c *BaseTenantProjectGrantCase) requireGrantPermission(ctx context.Context, permission string) error {
	identity, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx)
	user, err := query.BaseUser.WithContext(ctx).
		Select(query.BaseUser.ID, query.BaseUser.TenantID, query.BaseUser.RoleID, query.BaseUser.Status).
		Where(query.BaseUser.ID.Eq(identity.UserId)).First()
	if err != nil {
		return err
	}
	if user.Status != _const.STATUS_STATUS_ENABLE {
		return errorsx.PermissionDenied("用户身份已变更，请重新登录")
	}
	role, err := query.BaseRole.WithContext(ctx).Where(query.BaseRole.ID.Eq(user.RoleID)).First()
	if err != nil {
		return err
	}
	if role.Status != _const.STATUS_STATUS_ENABLE || role.TenantID != user.TenantID {
		return errorsx.PermissionDenied("角色状态或归属无效")
	}
	if role.Code == _const.BASE_ROLE_CODE_SUPER {
		return nil
	}
	var menuIDs []int64
	err = json.Unmarshal([]byte(role.Menus), &menuIDs)
	if err != nil {
		return errorsx.Internal("角色权限数据格式无效").WithCause(err)
	}
	if len(menuIDs) == 0 {
		return errorsx.PermissionDenied("无权管理项目授权")
	}
	count, err := query.BaseMenu.WithContext(ctx).Where(query.BaseMenu.ID.In(menuIDs...), query.BaseMenu.Path.Eq(permission), query.BaseMenu.Status.Eq(_const.STATUS_STATUS_ENABLE)).Count()
	if err != nil {
		return err
	}
	if count == 0 {
		return errorsx.PermissionDenied("无权管理项目授权")
	}
	return nil
}

// mapGrantPage 将授权记录补齐为管理端列表所需的展示字段。
func (c *BaseTenantProjectGrantCase) mapGrantPage(ctx context.Context, rows []*models.BaseTenantProjectGrant, total int64) (*adminv1.PageBaseTenantProjectGrantResponse, error) {
	tenantIDs := make([]int64, 0, len(rows))
	tenantSeen := make(map[int64]struct{}, len(rows))
	subjectIDs := make(map[int32][]int64)
	subjectSeen := make(map[int32]map[int64]struct{})
	projectIDs := make([]int64, 0)
	projectSeen := make(map[int64]struct{})
	decoded := make([][]int64, len(rows))
	for index, row := range rows {
		if _, exists := tenantSeen[row.TenantID]; !exists {
			tenantSeen[row.TenantID] = struct{}{}
			tenantIDs = append(tenantIDs, row.TenantID)
		}
		if subjectSeen[row.SubjectType] == nil {
			subjectSeen[row.SubjectType] = make(map[int64]struct{})
		}
		if _, exists := subjectSeen[row.SubjectType][row.SubjectID]; !exists {
			subjectSeen[row.SubjectType][row.SubjectID] = struct{}{}
			subjectIDs[row.SubjectType] = append(subjectIDs[row.SubjectType], row.SubjectID)
		}
		ids, err := projectauth.Decode(row.ProjectID)
		if err != nil {
			return nil, err
		}
		decoded[index] = ids
		for _, id := range ids {
			if id > 0 {
				if _, exists := projectSeen[id]; !exists {
					projectSeen[id] = struct{}{}
					projectIDs = append(projectIDs, id)
				}
			}
		}
	}
	query := c.Query(ctx)
	tenantNames := make(map[int64]string, len(tenantIDs))
	if len(tenantIDs) > 0 {
		tenants, err := query.BaseTenant.WithContext(ctx).Where(query.BaseTenant.ID.In(tenantIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, tenant := range tenants {
			tenantNames[tenant.ID] = tenant.Name
		}
	}
	subjectNames := make(map[int32]map[int64]string)
	subjectCodes := make(map[int32]map[int64]string)
	loadSubject := func(subjectType int32, names map[int64]string, codes map[int64]string) {
		subjectNames[subjectType] = names
		subjectCodes[subjectType] = codes
	}
	if ids := subjectIDs[1]; len(ids) > 0 {
		posts, err := query.BasePost.WithContext(ctx).Where(query.BasePost.ID.In(ids...)).Find()
		if err != nil {
			return nil, err
		}
		names, codes := make(map[int64]string), make(map[int64]string)
		for _, item := range posts {
			names[item.ID], codes[item.ID] = item.Name, item.Code
		}
		loadSubject(1, names, codes)
	}
	if ids := subjectIDs[2]; len(ids) > 0 {
		roles, err := query.BaseRole.WithContext(ctx).Where(query.BaseRole.ID.In(ids...)).Find()
		if err != nil {
			return nil, err
		}
		names, codes := make(map[int64]string), make(map[int64]string)
		for _, item := range roles {
			names[item.ID], codes[item.ID] = item.Name, item.Code
		}
		loadSubject(2, names, codes)
	}
	if ids := subjectIDs[3]; len(ids) > 0 {
		depts, err := query.BaseDept.WithContext(ctx).Where(query.BaseDept.ID.In(ids...)).Find()
		if err != nil {
			return nil, err
		}
		names := make(map[int64]string)
		for _, item := range depts {
			names[item.ID] = item.Name
		}
		loadSubject(3, names, make(map[int64]string))
	}
	if ids := subjectIDs[4]; len(ids) > 0 {
		users, err := query.BaseUser.WithContext(ctx).
			Select(query.BaseUser.ID, query.BaseUser.UserName, query.BaseUser.UserCode, query.BaseUser.NickName).
			Where(query.BaseUser.ID.In(ids...)).Find()
		if err != nil {
			return nil, err
		}
		names, codes := make(map[int64]string), make(map[int64]string)
		for _, item := range users {
			names[item.ID] = item.NickName
			if names[item.ID] == "" {
				names[item.ID] = item.UserName
			}
			codes[item.ID] = item.UserCode
		}
		loadSubject(4, names, codes)
	}
	projectNames := make(map[int64]string, len(projectIDs))
	if len(projectIDs) > 0 {
		projects, err := query.BaseTenantProject.WithContext(ctx).Where(query.BaseTenantProject.ID.In(projectIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			projectNames[project.ID] = project.Name
		}
	}
	result := make([]*adminv1.BaseTenantProjectGrant, 0, len(rows))
	for index, row := range rows {
		item := &adminv1.BaseTenantProjectGrant{
			TenantId:     row.TenantID,
			SubjectType:  adminv1.BaseTenantProjectGrantSubjectType(row.SubjectType),
			SubjectId:    row.SubjectID,
			ProjectId:    decoded[index],
			TenantName:   tenantNames[row.TenantID],
			SubjectName:  subjectNames[row.SubjectType][row.SubjectID],
			SubjectCode:  subjectCodes[row.SubjectType][row.SubjectID],
			ProjectNames: make([]string, 0, len(decoded[index])),
			GrantKey:     fmt.Sprintf("%d:%d:%d", row.TenantID, row.SubjectType, row.SubjectID),
		}
		for _, id := range decoded[index] {
			if id == 0 {
				continue
			}
			name := projectNames[id]
			if name == "" {
				name = fmt.Sprintf("ID:%d", id)
			}
			item.ProjectNames = append(item.ProjectNames, name)
		}
		result = append(result, item)
	}
	return &adminv1.PageBaseTenantProjectGrantResponse{Grants: result, Total: int32(total)}, nil
}
