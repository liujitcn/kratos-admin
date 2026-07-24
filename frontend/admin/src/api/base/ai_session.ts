import service from "@/utils/request";
import type {
  AiSessionService,
  CreateAiSessionBranchRequest,
  CreateAiSessionBranchResponse,
  CreateAiSessionRequest,
  CreateAiSessionResponse,
  DeleteAiSessionRequest,
  DeleteAiSessionResponse,
  ListAiMessageRequest,
  ListAiMessageResponse,
  ListAiSessionRequest,
  ListAiSessionResponse,
  UpdateAiSessionRequest,
  UpdateAiSessionResponse
} from "@/rpc/base/v1/ai_session";

const AI_SESSION_URL = "/v1/base/ai/session";

/** AI 助手会话服务。 */
export class AiSessionServiceImpl implements AiSessionService {
  /** 查询 AI 助手会话列表。 */
  ListAiSession(request: ListAiSessionRequest): Promise<ListAiSessionResponse> {
    return service<ListAiSessionRequest, ListAiSessionResponse>({
      url: `${AI_SESSION_URL}`,
      method: "get",
      params: request
    });
  }

  /** 创建 AI 助手会话。 */
  CreateAiSession(request: CreateAiSessionRequest): Promise<CreateAiSessionResponse> {
    return service<CreateAiSessionRequest, CreateAiSessionResponse>({
      url: `${AI_SESSION_URL}`,
      method: "post",
      data: request
    });
  }

  /** 更新 AI 助手会话。 */
  UpdateAiSession(request: UpdateAiSessionRequest): Promise<UpdateAiSessionResponse> {
    return service<UpdateAiSessionRequest, UpdateAiSessionResponse>({
      url: `${AI_SESSION_URL}/${request.id}`,
      method: "put",
      data: request
    });
  }

  /** 删除 AI 助手会话。 */
  DeleteAiSession(request: DeleteAiSessionRequest): Promise<DeleteAiSessionResponse> {
    return service<DeleteAiSessionRequest, DeleteAiSessionResponse>({
      url: `${AI_SESSION_URL}/${request.id}`,
      method: "delete",
      params: request
    });
  }

  /** 查询 AI 助手消息列表。 */
  ListAiMessage(request: ListAiMessageRequest): Promise<ListAiMessageResponse> {
    return service<ListAiMessageRequest, ListAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 从指定消息创建 AI 助手分支会话。 */
  CreateAiSessionBranch(request: CreateAiSessionBranchRequest): Promise<CreateAiSessionBranchResponse> {
    return service<CreateAiSessionBranchRequest, CreateAiSessionBranchResponse>({
      url: `${AI_SESSION_URL}/${request.source_session_id}/branch`,
      method: "post",
      data: request
    });
  }
}

export const defAiSessionService = new AiSessionServiceImpl();
