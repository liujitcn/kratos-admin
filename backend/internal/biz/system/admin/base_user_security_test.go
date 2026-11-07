package biz

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	corebiz "github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestCreateBaseUserRejectsReservedName 验证保留账号在访问数据库和密码策略前被拒绝。
func TestCreateBaseUserRejectsReservedName(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("保留账号未在业务边界拒绝，已访问其他依赖: %v", recovered)
		}
	}()
	userCase := &BaseUserCase{}
	err := userCase.CreateBaseUser(context.Background(), &adminv1.BaseUserForm{UserName: "AdMiN"})
	if err == nil || errors.Code(err) != 400 {
		t.Fatalf("保留账号必须返回参数错误，实际 %v", err)
	}
}

// TestTenantCreationRecognizesRenamedAdministrator 验证审批重试按内置角色识别已初始化租户。
func TestTenantCreationRecognizesRenamedAdministrator(t *testing.T) {
	storage := newUserSecurityStorage(t)
	var err error
	tenant := &models.BaseTenant{ID: 7, Code: "1007", Status: _const.STATUS_STATUS_DISABLE}
	role := &models.BaseRole{ID: 8, TenantID: 7, Code: _const.BASE_ROLE_CODE_TENANT, Menus: "[]"}
	user := &models.BaseUser{ID: 42, TenantID: 7, RoleID: 8, UserName: "r4nd0m_account", UserCode: "r4nd0m_account", PasswordHistory: "[]"}
	err = data.NewBaseTenantRepository(storage).Create(context.Background(), tenant)
	if err != nil {
		t.Fatal(err)
	}
	err = data.NewBaseRoleRepository(storage).Create(context.Background(), role)
	if err != nil {
		t.Fatal(err)
	}
	err = data.NewBaseUserRepository(storage).Create(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	tenantCase := &BaseTenantCase{
		tx: storage, BaseTenantRepository: data.NewBaseTenantRepository(storage),
		baseRoleRepo: data.NewBaseRoleRepository(storage), baseUserRepo: data.NewBaseUserRepository(storage),
	}
	_, err = tenantCase.completeTenantCreation(context.Background(), 7, 0, 0, 0, _const.STATUS_STATUS_ENABLE, "")
	if err != nil {
		t.Fatalf("已存在非固定名称管理员时审批重试失败: %v", err)
	}
	var persisted *models.BaseTenant
	persisted, err = tenantCase.FindByID(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != _const.STATUS_STATUS_ENABLE {
		t.Fatalf("已初始化租户未启用: %+v", persisted)
	}
}

// TestUserProfileChecksRenamedAccount 验证存量保留账号可以维护资料，改名时仍执行新规则。
func TestUserProfileChecksRenamedAccount(t *testing.T) {
	tests := []struct {
		name     string
		oldName  string
		newName  string
		rejected bool
	}{
		{"存量账号维护", "admin", "admin", false},
		{"改为保留账号", "alice", "ROOT", true},
		{"存量账号迁移", "admin", "alice", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := newUserSecurityStorage(t)
			userRepo := data.NewBaseUserRepository(storage)
			user := &models.BaseUser{ID: 42, TenantID: 7, UserName: test.oldName, UserCode: "unchanged-code", Avatar: "avatar.png", PasswordHistory: "[]"}
			err := userRepo.Create(context.Background(), user)
			if err != nil {
				t.Fatal(err)
			}
			identity := authData.UserTokenPayload{TenantId: 7, TenantCode: "1007", UserId: 42, UserName: test.oldName}
			ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
			authCase := &AuthCase{
				BaseCase: &corebiz.BaseCase{}, baseUserCase: &BaseUserCase{BaseUserRepository: userRepo}, fileCase: &biz.FileCase{},
			}
			err = authCase.UpdateUserProfile(ctx, &adminv1.UserProfileForm{UserName: test.newName, NickName: "更新昵称", Avatar: "avatar.png"})
			if test.rejected {
				if err == nil || errors.Code(err) != 400 {
					t.Fatalf("改为保留名必须返回参数错误，实际 %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var persisted *models.BaseUser
			persisted, err = userRepo.FindByID(context.Background(), 42)
			if err != nil {
				t.Fatal(err)
			}
			wantName := test.newName
			if test.rejected {
				wantName = test.oldName
			}
			if persisted.UserName != wantName || persisted.UserCode != "unchanged-code" {
				t.Fatalf("账号更新超出预期: %+v", persisted)
			}
		})
	}
}

// newUserSecurityStorage 创建隔离的真实用户、角色与租户仓储供安全边界测试复用。
func newUserSecurityStorage(t *testing.T) *data.Data {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var sqlDatabase *sql.DB
	sqlDatabase, err = database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDatabase.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	err = database.AutoMigrate(&models.BaseTenant{}, &models.BaseRole{}, &models.BaseUser{})
	if err != nil {
		t.Fatal(err)
	}
	var storage *data.Data
	storage, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: database}})
	if err != nil {
		t.Fatal(err)
	}
	return storage
}
