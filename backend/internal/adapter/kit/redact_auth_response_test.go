package kit

import (
	"context"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-kit/redact"
)

// TestAuthenticationResponsesKeepClientCredentials 验证登录响应中的客户端凭据不会被动态响应脱敏清空。
func TestAuthenticationResponsesKeepClientCredentials(t *testing.T) {
	resolver := NewRedactPolicyResolver(nil, nil, nil)
	resolver.loadedAt = time.Now()
	loginResponse := &basev1.LoginResponse{AccessToken: "access-token", RefreshToken: "refresh-token", TokenType: "Bearer", ExpiresIn: 3600}
	refreshResponse := &basev1.RefreshTokenResponse{AccessToken: "refreshed-access-token", RefreshToken: "refreshed-refresh-token", TokenType: "Bearer", ExpiresIn: 3600}
	captchaResponse := &basev1.VerifyCaptchaResponse{CaptchaToken: "captcha-token", ExpiresIn: 120}

	redact.ApplyWith(
		redact.WithDirection(redact.WithOperation(context.Background(), "/base.v1.LoginService/Login"), redact.DirectionResponse),
		resolver,
		loginResponse,
	)
	redact.ApplyWith(
		redact.WithDirection(redact.WithOperation(context.Background(), "/base.v1.LoginService/RefreshToken"), redact.DirectionResponse),
		resolver,
		refreshResponse,
	)
	redact.ApplyWith(
		redact.WithDirection(redact.WithOperation(context.Background(), "/base.v1.LoginService/VerifyCaptcha"), redact.DirectionResponse),
		resolver,
		captchaResponse,
	)

	if loginResponse.AccessToken != "access-token" || loginResponse.RefreshToken != "refresh-token" {
		t.Fatalf("login credentials were redacted: access=%q refresh=%q", loginResponse.AccessToken, loginResponse.RefreshToken)
	}
	if refreshResponse.AccessToken != "refreshed-access-token" || refreshResponse.RefreshToken != "refreshed-refresh-token" {
		t.Fatalf("refresh credentials were redacted: access=%q refresh=%q", refreshResponse.AccessToken, refreshResponse.RefreshToken)
	}
	if captchaResponse.CaptchaToken != "captcha-token" {
		t.Fatalf("captcha token was redacted: %q", captchaResponse.CaptchaToken)
	}
}
