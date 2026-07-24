import service, { getRequestAccessToken, handleAuthExpired, requestBaseURL } from "@/utils/request";
import type { ListAiMessageRequest, ListAiMessageResponse } from "@/rpc/base/v1/ai_session";
import type {
  AiMessageService,
  DeleteAiMessageRequest,
  DeleteAiMessageResponse,
  RegenerateAiMessageRequest,
  RetryAiUserMessageRequest,
  SendAiMessageRequest,
  SendAiMessageResponse,
  UpdateAiMessageRequest
} from "@/rpc/base/v1/ai_message";

const AI_SESSION_URL = "/v1/base/ai/session";

/** direct stream 请求控制选项。 */
export type AiMessageStreamOptions = {
  /** 外部取消信号，用于页面卸载或会话删除时终止流式请求。 */
  signal?: AbortSignal;
};

/** 从 direct stream 错误响应中提取后端业务提示。 */
async function resolveStreamErrorMessage(response: Response): Promise<string> {
  const fallbackMessage = `AI 助手请求失败（${response.status}）`;
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const payload = await response.json();
      return String(payload?.message || payload?.error || fallbackMessage);
    } catch {
      return fallbackMessage;
    }
  }
  try {
    const text = (await response.text()).trim();
    return text || fallbackMessage;
  } catch {
    return fallbackMessage;
  }
}

/** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供调用方消费。 */
export async function SendAiMessageStream(
  request: SendAiMessageRequest,
  options?: AiMessageStreamOptions
): Promise<Response> {
  const accessToken = await getRequestAccessToken();
  const headers: Record<string, string> = {
    Accept: "text/event-stream",
    "Content-Type": "application/json;charset=utf-8"
  };
  if (accessToken) headers.Authorization = accessToken;

  const response = await fetch(`${requestBaseURL}${AI_SESSION_URL}/${request.session_id}/message`, {
    method: "POST",
    headers,
    body: JSON.stringify(request),
    signal: options?.signal
  });

  // direct stream 不经过 axios 响应拦截器，需要在这里补齐登录失效处理。
  if (response.status === 401 || response.status === 403) {
    handleAuthExpired();
    throw new Error("登录状态已失效，请重新登录");
  }
  if (!response.ok) {
    throw new Error(await resolveStreamErrorMessage(response));
  }

  return response;
}

/** AI 助手消息服务。 */
export class AiMessageServiceImpl implements AiMessageService {
  /** 查询 AI 助手消息列表。 */
  ListAiMessage(request: ListAiMessageRequest): Promise<ListAiMessageResponse> {
    return service<ListAiMessageRequest, ListAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: "get",
      params: request
    });
  }

  /** 发送 AI 助手消息并等待完整响应。 */
  SendAiMessage(request: SendAiMessageRequest): Promise<SendAiMessageResponse> {
    return service<SendAiMessageRequest, SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: "post",
      data: request
    });
  }

  /** 删除 AI 助手消息。 */
  DeleteAiMessage(request: DeleteAiMessageRequest): Promise<DeleteAiMessageResponse> {
    return service<DeleteAiMessageRequest, DeleteAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}`,
      method: "delete",
      params: request
    });
  }

  /** 更新 AI 助手消息文本并重新生成输出。 */
  UpdateAiMessage(request: UpdateAiMessageRequest): Promise<SendAiMessageResponse> {
    return service<UpdateAiMessageRequest, SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}`,
      method: "put",
      data: request
    });
  }

  /** 重试失败的 AI 助手消息。 */
  RetryAiUserMessage(request: RetryAiUserMessageRequest): Promise<SendAiMessageResponse> {
    return service<RetryAiUserMessageRequest, SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}/retry`,
      method: "post",
      data: request
    });
  }

  /** 重新生成 AI 助手输出。 */
  RegenerateAiMessage(request: RegenerateAiMessageRequest): Promise<SendAiMessageResponse> {
    return service<RegenerateAiMessageRequest, SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}/regeneration`,
      method: "post",
      data: request
    });
  }

  /** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供调用方消费。 */
  async StreamAiMessage(
    request: SendAiMessageRequest,
    options?: AiMessageStreamOptions
  ): Promise<Response> {
    return SendAiMessageStream(request, options);
  }
}

export const defAiMessageService = new AiMessageServiceImpl();
