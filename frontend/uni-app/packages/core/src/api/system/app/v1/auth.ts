import { http } from '../../../../utils/http'
import {
  type AuthService,
  type BindUserPhoneRequest,
  type BindUserPhoneResponse,
  type GetUserProfileRequest,
  type UpdateUserProfileRequest,
  type UserProfileForm,
} from '../../../../rpc/system/app/v1/auth'
import type { Empty } from '../../../../rpc/google/protobuf/empty'

const AUTH_URL = '/v1/app/auth'

/** 用户认证资料服务 */
export class AuthServiceImpl implements AuthService {
  /** 获取已登录用户资料 */
  GetUserProfile(request: GetUserProfileRequest): Promise<UserProfileForm> {
    return http<UserProfileForm>({
      url: `${AUTH_URL}/profile`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 修改个人中心用户信息 */
  UpdateUserProfile(request: UpdateUserProfileRequest): Promise<Empty> {
    return http<Empty>({
      url: `${AUTH_URL}/profile`,
      method: 'PUT',
      authMode: 'required',
      data: request.user_profile,
    })
  }

  /** 手机号授权 */
  BindUserPhone(request: BindUserPhoneRequest): Promise<BindUserPhoneResponse> {
    return http<BindUserPhoneResponse>({
      url: `${AUTH_URL}/phone`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAuthService = new AuthServiceImpl()
