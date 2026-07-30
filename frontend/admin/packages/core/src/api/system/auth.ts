import service from "@/utils/request";
import type {
  GetUserInfoRequest,
  ListUserButtonRequest,
  TreeRouteResponse,
  TreeUserMenuRequest,
  UserInfoForm
} from "@/rpc/system/admin/v1/auth";
import type { StringValues } from "@/rpc/common/v1/types";

const AUTH_URL = "/v1/admin/auth";

/** AuthServiceImpl 管理端运行壳认证服务。 */
export class AuthServiceImpl {
  /** 获取当前登录用户。 */
  GetUserInfo(request: GetUserInfoRequest): Promise<UserInfoForm> {
    return service<GetUserInfoRequest, UserInfoForm>({
      url: `${AUTH_URL}/user`,
      method: "get",
      params: request
    });
  }

  /** 获取当前用户菜单。 */
  TreeUserMenu(request: TreeUserMenuRequest): Promise<TreeRouteResponse> {
    return service<TreeUserMenuRequest, TreeRouteResponse>({
      url: `${AUTH_URL}/menu/tree`,
      method: "get",
      params: request
    });
  }

  /** 获取当前用户按钮权限。 */
  ListUserButton(request: ListUserButtonRequest): Promise<StringValues> {
    return service<ListUserButtonRequest, StringValues>({
      url: `${AUTH_URL}/buttons`,
      method: "get",
      params: request
    });
  }

}

/** defAuthService 管理端运行壳认证服务实例。 */
export const defAuthService = new AuthServiceImpl();
