import { http } from '../../../utils/http'
import type { LoginResponse } from '../../../rpc/base/v1/login'
import type {
  BeginMfaDisableRequest,
  BeginMfaDisableResponse,
  BeginMfaEnrollmentRequest,
  BeginMfaEnrollmentResponse,
  BeginMfaSetupRequest,
  BeginMfaSetupResponse,
  ConfirmMfaEnrollmentRequest,
  ConfirmMfaEnrollmentResponse,
  ConfirmMfaSetupRequest,
  ConfirmMfaSetupResponse,
  DisableMfaRequest,
  GetMfaStatusRequest,
  MfaService,
  MfaStatusResponse,
  RecoveryCodesResponse,
  RegenerateMfaRecoveryCodesRequest,
  VerifyMfaRequest,
} from '../../../rpc/base/v1/mfa'
import type { Empty } from '../../../rpc/google/protobuf/empty'

const MFA_URL = '/v1/base/mfa'

/** 多因素认证公共服务。 */
export class MfaServiceImpl implements MfaService {
  /** 校验登录阶段多因素认证。 */
  VerifyMfa(request: VerifyMfaRequest): Promise<LoginResponse> {
    return http<LoginResponse>({
      url: `${MFA_URL}/verify`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 开始登录阶段的强制多因素认证绑定。 */
  BeginMfaEnrollment(request: BeginMfaEnrollmentRequest): Promise<BeginMfaEnrollmentResponse> {
    return http<BeginMfaEnrollmentResponse>({
      url: `${MFA_URL}/enrollment`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 确认登录阶段的强制多因素认证绑定。 */
  ConfirmMfaEnrollment(
    request: ConfirmMfaEnrollmentRequest,
  ): Promise<ConfirmMfaEnrollmentResponse> {
    return http<ConfirmMfaEnrollmentResponse>({
      url: `${MFA_URL}/enrollment/confirm`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 查询当前用户多因素认证状态。 */
  GetMfaStatus(request: GetMfaStatusRequest): Promise<MfaStatusResponse> {
    return http<MfaStatusResponse>({
      url: MFA_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 开始绑定当前用户的多因素认证方式。 */
  BeginMfaSetup(request: BeginMfaSetupRequest): Promise<BeginMfaSetupResponse> {
    return http<BeginMfaSetupResponse>({
      url: `${MFA_URL}/setup`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 确认绑定当前用户的多因素认证方式。 */
  ConfirmMfaSetup(request: ConfirmMfaSetupRequest): Promise<ConfirmMfaSetupResponse> {
    return http<ConfirmMfaSetupResponse>({
      url: `${MFA_URL}/setup/confirm`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 开始禁用 WebAuthn 多因素认证的 Passkey 验证。 */
  BeginMfaDisable(request: BeginMfaDisableRequest): Promise<BeginMfaDisableResponse> {
    return http<BeginMfaDisableResponse>({
      url: `${MFA_URL}/disable/challenge`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 禁用当前用户的多因素认证。 */
  DisableMfa(request: DisableMfaRequest): Promise<Empty> {
    return http<Empty>({
      url: `${MFA_URL}/disable`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 重新生成当前用户的多因素认证恢复码。 */
  RegenerateMfaRecoveryCodes(
    request: RegenerateMfaRecoveryCodesRequest,
  ): Promise<RecoveryCodesResponse> {
    return http<RecoveryCodesResponse>({
      url: `${MFA_URL}/recovery-codes`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
}

/** defMfaService 多因素认证公共服务实例。 */
export const defMfaService = new MfaServiceImpl()
