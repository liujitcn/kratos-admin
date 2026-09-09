package biz

import (
	"context"
	"fmt"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestOauthLoginTicketPreservesMfaRememberDays 验证三方登录票据缓存及兑换完整保留设备免验证策略。
func TestOauthLoginTicketPreservesMfaRememberDays(t *testing.T) {
	for _, days := range []int32{0, 14} {
		t.Run(fmt.Sprintf("days_%d", days), func(t *testing.T) {
			store, cleanup, err := memory.NewMemory()
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			oauthCase := &OauthCase{BaseCase: &biz.BaseCase{Cache: store}}
			var ticket string
			ticket, err = oauthCase.createOauthLoginTicket(&basev1.LoginResponse{
				Status:          basev1.LoginStatus_LOGIN_STATUS_MFA_REQUIRED,
				MfaChallengeId:  "mfa-challenge",
				MfaMethod:       "totp",
				MfaRememberDays: days,
			})
			if err != nil {
				t.Fatal(err)
			}
			var response *basev1.ExchangeOauthTicketResponse
			response, err = oauthCase.ExchangeOauthTicket(context.Background(), &basev1.ExchangeOauthTicketRequest{Ticket: ticket})
			if err != nil {
				t.Fatal(err)
			}
			if response.GetMfaRememberDays() != days {
				t.Fatalf("设备免验证天数丢失：得到 %d，期望 %d", response.GetMfaRememberDays(), days)
			}
			if response.GetStatus() != basev1.LoginStatus_LOGIN_STATUS_MFA_REQUIRED || response.GetMfaChallengeId() != "mfa-challenge" {
				t.Fatal("MFA 挑战状态或编号丢失")
			}
		})
	}
}
