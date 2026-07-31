import service from "@liujitcn/kratos-admin-core/request";
import type {
  CreateOauthBindingAuthorizationRequest,
  CreateOauthBindingAuthorizationResponse,
  ListOauthBindingRequest,
  ListOauthBindingResponse,
  UnbindOauthAccountRequest
} from "@liujitcn/kratos-admin-system/rpc/base/v1/oauth";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const OAUTH_BINDING_URL = "/v1/base/oauth/binding";
const OAUTH_BINDING_AUTHORIZATION_URL = "/v1/base/oauth/binding/authorization";

/** ProfileOauthServiceImpl 个人中心三方账号服务。 */
export class ProfileOauthServiceImpl {
  /** 查询个人中心三方账号绑定列表。 */
  ListOauthBinding(request: ListOauthBindingRequest): Promise<ListOauthBindingResponse> {
    return service<ListOauthBindingRequest, ListOauthBindingResponse>({
      url: OAUTH_BINDING_URL,
      method: "get",
      params: request
    });
  }

  /** 创建个人中心三方账号绑定授权地址。 */
  CreateOauthBindingAuthorization(
    request: CreateOauthBindingAuthorizationRequest
  ): Promise<CreateOauthBindingAuthorizationResponse> {
    return service<CreateOauthBindingAuthorizationRequest, CreateOauthBindingAuthorizationResponse>({
      url: OAUTH_BINDING_AUTHORIZATION_URL,
      method: "post",
      data: request
    });
  }

  /** 解绑个人中心三方账号。 */
  UnbindOauthAccount(request: UnbindOauthAccountRequest): Promise<Empty> {
    return service<UnbindOauthAccountRequest, Empty>({
      url: `${OAUTH_BINDING_URL}/${request.provider}`,
      method: "delete",
      params: request
    });
  }
}

/** defProfileOauthService 个人中心三方账号服务实例。 */
export const defProfileOauthService = new ProfileOauthServiceImpl();
