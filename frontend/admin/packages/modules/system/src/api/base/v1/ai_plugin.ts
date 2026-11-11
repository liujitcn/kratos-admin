import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiPlugin,
  AiPluginConnection,
  AiPluginInstallation,
  AiPluginService,
  AiPluginTool,
  CompleteAiPluginOAuthRequest,
  CreateAiPluginConnectionRequest,
  CreateAiPluginRequest,
  DeleteAiPluginRequest,
  DisconnectAiPluginConnectionRequest,
  GetAiPluginRequest,
  ListAiPluginConnectionRequest,
  ListAiPluginConnectionResponse,
  ListAiPluginRequest,
  ListAiPluginResponse,
  ListAiPluginToolRequest,
  ListAiPluginToolResponse,
  RefreshAiPluginConnectionRequest,
  ReauthorizeAiPluginConnectionRequest,
  SetAiPluginConnectionSharingRequest,
  SetAiPluginInstallationRequest,
  SetAiPluginToolPolicyRequest,
  StartAiPluginOAuthRequest,
  StartAiPluginOAuthResponse,
  UpdateAiPluginRequest
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";

/** AI 助手插件服务。 */
export class AiPluginServiceImpl implements AiPluginService {
  /** ListAiPlugin 调用服务端插件能力。 */
  ListAiPlugin(request: ListAiPluginRequest): Promise<ListAiPluginResponse> {
    return service<ListAiPluginRequest, ListAiPluginResponse>({ url: `/v1/base/ai/plugin`, method: "get", params: request });
  }
  /** GetAiPlugin 调用服务端插件能力。 */
  GetAiPlugin(request: GetAiPluginRequest): Promise<AiPlugin> {
    return service<GetAiPluginRequest, AiPlugin>({ url: `/v1/base/ai/plugin/${request.id}`, method: "get", params: request });
  }
  /** CreateAiPlugin 调用服务端插件能力。 */
  CreateAiPlugin(request: CreateAiPluginRequest): Promise<AiPlugin> {
    return service<CreateAiPluginRequest, AiPlugin>({ url: `/v1/base/ai/plugin`, method: "post", data: request });
  }
  /** UpdateAiPlugin 调用服务端插件能力。 */
  UpdateAiPlugin(request: UpdateAiPluginRequest): Promise<AiPlugin> {
    return service<UpdateAiPluginRequest, AiPlugin>({ url: `/v1/base/ai/plugin/${request.id}`, method: "put", data: request });
  }
  /** DeleteAiPlugin 调用服务端插件能力。 */
  DeleteAiPlugin(request: DeleteAiPluginRequest): Promise<Empty> {
    return service<DeleteAiPluginRequest, Empty>({ url: `/v1/base/ai/plugin/${request.id}`, method: "delete", params: request });
  }
  /** SetAiPluginInstallation 调用服务端插件能力。 */
  SetAiPluginInstallation(request: SetAiPluginInstallationRequest): Promise<AiPluginInstallation> {
    return service<SetAiPluginInstallationRequest, AiPluginInstallation>({
      url: `/v1/base/ai/plugin/${request.plugin_id}/installation`,
      method: "put",
      data: request
    });
  }
  /** ListAiPluginTool 调用服务端插件能力。 */
  ListAiPluginTool(request: ListAiPluginToolRequest): Promise<ListAiPluginToolResponse> {
    return service<ListAiPluginToolRequest, ListAiPluginToolResponse>({
      url: `/v1/base/ai/plugin/${request.plugin_id}/tool`,
      method: "get",
      params: request
    });
  }
  /** SetAiPluginToolPolicy 调用服务端插件能力。 */
  SetAiPluginToolPolicy(request: SetAiPluginToolPolicyRequest): Promise<AiPluginTool> {
    return service<SetAiPluginToolPolicyRequest, AiPluginTool>({
      url: `/v1/base/ai/plugin/${request.plugin_id}/tool/${request.tool_id}/policy`,
      method: "put",
      data: request
    });
  }
  /** ListAiPluginConnection 调用服务端插件能力。 */
  ListAiPluginConnection(request: ListAiPluginConnectionRequest): Promise<ListAiPluginConnectionResponse> {
    return service<ListAiPluginConnectionRequest, ListAiPluginConnectionResponse>({
      url: `/v1/base/ai/plugin-connection`,
      method: "get",
      params: request
    });
  }
  /** CreateAiPluginConnection 调用服务端插件能力。 */
  CreateAiPluginConnection(request: CreateAiPluginConnectionRequest): Promise<AiPluginConnection> {
    return service<CreateAiPluginConnectionRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-connection`,
      method: "post",
      data: request
    });
  }
  /** RefreshAiPluginConnection 调用服务端插件能力。 */
  RefreshAiPluginConnection(request: RefreshAiPluginConnectionRequest): Promise<AiPluginConnection> {
    return service<RefreshAiPluginConnectionRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-connection/${request.id}/refresh`,
      method: "post",
      data: request
    });
  }
  /** DisconnectAiPluginConnection 调用服务端插件能力。 */
  DisconnectAiPluginConnection(request: DisconnectAiPluginConnectionRequest): Promise<AiPluginConnection> {
    return service<DisconnectAiPluginConnectionRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-connection/${request.id}`,
      method: "delete",
      params: request
    });
  }
  /** ReauthorizeAiPluginConnection 更换原账号令牌并使旧许可失效。 */
  ReauthorizeAiPluginConnection(request: ReauthorizeAiPluginConnectionRequest): Promise<AiPluginConnection> {
    return service<ReauthorizeAiPluginConnectionRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-connection/${request.id}/credential`,
      method: "put",
      data: request
    });
  }
  /** SetAiPluginConnectionSharing 设置本人账号的租户共享范围。 */
  SetAiPluginConnectionSharing(request: SetAiPluginConnectionSharingRequest): Promise<AiPluginConnection> {
    return service<SetAiPluginConnectionSharingRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-connection/${request.id}/sharing`,
      method: "put",
      data: request
    });
  }
  /** StartAiPluginOAuth 调用服务端插件能力。 */
  StartAiPluginOAuth(request: StartAiPluginOAuthRequest): Promise<StartAiPluginOAuthResponse> {
    return service<StartAiPluginOAuthRequest, StartAiPluginOAuthResponse>({
      url: `/v1/base/ai/plugin-connection/${request.id}/authorization`,
      method: "post",
      data: request
    });
  }
  /** CompleteAiPluginOAuth 调用服务端插件能力。 */
  CompleteAiPluginOAuth(request: CompleteAiPluginOAuthRequest): Promise<AiPluginConnection> {
    return service<CompleteAiPluginOAuthRequest, AiPluginConnection>({
      url: `/v1/base/ai/plugin-oauth/callback`,
      method: "get",
      params: request,
      headers: { Authorization: "no-auth" }
    });
  }
}
export const defAiPluginService = new AiPluginServiceImpl();
