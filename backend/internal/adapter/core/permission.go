package core

import (
	"context"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coredata "github.com/liujitcn/kratos-core/data"
)

// PermissionStoreAdapter 将 Admin 的权限生成仓储适配为 Core 权限资源接口。
type PermissionStoreAdapter struct {
	menuRepository   *data.BaseMenuRepository
	roleRepository   *data.BaseRoleRepository
	tenantRepository *data.BaseTenantRepository
	policyRepository *data.CasbinRuleRepository
}

// NewPermissionStoreAdapter 创建 Core 权限资源存储适配器。
func NewPermissionStoreAdapter(
	menuRepository *data.BaseMenuRepository,
	roleRepository *data.BaseRoleRepository,
	tenantRepository *data.BaseTenantRepository,
	policyRepository *data.CasbinRuleRepository,
) *PermissionStoreAdapter {
	return &PermissionStoreAdapter{
		menuRepository:   menuRepository,
		roleRepository:   roleRepository,
		tenantRepository: tenantRepository,
		policyRepository: policyRepository,
	}
}

// FindTenantByCode 按编码查询租户权限字段。
func (s *PermissionStoreAdapter) FindTenantByCode(ctx context.Context, code string) (coredata.TenantRecord, error) {
	query := s.tenantRepository.Query(ctx).BaseTenant
	item, err := query.WithContext(ctx).Where(query.Code.Eq(code)).First()
	if err != nil {
		return coredata.TenantRecord{}, err
	}
	return coredata.TenantRecord{ID: item.ID, Code: item.Code}, nil
}

// ListTenants 查询权限重建所需的全部租户字段。
func (s *PermissionStoreAdapter) ListTenants(ctx context.Context) ([]coredata.TenantRecord, error) {
	items, err := s.tenantRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]coredata.TenantRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.TenantRecord{ID: item.ID, Code: item.Code})
	}
	return result, nil
}

// FindRoleByTenantIDAndCode 按租户和角色编码查询角色权限字段。
func (s *PermissionStoreAdapter) FindRoleByTenantIDAndCode(ctx context.Context, tenantID int64, code string) (coredata.RoleRecord, error) {
	query := s.roleRepository.Query(ctx).BaseRole
	item, err := query.WithContext(ctx).Where(query.TenantID.Eq(tenantID), query.Code.Eq(code)).First()
	if err != nil {
		return coredata.RoleRecord{}, err
	}
	return coredata.RoleRecord{ID: item.ID, TenantID: item.TenantID, Code: item.Code, Menus: item.Menus}, nil
}

// ListRoles 查询权限重建所需的全部角色字段。
func (s *PermissionStoreAdapter) ListRoles(ctx context.Context) ([]coredata.RoleRecord, error) {
	items, err := s.roleRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]coredata.RoleRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.RoleRecord{ID: item.ID, TenantID: item.TenantID, Code: item.Code, Menus: item.Menus})
	}
	return result, nil
}

// ListRolesByCode 查询指定编码的全部角色字段。
func (s *PermissionStoreAdapter) ListRolesByCode(ctx context.Context, code string) ([]coredata.RoleRecord, error) {
	query := s.roleRepository.Query(ctx).BaseRole
	items, err := query.WithContext(ctx).Where(query.Code.Eq(code)).Find()
	if err != nil {
		return nil, err
	}
	result := make([]coredata.RoleRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.RoleRecord{ID: item.ID, TenantID: item.TenantID, Code: item.Code, Menus: item.Menus})
	}
	return result, nil
}

// UpdateRoleMenus 更新角色的菜单列表。
func (s *PermissionStoreAdapter) UpdateRoleMenus(ctx context.Context, item coredata.RoleRecord) error {
	query := s.roleRepository.Query(ctx).BaseRole
	_, err := query.WithContext(ctx).Where(query.ID.Eq(item.ID)).UpdateSimple(query.Menus.Value(item.Menus))
	return err
}

// ListMenusByIDs 查询权限重建所需的菜单字段。
func (s *PermissionStoreAdapter) ListMenusByIDs(ctx context.Context, ids []int64) ([]coredata.MenuRecord, error) {
	items, err := s.menuRepository.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]coredata.MenuRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.MenuRecord{ID: item.ID, API: item.API})
	}
	return result, nil
}

// ListPolicies 查询全部 Casbin 规则字段。
func (s *PermissionStoreAdapter) ListPolicies(ctx context.Context) ([]coredata.PolicyRecord, error) {
	items, err := s.policyRepository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]coredata.PolicyRecord, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, coredata.PolicyRecord{
			Ptype: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
		})
	}
	return result, nil
}

// ReplacePolicies 替换全部 Casbin 规则。
func (s *PermissionStoreAdapter) ReplacePolicies(ctx context.Context, items []coredata.PolicyRecord) error {
	records := make([]*models.CasbinRule, 0, len(items))
	for _, item := range items {
		records = append(records, &models.CasbinRule{
			Ptype: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
		})
	}
	query := s.policyRepository.Query(ctx).CasbinRule
	err := s.policyRepository.Delete(ctx, repository.Where(query.ID.Gt(0)))
	if err != nil {
		return err
	}
	return s.policyRepository.BatchCreate(ctx, records)
}

var _ coredata.PermissionStore = (*PermissionStoreAdapter)(nil)
