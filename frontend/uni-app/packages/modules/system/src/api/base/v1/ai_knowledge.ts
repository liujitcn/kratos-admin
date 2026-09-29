import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiKnowledgeService,
  ListAiKnowledgeOptionsRequest,
  ListAiKnowledgeOptionsResponse,
} from '../../../rpc/base/v1/ai_knowledge'

const AI_KNOWLEDGE_OPTIONS_URL = '/v1/base/ai/knowledge/options'

/** AI知识库选项服务。 */
export class AiKnowledgeServiceImpl implements AiKnowledgeService {
  /** 查询可用的AI知识库选项。 */
  ListAiKnowledgeOptions(
    request: ListAiKnowledgeOptionsRequest,
  ): Promise<ListAiKnowledgeOptionsResponse> {
    return http<ListAiKnowledgeOptionsResponse>({
      url: AI_KNOWLEDGE_OPTIONS_URL,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAiKnowledgeService = new AiKnowledgeServiceImpl()
