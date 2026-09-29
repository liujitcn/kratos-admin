import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiQueryService,
  GetAiQueryRequest,
  PageAiQueryRequest,
  PageAiQueryResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_query";
import type { AiQueryRecord } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_query";

const AI_QUERY_URL = "/v1/admin/base/ai-query";

/** 智能问数记录管理服务实现。 */
export class AiQueryServiceImpl implements AiQueryService {
  /** 分页查询智能问数记录。 */
  PageAiQuery(request: PageAiQueryRequest): Promise<PageAiQueryResponse> {
    return service({ url: AI_QUERY_URL, method: "get", params: request });
  }

  /** 查询智能问数记录详情。 */
  GetAiQuery(request: GetAiQueryRequest): Promise<AiQueryRecord> {
    return service({ url: `${AI_QUERY_URL}/${request.id}`, method: "get" });
  }
}

export const defAiQueryService = new AiQueryServiceImpl();
