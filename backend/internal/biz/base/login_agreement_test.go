package biz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/adminagreement"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginaudit"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/query"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAdminAgreementDoesNotUseAppConfiguration 验证管理端缺少独立协议时不回退到移动端配置。
func TestAdminAgreementDoesNotUseAppConfiguration(t *testing.T) {
	c := newAgreementLoginCase(t)
	var err error
	for _, item := range []*models.BaseConfig{
		{Site: 3, Key: "serviceProtocol", Value: "应用服务正文", Status: 1},
		{Site: 3, Key: "privacyProtocol", Value: "应用隐私正文", Status: 1},
	} {
		err = c.baseConfigRepo.Create(context.Background(), item)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = c.ConfirmAdminAgreement(context.Background(), &basev1.AdminAgreementConsent{Accepted: true, ServiceVersion: "v1", PrivacyVersion: "v1"})
	if err == nil || errors.FromError(err).Reason != "PERMISSION_DENIED" {
		t.Fatalf("管理端缺少专属协议时未阻止登录: %v", err)
	}
}

// TestAdminMfaMissingAgreementFailsClosed 验证管理端挑战丢失确认字段时不能降级为移动端登录。
func TestAdminMfaMissingAgreementFailsClosed(t *testing.T) {
	c := newAgreementLoginCase(t)
	raw := `{"user_id":42,"tenant_id":7,"expires_at":4102444800,"method":"totp","admin_login":true}`
	err := c.Cache.Set(mfaLoginChallengeKey("challenge"), raw, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("缺少管理端确认时仍进入 MFA 用户认证: %v", recovered)
		}
	}()
	_, err = c.VerifyMfa(context.Background(), &basev1.VerifyMfaRequest{ChallengeId: "challenge", Code: "123456"})
	if err == nil || errors.FromError(err).Reason != "PERMISSION_DENIED" {
		t.Fatalf("管理端挑战缺少协议确认时未拒绝: %v", err)
	}
}

// TestMfaChallengeSeparatesAdminAgreement 验证管理端确认与 MFA 挑战原子保存，移动端挑战不附带管理端内容。
func TestMfaChallengeSeparatesAdminAgreement(t *testing.T) {
	c := newAgreementLoginCase(t)
	receipt := adminagreement.Receipt{ServiceVersion: "v1", PrivacyVersion: "v1", AcceptedAt: time.Now()}
	for _, adminLogin := range []bool{true, false} {
		ctx := context.Background()
		if adminLogin {
			ctx = adminagreement.WithReceipt(ctx, receipt)
		}
		response, err := c.mfaCase.beginMfaLoginChallenge(ctx, &models.BaseUser{ID: 42, TenantID: 7}, "totp", 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		var challenge dto.MfaLoginChallenge
		err = c.mfaCase.loadJSON(mfaLoginChallengeKey(response.GetMfaChallengeId()), &challenge)
		if err != nil {
			t.Fatal(err)
		}
		if challenge.AdminLogin != adminLogin || (challenge.AdminAgreement != nil) != adminLogin {
			t.Fatalf("MFA 挑战的终端或确认信息错误: admin=%v, challenge=%+v", adminLogin, challenge)
		}
	}
}

// TestAdminOauthExchangePreservesRefreshIdentity 验证协议审计只读刷新身份，并保留原确认时间。
func TestAdminOauthExchangePreservesRefreshIdentity(t *testing.T) {
	c := newAgreementLoginCase(t)
	var err error
	for key, value := range map[string]string{
		"adminServiceProtocol": "管理端服务正文", "adminServiceProtocolVersion": "v1",
		"adminPrivacyProtocol": "管理端隐私正文", "adminPrivacyProtocolVersion": "v1",
	} {
		err = c.baseConfigRepo.Create(context.Background(), &models.BaseConfig{Site: 2, Key: key, Value: value, Status: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := loginaudit.WithCapture(adminagreement.WithCapture(context.Background()))
	var receipt adminagreement.Receipt
	receipt, err = c.ConfirmAdminAgreement(ctx, &basev1.AdminAgreementConsent{Accepted: true, ServiceVersion: "v1", PrivacyVersion: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	identity := `{"TenantId":7,"TenantCode":"0000","UserId":42,"UserCode":"U42","UserName":"manager","RoleId":8,"RoleCode":"admin","RoleName":"管理员","DeptId":9,"DeptName":"运营","DataScope":1,"TenantProjects":[]}`
	if err = c.Cache.Set(refreshTokenAuthKey("refresh"), identity, time.Minute); err != nil {
		t.Fatal(err)
	}
	oauthCase := &OauthCase{BaseCase: c.BaseCase, loginCase: c}
	var ticket string
	ticket, err = oauthCase.createOauthLoginTicket(adminagreement.WithReceipt(ctx, receipt), &basev1.LoginResponse{AccessToken: "access", RefreshToken: "refresh", Status: basev1.LoginStatus_LOGIN_STATUS_AUTHENTICATED})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("兑换协议确认票据调用了刷新令牌的消费认证: %v", recovered)
		}
	}()
	_, err = oauthCase.ExchangeOauthTicket(ctx, &basev1.ExchangeOauthTicketRequest{Ticket: ticket})
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	raw, err = c.Cache.Get(refreshTokenAuthKey("refresh"))
	if err != nil || raw != identity {
		t.Fatalf("协议审计消费或改写了刷新身份: %q, %v", raw, err)
	}
	if captured := adminagreement.FromContext(ctx); captured == nil || !captured.AcceptedAt.Equal(receipt.AcceptedAt) {
		t.Fatalf("OAuth 兑换未提交原始确认时间: %+v", captured)
	}
	if captured := loginaudit.FromContext(ctx); captured == nil || captured.UserId != 42 {
		t.Fatalf("OAuth 协议确认未绑定认证用户: %+v", captured)
	}
}

// TestAdminOauthTicketMissingAgreementFailsClosed 验证管理端票据缺少确认时不能降级为普通票据。
func TestAdminOauthTicketMissingAgreementFailsClosed(t *testing.T) {
	c := newAgreementLoginCase(t)
	raw := `{"access_token":"access","refresh_token":"refresh","status":1,"admin_login":true}`
	err := c.Cache.Set(oauthLoginTicketKey("ticket"), raw, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	oauthCase := &OauthCase{BaseCase: c.BaseCase, loginCase: c}
	_, err = oauthCase.ExchangeOauthTicket(context.Background(), &basev1.ExchangeOauthTicketRequest{Ticket: "ticket"})
	if err == nil || errors.FromError(err).Reason != "PERMISSION_DENIED" {
		t.Fatalf("管理端票据缺少协议确认时未拒绝: %v", err)
	}
}

// newAgreementLoginCase 创建独立 SQLite 配置仓库和内存挑战缓存。
func newAgreementLoginCase(t *testing.T) *LoginCase {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("关闭协议测试数据库失败: %v", closeErr)
		}
	})
	if err = db.AutoMigrate(&models.BaseConfig{}); err != nil {
		t.Fatal(err)
	}
	var store *memory.Memory
	var cleanup func()
	store, cleanup, err = memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	baseCase := &biz.BaseCase{Cache: store}
	return &LoginCase{
		BaseCase:       baseCase,
		baseConfigRepo: data.NewBaseConfigRepository(agreementQueryProvider{query: query.Use(db)}),
		mfaCase:        &MfaCase{BaseCase: baseCase, runtimeConfig: newMfaRuntimeConfig(nil)},
	}
}

type agreementQueryProvider struct {
	query *query.Query
}

// Query 为协议测试提供真实的类型化配置查询。
func (p agreementQueryProvider) Query(context.Context) *query.Query {
	return p.query
}
