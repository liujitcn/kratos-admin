import service from "@/utils/request";
import {
  type AuthService,
  type ListUserButtonRequest,
  type GetUserInfoRequest,
  type TreeUserMenuRequest,
  type GetUserProfileRequest,
  type SendPhoneCodeRequest,
  type TreeRouteResponse,
  type UpdateUserPasswordRequest,
  type UpdateUserPhoneRequest,
  type UpdateUserProfileRequest,
  type UserInfoForm,
  type UserPasswordForm,
  type UserPhoneForm,
  type UserProfileForm
} from "@/rpc/system/admin/v1/auth";
import type { StringValues } from "@/rpc/common/v1/types";
import type { Empty } from "@/rpc/google/protobuf/empty";

const AUTH_URL = "/v1/admin/auth";

/** Admin用户登录认证服务 */
export class AuthServiceImpl implements AuthService {
  /** 获取已经登录的用户的数据 */
  GetUserInfo(request: GetUserInfoRequest): Promise<UserInfoForm> {
    return service<GetUserInfoRequest, UserInfoForm>({
      url: `${AUTH_URL}/user`,
      method: "get",
      params: request
    });
  }

  /** 获取已经登录的用户菜单 */
  TreeUserMenu(request: TreeUserMenuRequest): Promise<TreeRouteResponse> {
    return service<TreeUserMenuRequest, TreeRouteResponse>({
      url: `${AUTH_URL}/menu/tree`,
      method: "get",
      params: request
    });
  }

  /** 获取已经登录的用户按钮权限 */
  ListUserButton(request: ListUserButtonRequest): Promise<StringValues> {
    return service<ListUserButtonRequest, StringValues>({
      url: `${AUTH_URL}/buttons`,
      method: "get",
      params: request
    });
  }

  /** 获取个人中心用户信息 */
  GetUserProfile(request: GetUserProfileRequest): Promise<UserProfileForm> {
    return service<GetUserProfileRequest, UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: "get",
      params: request
    });
  }

  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UpdateUserProfileRequest): Promise<Empty> {
    return service<UserProfileForm | undefined, Empty>({
      url: `${AUTH_URL}/profile`,
      method: "put",
      data: request.user_profile
    });
  }

  /** 发送手机号验证码 */
  SendPhoneCode(request: SendPhoneCodeRequest): Promise<Empty> {
    return service<SendPhoneCodeRequest, Empty>({
      url: `${AUTH_URL}/phone/code`,
      method: "post",
      data: request
    });
  }

  /** 修改个人中心手机号 */
  UpdateUserPhone(request: UpdateUserPhoneRequest): Promise<Empty> {
    return service<UserPhoneForm | undefined, Empty>({
      url: `${AUTH_URL}/phone`,
      method: "put",
      data: request.user_phone
    });
  }

  /** 修改个人中心密码 */
  UpdateUserPassword(request: UpdateUserPasswordRequest): Promise<Empty> {
    return service<UserPasswordForm | undefined, Empty>({
      url: `${AUTH_URL}/password`,
      method: "put",
      data: request.user_password
    });
  }
}

export const defAuthService = new AuthServiceImpl();
