package biz

import (
	"path/filepath"
	"testing"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestTenantMaskedEdit 验证脱敏联系人可以原样保存，真实变更仍校验号码且不污染请求。
func TestTenantMaskedEdit(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseTenant), new(models.BaseRedactRule), new(models.BaseRedactStoragePolicy), new(models.BaseRedactOutputPolicy))
	row := &models.BaseTenant{ID: 2, Code: "1000", Name: "原租户", ContactName: "张三", ContactPhone: "19999911111", Status: 1}
	err := client.DB.Create(row).Error
	if err != nil {
		t.Fatal(err)
	}
	err = client.DB.Create(&models.BaseRedactRule{ID: 2, Code: "mask", RuleType: "MASK", DefaultParams: `{"mask":{"keep_first":1,"keep_last":0,"mask_char":"*"}}`, Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	for index, column := range []string{"contact_name", "contact_phone"} {
		params := `{"mask":{"keep_first":1,"keep_last":0,"mask_char":"*"}}`
		if column == "contact_phone" {
			params = `{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}`
		}
		err = client.DB.Create(&models.BaseRedactOutputPolicy{ID: int64(30 + index), TenantID: 1, Operation: "/system.admin.v1.BaseTenantService/GetBaseTenant", ServiceName: "system.admin.v1.BaseTenantService", MessageRef: "system.admin.v1.BaseTenantForm", FieldPath: column, Mode: 1, RuleID: 2, RuleParams: params, Status: 1}).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	var resolver *kit.RedactPolicyResolver
	resolver, err = kit.NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	err = resolver.Refresh(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	identity := &authdata.UserTokenPayload{UserId: 1, TenantId: 1, TenantCode: kitgorm.DefaultTenantCode, RoleCode: "admin"}
	ctx := redact.WithDirection(engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims()), redact.DirectionRequest)
	service := &BaseTenantCase{BaseCase: &biz.BaseCase{}, BaseTenantRepository: data.NewBaseTenantRepository(database), redactResolver: resolver, formMapper: mapper.NewCopierMapper[adminv1.BaseTenantForm, models.BaseTenant]()}
	form := &adminv1.BaseTenantForm{Id: 2, Code: "ignored", Name: "修改名称", ContactName: "张*", ContactPhone: "199****1111", Status: new(commonv1.Status_STATUS_ENABLE)}
	err = service.UpdateBaseTenant(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	var stored *models.BaseTenant
	stored, err = service.FindByID(ctx, 2)
	if err != nil || stored.ContactName != row.ContactName || stored.ContactPhone != row.ContactPhone || stored.Code != row.Code || stored.Name != form.Name || form.ContactPhone != "199****1111" {
		t.Fatalf("脱敏编辑破坏原值或请求: %+v, %v", stored, err)
	}
	form.ContactPhone = "bad-phone"
	if service.UpdateBaseTenant(ctx, form) == nil {
		t.Fatal("无效新号码被接受")
	}
	form.ContactPhone = "18888822222"
	err = service.UpdateBaseTenant(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	stored, err = service.FindByID(ctx, 2)
	if err != nil || stored.ContactPhone != form.ContactPhone {
		t.Fatalf("合法新号码无法保存: %v", err)
	}
}

// TestDefaultTenantProjectScope 验证默认租户普通账号按实际授权计算，平台管理员继续拥有全部项目。
func TestDefaultTenantProjectScope(t *testing.T) {
	client, database := newSecurityData(t, new(models.BaseUser), new(models.BaseRole), new(models.BaseDept), new(models.BasePost), new(models.BaseTenant), new(models.BaseTenantProjectGrant))
	var err error
	for _, row := range []any{
		&models.BaseTenant{ID: 1, Code: kitgorm.DefaultTenantCode, Status: 1},
		&models.BaseTenant{ID: 2, Code: "1000", Status: 1},
		&models.BaseRole{ID: 5, TenantID: 1, Code: "operator", Status: 1},
		&models.BaseDept{ID: 3, TenantID: 1, Status: 1},
		&models.BaseUser{ID: 9, TenantID: 1, RoleID: 5, DeptID: 3, Status: 1},
		&models.BaseTenantProjectGrant{TenantID: 2, SubjectType: 4, SubjectID: 9, ProjectID: "[7]"},
	} {
		err = client.DB.Create(row).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	identity := &authdata.UserTokenPayload{UserId: 9, TenantId: 1, TenantCode: kitgorm.DefaultTenantCode, RoleCode: "operator"}
	ctx := engine.ContextWithAuthClaims(t.Context(), identity.MakeAuthClaims())
	service := NewBaseTenantProjectGrantCase(&biz.BaseCase{}, data.NewTransaction(database), data.NewBaseTenantProjectGrantRepository(database))
	var allowed map[int64][]int64
	allowed, err = service.EffectiveProjects(ctx)
	if err != nil || len(allowed) != 1 || len(allowed[2]) != 1 || allowed[2][0] != 7 {
		t.Fatalf("普通账号权限未按真实授权计算: %v, %v", allowed, err)
	}
	table := database.Query(ctx).BaseRole
	_, err = table.WithContext(ctx).Where(table.ID.Eq(5)).UpdateSimple(table.Code.Value("admin"))
	if err != nil {
		t.Fatal(err)
	}
	allowed, err = service.EffectiveProjects(ctx)
	if err != nil || len(allowed) != 2 || allowed[1][0] != 0 || allowed[2][0] != 0 {
		t.Fatalf("平台管理员完整权限回归: %v, %v", allowed, err)
	}
}

// newSecurityData 为安全业务回归建立独立 SQLite 数据库，避免连接真实业务环境。
func newSecurityData(t *testing.T, registered ...any) (*kitgorm.Client, *data.Data) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "security.db")), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range registered {
		statement := &gorm.Statement{DB: db}
		err = statement.Parse(model)
		if err != nil {
			t.Fatal(err)
		}
		if field := statement.Schema.FieldsByName["ID"]; field != nil {
			field.DataType = schema.Int
		}
	}
	err = db.AutoMigrate(registered...)
	if err != nil {
		t.Fatal(err)
	}
	client := &kitgorm.Client{DB: db}
	var database *data.Data
	database, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	return client, database
}
