import service from "@liujitcn/kratos-admin-core/request";
import type {
  CreateOauthClientRequest,
  DeleteOauthClientRequest,
  GetOauthClientCredentialsRequest,
  GetOauthClientRequest,
  OauthClientCredentials,
  OauthClientForm,
  OauthClientService,
  OptionOauthClientApiRequest,
  OptionOauthClientApiResponse,
  PageOauthClientRequest,
  PageOauthClientResponse,
  RotateOauthClientCredentialsRequest,
  SetOauthClientStatusRequest,
  UpdateOauthClientRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/oauth_client";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const OAUTH_CLIENT_URL = "/v1/admin/base/oauth-client";

/** 开放授权客户端管理服务实现。 */
export class OauthClientServiceImpl implements OauthClientService {
  /** 查询可授权的开发接口。 */
  OptionOauthClientApi(request: OptionOauthClientApiRequest): Promise<OptionOauthClientApiResponse> {
    return service<OptionOauthClientApiRequest, OptionOauthClientApiResponse>({
      url: `${OAUTH_CLIENT_URL}/api`,
      method: "get",
      params: request
    });
  }

  /** 查询开放授权客户端分页列表。 */
  PageOauthClient(request: PageOauthClientRequest): Promise<PageOauthClientResponse> {
    return service<PageOauthClientRequest, PageOauthClientResponse>({
      url: OAUTH_CLIENT_URL,
      method: "get",
      params: request
    });
  }

  /** 查询开放授权客户端详情。 */
  GetOauthClient(request: GetOauthClientRequest): Promise<OauthClientForm> {
    return service<GetOauthClientRequest, OauthClientForm>({
      url: `${OAUTH_CLIENT_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 查询开放授权客户端非敏感凭据元数据。 */
  GetOauthClientCredentials(request: GetOauthClientCredentialsRequest): Promise<OauthClientCredentials> {
    return service<GetOauthClientCredentialsRequest, OauthClientCredentials>({
      url: `${OAUTH_CLIENT_URL}/${request.id}/credentials`,
      method: "get"
    });
  }

  /** 轮换开放授权客户端凭据并一次性返回新值。 */
  RotateOauthClientCredentials(request: RotateOauthClientCredentialsRequest): Promise<OauthClientCredentials> {
    return service<RotateOauthClientCredentialsRequest, OauthClientCredentials>({
      url: `${OAUTH_CLIENT_URL}/${request.id}/credentials/rotate`,
      method: "post"
    });
  }

  /** 创建开放授权客户端。 */
  CreateOauthClient(request: CreateOauthClientRequest): Promise<Empty> {
    return service<OauthClientForm | undefined, Empty>({
      url: OAUTH_CLIENT_URL,
      method: "post",
      data: request.oauth_client
    });
  }

  /** 更新开放授权客户端。 */
  UpdateOauthClient(request: UpdateOauthClientRequest): Promise<Empty> {
    return service<OauthClientForm | undefined, Empty>({
      url: `${OAUTH_CLIENT_URL}/${request.oauth_client?.id ?? ""}`,
      method: "put",
      data: request.oauth_client
    });
  }

  /** 删除开放授权客户端。 */
  DeleteOauthClient(request: DeleteOauthClientRequest): Promise<Empty> {
    return service<DeleteOauthClientRequest, Empty>({
      url: `${OAUTH_CLIENT_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置开放授权客户端状态。 */
  SetOauthClientStatus(request: SetOauthClientStatusRequest): Promise<Empty> {
    return service<SetOauthClientStatusRequest, Empty>({
      url: `${OAUTH_CLIENT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defOauthClientService = new OauthClientServiceImpl();
