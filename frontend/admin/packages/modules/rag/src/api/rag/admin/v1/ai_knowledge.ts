import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiKnowledgeForm,
  AiKnowledgeService,
  CreateAiKnowledgeRequest,
  DeleteAiKnowledgeRequest,
  GetAiKnowledgeRequest,
  ListAiKnowledgeModelsRequest,
  ListAiKnowledgeModelsResponse,
  PageAiKnowledgeRequest,
  PageAiKnowledgeResponse,
  UpdateAiKnowledgeRequest
} from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge";
import type { Empty } from "@liujitcn/kratos-admin-rag/rpc/google/protobuf/empty";

const AI_KNOWLEDGE_URL = "/v1/admin/rag/knowledge";

/** AI知识库管理服务实现。 */
export class AiKnowledgeServiceImpl implements AiKnowledgeService {
  /** 分页查询AI知识库。 */
  PageAiKnowledge(request: PageAiKnowledgeRequest): Promise<PageAiKnowledgeResponse> {
    return service({ url: AI_KNOWLEDGE_URL, method: "get", params: request });
  }

  /** 查询AI知识库表单。 */
  GetAiKnowledge(request: GetAiKnowledgeRequest): Promise<AiKnowledgeForm> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}`, method: "get" });
  }

  /** 创建AI知识库。 */
  CreateAiKnowledge(request: CreateAiKnowledgeRequest): Promise<Empty> {
    return service({ url: AI_KNOWLEDGE_URL, method: "post", data: request.ai_knowledge });
  }

  /** 更新AI知识库。 */
  UpdateAiKnowledge(request: UpdateAiKnowledgeRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.ai_knowledge?.id ?? ""}`, method: "put", data: request.ai_knowledge });
  }

  /** 删除AI知识库。 */
  DeleteAiKnowledge(request: DeleteAiKnowledgeRequest): Promise<Empty> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}`, method: "delete" });
  }

  /** 查询已启用的embedding模型选项。 */
  ListAiKnowledgeModels(request: ListAiKnowledgeModelsRequest): Promise<ListAiKnowledgeModelsResponse> {
    return service({ url: `${AI_KNOWLEDGE_URL}/models`, method: "get", params: request });
  }
}

export const defAiKnowledgeService = new AiKnowledgeServiceImpl();
