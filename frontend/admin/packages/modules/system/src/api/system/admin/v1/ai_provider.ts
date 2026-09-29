import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiProviderForm,
  AiProviderService,
  CreateAiProviderRequest,
  DeleteAiProviderRequest,
  GetAiProviderRequest,
  ListAiProviderOptionsRequest,
  ListAiProviderOptionsResponse,
  PageAiProviderRequest,
  PageAiProviderResponse,
  SetAiProviderStatusRequest,
  TestAiProviderModelsRequest,
  TestAiProviderModelsResponse,
  UpdateAiProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_provider";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const AI_PROVIDER_URL = "/v1/admin/base/ai-provider";

/** AI供应商管理服务实现。 */
export class AiProviderServiceImpl implements AiProviderService {
  /** 分页查询AI供应商。 */
  PageAiProvider(request: PageAiProviderRequest): Promise<PageAiProviderResponse> {
    return service({ url: AI_PROVIDER_URL, method: "get", params: request });
  }

  /** 查询AI供应商表单。 */
  GetAiProvider(request: GetAiProviderRequest): Promise<AiProviderForm> {
    return service({ url: `${AI_PROVIDER_URL}/${request.id}`, method: "get" });
  }

  /** 创建AI供应商。 */
  CreateAiProvider(request: CreateAiProviderRequest): Promise<Empty> {
    return service({ url: AI_PROVIDER_URL, method: "post", data: request.ai_provider });
  }

  /** 更新AI供应商。 */
  UpdateAiProvider(request: UpdateAiProviderRequest): Promise<Empty> {
    return service({ url: `${AI_PROVIDER_URL}/${request.ai_provider?.id ?? ""}`, method: "put", data: request.ai_provider });
  }

  /** 查询已启用的AI供应商选项。 */
  ListAiProviderOptions(request: ListAiProviderOptionsRequest): Promise<ListAiProviderOptionsResponse> {
    return service({ url: `${AI_PROVIDER_URL}/options`, method: "get", params: request });
  }

  /** 测试供应商表单中的全部模型。 */
  TestAiProviderModels(request: TestAiProviderModelsRequest): Promise<TestAiProviderModelsResponse> {
    return service({ url: `${AI_PROVIDER_URL}/test-models`, method: "post", data: request.ai_provider });
  }

  /** 删除AI供应商。 */
  DeleteAiProvider(request: DeleteAiProviderRequest): Promise<Empty> {
    return service({ url: `${AI_PROVIDER_URL}/${request.id}`, method: "delete" });
  }

  /** 设置AI供应商状态。 */
  SetAiProviderStatus(request: SetAiProviderStatusRequest): Promise<Empty> {
    return service({ url: `${AI_PROVIDER_URL}/${request.id}/status`, method: "put", data: request });
  }
}

export const defAiProviderService = new AiProviderServiceImpl();
