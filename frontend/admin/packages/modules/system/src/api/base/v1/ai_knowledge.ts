import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiKnowledgeService,
  ListAiKnowledgeOptionsRequest,
  ListAiKnowledgeOptionsResponse
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_knowledge";

const AI_KNOWLEDGE_OPTIONS_URL = "/v1/base/ai/knowledge/options";

/** AI知识库选项服务。 */
export class AiKnowledgeServiceImpl implements AiKnowledgeService {
  /** 查询可用的AI知识库选项。 */
  ListAiKnowledgeOptions(request: ListAiKnowledgeOptionsRequest): Promise<ListAiKnowledgeOptionsResponse> {
    return service<ListAiKnowledgeOptionsRequest, ListAiKnowledgeOptionsResponse>({
      url: AI_KNOWLEDGE_OPTIONS_URL,
      method: "get",
      params: request
    });
  }
}

export const defAiKnowledgeService = new AiKnowledgeServiceImpl();
