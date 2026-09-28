import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiModelService,
  ListAiProviderModelOptionsRequest,
  ListAiProviderModelOptionsResponse
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";

const AI_PROVIDER_MODELS_URL = "/v1/base/ai/provider-models";

/** AI Provider模型选项服务。 */
export class AiModelServiceImpl implements AiModelService {
  /** 查询已启用Provider及其可用模型。 */
  ListAiProviderModelOptions(request: ListAiProviderModelOptionsRequest): Promise<ListAiProviderModelOptionsResponse> {
    return service<ListAiProviderModelOptionsRequest, ListAiProviderModelOptionsResponse>({
      url: AI_PROVIDER_MODELS_URL,
      method: "get",
      params: request
    });
  }
}

export const defAiModelService = new AiModelServiceImpl();
