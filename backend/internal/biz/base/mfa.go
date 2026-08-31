package biz

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/go-webauthn/webauthn/protocol"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/utils"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen"
	"gorm.io/gorm"
)

const (
	mfaMethodTOTP                  = "totp"
	mfaMethodWebAuthn              = "webauthn"
	mfaStatusEnabled               = int32(1)
	mfaStatusDisabled              = int32(2)
	mfaPolicyDisabled              = "disabled"
	mfaPolicyOptional              = "optional"
	mfaPolicyAllRequired           = "all_required"
	mfaLoginChallengePrefix        = "mfa:login:"
	mfaSetupTicketPrefix           = "mfa:setup:"
	defaultMfaLoginChallengeExpire = 5 * time.Minute
	defaultMfaSetupTicketExpire    = 10 * time.Minute
	defaultMfaLoginMaxAttempts     = 5
	defaultMfaRecoveryCodeCount    = 10
	defaultMfaRecoveryCodeLength   = 10
	defaultMfaTotpIssuer           = "Kratos Admin"
	defaultMfaTotpPeriod           = 30
	defaultMfaTotpSkew             = 1
	defaultMfaTotpSecretSize       = 20
	mfaDisableChallengePrefix      = "mfa:disable:"
)

var mfaRecoveryAlphabet = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

// MfaCase 处理 TOTP、WebAuthn 多因素认证、恢复码和登录挑战。
type MfaCase struct {
	*biz.BaseCase
	tx                      data.Transaction
	baseUserMFARepo         *data.BaseUserMFARepository
	baseUserMFARecoveryRepo *data.BaseUserMFARecoveryRepository
	baseUserMFATotpRepo     *data.BaseUserMFATotpRepository
	baseUserMFAWebauthnRepo *data.BaseUserMFAWebauthnRepository
	baseUserCase            *BaseUserCase
	configCase              *ConfigCase
	userToken               *authData.UserToken
	runtimeConfig           mfaRuntimeConfig
	webAuthn                *webauthn.WebAuthn
	webAuthnErr             error
}

// mfaRuntimeConfig 汇总 MFA 运行时安全参数和 WebAuthn 依赖方配置。
// 该配置由 Bootstrap 解析并在构造 Case 时固定，避免请求期间被客户端输入覆盖。
type mfaRuntimeConfig struct {
	encryptionKey        string        // TOTP secret 的服务端加密密钥。
	loginChallengeExpire time.Duration // 登录 MFA 挑战的最大有效期。
	setupTicketExpire    time.Duration // MFA 绑定票据的最大有效期。
	loginMaxAttempts     int           // 单个登录挑战允许的最大失败次数。
	recoveryCodeCount    int           // 每次生成的恢复码数量。
	recoveryCodeLength   int           // 单个恢复码的字符长度。
	totpIssuer           string        // TOTP 客户端展示的发行方名称。
	totpPeriod           uint          // TOTP 动态码的时间步长。
	totpSkew             uint          // 允许的 TOTP 时间窗口偏移量。
	totpSecretSize       uint          // 新建 TOTP secret 的字节长度。
	totpDigits           otp.Digits    // TOTP 动态码位数。
	totpAlgorithm        otp.Algorithm // TOTP 使用的哈希算法。
	webauthnRPID         string        // WebAuthn 依赖方 ID。
	webauthnOrigins      []string      // WebAuthn 允许的来源列表。
}

// mfaWebAuthnUser 将系统用户转换为 WebAuthn 用户模型。
type mfaWebAuthnUser struct {
	user        *models.BaseUser
	credentials []webauthn.Credential
}

// WebAuthnID 返回稳定的 WebAuthn 用户句柄。
func (u *mfaWebAuthnUser) WebAuthnID() []byte {
	return []byte(strconv.FormatInt(u.user.ID, 10))
}

// WebAuthnName 返回 WebAuthn 用户名。
func (u *mfaWebAuthnUser) WebAuthnName() string {
	return u.user.UserName
}

// WebAuthnDisplayName 返回 WebAuthn 显示名称。
func (u *mfaWebAuthnUser) WebAuthnDisplayName() string {
	if u.user.NickName != "" {
		return u.user.NickName
	}
	return u.user.UserName
}

// WebAuthnCredentials 返回当前用户已绑定的 WebAuthn 凭据。
func (u *mfaWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// NewMfaCase 创建多因素认证业务实例。
func NewMfaCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseUserMFARepo *data.BaseUserMFARepository,
	baseUserMFARecoveryRepo *data.BaseUserMFARecoveryRepository,
	baseUserMFATotpRepo *data.BaseUserMFATotpRepository,
	baseUserMFAWebauthnRepo *data.BaseUserMFAWebauthnRepository,
	baseUserCase *BaseUserCase,
	configCase *ConfigCase,
	userToken *authData.UserToken,
	mfaConfig *configv1.Mfa,
) *MfaCase {
	runtimeConfig := newMfaRuntimeConfig(mfaConfig)
	webAuthnConfig, webAuthnErr := newMfaWebAuthn(runtimeConfig)
	return &MfaCase{
		BaseCase:                baseCase,
		tx:                      tx,
		baseUserMFARepo:         baseUserMFARepo,
		baseUserMFARecoveryRepo: baseUserMFARecoveryRepo,
		baseUserMFATotpRepo:     baseUserMFATotpRepo,
		baseUserMFAWebauthnRepo: baseUserMFAWebauthnRepo,
		baseUserCase:            baseUserCase,
		configCase:              configCase,
		userToken:               userToken,
		runtimeConfig:           runtimeConfig,
		webAuthn:                webAuthnConfig,
		webAuthnErr:             webAuthnErr,
	}
}

// PrepareLogin 根据全局策略和用户 MFA 状态准备登录结果。
func (c *MfaCase) PrepareLogin(ctx context.Context, user *models.BaseUser) (*basev1.LoginResponse, error) {
	var err error
	var policy string
	policy, err = c.policy(ctx)
	if err != nil {
		return nil, err
	}
	if policy == mfaPolicyDisabled {
		return nil, nil
	}
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return nil, err
	}
	method, err = c.effectiveMethod(ctx, user.ID, method)
	if err != nil {
		return nil, err
	}
	method, err = c.resolveEnabledMethod(ctx, user.ID, method)
	if err != nil {
		return nil, err
	}
	var enabled bool
	enabled, err = c.hasEnabledMFA(ctx, user.ID, method)
	if err != nil {
		return nil, err
	}
	if policy == mfaPolicyOptional && !enabled {
		return nil, nil
	}
	if enabled {
		if method == mfaMethodWebAuthn {
			if !mfaWebAuthnAvailable(ctx) {
				var mfa *models.BaseUserMFA
				mfa, err = c.findEnabledMFA(ctx, user.ID, method)
				if err != nil {
					return nil, errorsx.Internal("查询 WebAuthn 多因素认证配置失败").WithCause(err)
				}
				return c.beginMfaLoginChallenge(user, method, mfa.ID)
			}
			return c.beginWebAuthnLogin(ctx, user)
		}
		return c.beginMfaLoginChallenge(user, method, 0)
	}
	if policy == mfaPolicyAllRequired {
		setupTicket := id.NewGUIDv4NoHyphen()
		payload := dto.MfaSetupTicket{UserID: user.ID, TenantID: user.TenantID, Method: method}
		if err = c.saveJSON(mfaSetupTicketKey(setupTicket), payload, c.runtimeConfig.setupTicketExpire); err != nil {
			return nil, errorsx.Internal("创建多因素认证绑定票据失败").WithCause(err)
		}
		return &basev1.LoginResponse{
			Status:         basev1.LoginStatus_LOGIN_STATUS_MFA_ENROLLMENT_REQUIRED,
			MfaSetupTicket: setupTicket,
			MfaExpiresIn:   int64(c.runtimeConfig.setupTicketExpire / time.Second),
			MfaMethod:      method,
		}, nil
	}
	return nil, nil
}

// beginMfaLoginChallenge 创建 TOTP 或恢复码登录挑战。
func (c *MfaCase) beginMfaLoginChallenge(user *models.BaseUser, method string, mfaID int64) (*basev1.LoginResponse, error) {
	challengeID := id.NewGUIDv4NoHyphen()
	payload := dto.MfaLoginChallenge{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		ExpiresAt: time.Now().Add(c.runtimeConfig.loginChallengeExpire).Unix(),
		Method:    method,
		MFAID:     mfaID,
	}
	if err := c.saveJSON(mfaLoginChallengeKey(challengeID), payload, c.runtimeConfig.loginChallengeExpire); err != nil {
		return nil, errorsx.Internal("创建多因素认证挑战失败").WithCause(err)
	}
	return &basev1.LoginResponse{
		Status:         basev1.LoginStatus_LOGIN_STATUS_MFA_REQUIRED,
		MfaChallengeId: challengeID,
		MfaExpiresIn:   int64(c.runtimeConfig.loginChallengeExpire / time.Second),
		MfaMethod:      method,
	}, nil
}

// method 读取并校验当前系统配置的 MFA 认证方式。
func (c *MfaCase) method(ctx context.Context) (string, error) {
	values, err := c.runtimeConfigValues(ctx, _const.BASE_CONFIG_KEY_SECURITY_MFA_METHOD)
	if err != nil {
		return "", errorsx.Internal("读取多因素认证方式失败").WithCause(err)
	}
	method := mfaMethodTOTP
	for _, value := range values {
		if value == "" {
			continue
		}
		if value == mfaMethodWebAuthn {
			method = mfaMethodWebAuthn
			continue
		}
		if value != mfaMethodTOTP {
			return "", errorsx.Internal("多因素认证方式配置无效")
		}
	}
	return method, nil
}

// effectiveMethod 保持全局配置的认证方式，避免通过可伪造的客户端头降级因子。
func (c *MfaCase) effectiveMethod(_ context.Context, _ int64, configured string) (string, error) {
	return configured, nil
}

// resolveEnabledMethod 在配置方式未绑定时回退到用户已有的启用方式。
func (c *MfaCase) resolveEnabledMethod(ctx context.Context, userID int64, configured string) (string, error) {
	enabled, err := c.hasEnabledMFA(ctx, userID, configured)
	if err != nil {
		return "", err
	}
	if enabled {
		return configured, nil
	}
	fallback := mfaMethodTOTP
	if configured == mfaMethodTOTP {
		fallback = mfaMethodWebAuthn
	}
	enabled, err = c.hasEnabledMFA(ctx, userID, fallback)
	if err != nil {
		return "", err
	}
	if enabled {
		return fallback, nil
	}
	return configured, nil
}

// beginWebAuthnLogin 创建 WebAuthn 登录挑战并保存服务端会话数据。
func (c *MfaCase) beginWebAuthnLogin(ctx context.Context, user *models.BaseUser) (*basev1.LoginResponse, error) {
	if c.webAuthnErr != nil || c.webAuthn == nil {
		return nil, errorsx.Internal("WebAuthn 配置无效").WithCause(c.webAuthnErr)
	}
	var err error
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, user.ID, mfaMethodWebAuthn)
	if err != nil {
		return nil, errorsx.Internal("查询 WebAuthn 配置失败").WithCause(err)
	}
	var credential webauthn.Credential
	credential, err = c.loadWebAuthnCredential(ctx, mfa.ID)
	if err != nil {
		return nil, err
	}
	var assertion *protocol.CredentialAssertion
	var session *webauthn.SessionData
	assertion, session, err = c.webAuthn.BeginLogin(&mfaWebAuthnUser{user: user, credentials: []webauthn.Credential{credential}})
	if err != nil {
		return nil, errorsx.Internal("创建 WebAuthn 登录挑战失败").WithCause(err)
	}
	var options []byte
	options, err = json.Marshal(assertion)
	if err != nil {
		return nil, errorsx.Internal("编码 WebAuthn 登录选项失败").WithCause(err)
	}
	challengeID := id.NewGUIDv4NoHyphen()
	payload := dto.MfaLoginChallenge{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		ExpiresAt: time.Now().Add(c.runtimeConfig.loginChallengeExpire).Unix(),
		Method:    mfaMethodWebAuthn,
		MFAID:     mfa.ID,
		WebAuthn:  session,
	}
	if err = c.saveJSON(mfaLoginChallengeKey(challengeID), payload, c.runtimeConfig.loginChallengeExpire); err != nil {
		return nil, errorsx.Internal("保存 WebAuthn 登录挑战失败").WithCause(err)
	}
	return &basev1.LoginResponse{
		Status:                 basev1.LoginStatus_LOGIN_STATUS_MFA_REQUIRED,
		MfaChallengeId:         challengeID,
		MfaExpiresIn:           int64(c.runtimeConfig.loginChallengeExpire / time.Second),
		MfaMethod:              mfaMethodWebAuthn,
		MfaWebauthnOptionsJson: string(options),
	}, nil
}

// VerifyLoginChallenge 校验登录阶段的 TOTP 或恢复码并返回用户。
func (c *MfaCase) VerifyLoginChallenge(ctx context.Context, req *basev1.VerifyMfaRequest) (*models.BaseUser, error) {
	challengeKey := mfaLoginChallengeKey(req.GetChallengeId())
	var challenge dto.MfaLoginChallenge
	err := c.loadJSON(challengeKey, &challenge)
	if err != nil || challenge.UserID <= 0 || challenge.ExpiresAt <= time.Now().Unix() {
		return nil, errorsx.Unauthenticated("多因素认证挑战已失效")
	}
	if challenge.Method == mfaMethodWebAuthn {
		if req.GetRecoveryCode() != "" {
			var user *models.BaseUser
			user, err = c.baseUserCase.FindByID(ctx, challenge.UserID)
			if err != nil {
				return nil, errorsx.Unauthenticated("多因素认证失败")
			}
			var mfa *models.BaseUserMFA
			mfa, err = c.findEnabledMFA(ctx, user.ID, mfaMethodWebAuthn)
			if err != nil {
				return nil, errorsx.Unauthenticated("多因素认证未启用")
			}
			var verified bool
			verified, err = c.verifyRecoveryCode(ctx, mfa, req.GetRecoveryCode())
			if err != nil {
				return nil, err
			}
			if !verified {
				return nil, c.recordLoginChallengeFailure(challengeKey, &challenge, errorsx.Unauthenticated("多因素认证验证码错误"))
			}
			if _, err = c.Cache.GetDel(challengeKey); err != nil {
				return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
			}
			if err = c.Cache.Del(mfaLoginChallengeAttemptsKeyFromKey(challengeKey)); err != nil {
				return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
			}
			return user, nil
		}
		return c.verifyWebAuthnLogin(ctx, req.GetWebauthnResponseJson(), challengeKey, challenge)
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, challenge.UserID)
	if err != nil {
		return nil, errorsx.Unauthenticated("多因素认证失败")
	}
	if challenge.Method != mfaMethodTOTP {
		return nil, errorsx.Unauthenticated("多因素认证方式已失效")
	}
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, user.ID, challenge.Method)
	if err != nil {
		return nil, errorsx.Unauthenticated("多因素认证未启用")
	}
	var verified bool
	verified, err = c.verifyFactor(ctx, mfa, req.GetCode(), req.GetRecoveryCode())
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, c.recordLoginChallengeFailure(challengeKey, &challenge, errorsx.Unauthenticated("多因素认证验证码错误"))
	}
	if _, err = c.Cache.GetDel(challengeKey); err != nil {
		return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
	}
	if err = c.Cache.Del(mfaLoginChallengeAttemptsKey(req.GetChallengeId())); err != nil {
		return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
	}
	return user, nil
}

// recordLoginChallengeFailure 记录 MFA 登录失败次数并在达到上限时消费挑战。
func (c *MfaCase) recordLoginChallengeFailure(challengeKey string, challenge *dto.MfaLoginChallenge, failure error) error {
	attemptsKey := mfaLoginChallengeAttemptsKeyFromKey(challengeKey)
	attempts, err := c.Cache.Incr(attemptsKey)
	if err != nil {
		return errorsx.Internal("记录多因素认证失败次数失败").WithCause(err)
	}
	if attempts == 1 {
		if err = c.Cache.Expire(attemptsKey, time.Until(time.Unix(challenge.ExpiresAt, 0))); err != nil {
			return errorsx.Internal("记录多因素认证失败次数失败").WithCause(err)
		}
	}
	if attempts >= int64(c.runtimeConfig.loginMaxAttempts) {
		if err = c.Cache.Del(challengeKey); err != nil {
			return errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
		}
		if err = c.Cache.Del(attemptsKey); err != nil {
			return errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
		}
		return errorsx.Unauthenticated("多因素认证失败次数过多")
	}
	return failure
}

// verifyWebAuthnLogin 校验 WebAuthn 登录响应并更新签名计数器。
func (c *MfaCase) verifyWebAuthnLogin(ctx context.Context, responseJSON string, challengeKey string, challenge dto.MfaLoginChallenge) (*models.BaseUser, error) {
	if responseJSON == "" {
		return nil, c.recordLoginChallengeFailure(challengeKey, &challenge, errorsx.InvalidArgument("WebAuthn认证响应不能为空"))
	}
	if challenge.WebAuthn == nil || challenge.MFAID <= 0 || c.webAuthn == nil {
		return nil, errorsx.InvalidArgument("WebAuthn认证响应不能为空")
	}
	user, err := c.baseUserCase.FindByID(ctx, challenge.UserID)
	if err != nil {
		return nil, errorsx.Unauthenticated("多因素认证失败")
	}
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, user.ID, mfaMethodWebAuthn)
	if err != nil || mfa.ID != challenge.MFAID {
		return nil, errorsx.Unauthenticated("多因素认证未启用")
	}
	var verifiedCredential *webauthn.Credential
	verifiedCredential, err = c.verifyWebAuthnAssertion(ctx, user, mfa.ID, *challenge.WebAuthn, responseJSON)
	if err != nil {
		if kratosErrors.Code(err) >= 500 {
			return nil, err
		}
		return nil, c.recordLoginChallengeFailure(challengeKey, &challenge, err)
	}
	if err = c.updateWebAuthnCredential(ctx, challenge.MFAID, verifiedCredential); err != nil {
		return nil, errorsx.Internal("更新 WebAuthn 签名计数器失败").WithCause(err)
	}
	if _, err = c.Cache.GetDel(challengeKey); err != nil {
		return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
	}
	if err = c.Cache.Del(mfaLoginChallengeAttemptsKeyFromKey(challengeKey)); err != nil {
		return nil, errorsx.Internal("消费多因素认证挑战失败").WithCause(err)
	}
	return user, nil
}

// verifyWebAuthnAssertion 校验指定用户的 WebAuthn 断言响应。
func (c *MfaCase) verifyWebAuthnAssertion(ctx context.Context, user *models.BaseUser, mfaID int64, session webauthn.SessionData, responseJSON string) (*webauthn.Credential, error) {
	credential, err := c.loadWebAuthnCredential(ctx, mfaID)
	if err != nil {
		return nil, err
	}
	var parsed *protocol.ParsedCredentialAssertionData
	parsed, err = protocol.ParseCredentialRequestResponseBytes([]byte(responseJSON))
	if err != nil {
		return nil, errorsx.InvalidArgument("WebAuthn认证响应无效").WithCause(err)
	}
	var verifiedCredential *webauthn.Credential
	verifiedCredential, err = c.webAuthn.ValidateLogin(
		&mfaWebAuthnUser{user: user, credentials: []webauthn.Credential{credential}},
		session,
		parsed,
	)
	if err != nil {
		return nil, errorsx.Unauthenticated("WebAuthn认证失败").WithCause(err)
	}
	return verifiedCredential, nil
}

// updateWebAuthnCredential 更新 WebAuthn 签名计数器和备份状态。
func (c *MfaCase) updateWebAuthnCredential(ctx context.Context, mfaID int64, credential *webauthn.Credential) error {
	row, err := c.baseUserMFAWebauthnRepo.FindByID(ctx, mfaID)
	if err != nil {
		return err
	}
	row.SignCount = int64(credential.Authenticator.SignCount)
	row.BackupEligible = credential.Flags.BackupEligible
	row.BackupState = credential.Flags.BackupState
	return c.baseUserMFAWebauthnRepo.UpdateByID(ctx, row)
}

// GetMfaStatus 查询当前用户的 MFA 状态。
func (c *MfaCase) GetMfaStatus(ctx context.Context) (*basev1.MfaStatusResponse, error) {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return nil, err
	}
	method, err = c.effectiveMethod(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, err
	}
	method, err = c.resolveEnabledMethod(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, err
	}
	var enabled bool
	enabled, err = c.hasEnabledMFA(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, err
	}
	var policy string
	policy, err = c.policy(ctx)
	if err != nil {
		return nil, err
	}
	return &basev1.MfaStatusResponse{Enabled: enabled, Policy: policy, Method: method}, nil
}

// BeginMfaSetup 开始绑定当前配置的多因素认证方式。
func (c *MfaCase) BeginMfaSetup(ctx context.Context, req *basev1.BeginMfaSetupRequest) (*basev1.BeginMfaSetupResponse, error) {
	var user *models.BaseUser
	var err error
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetSetupTicket() != "" {
		var ticket dto.MfaSetupTicket
		err = c.loadJSON(mfaSetupTicketKey(req.GetSetupTicket()), &ticket)
		if err != nil || ticket.UserID <= 0 {
			return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
		}
		user, err = c.baseUserCase.FindByID(ctx, ticket.UserID)
		if err != nil {
			return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
		}
		if ticket.Method != "" {
			method = ticket.Method
		}
	} else {
		var authInfo *authData.UserTokenPayload
		authInfo, err = c.GetAuthInfo(ctx)
		if err != nil {
			return nil, err
		}
		user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
		if err != nil {
			return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
		}
		var password string
		password, err = utils.DecryptPassword(c.Cache, req.GetPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_MFA)
		if err != nil {
			return nil, err
		}
		if err = crypto.Verify(password, user.Password); err != nil {
			return nil, errorsx.InvalidArgument("当前密码错误")
		}
	}
	if method == mfaMethodWebAuthn && !mfaWebAuthnAvailable(ctx) {
		return nil, errorsx.PermissionDenied("当前客户端不支持 WebAuthn，请使用支持 Passkey 的客户端完成绑定")
	}
	if _, err = c.findEnabledMFA(ctx, user.ID, method); err == nil {
		return nil, errorsx.Conflict("多因素认证已启用")
	}
	if !isRecordNotFound(err) {
		return nil, errorsx.Internal("查询多因素认证状态失败").WithCause(err)
	}
	if method == mfaMethodWebAuthn {
		return c.beginWebAuthnSetup(ctx, user, req.GetSetupTicket())
	}
	var key *otp.Key
	key, err = totp.Generate(totp.GenerateOpts{
		Issuer:      c.runtimeConfig.totpIssuer,
		AccountName: user.UserName,
		Period:      c.runtimeConfig.totpPeriod,
		SecretSize:  c.runtimeConfig.totpSecretSize,
		Digits:      c.runtimeConfig.totpDigits,
		Algorithm:   c.runtimeConfig.totpAlgorithm,
	})
	if err != nil {
		return nil, errorsx.Internal("生成多因素认证密钥失败").WithCause(err)
	}
	var protected string
	protected, err = c.protectMFASecret(key.Secret(), user.ID)
	if err != nil {
		return nil, errorsx.Internal("保护多因素认证密钥失败").WithCause(err)
	}
	setupTicket := req.GetSetupTicket()
	if setupTicket == "" {
		setupTicket = id.NewGUIDv4NoHyphen()
	}
	payload := dto.MfaSetupTicket{UserID: user.ID, TenantID: user.TenantID, Method: mfaMethodTOTP, EncryptedSecret: protected}
	if err = c.saveJSON(mfaSetupTicketKey(setupTicket), payload, c.runtimeConfig.setupTicketExpire); err != nil {
		return nil, errorsx.Internal("保存多因素认证绑定票据失败").WithCause(err)
	}
	return &basev1.BeginMfaSetupResponse{
		SetupTicket: setupTicket,
		OtpauthUri:  key.URL(),
		ExpiresIn:   int64(c.runtimeConfig.setupTicketExpire / time.Second),
		Method:      mfaMethodTOTP,
	}, nil
}

// ConfirmMfaSetup 确认并启用当前配置的多因素认证方式。
func (c *MfaCase) ConfirmMfaSetup(ctx context.Context, req *basev1.ConfirmMfaSetupRequest) (*basev1.ConfirmMfaSetupResponse, error) {
	key := mfaSetupTicketKey(req.GetSetupTicket())
	var payload dto.MfaSetupTicket
	err := c.loadJSON(key, &payload)
	if err != nil || payload.UserID <= 0 {
		return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
	}
	if payload.Method == mfaMethodWebAuthn {
		return c.confirmWebAuthnSetup(ctx, key, payload, req.GetWebauthnResponseJson())
	}
	if payload.EncryptedSecret == "" {
		return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
	}
	var secret string
	secret, err = c.unprotectMFASecret(payload.EncryptedSecret, payload.UserID)
	if err != nil {
		return nil, errorsx.Internal("读取多因素认证密钥失败").WithCause(err)
	}
	var valid bool
	valid, err = totp.ValidateCustom(req.GetCode(), secret, time.Now().UTC(), totp.ValidateOpts{Period: c.runtimeConfig.totpPeriod, Skew: c.runtimeConfig.totpSkew, Digits: c.runtimeConfig.totpDigits, Algorithm: c.runtimeConfig.totpAlgorithm})
	if err != nil || !valid {
		return nil, errorsx.WithMessageKey(errorsx.InvalidArgument("动态口令错误"), "base.mfa.confirm_enrollment.code.invalid", nil)
	}
	if _, err = c.Cache.Get(key); err != nil {
		return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
	}
	var recoveryCodes []string
	var hashes []string
	recoveryCodes, hashes, err = c.generateRecoveryCodes()
	if err != nil {
		return nil, errorsx.Internal("生成多因素认证恢复码失败").WithCause(err)
	}
	now := time.Now()
	var mfa *models.BaseUserMFA
	mfa, err = c.findMFA(ctx, payload.UserID, mfaMethodTOTP)
	if isRecordNotFound(err) {
		mfa = &models.BaseUserMFA{TenantID: payload.TenantID, UserID: payload.UserID, Method: mfaMethodTOTP, Status: mfaStatusEnabled, ConfirmedAt: now, CreatedBy: payload.UserID, UpdatedBy: payload.UserID, CreatedAt: now, UpdatedAt: now}
		err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
			if err = c.baseUserMFARepo.Create(txCtx, mfa); err != nil {
				return err
			}
			if err = c.baseUserMFATotpRepo.Create(txCtx, &models.BaseUserMFATotp{MFAID: mfa.ID, SecretCiphertext: payload.EncryptedSecret, LastUsedStep: 0}); err != nil {
				return err
			}
			return c.createRecoveryCodes(txCtx, mfa, hashes, now)
		})
	} else if err == nil {
		mfa.Status = mfaStatusEnabled
		mfa.ConfirmedAt = now
		mfa.UpdatedBy = payload.UserID
		mfa.UpdatedAt = now
		err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
			if err = c.baseUserMFARepo.UpdateByID(txCtx, mfa); err != nil {
				return err
			}
			if err = c.baseUserMFATotpRepo.DeleteByID(txCtx, mfa.ID); err != nil {
				return err
			}
			if err = c.baseUserMFATotpRepo.Create(txCtx, &models.BaseUserMFATotp{MFAID: mfa.ID, SecretCiphertext: payload.EncryptedSecret, LastUsedStep: 0}); err != nil {
				return err
			}
			if err = c.deleteRecoveryCodes(txCtx, mfa.ID); err != nil {
				return err
			}
			return c.createRecoveryCodes(txCtx, mfa, hashes, now)
		})
	} else {
		return nil, errorsx.Internal("查询多因素认证配置失败").WithCause(err)
	}
	if err != nil {
		return nil, errorsx.Internal("保存多因素认证配置失败").WithCause(err)
	}
	if err = c.Cache.Del(key); err != nil {
		return nil, errorsx.Internal("消费多因素认证绑定票据失败").WithCause(err)
	}
	if err = c.revokeUserSession(payload.UserID); err != nil {
		return nil, errorsx.Internal("撤销旧登录会话失败").WithCause(err)
	}
	return &basev1.ConfirmMfaSetupResponse{Enabled: true, RecoveryCodes: recoveryCodes}, nil
}

// beginWebAuthnSetup 创建 WebAuthn 注册选项并保存服务端会话数据。
func (c *MfaCase) beginWebAuthnSetup(ctx context.Context, user *models.BaseUser, setupTicket string) (*basev1.BeginMfaSetupResponse, error) {
	if c.webAuthnErr != nil || c.webAuthn == nil {
		return nil, errorsx.Internal("WebAuthn 配置无效").WithCause(c.webAuthnErr)
	}
	if setupTicket == "" {
		setupTicket = id.NewGUIDv4NoHyphen()
	}
	var err error
	var registration *protocol.CredentialCreation
	var session *webauthn.SessionData
	registration, session, err = c.webAuthn.BeginRegistration(&mfaWebAuthnUser{user: user})
	if err != nil {
		return nil, errorsx.Internal("创建 WebAuthn 注册选项失败").WithCause(err)
	}
	var options []byte
	options, err = json.Marshal(registration)
	if err != nil {
		return nil, errorsx.Internal("编码 WebAuthn 注册选项失败").WithCause(err)
	}
	payload := dto.MfaSetupTicket{UserID: user.ID, TenantID: user.TenantID, Method: mfaMethodWebAuthn, WebAuthn: session}
	if err = c.saveJSON(mfaSetupTicketKey(setupTicket), payload, c.runtimeConfig.setupTicketExpire); err != nil {
		return nil, errorsx.Internal("保存 WebAuthn 绑定票据失败").WithCause(err)
	}
	return &basev1.BeginMfaSetupResponse{
		SetupTicket:         setupTicket,
		ExpiresIn:           int64(c.runtimeConfig.setupTicketExpire / time.Second),
		Method:              mfaMethodWebAuthn,
		WebauthnOptionsJson: string(options),
	}, nil
}

// confirmWebAuthnSetup 校验 WebAuthn 注册响应并保存凭据。
func (c *MfaCase) confirmWebAuthnSetup(ctx context.Context, ticketKey string, payload dto.MfaSetupTicket, responseJSON string) (*basev1.ConfirmMfaSetupResponse, error) {
	if c.webAuthnErr != nil || c.webAuthn == nil || payload.WebAuthn == nil || responseJSON == "" {
		return nil, errorsx.InvalidArgument("WebAuthn注册响应不能为空")
	}
	var err error
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, payload.UserID)
	if err != nil {
		return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
	}
	var parsed *protocol.ParsedCredentialCreationData
	parsed, err = protocol.ParseCredentialCreationResponseBytes([]byte(responseJSON))
	if err != nil {
		return nil, errorsx.InvalidArgument("WebAuthn注册响应无效").WithCause(err)
	}
	var credential *webauthn.Credential
	credential, err = c.webAuthn.CreateCredential(&mfaWebAuthnUser{user: user}, *payload.WebAuthn, parsed)
	if err != nil {
		return nil, errorsx.InvalidArgument("WebAuthn注册验证失败").WithCause(err)
	}
	if _, err = c.Cache.Get(ticketKey); err != nil {
		return nil, errorsx.Unauthenticated("多因素认证绑定票据已失效")
	}
	var transports []byte
	transports, err = json.Marshal(credential.Transport)
	if err != nil {
		return nil, errorsx.Internal("编码 WebAuthn 传输方式失败").WithCause(err)
	}
	var recoveryCodes []string
	var hashes []string
	recoveryCodes, hashes, err = c.generateRecoveryCodes()
	if err != nil {
		return nil, errorsx.Internal("生成多因素认证恢复码失败").WithCause(err)
	}
	now := time.Now()
	var mfa *models.BaseUserMFA
	mfa, err = c.findMFA(ctx, payload.UserID, mfaMethodWebAuthn)
	if isRecordNotFound(err) {
		mfa = &models.BaseUserMFA{TenantID: payload.TenantID, UserID: payload.UserID, Method: mfaMethodWebAuthn, Status: mfaStatusEnabled, ConfirmedAt: now, CreatedBy: payload.UserID, UpdatedBy: payload.UserID, CreatedAt: now, UpdatedAt: now}
		err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
			if err = c.baseUserMFARepo.Create(txCtx, mfa); err != nil {
				return err
			}
			if err = c.saveWebAuthnCredential(txCtx, mfa.ID, credential, string(transports), now); err != nil {
				return err
			}
			return c.createRecoveryCodes(txCtx, mfa, hashes, now)
		})
	} else if err == nil {
		mfa.Status = mfaStatusEnabled
		mfa.ConfirmedAt = now
		mfa.UpdatedBy = payload.UserID
		mfa.UpdatedAt = now
		err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
			if err = c.baseUserMFARepo.UpdateByID(txCtx, mfa); err != nil {
				return err
			}
			if err = c.baseUserMFAWebauthnRepo.DeleteByID(txCtx, mfa.ID); err != nil {
				return err
			}
			if err = c.saveWebAuthnCredential(txCtx, mfa.ID, credential, string(transports), now); err != nil {
				return err
			}
			if err = c.deleteRecoveryCodes(txCtx, mfa.ID); err != nil {
				return err
			}
			return c.createRecoveryCodes(txCtx, mfa, hashes, now)
		})
	} else {
		return nil, errorsx.Internal("查询多因素认证配置失败").WithCause(err)
	}
	if err != nil {
		return nil, errorsx.Internal("保存 WebAuthn 配置失败").WithCause(err)
	}
	if err = c.Cache.Del(ticketKey); err != nil {
		return nil, errorsx.Internal("消费多因素认证绑定票据失败").WithCause(err)
	}
	if err = c.revokeUserSession(payload.UserID); err != nil {
		return nil, errorsx.Internal("撤销旧登录会话失败").WithCause(err)
	}
	return &basev1.ConfirmMfaSetupResponse{Enabled: true, RecoveryCodes: recoveryCodes}, nil
}

// saveWebAuthnCredential 保存 WebAuthn 凭据的持久化字段。
func (c *MfaCase) saveWebAuthnCredential(ctx context.Context, mfaID int64, credential *webauthn.Credential, transports string, now time.Time) error {
	return c.baseUserMFAWebauthnRepo.Create(ctx, &models.BaseUserMFAWebauthn{
		MFAID:          mfaID,
		CredentialID:   credential.ID,
		PublicKey:      credential.PublicKey,
		Aaguid:         credential.Authenticator.AAGUID,
		SignCount:      int64(credential.Authenticator.SignCount),
		Transports:     transports,
		BackupEligible: credential.Flags.BackupEligible,
		BackupState:    credential.Flags.BackupState,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
}

// loadWebAuthnCredential 读取指定 MFA 记录对应的 WebAuthn 凭据。
func (c *MfaCase) loadWebAuthnCredential(ctx context.Context, mfaID int64) (webauthn.Credential, error) {
	row, err := c.baseUserMFAWebauthnRepo.FindByID(ctx, mfaID)
	if err != nil {
		return webauthn.Credential{}, errorsx.Internal("读取 WebAuthn 凭据失败").WithCause(err)
	}
	transports := make([]protocol.AuthenticatorTransport, 0)
	if row.Transports != "" {
		if err = json.Unmarshal([]byte(row.Transports), &transports); err != nil {
			return webauthn.Credential{}, errorsx.Internal("解析 WebAuthn 传输方式失败").WithCause(err)
		}
	}
	return webauthn.Credential{
		ID:        row.CredentialID,
		PublicKey: row.PublicKey,
		Transport: transports,
		Flags: webauthn.CredentialFlags{
			BackupEligible: row.BackupEligible,
			BackupState:    row.BackupState,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:    row.Aaguid,
			SignCount: uint32(row.SignCount),
		},
	}, nil
}

// BeginMfaDisable 开始禁用 WebAuthn 多因素认证的 Passkey 验证。
func (c *MfaCase) BeginMfaDisable(ctx context.Context, _ *basev1.BeginMfaDisableRequest) (*basev1.BeginMfaDisableResponse, error) {
	if c.webAuthnErr != nil || c.webAuthn == nil {
		return nil, errorsx.Internal("WebAuthn 配置无效").WithCause(c.webAuthnErr)
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return nil, err
	}
	method, err = c.effectiveMethod(ctx, user.ID, method)
	if err != nil {
		return nil, err
	}
	method, err = c.resolveEnabledMethod(ctx, user.ID, method)
	if err != nil {
		return nil, err
	}
	if method != mfaMethodWebAuthn {
		return nil, errorsx.InvalidArgument("当前认证方式不需要 WebAuthn 验证")
	}
	if !mfaWebAuthnAvailable(ctx) {
		return nil, errorsx.PermissionDenied("当前客户端不支持 WebAuthn，请使用支持 Passkey 的客户端")
	}
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, user.ID, method)
	if err != nil {
		return nil, errorsx.InvalidArgument("多因素认证未启用")
	}
	var credential webauthn.Credential
	credential, err = c.loadWebAuthnCredential(ctx, mfa.ID)
	if err != nil {
		return nil, err
	}
	var assertion *protocol.CredentialAssertion
	var session *webauthn.SessionData
	assertion, session, err = c.webAuthn.BeginLogin(&mfaWebAuthnUser{user: user, credentials: []webauthn.Credential{credential}})
	if err != nil {
		return nil, errorsx.Internal("创建 WebAuthn 禁用挑战失败").WithCause(err)
	}
	var options []byte
	options, err = json.Marshal(assertion)
	if err != nil {
		return nil, errorsx.Internal("编码 WebAuthn 禁用选项失败").WithCause(err)
	}
	challengeID := id.NewGUIDv4NoHyphen()
	expiresIn := c.runtimeConfig.loginChallengeExpire
	payload := dto.MfaDisableChallenge{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		MFAID:     mfa.ID,
		ExpiresAt: time.Now().Add(expiresIn).Unix(),
		WebAuthn:  session,
	}
	if err = c.saveJSON(mfaDisableChallengeKey(challengeID), payload, expiresIn); err != nil {
		return nil, errorsx.Internal("保存 WebAuthn 禁用挑战失败").WithCause(err)
	}
	return &basev1.BeginMfaDisableResponse{
		ChallengeId:         challengeID,
		ExpiresIn:           int64(expiresIn / time.Second),
		WebauthnOptionsJson: string(options),
	}, nil
}

// DisableMfa 禁用当前用户的多因素认证。
func (c *MfaCase) DisableMfa(ctx context.Context, req *basev1.DisableMfaRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}
	var password string
	password, err = utils.DecryptPassword(c.Cache, req.GetPassword(), basev1.PasswordCryptoScene_PASSWORD_CRYPTO_SCENE_MFA)
	if err != nil {
		return err
	}
	if err = crypto.Verify(password, user.Password); err != nil {
		return errorsx.InvalidArgument("当前密码错误")
	}
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return err
	}
	method, err = c.effectiveMethod(ctx, user.ID, method)
	if err != nil {
		return err
	}
	method, err = c.resolveEnabledMethod(ctx, user.ID, method)
	if err != nil {
		return err
	}
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, user.ID, method)
	if err != nil {
		return errorsx.InvalidArgument("多因素认证未启用")
	}
	if req.GetRecoveryCode() != "" {
		var verified bool
		verified, err = c.verifyRecoveryCode(ctx, mfa, req.GetRecoveryCode())
		if err != nil {
			return err
		}
		if !verified {
			return errorsx.InvalidArgument("多因素认证恢复码错误")
		}
	} else if method == mfaMethodWebAuthn {
		if req.GetWebauthnChallengeId() == "" || req.GetWebauthnResponseJson() == "" {
			return errorsx.InvalidArgument("WebAuthn认证响应不能为空")
		}
		challengeKey := mfaDisableChallengeKey(req.GetWebauthnChallengeId())
		var challenge dto.MfaDisableChallenge
		var rawChallenge string
		rawChallenge, err = c.Cache.Get(challengeKey)
		if err == nil {
			err = json.Unmarshal([]byte(rawChallenge), &challenge)
		}
		if err != nil || challenge.UserID != user.ID || challenge.TenantID != user.TenantID || challenge.MFAID != mfa.ID || challenge.ExpiresAt <= time.Now().Unix() || challenge.WebAuthn == nil {
			return errorsx.Unauthenticated("WebAuthn禁用挑战已失效")
		}
		var verifiedCredential *webauthn.Credential
		verifiedCredential, err = c.verifyWebAuthnAssertion(ctx, user, mfa.ID, *challenge.WebAuthn, req.GetWebauthnResponseJson())
		if err != nil {
			return err
		}
		if err = c.updateWebAuthnCredential(ctx, mfa.ID, verifiedCredential); err != nil {
			return errorsx.Internal("更新 WebAuthn 签名计数器失败").WithCause(err)
		}
		if err = c.Cache.Del(challengeKey); err != nil {
			return errorsx.Internal("消费 WebAuthn 禁用挑战失败").WithCause(err)
		}
	} else {
		if req.GetCode() == "" {
			return errorsx.InvalidArgument("动态口令或恢复码不能为空")
		}
		var verified bool
		verified, err = c.verifyFactor(ctx, mfa, req.GetCode(), "")
		if err != nil {
			return err
		}
		if !verified {
			return errorsx.InvalidArgument("多因素认证验证码错误")
		}
	}
	err = c.baseUserMFARepo.UpdateByID(ctx, &models.BaseUserMFA{ID: mfa.ID, Status: mfaStatusDisabled, UpdatedBy: authInfo.UserId, UpdatedAt: time.Now()})
	if err != nil {
		return errorsx.Internal("禁用多因素认证失败").WithCause(err)
	}
	return c.revokeUserSession(user.ID)
}

// RegenerateRecoveryCodes 重新生成当前用户的 MFA 恢复码。
func (c *MfaCase) RegenerateRecoveryCodes(ctx context.Context, req *basev1.RegenerateMfaRecoveryCodesRequest) (*basev1.RecoveryCodesResponse, error) {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var method string
	method, err = c.method(ctx)
	if err != nil {
		return nil, err
	}
	method, err = c.effectiveMethod(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, err
	}
	method, err = c.resolveEnabledMethod(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, err
	}
	var mfa *models.BaseUserMFA
	mfa, err = c.findEnabledMFA(ctx, authInfo.UserId, method)
	if err != nil {
		return nil, errorsx.InvalidArgument("多因素认证未启用")
	}
	if method != mfaMethodTOTP {
		return nil, errorsx.InvalidArgument("当前认证方式暂不支持重新生成恢复码")
	}
	var verified bool
	verified, err = c.verifyFactor(ctx, mfa, req.GetCode(), "")
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, errorsx.InvalidArgument("多因素认证验证码错误")
	}
	var codes []string
	var hashes []string
	codes, hashes, err = c.generateRecoveryCodes()
	if err != nil {
		return nil, errorsx.Internal("生成多因素认证恢复码失败").WithCause(err)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.deleteRecoveryCodes(txCtx, mfa.ID); err != nil {
			return err
		}
		return c.createRecoveryCodes(txCtx, mfa, hashes, time.Now())
	})
	if err != nil {
		return nil, errorsx.Internal("保存多因素认证恢复码失败").WithCause(err)
	}
	return &basev1.RecoveryCodesResponse{RecoveryCodes: codes}, nil
}

// policy 读取并校验全局 MFA 策略，管理端与应用端配置取更严格值。
func (c *MfaCase) policy(ctx context.Context) (string, error) {
	values, err := c.runtimeConfigValues(ctx, _const.BASE_CONFIG_KEY_SECURITY_MFA_POLICY)
	if err != nil {
		return "", errorsx.Internal("读取多因素认证策略失败").WithCause(err)
	}
	policy := mfaPolicyDisabled
	hasValue := false
	for _, value := range values {
		if value == "" {
			continue
		}
		hasValue = true
		if value != mfaPolicyDisabled && value != mfaPolicyOptional && value != mfaPolicyAllRequired {
			return "", errorsx.Internal("多因素认证策略配置无效")
		}
		if mfaPolicyRank(value) > mfaPolicyRank(policy) {
			policy = value
		}
	}
	if !hasValue {
		return mfaPolicyOptional, nil
	}
	return policy, nil
}

// runtimeConfigValues 读取管理端和应用端的同名运行时配置。
func (c *MfaCase) runtimeConfigValues(ctx context.Context, key string) ([]string, error) {
	sites := []basev1.BaseConfigSite{
		basev1.BaseConfigSite_BASE_CONFIG_SITE_ADMIN,
		basev1.BaseConfigSite_BASE_CONFIG_SITE_APP,
	}
	values := make([]string, 0, len(sites))
	for _, site := range sites {
		config, err := c.configCase.GetConfig(ctx, &basev1.GetConfigRequest{Site: site})
		if err != nil {
			return nil, err
		}
		for _, item := range config.GetConfigs() {
			if item.GetKey() == key {
				values = append(values, item.GetValue())
				break
			}
		}
	}
	return values, nil
}

// mfaPolicyRank 返回策略严格程度，数值越大要求越高。
func mfaPolicyRank(policy string) int {
	switch policy {
	case mfaPolicyAllRequired:
		return 2
	case mfaPolicyOptional:
		return 1
	default:
		return 0
	}
}

// findMFA 查询用户的 MFA 配置。
func (c *MfaCase) findMFA(ctx context.Context, userID int64, method string) (*models.BaseUserMFA, error) {
	query := c.baseUserMFARepo.Query(ctx).BaseUserMFA
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Method.Eq(method)))
	return c.baseUserMFARepo.Find(ctx, opts...)
}

// findEnabledMFA 查询用户当前启用的 MFA 配置。
func (c *MfaCase) findEnabledMFA(ctx context.Context, userID int64, method string) (*models.BaseUserMFA, error) {
	query := c.baseUserMFARepo.Query(ctx).BaseUserMFA
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Method.Eq(method)))
	opts = append(opts, repository.Where(query.Status.Eq(mfaStatusEnabled)))
	return c.baseUserMFARepo.Find(ctx, opts...)
}

// hasEnabledMFA 判断用户是否已启用 MFA。
func (c *MfaCase) hasEnabledMFA(ctx context.Context, userID int64, method string) (bool, error) {
	_, err := c.findEnabledMFA(ctx, userID, method)
	if isRecordNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errorsx.Internal("查询多因素认证状态失败").WithCause(err)
	}
	return true, nil
}

// verifyFactor 校验 TOTP 或恢复码。
func (c *MfaCase) verifyFactor(ctx context.Context, mfa *models.BaseUserMFA, code string, recoveryCode string) (bool, error) {
	if recoveryCode != "" {
		return c.verifyRecoveryCode(ctx, mfa, recoveryCode)
	}
	if code == "" {
		return false, errorsx.InvalidArgument("动态口令不能为空")
	}
	if mfa.Method != mfaMethodTOTP {
		return false, errorsx.InvalidArgument("当前认证方式不支持动态口令")
	}
	totpConfig, err := c.baseUserMFATotpRepo.FindByID(ctx, mfa.ID)
	if err != nil {
		return false, errorsx.Internal("读取 TOTP 配置失败").WithCause(err)
	}
	var secret string
	secret, err = c.unprotectMFASecret(totpConfig.SecretCiphertext, mfa.UserID)
	if err != nil {
		return false, errorsx.Internal("读取多因素认证密钥失败").WithCause(err)
	}
	now := time.Now().UTC()
	currentStep := now.Unix() / int64(c.runtimeConfig.totpPeriod)
	skew := int64(c.runtimeConfig.totpSkew)
	for step := currentStep - skew; step <= currentStep+skew; step++ {
		var generated string
		generated, err = totp.GenerateCodeCustom(secret, time.Unix(step*int64(c.runtimeConfig.totpPeriod), 0).UTC(), totp.ValidateOpts{Period: c.runtimeConfig.totpPeriod, Skew: c.runtimeConfig.totpSkew, Digits: c.runtimeConfig.totpDigits, Algorithm: c.runtimeConfig.totpAlgorithm})
		if err != nil {
			return false, errorsx.Internal("生成动态口令校验值失败").WithCause(err)
		}
		if generated != code || step <= totpConfig.LastUsedStep {
			continue
		}
		query := c.baseUserMFATotpRepo.Query(ctx).BaseUserMFATotp
		var result gen.ResultInfo
		result, err = query.WithContext(ctx).Where(query.MFAID.Eq(mfa.ID), query.LastUsedStep.Lt(step)).UpdateSimple(query.LastUsedStep.Value(step))
		if err != nil {
			return false, errorsx.Internal("记录动态口令使用窗口失败").WithCause(err)
		}
		if result.RowsAffected == 0 {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// verifyRecoveryCode 校验并消费一次性恢复码。
func (c *MfaCase) verifyRecoveryCode(ctx context.Context, mfa *models.BaseUserMFA, recoveryCode string) (bool, error) {
	normalized := normalizeRecoveryCode(recoveryCode)
	query := c.baseUserMFARecoveryRepo.Query(ctx).BaseUserMFARecovery
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.MFAID.Eq(mfa.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(mfa.UserID)))
	opts = append(opts, repository.Where(query.UsedAt.IsNull()))
	// used_at 在数据库中允许 NULL，生成模型使用 time.Time；恢复码校验不需要该列，避免 NULL 扫描失败。
	opts = append(opts, repository.Select(query.ID, query.MFAID, query.UserID, query.CodeHash))
	rows, err := c.baseUserMFARecoveryRepo.List(ctx, opts...)
	if err != nil {
		return false, errorsx.Internal("查询多因素认证恢复码失败").WithCause(err)
	}
	for _, row := range rows {
		if err = crypto.Verify(normalized, row.CodeHash); err != nil {
			continue
		}
		var result gen.ResultInfo
		result, err = query.WithContext(ctx).Where(query.ID.Eq(row.ID), query.UsedAt.IsNull()).UpdateSimple(query.UsedAt.Value(time.Now()))
		if err != nil {
			return false, errorsx.Internal("消费多因素认证恢复码失败").WithCause(err)
		}
		return result.RowsAffected > 0, nil
	}
	return false, nil
}

// createRecoveryCodes 保存恢复码哈希。
func (c *MfaCase) createRecoveryCodes(ctx context.Context, mfa *models.BaseUserMFA, hashes []string, now time.Time) error {
	rows := make([]*models.BaseUserMFARecovery, 0, len(hashes))
	for _, hash := range hashes {
		rows = append(rows, &models.BaseUserMFARecovery{MFAID: mfa.ID, UserID: mfa.UserID, CodeHash: hash, CreatedAt: now, UpdatedAt: now})
	}
	// 生成模型将可空时间字段声明为 time.Time，必须省略 used_at 才能让 MySQL 写入 NULL。
	query := c.baseUserMFARecoveryRepo.Query(ctx).BaseUserMFARecovery
	return query.WithContext(ctx).Omit(query.UsedAt).CreateInBatches(rows, len(rows))
}

// deleteRecoveryCodes 删除用户旧恢复码。
func (c *MfaCase) deleteRecoveryCodes(ctx context.Context, mfaID int64) error {
	query := c.baseUserMFARecoveryRepo.Query(ctx).BaseUserMFARecovery
	return c.baseUserMFARecoveryRepo.Delete(ctx, repository.Where(query.MFAID.Eq(mfaID)))
}

// revokeUserSession 撤销用户访问令牌、刷新令牌和刷新令牌认证缓存。
func (c *MfaCase) revokeUserSession(userID int64) error {
	if c.userToken == nil {
		return nil
	}
	refreshToken := c.userToken.GetRefreshToken(userID)
	if err := c.userToken.RemoveToken(userID); err != nil {
		return err
	}
	if refreshToken == "" {
		return nil
	}
	return c.Cache.Del(refreshTokenAuthKey(refreshToken))
}

// saveJSON 保存带过期时间的 JSON 缓存值。
func (c *MfaCase) saveJSON(key string, value interface{}, expire time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Cache.Set(key, string(payload), expire)
}

// loadJSON 读取 JSON 缓存值。
func (c *MfaCase) loadJSON(key string, value interface{}) error {
	payload, err := c.Cache.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), value)
}

// newMfaRuntimeConfig 将 Bootstrap MFA 配置转换为带默认值的运行时配置。
func newMfaRuntimeConfig(config *configv1.Mfa) mfaRuntimeConfig {
	runtimeConfig := mfaRuntimeConfig{
		loginChallengeExpire: defaultMfaLoginChallengeExpire,
		setupTicketExpire:    defaultMfaSetupTicketExpire,
		loginMaxAttempts:     defaultMfaLoginMaxAttempts,
		recoveryCodeCount:    defaultMfaRecoveryCodeCount,
		recoveryCodeLength:   defaultMfaRecoveryCodeLength,
		totpIssuer:           defaultMfaTotpIssuer,
		totpPeriod:           defaultMfaTotpPeriod,
		totpSkew:             defaultMfaTotpSkew,
		totpSecretSize:       defaultMfaTotpSecretSize,
		totpDigits:           otp.DigitsSix,
		totpAlgorithm:        otp.AlgorithmSHA1,
		webauthnRPID:         "localhost",
		webauthnOrigins: []string{
			"http://localhost:3000", "http://localhost:5173", "http://localhost:5002", "http://localhost:5004",
			"http://127.0.0.1:3000", "http://127.0.0.1:5173", "http://127.0.0.1:5002", "http://127.0.0.1:5004",
		},
	}
	if config == nil {
		return runtimeConfig
	}

	runtimeConfig.encryptionKey = config.GetEncryptionKey()
	if value := config.GetLoginChallengeExpire(); value != nil && value.AsDuration() > 0 {
		runtimeConfig.loginChallengeExpire = value.AsDuration()
	}
	if value := config.GetSetupTicketExpire(); value != nil && value.AsDuration() > 0 {
		runtimeConfig.setupTicketExpire = value.AsDuration()
	}
	if value := config.GetLoginMaxAttempts(); value > 0 {
		runtimeConfig.loginMaxAttempts = int(value)
	}
	if value := config.GetRecoveryCodeCount(); value > 0 {
		runtimeConfig.recoveryCodeCount = int(value)
	}
	if value := config.GetRecoveryCodeLength(); value > 0 {
		runtimeConfig.recoveryCodeLength = int(value)
	}
	if totpConfig := config.GetTotp(); totpConfig != nil {
		if value := totpConfig.GetIssuer(); value != "" {
			runtimeConfig.totpIssuer = value
		}
		if value := totpConfig.GetPeriod(); value > 0 {
			runtimeConfig.totpPeriod = uint(value)
		}
		if value := totpConfig.GetSkew(); value > 0 {
			runtimeConfig.totpSkew = uint(value)
		}
		if value := totpConfig.GetSecretSize(); value > 0 {
			runtimeConfig.totpSecretSize = uint(value)
		}
		switch totpConfig.GetDigits() {
		case 6:
			runtimeConfig.totpDigits = otp.DigitsSix
		case 8:
			runtimeConfig.totpDigits = otp.DigitsEight
		}
		runtimeConfig.totpAlgorithm = parseMfaTotpAlgorithm(totpConfig.GetAlgorithm())
	}
	if webauthnConfig := config.GetWebauthn(); webauthnConfig != nil {
		if value := webauthnConfig.GetRpId(); value != "" {
			runtimeConfig.webauthnRPID = value
		}
		if values := webauthnConfig.GetRpOrigins(); len(values) > 0 {
			runtimeConfig.webauthnOrigins = append([]string(nil), values...)
		}
	}
	return runtimeConfig
}

// parseMfaTotpAlgorithm 解析 TOTP 哈希算法配置，未知值回退到兼容性最好的 SHA1。
func parseMfaTotpAlgorithm(value string) otp.Algorithm {
	switch strings.ToUpper(value) {
	case "SHA256", "SHA-256":
		return otp.AlgorithmSHA256
	case "SHA512", "SHA-512":
		return otp.AlgorithmSHA512
	case "MD5":
		return otp.AlgorithmMD5
	default:
		return otp.AlgorithmSHA1
	}
}

// newMfaWebAuthn 创建 WebAuthn 依赖实例。
func newMfaWebAuthn(config mfaRuntimeConfig) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPID:          config.webauthnRPID,
		RPDisplayName: "Kratos Admin",
		RPOrigins:     config.webauthnOrigins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
		},
	})
}

// generateRecoveryCodes 生成恢复码及其哈希。
func (c *MfaCase) generateRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 0, c.runtimeConfig.recoveryCodeCount)
	hashes := make([]string, 0, c.runtimeConfig.recoveryCodeCount)
	var err error
	for i := 0; i < c.runtimeConfig.recoveryCodeCount; i++ {
		buf := make([]byte, c.runtimeConfig.recoveryCodeLength)
		for index := range buf {
			var randomByte [1]byte
			_, err = rand.Read(randomByte[:])
			if err != nil {
				return nil, nil, err
			}
			buf[index] = mfaRecoveryAlphabet[int(randomByte[0])%len(mfaRecoveryAlphabet)]
		}
		code := string(buf)
		var hash string
		hash, err = crypto.Encrypt(code)
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, code)
		hashes = append(hashes, hash)
	}
	return codes, hashes, nil
}

// protectMFASecret 使用配置密钥加密 TOTP secret。
func (c *MfaCase) protectMFASecret(secret string, userID int64) (string, error) {
	var err error
	var key []byte
	key, err = c.loadMFAEncryptionKey()
	if err != nil {
		return "", err
	}
	var block cipher.Block
	block, err = aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	var gcm cipher.AEAD
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(secret), []byte(strconv.FormatInt(userID, 10)))
	value := base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...))
	return value, nil
}

// unprotectMFASecret 解密 TOTP secret。
func (c *MfaCase) unprotectMFASecret(value string, userID int64) (string, error) {
	var err error
	var key []byte
	key, err = c.loadMFAEncryptionKey()
	if err != nil {
		return "", err
	}
	var encoded []byte
	encoded, err = base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	var block cipher.Block
	block, err = aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	var gcm cipher.AEAD
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(encoded) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid MFA ciphertext")
	}
	var plaintext []byte
	plaintext, err = gcm.Open(nil, encoded[:gcm.NonceSize()], encoded[gcm.NonceSize():], []byte(strconv.FormatInt(userID, 10)))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// loadMFAEncryptionKey 读取配置文件中的 MFA 加密密钥。
func (c *MfaCase) loadMFAEncryptionKey() ([]byte, error) {
	encoded := c.runtimeConfig.encryptionKey
	if encoded == "" {
		return nil, fmt.Errorf("mfa.encryption_key is not configured")
	}
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		key, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("mfa.encryption_key must be a base64 encoded 32-byte key")
	}
	return key, nil
}

// mfaRequestSource 读取当前 HTTP 请求携带的平台来源标识。
func mfaRequestSource(ctx context.Context) string {
	request, ok := http.RequestFromServerContext(ctx)
	if !ok || request == nil {
		return ""
	}
	return request.Header.Get("source-client")
}

// mfaWebAuthnAvailable 判断当前 HTTP 客户端是否具备浏览器 WebAuthn 能力。
// 该值只影响客户端能力分支，不参与认证策略或认证因子选择。
func mfaWebAuthnAvailable(ctx context.Context) bool {
	switch mfaRequestSource(ctx) {
	case "uni-weapp", "taro-weapp":
		return false
	default:
		return true
	}
}

// normalizeRecoveryCode 规范化用户输入的恢复码。
func normalizeRecoveryCode(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", ""))
}

// mfaLoginChallengeKey 生成登录挑战缓存键。
func mfaLoginChallengeKey(id string) string {
	return mfaLoginChallengePrefix + id
}

// mfaLoginChallengeAttemptsKeyFromKey 生成登录挑战失败计数缓存键。
func mfaLoginChallengeAttemptsKeyFromKey(challengeKey string) string {
	return challengeKey + ":attempts"
}

// mfaLoginChallengeAttemptsKey 生成指定登录挑战的失败计数缓存键。
func mfaLoginChallengeAttemptsKey(challengeID string) string {
	return mfaLoginChallengeAttemptsKeyFromKey(mfaLoginChallengeKey(challengeID))
}

// mfaDisableChallengeKey 生成禁用挑战缓存键。
func mfaDisableChallengeKey(id string) string {
	return mfaDisableChallengePrefix + id
}

// mfaSetupTicketKey 生成 MFA 绑定票据缓存键。
func mfaSetupTicketKey(id string) string {
	return mfaSetupTicketPrefix + id
}

// isRecordNotFound 判断仓储查询是否未找到记录。
func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
