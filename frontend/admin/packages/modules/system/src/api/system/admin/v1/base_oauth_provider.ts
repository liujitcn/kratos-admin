import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseOauthProviderForm,
  BaseOauthProviderService,
  CreateBaseOauthProviderRequest,
  DeleteBaseOauthProviderRequest,
  GetBaseOauthProviderRequest,
  PageBaseOauthProviderRequest,
  PageBaseOauthProviderResponse,
  SetBaseOauthProviderStatusRequest,
  UpdateBaseOauthProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_oauth_provider";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const OAUTH_PROVIDER_URL = "/v1/admin/base/oauth-provider";

/** OAuth第三方登录方式管理服务实现。 */
export class BaseOauthProviderServiceImpl implements BaseOauthProviderService {
  /** 分页查询OAuth登录方式。 */
  PageBaseOauthProvider(request: PageBaseOauthProviderRequest): Promise<PageBaseOauthProviderResponse> {
    return service({ url: OAUTH_PROVIDER_URL, method: "get", params: request });
  }

  /** 查询OAuth登录方式详情。 */
  GetBaseOauthProvider(request: GetBaseOauthProviderRequest): Promise<BaseOauthProviderForm> {
    return service({ url: `${OAUTH_PROVIDER_URL}/${request.id}`, method: "get" });
  }

  /** 创建OAuth登录方式。 */
  CreateBaseOauthProvider(request: CreateBaseOauthProviderRequest): Promise<Empty> {
    return service({ url: OAUTH_PROVIDER_URL, method: "post", data: request.base_oauth_provider });
  }

  /** 更新OAuth登录方式。 */
  UpdateBaseOauthProvider(request: UpdateBaseOauthProviderRequest): Promise<Empty> {
    return service({ url: `${OAUTH_PROVIDER_URL}/${request.base_oauth_provider?.id ?? ""}`, method: "put", data: request.base_oauth_provider });
  }

  /** 删除OAuth登录方式。 */
  DeleteBaseOauthProvider(request: DeleteBaseOauthProviderRequest): Promise<Empty> {
    return service({ url: `${OAUTH_PROVIDER_URL}/${request.id}`, method: "delete" });
  }

  /** 设置OAuth登录方式状态。 */
  SetBaseOauthProviderStatus(request: SetBaseOauthProviderStatusRequest): Promise<Empty> {
    return service({ url: `${OAUTH_PROVIDER_URL}/${request.id}/status`, method: "put", data: request });
  }
}

export const defBaseOauthProviderService = new BaseOauthProviderServiceImpl();
