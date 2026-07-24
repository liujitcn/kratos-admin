import { http } from '@/utils/http'
import type {
  CreateOauthAuthorizationRequest,
  CreateOauthAuthorizationResponse,
  CreateOauthBindingAuthorizationRequest,
  CreateOauthBindingAuthorizationResponse,
  CreateOauthSessionRequest,
  CreateOauthSessionResponse,
  ExchangeOauthTicketRequest,
  ExchangeOauthTicketResponse,
  HandleOauthBindingCallbackRequest,
  HandleOauthBindingCallbackResponse,
  HandleOauthCallbackRequest,
  HandleOauthCallbackResponse,
  ListOauthBindingRequest,
  ListOauthBindingResponse,
  ListOauthProviderRequest,
  ListOauthProviderResponse,
  OauthService,
  UnbindOauthAccountRequest,
} from '@/rpc/base/v1/oauth'
import type { Empty } from '@/rpc/google/protobuf/empty'

const OAUTH_PROVIDER_URL = '/v1/base/oauth/provider'
const OAUTH_AUTHORIZATION_URL = '/v1/base/oauth/authorization'
const OAUTH_CALLBACK_URL = '/v1/base/oauth'
const OAUTH_TICKET_URL = '/v1/base/oauth/ticket'
const OAUTH_SESSION_URL = '/v1/base/oauth/session'
const OAUTH_BINDING_URL = '/v1/base/oauth/binding'
const OAUTH_BINDING_AUTHORIZATION_URL = '/v1/base/oauth/binding/authorization'

/** 三方登录公共服务 */
export class OauthServiceImpl implements OauthService {
  /** 查询三方登录方式 */
  ListOauthProvider(request: ListOauthProviderRequest): Promise<ListOauthProviderResponse> {
    return http<ListOauthProviderResponse>({
      url: `${OAUTH_PROVIDER_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 创建三方登录授权地址 */
  CreateOauthAuthorization(
    request: CreateOauthAuthorizationRequest,
  ): Promise<CreateOauthAuthorizationResponse> {
    return http<CreateOauthAuthorizationResponse>({
      url: `${OAUTH_AUTHORIZATION_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 处理三方登录回调 */
  HandleOauthCallback(request: HandleOauthCallbackRequest): Promise<HandleOauthCallbackResponse> {
    return http<HandleOauthCallbackResponse>({
      url: `${OAUTH_CALLBACK_URL}/${request.provider}/callback`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 兑换三方登录票据 */
  ExchangeOauthTicket(request: ExchangeOauthTicketRequest): Promise<ExchangeOauthTicketResponse> {
    return http<ExchangeOauthTicketResponse>({
      url: `${OAUTH_TICKET_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 创建三方登录会话 */
  CreateOauthSession(request: CreateOauthSessionRequest): Promise<CreateOauthSessionResponse> {
    return http<CreateOauthSessionResponse>({
      url: `${OAUTH_SESSION_URL}`,
      method: 'POST',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 查询个人中心三方账号绑定列表 */
  ListOauthBinding(request: ListOauthBindingRequest): Promise<ListOauthBindingResponse> {
    return http<ListOauthBindingResponse>({
      url: `${OAUTH_BINDING_URL}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 创建个人中心三方账号绑定授权地址 */
  CreateOauthBindingAuthorization(
    request: CreateOauthBindingAuthorizationRequest,
  ): Promise<CreateOauthBindingAuthorizationResponse> {
    return http<CreateOauthBindingAuthorizationResponse>({
      url: `${OAUTH_BINDING_AUTHORIZATION_URL}`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 处理个人中心三方账号绑定回调 */
  HandleOauthBindingCallback(
    request: HandleOauthBindingCallbackRequest,
  ): Promise<HandleOauthBindingCallbackResponse> {
    return http<HandleOauthBindingCallbackResponse>({
      url: `${OAUTH_CALLBACK_URL}/${request.provider}/binding/callback`,
      method: 'GET',
      authMode: 'none',
      data: request,
      header: { Authorization: 'no-auth' },
    })
  }

  /** 解绑个人中心三方账号 */
  UnbindOauthAccount(request: UnbindOauthAccountRequest): Promise<Empty> {
    return http<Empty>({
      url: `${OAUTH_BINDING_URL}/${request.provider}`,
      method: 'DELETE',
      authMode: 'required',
      data: request,
    })
  }
}

export const defOauthService = new OauthServiceImpl()
