import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiSessionService,
  PageAiSessionMessageRequest,
  PageAiSessionMessageResponse,
  PageAiSessionRequest,
  PageAiSessionResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_session";

const AI_SESSION_URL = "/v1/admin/base/ai-session";

/** AI会话管理服务实现。 */
export class AiSessionServiceImpl implements AiSessionService {
  /** 分页查询AI会话。 */
  PageAiSession(request: PageAiSessionRequest): Promise<PageAiSessionResponse> {
    return service({ url: AI_SESSION_URL, method: "get", params: request });
  }

  /** 分页查询AI会话消息。 */
  PageAiSessionMessage(request: PageAiSessionMessageRequest): Promise<PageAiSessionMessageResponse> {
    return service({ url: `${AI_SESSION_URL}/${request.id}/message`, method: "get", params: { page_num: request.page_num, page_size: request.page_size } });
  }
}

export const defAiSessionService = new AiSessionServiceImpl();
