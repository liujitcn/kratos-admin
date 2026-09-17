package biz

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/projectaccess"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newGrantTestCase 构造真实四维组织关系和授权表，使用独立数据库验证业务边界。
func newGrantTestCase(t *testing.T) (*BaseTenantProjectGrantCase, *gorm.DB, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Error(err)
		}
	})
	err = db.AutoMigrate(&models.BaseTenant{}, &models.BaseUser{}, &models.BaseRole{}, &models.BaseDept{}, &models.BasePost{}, &models.BaseTenantProject{}, &models.BaseTenantProjectGrant{})
	if err != nil {
		t.Fatal(err)
	}
	rows := []interface{}{
		&models.BaseTenant{ID: 1, Code: "tenant", Name: "租户", Status: 1},
		&models.BaseTenant{ID: 99, Code: kitgorm.DefaultTenantCode, Name: "默认租户", Status: 1},
		&models.BaseRole{ID: 10, TenantID: 1, Code: "viewer", Name: "角色", Status: 1},
		&models.BaseRole{ID: 11, TenantID: 1, Code: "target", Name: "目标角色", Status: 1, Menus: "[]"},
		&models.BaseRole{ID: 90, TenantID: 99, Code: "tenant", Name: "默认租户角色", Status: 1},
		&models.BaseDept{ID: 20, TenantID: 1, Name: "直属部门", Status: 1},
		&models.BaseDept{ID: 21, TenantID: 1, ParentID: 20, Name: "子部门", Status: 1},
		&models.BaseDept{ID: 90, TenantID: 99, Name: "默认租户部门", Status: 1},
		&models.BasePost{ID: 30, TenantID: 1, Code: "post", Name: "岗位", Status: 1},
		&models.BaseUser{ID: 100, TenantID: 1, UserName: "actor", UserCode: "actor", RoleID: 10, DeptID: 20, PostID: 30, Status: 1},
		&models.BaseUser{ID: 101, TenantID: 1, UserName: "target", UserCode: "target", RoleID: 11, DeptID: 21, Status: 1},
		&models.BaseUser{ID: 200, TenantID: 99, UserName: "default-actor", UserCode: "default-actor", RoleID: 90, DeptID: 90, Status: 1},
		&models.BaseTenantProjectGrant{TenantID: 1, SubjectType: 1, SubjectID: 30, ProjectID: "[101]"},
		&models.BaseTenantProjectGrant{TenantID: 1, SubjectType: 2, SubjectID: 10, ProjectID: "[102]"},
		&models.BaseTenantProjectGrant{TenantID: 1, SubjectType: 3, SubjectID: 20, ProjectID: "[103]"},
		&models.BaseTenantProjectGrant{TenantID: 1, SubjectType: 4, SubjectID: 100, ProjectID: "[104]"},
		&models.BaseTenantProjectGrant{TenantID: 1, SubjectType: 3, SubjectID: 21, ProjectID: "[105]"},
	}
	for _, row := range rows {
		if err = db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for id := int64(101); id <= 105; id++ {
		if err = db.Create(&models.BaseTenantProject{ID: id, TenantID: 1, Code: string(rune('A' + id - 101)), Name: "项目", Status: 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	service := NewBaseTenantProjectGrantCase(&biz.BaseCase{}, data.NewTransaction(store), data.NewBaseTenantProjectGrantRepository(store))
	identity := &authdata.UserTokenPayload{TenantId: 1, TenantCode: "tenant", UserId: 100, RoleId: 10, RoleCode: "super", DeptId: 20, DataScope: 1}
	return service, db, engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
}

// TestEffectiveProjectsSources 验证四维合并、直属部门规则及组织关系变更立即生效。
func TestEffectiveProjectsSources(t *testing.T) {
	service, db, ctx := newGrantTestCase(t)
	scope, err := service.EffectiveProjects(ctx)
	if err != nil || !reflect.DeepEqual(scope[1], []int64{101, 102, 103, 104}) {
		t.Fatalf("四维合并错误: %v %v", scope, err)
	}
	err = db.Model(&models.BaseUser{}).Where("id = ?", 100).Update("post_id", 0).Error
	if err != nil {
		t.Fatal(err)
	}
	scope, err = service.EffectiveProjects(ctx)
	if err != nil || !reflect.DeepEqual(scope[1], []int64{102, 103, 104}) {
		t.Fatalf("岗位变更未生效: %v %v", scope, err)
	}
	err = db.Model(&models.BaseTenantProjectGrant{}).Where("tenant_id = ? AND subject_type = ? AND subject_id = ?", 1, 2, 10).Update("project_id", "[0]").Error
	if err != nil {
		t.Fatal(err)
	}
	scope, err = service.EffectiveProjects(ctx)
	if err != nil || !reflect.DeepEqual(scope[1], []int64{0}) {
		t.Fatalf("全部授权未合并: %v %v", scope, err)
	}
}

// TestProjectGrantCRUD 验证项目授权新增、更新、查询、删除及租户边界。
func TestProjectGrantCRUD(t *testing.T) {
	service, _, ctx := newGrantTestCase(t)
	roleGrant := &adminv1.BaseTenantProjectGrant{TenantId: 1, SubjectType: adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_ROLE, SubjectId: 11, ProjectId: []int64{105}}
	if err := service.CreateBaseTenantProjectGrant(ctx, roleGrant); err != nil {
		t.Fatalf("合法角色授权失败: %v", err)
	}
	if err := service.CreateBaseTenantProjectGrant(ctx, roleGrant); err == nil {
		t.Fatal("重复新增应失败")
	}
	userGrant := &adminv1.BaseTenantProjectGrant{TenantId: 1, SubjectType: adminv1.BaseTenantProjectGrantSubjectType_BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_USER, SubjectId: 101}
	userGrant.ProjectId = []int64{104}
	if err := service.CreateBaseTenantProjectGrant(ctx, userGrant); err != nil {
		t.Fatalf("合法用户授权失败: %v", err)
	}
	saved, err := service.GetBaseTenantProjectGrant(ctx, &adminv1.GetBaseTenantProjectGrantRequest{TenantId: 1, SubjectType: userGrant.SubjectType, SubjectId: userGrant.SubjectId})
	if err != nil || !reflect.DeepEqual(saved.ProjectId, []int64{104}) {
		t.Fatalf("联合主键查询失败: %v %v", saved, err)
	}
	userGrant.ProjectId = []int64{101}
	if err = service.UpdateBaseTenantProjectGrant(ctx, userGrant); err != nil {
		t.Fatalf("更新项目授权失败: %v", err)
	}
	if err = service.DeleteBaseTenantProjectGrant(ctx, &adminv1.DeleteBaseTenantProjectGrantRequest{TenantId: 1, SubjectType: userGrant.SubjectType, SubjectId: userGrant.SubjectId}); err != nil {
		t.Fatalf("删除项目授权失败: %v", err)
	}
	if _, err = service.GetBaseTenantProjectGrant(ctx, &adminv1.GetBaseTenantProjectGrantRequest{TenantId: 1, SubjectType: userGrant.SubjectType, SubjectId: userGrant.SubjectId}); err == nil {
		t.Fatal("删除后查询应返回不存在")
	}
}

// TestDefaultTenantProjectScope 验证默认租户不依赖四维授权即可获得全部租户项目范围。
func TestDefaultTenantProjectScope(t *testing.T) {
	service, _, _ := newGrantTestCase(t)
	identity := &authdata.UserTokenPayload{TenantId: 99, TenantCode: kitgorm.DefaultTenantCode, UserId: 200, RoleId: 90, RoleCode: "tenant", DeptId: 90, DataScope: 1}
	ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
	scope, err := service.EffectiveProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scope[1], []int64{0}) {
		t.Fatalf("默认租户项目范围错误: %v", scope)
	}
}

// TestProjectCatalogManagement 验证普通租户可在自身租户创建项目但不能跨租户创建。
func TestProjectCatalogManagement(t *testing.T) {
	grantService, db, normalCtx := newGrantTestCase(t)
	store, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	projectService := NewBaseTenantProjectCase(projectaccess.NewLifecycle(), &biz.BaseCase{}, grantService, data.NewTransaction(store), data.NewBaseTenantProjectRepository(store))
	options, err := projectService.OptionBaseTenantProject(normalCtx, &adminv1.OptionBaseTenantProjectRequest{})
	if err != nil || len(options.List) != 5 {
		t.Fatalf("普通租户项目目录错误: len=%d err=%v", len(options.List), err)
	}
	form := &adminv1.BaseTenantProjectForm{TenantId: 1, Code: "normal-create", Name: "普通租户创建", Sort: 1}
	if err = projectService.CreateBaseTenantProject(normalCtx, form); err != nil {
		t.Fatalf("普通租户应可在自身租户创建项目: %v", err)
	}
	crossTenantForm := &adminv1.BaseTenantProjectForm{TenantId: 99, Code: "cross-tenant-create", Name: "跨租户创建", Sort: 1}
	if err = projectService.CreateBaseTenantProject(normalCtx, crossTenantForm); err == nil {
		t.Fatal("普通租户不应跨租户创建项目")
	}
}

// TestPageProjectGrant 验证分页列表补齐主体、项目和联合键展示字段。
func TestPageProjectGrant(t *testing.T) {
	service, _, ctx := newGrantTestCase(t)
	result, err := service.PageBaseTenantProjectGrant(ctx, &adminv1.PageBaseTenantProjectGrantRequest{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("查询项目授权列表失败: %v", err)
	}
	if result.Total != 5 || len(result.Grants) != 5 {
		t.Fatalf("分页结果错误: total=%d len=%d", result.Total, len(result.Grants))
	}
	if result.Grants[0].GrantKey == "" || result.Grants[0].SubjectName == "" || len(result.Grants[0].ProjectNames) != 1 {
		t.Fatalf("列表展示字段未补齐: %+v", result.Grants[0])
	}
}
