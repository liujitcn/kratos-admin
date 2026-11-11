import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiPluginService,
  AiPlugin,
  GetAiPluginRequest,
  ListAiPluginRequest,
  ListAiPluginResponse,
  ListAiPluginToolRequest,
  ListAiPluginToolResponse,
  ListAiPluginConnectionRequest,
  ListAiPluginConnectionResponse,
} from '../../../rpc/base/v1/ai_plugin'

/** 移动端只读插件目录和已有账号，不提供管理写操作。 */
class AiPluginUseService
  implements
    Pick<
      AiPluginService,
      'ListAiPlugin' | 'GetAiPlugin' | 'ListAiPluginTool' | 'ListAiPluginConnection'
    >
{
  /** 查询插件目录。 */
  ListAiPlugin(request: ListAiPluginRequest): Promise<ListAiPluginResponse> {
    return http({ url: '/v1/base/ai/plugin', method: 'GET', authMode: 'required', data: request })
  }
  /** 查询插件说明。 */
  GetAiPlugin(request: GetAiPluginRequest): Promise<AiPlugin> {
    return http({
      url: `/v1/base/ai/plugin/${request.id}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询工具说明与可用状态。 */
  ListAiPluginTool(request: ListAiPluginToolRequest): Promise<ListAiPluginToolResponse> {
    return http({
      url: `/v1/base/ai/plugin/${request.plugin_id}/tool`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询本人及授权共享账号。 */
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

export const defAiPluginService = new AiPluginUseService()
