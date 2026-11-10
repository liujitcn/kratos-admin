package biz

import (
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginaudit"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestOauthTicketAuditIdentity 验证跨回调的一次性票据保留可信身份，兑换后才能记录本人协议确认。
func TestOauthTicketAuditIdentity(t *testing.T) {
	cache, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	service := &OauthCase{BaseCase: &biz.BaseCase{Cache: cache}}
	callback := loginaudit.WithCapture(t.Context())
	loginaudit.Record(callback, &data.UserTokenPayload{UserId: 9, TenantId: 2, UserName: "operator"})
	var ticket string
	ticket, err = service.createOauthLoginTicket(callback, &basev1.LoginResponse{Status: basev1.LoginStatus_LOGIN_STATUS_AUTHENTICATED, AccessToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := loginaudit.WithCapture(t.Context())
	var response *basev1.ExchangeOauthTicketResponse
	response, err = service.ExchangeOauthTicket(ctx, &basev1.ExchangeOauthTicketRequest{Ticket: ticket})
	identity := loginaudit.FromContext(ctx)
	if err != nil || response.GetAccessToken() != "test-token" || identity == nil || identity.UserId != 9 || identity.TenantId != 2 {
		t.Fatalf("票据兑换丢失真实身份: %v", err)
	}
	_, err = service.ExchangeOauthTicket(ctx, &basev1.ExchangeOauthTicketRequest{Ticket: ticket})
	if err == nil {
		t.Fatal("同一票据被重复兑换")
	}
}
