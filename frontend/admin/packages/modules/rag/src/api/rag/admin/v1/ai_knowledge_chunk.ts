import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiKnowledgeChunkService,
  SearchAiKnowledgeRequest,
  SearchAiKnowledgeResponse
} from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge_chunk";

const AI_KNOWLEDGE_URL = "/v1/admin/rag/knowledge";

/** AI知识库切片检索服务实现。 */
export class AiKnowledgeChunkServiceImpl implements AiKnowledgeChunkService {
  /** 检索测试。 */
  SearchAiKnowledge(request: SearchAiKnowledgeRequest): Promise<SearchAiKnowledgeResponse> {
    return service({ url: `${AI_KNOWLEDGE_URL}/${request.id}/search`, method: "post", data: { query: request.query, top_k: request.top_k } });
  }
}

export const defAiKnowledgeChunkService = new AiKnowledgeChunkServiceImpl();
