import { http } from '@/utils/http'
import type {
  CaptchaRequest,
  CaptchaResponse,
  LogoutRequest,
  LoginRequest,
  LoginResponse,
  PasswordPublicKeyRequest,
  PasswordPublicKeyResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  VerifyCaptchaRequest,
  VerifyCaptchaResponse,
  LoginService,
} from '@/rpc/base/v1/login'
import type { Empty } from '@/rpc/google/protobuf/empty'

const CAPTCHA_URL = '/v1/base/captcha'
const CAPTCHA_VERIFY_URL = '/v1/base/captcha/verify'
const PASSWORD_PUBLIC_KEY_URL = '/v1/base/password-public-key'
const SESSION_URL = '/v1/base/session'
const TOKEN_URL = '/v1/base/token'

/** 登录公共服务 */
export class LoginServiceImpl implements LoginService {
  /** 验证码 */
  Captcha(request: CaptchaRequest): Promise<CaptchaResponse> {
    return http<CaptchaResponse>({
      url: `${CAPTCHA_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 验证码预校验 */
  VerifyCaptcha(request: VerifyCaptchaRequest): Promise<VerifyCaptchaResponse> {
    return http<VerifyCaptchaResponse>({
      url: `${CAPTCHA_VERIFY_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 获取密码临时公钥 */
  PasswordPublicKey(request: PasswordPublicKeyRequest): Promise<PasswordPublicKeyResponse> {
    return http<PasswordPublicKeyResponse>({
      url: `${PASSWORD_PUBLIC_KEY_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登出 */
  Logout(request: LogoutRequest): Promise<Empty> {
    return http<Empty>({
      url: `${SESSION_URL}`,
      method: 'DELETE',
      authMode: 'required',
      data: request,
    })
  }
  /** 刷新认证令牌 */
  RefreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    return http<RefreshTokenResponse>({
      url: `${TOKEN_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
  /** 登录 */
  Login(request: LoginRequest): Promise<LoginResponse> {
    return http<LoginResponse>({
      url: `${SESSION_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }
}

export const defLoginService = new LoginServiceImpl()
