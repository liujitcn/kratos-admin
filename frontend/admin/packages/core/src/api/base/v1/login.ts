import service from "@/utils/request";
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
  LoginService
} from "@/rpc/base/v1/login";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CAPTCHA_URL = "/v1/base/captcha";
const CAPTCHA_VERIFY_URL = "/v1/base/captcha/verify";
const PASSWORD_PUBLIC_KEY_URL = "/v1/base/password-public-key";
const SESSION_URL = "/v1/base/session";
const TOKEN_URL = "/v1/base/token";

/** 登录公共服务 */
export class LoginServiceImpl implements LoginService {
  /** 验证码 */
  Captcha(request: CaptchaRequest): Promise<CaptchaResponse> {
    return service<CaptchaRequest, CaptchaResponse>({
      url: `${CAPTCHA_URL}`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }
  /** 验证码预校验 */
  VerifyCaptcha(request: VerifyCaptchaRequest): Promise<VerifyCaptchaResponse> {
    return service<VerifyCaptchaRequest, VerifyCaptchaResponse>({
      url: `${CAPTCHA_VERIFY_URL}`,
      method: "post",
      data: request,
      headers: { Authorization: "no-auth" }
    });
  }
  /** 获取密码临时公钥 */
  PasswordPublicKey(request: PasswordPublicKeyRequest): Promise<PasswordPublicKeyResponse> {
    return service<PasswordPublicKeyRequest, PasswordPublicKeyResponse>({
      url: `${PASSWORD_PUBLIC_KEY_URL}`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }
  /** 登录 */
  Login(request: LoginRequest): Promise<LoginResponse> {
    return service<LoginRequest, LoginResponse>({
      url: `${SESSION_URL}`,
      method: "post",
      data: request,
      headers: { Authorization: "no-auth" }
    });
  }
  /** 登出 */
  Logout(request: LogoutRequest): Promise<Empty> {
    return service<LogoutRequest, Empty>({
      url: `${SESSION_URL}`,
      method: "delete",
      data: request
    });
  }
  /** 刷新认证令牌 */
  RefreshToken(request: RefreshTokenRequest): Promise<RefreshTokenResponse> {
    return service<RefreshTokenRequest, RefreshTokenResponse>({
      url: `${TOKEN_URL}`,
      method: "post",
      data: request,
      headers: { Authorization: "no-auth" }
    });
  }
}

export const defLoginService = new LoginServiceImpl();
