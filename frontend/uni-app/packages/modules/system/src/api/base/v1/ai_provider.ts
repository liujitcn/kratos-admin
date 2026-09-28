import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiModelService,
  ListAiProviderModelOptionsRequest,
  ListAiProviderModelOptionsResponse,
} from '../../../rpc/base/v1/ai_provider'

const AI_PROVIDER_MODELS_URL = '/v1/base/ai/provider-models'

/** AI Provider模型选项服务。 */
export class AiModelServiceImpl implements AiModelService {
  /** 查询已启用Provider及其可用模型。 */
  ListAiProviderModelOptions(
    request: ListAiProviderModelOptionsRequest,
  ): Promise<ListAiProviderModelOptionsResponse> {
    return http<ListAiProviderModelOptionsResponse>({
      url: AI_PROVIDER_MODELS_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAiModelService = new AiModelServiceImpl()
