package dto

import webauthn "github.com/go-webauthn/webauthn/webauthn"

// MfaLoginChallenge 表示登录阶段暂存的多因素认证挑战。
// 挑战仅用于短期缓存，不包含口令明文；Attempts 由独立的原子计数键维护。
type MfaLoginChallenge struct {
	UserID    int64                 `json:"user_id"`            // 发起挑战的用户编号。
	TenantID  int64                 `json:"tenant_id"`          // 用户所属租户编号。
	ExpiresAt int64                 `json:"expires_at"`         // 挑战过期时间，Unix 秒。
	Method    string                `json:"method"`             // 因子类型，例如 totp 或 webauthn。
	MFAID     int64                 `json:"mfa_id"`             // 绑定的 MFA 配置编号。
	WebAuthn  *webauthn.SessionData `json:"webauthn,omitempty"` // WebAuthn 服务端会话数据。
}

// MfaDisableChallenge 表示禁用 WebAuthn 时暂存的认证挑战。
// 服务端使用用户、租户和 MFA 配置编号做一致性校验，防止跨用户复用挑战。
type MfaDisableChallenge struct {
	UserID    int64                 `json:"user_id"`            // 发起禁用操作的用户编号。
	TenantID  int64                 `json:"tenant_id"`          // 用户所属租户编号。
	MFAID     int64                 `json:"mfa_id"`             // 待禁用 MFA 配置编号。
	ExpiresAt int64                 `json:"expires_at"`         // 挑战过期时间，Unix 秒。
	WebAuthn  *webauthn.SessionData `json:"webauthn,omitempty"` // WebAuthn 服务端会话数据。
}

// MfaSetupTicket 表示 MFA 绑定阶段暂存的票据和注册数据。
// 票据绑定用户和租户，并在短期内保存加密后的 TOTP secret 或 WebAuthn 注册会话。
type MfaSetupTicket struct {
	UserID          int64                 `json:"user_id"`            // 发起绑定的用户编号。
	TenantID        int64                 `json:"tenant_id"`          // 用户所属租户编号。
	EncryptedSecret string                `json:"encrypted_secret"`   // 加密保存的 TOTP secret。
	Method          string                `json:"method"`             // 因子类型，例如 totp 或 webauthn。
	WebAuthn        *webauthn.SessionData `json:"webauthn,omitempty"` // WebAuthn 服务端注册数据。
}
