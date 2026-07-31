import service from "@liujitcn/kratos-admin-core/request";
import type {
  GetUserProfileRequest,
  SendPhoneCodeRequest,
  UpdateUserPasswordRequest,
  UpdateUserPhoneRequest,
  UpdateUserProfileRequest,
  UserPasswordForm,
  UserPhoneForm,
  UserProfileForm
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const AUTH_URL = "/v1/admin/auth";

/** ProfileAuthServiceImpl 个人中心认证服务。 */
export class ProfileAuthServiceImpl {
  /** 获取个人中心用户信息。 */
  GetUserProfile(request: GetUserProfileRequest): Promise<UserProfileForm> {
    return service<GetUserProfileRequest, UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: "get",
      params: request
    });
  }

  /** 修改个人中心密码。 */
  UpdateUserPassword(request: UpdateUserPasswordRequest): Promise<Empty> {
    return service<UserPasswordForm | undefined, Empty>({
      url: `${AUTH_URL}/password`,
      method: "put",
      data: request.user_password
    });
  }

  /** 修改个人中心手机号。 */
  UpdateUserPhone(request: UpdateUserPhoneRequest): Promise<Empty> {
    return service<UserPhoneForm | undefined, Empty>({
      url: `${AUTH_URL}/phone`,
      method: "put",
      data: request.user_phone
    });
  }

  /** 修改个人中心用户信息。 */
  UpdateUserProfile(request: UpdateUserProfileRequest): Promise<Empty> {
    return service<UserProfileForm | undefined, Empty>({
      url: `${AUTH_URL}/profile`,
      method: "put",
      data: request.user_profile
    });
  }

  /** 发送手机号验证码。 */
  SendPhoneCode(request: SendPhoneCodeRequest): Promise<Empty> {
    return service<SendPhoneCodeRequest, Empty>({
      url: `${AUTH_URL}/phone/code`,
      method: "post",
      data: request
    });
  }
}

/** defProfileAuthService 个人中心认证服务实例。 */
export const defProfileAuthService = new ProfileAuthServiceImpl();
