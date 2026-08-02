import service from "@/utils/request";
import type {
  CreateOauthAuthorizationRequest,
  CreateOauthAuthorizationResponse,
  ExchangeOauthTicketRequest,
  ExchangeOauthTicketResponse,
  ListOauthProviderRequest,
  ListOauthProviderResponse
} from "@/rpc/base/v1/oauth";

const OAUTH_PROVIDER_URL = "/v1/base/oauth/provider";
const OAUTH_AUTHORIZATION_URL = "/v1/base/oauth/authorization";
const OAUTH_TICKET_URL = "/v1/base/oauth/ticket";

/** OauthServiceImpl 三方登录服务。 */
export class OauthServiceImpl {
  /** 查询三方登录方式 */
  ListOauthProvider(request: ListOauthProviderRequest): Promise<ListOauthProviderResponse> {
    return service<ListOauthProviderRequest, ListOauthProviderResponse>({
      url: `${OAUTH_PROVIDER_URL}`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }

  /** 创建三方登录授权地址 */
  CreateOauthAuthorization(request: CreateOauthAuthorizationRequest): Promise<CreateOauthAuthorizationResponse> {
    return service<CreateOauthAuthorizationRequest, CreateOauthAuthorizationResponse>({
      url: `${OAUTH_AUTHORIZATION_URL}`,
      method: "post",
      data: request,
      headers: { Authorization: "no-auth" }
    });
  }

  /** 兑换三方登录票据 */
  ExchangeOauthTicket(request: ExchangeOauthTicketRequest): Promise<ExchangeOauthTicketResponse> {
    return service<ExchangeOauthTicketRequest, ExchangeOauthTicketResponse>({
      url: `${OAUTH_TICKET_URL}`,
      method: "post",
      data: request,
      headers: { Authorization: "no-auth" }
    });
  }
}

/** defOauthService 三方登录服务实例。 */
export const defOauthService = new OauthServiceImpl();
