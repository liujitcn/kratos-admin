package dto

import basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

// OauthLoginTicketPayload 表示三方登录一次性票据缓存的令牌信息。
// 票据只在短时间内使用一次，既可携带最终令牌，也可携带 MFA 或首次绑定的后续步骤状态。
type OauthLoginTicketPayload struct {
	AccessToken     string             `json:"access_token"`              // 登录成功后签发的访问令牌。
	RefreshToken    string             `json:"refresh_token"`             // 登录成功后签发的刷新令牌。
	TokenType       string             `json:"token_type"`                // 令牌类型，通常为 Bearer。
	ExpiresIn       int64              `json:"expires_in"`                // 访问令牌有效期，单位秒。
	Status          basev1.LoginStatus `json:"status"`                    // 当前登录流程状态。
	MfaChallengeID  string             `json:"mfa_challenge_id"`          // 待校验的 MFA 挑战编号。
	MfaSetupTicket  string             `json:"mfa_setup_ticket"`          // 强制绑定 MFA 时使用的一次性票据。
	MfaExpiresIn    int64              `json:"mfa_expires_in"`            // MFA 挑战剩余有效期，单位秒。
	MfaMethod       string             `json:"mfa_method"`                // 待校验的 MFA 方式。
	MfaWebAuthnJSON string             `json:"mfa_webauthn_options_json"` // WebAuthn 前端断言选项 JSON。
}
