package biz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUpdateBaseRedactOutputPolicyPreservesTenant 验证编辑响应脱敏策略时忽略请求中的租户漂移。
func TestUpdateBaseRedactOutputPolicyPreservesTenant(t *testing.T) {
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
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseAPI{}, &models.BaseRedactRule{}, &models.BaseRedactOutputPolicy{}); err != nil {
		t.Fatal(err)
	}
	operation := "/system.admin.v1.BaseRedactOutputPolicyService/GetBaseRedactOutputPolicy"
	serviceName := "system.admin.v1.BaseRedactOutputPolicyService"
	if err = db.Create(&models.BaseAPI{ID: 1, Operation: operation, ServiceName: serviceName, Method: "GET", TenantResponse: true}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	policy := &models.BaseRedactOutputPolicy{ID: 10, TenantID: 2, Operation: operation, ServiceName: serviceName, MessageRef: "system.admin.v1.BaseRedactOutputPolicyForm", FieldPath: "remark", Mode: int32(adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_FULL), RuleParams: "{}", Status: int32(commonv1.Status_STATUS_ENABLE), Remark: "旧备注", CreatedBy: 1, UpdatedBy: 1, CreatedAt: now, UpdatedAt: now}
	if err = db.Create(policy).Error; err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	policyCase := NewBaseRedactOutputPolicyCase(&biz.BaseCase{}, data.NewTransaction(store), data.NewBaseRedactOutputPolicyRepository(store), data.NewBaseAPIRepository(store), nil, data.NewBaseRedactRuleRepository(store), nil)
	identity := &authdata.UserTokenPayload{TenantId: 1, TenantCode: kitgorm.DefaultTenantCode, UserId: 1, RoleCode: "super"}
	ctx := engine.ContextWithAuthClaims(context.Background(), identity.MakeAuthClaims())
	input := &adminv1.BaseRedactOutputPolicyForm{Id: policy.ID, TenantId: 1, Operation: operation, ServiceName: serviceName, MessageRef: policy.MessageRef, FieldPath: policy.FieldPath, Mode: adminv1.BaseRedactOutputPolicyMode_BASE_REDACT_OUTPUT_POLICY_MODE_FULL, RuleParams: "{}", Status: commonv1.Status_STATUS_ENABLE, Remark: "新备注"}
	if err = policyCase.UpdateBaseRedactOutputPolicy(ctx, []*adminv1.BaseRedactOutputPolicyForm{input}); err != nil {
		t.Fatalf("编辑响应脱敏策略失败: %v", err)
	}
	var updated models.BaseRedactOutputPolicy
	if err = db.First(&updated, policy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.TenantID != policy.TenantID {
		t.Fatalf("响应脱敏策略租户被修改: %d", updated.TenantID)
	}
	if updated.Remark != input.Remark {
		t.Fatalf("响应脱敏策略备注 = %q, want %q", updated.Remark, input.Remark)
	}
}
