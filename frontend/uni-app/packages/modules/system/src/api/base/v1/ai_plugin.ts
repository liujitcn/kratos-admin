import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiPluginService,
  ListAiPluginRequest,
  ListAiPluginResponse,
  ListAiPluginToolRequest,
  ListAiPluginToolResponse,
  ListAiPluginConnectionRequest,
  ListAiPluginConnectionResponse,
} from '../../../rpc/base/v1/ai_plugin'

/** 应用端插件目录、连接和工具只读服务。 */
class AiPluginReadService
  implements Pick<AiPluginService, 'ListAiPlugin' | 'ListAiPluginTool' | 'ListAiPluginConnection'>
{
  /** 查询当前用户可见的应用端插件目录。 */
  ListAiPlugin(request: ListAiPluginRequest): Promise<ListAiPluginResponse> {
    return http({ url: '/v1/base/ai/plugin', method: 'GET', authMode: 'required', data: request })
  }
  /** 查询工具参数及当前终端的实际可用状态。 */
  ListAiPluginTool(request: ListAiPluginToolRequest): Promise<ListAiPluginToolResponse> {
    return http({
      url: `/v1/base/ai/plugin/${request.plugin_id}/tool`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询本人和授权共享账号，不返回凭据。 */
  ListAiPluginConnection(
    request: ListAiPluginConnectionRequest,
  ): Promise<ListAiPluginConnectionResponse> {
    return http({
      url: '/v1/base/ai/plugin-connection',
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
}

/** 默认应用端插件只读服务。 */
export const defAiPluginService = new AiPluginReadService()
