import { onBeforeUnmount, type Ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiMessageService } from "../../api/base/v1/ai_message";
import type { AiSession } from "../../rpc/base/v1/ai_session";
import {
  appendStreamingDelta,
  completeStreamingActivities,
  ensureStreamingMessage,
  hasStreamingDelta,
  markStreamingError,
  markThinkingMessageFailed,
  normalizeMessageList,
  replacePendingMessages,
  upsertStreamingActivity
} from "../../views/ai/chat/message";
import { readAiEventStream, readAiMessageEvents } from "../../views/ai/chat/stream";
import type { AiStreamEvent, AiStreamPayload, ChatMessageItem, SubmitPayload } from "../../views/ai/chat/types";

/** AI 聊天流式请求使用的模型和知识库选择。 */
export interface AiChatStreamRequest {
  /** 供应商编号。 */
  providerID: number;
  /** 模型名称。 */
  modelName: string;
  /** 检索使用的知识库编号。 */
  knowledgeBaseIDs: number[];
}

/** useAiChatStream 的状态回调和消息存储。 */
export interface AiChatStreamOptions {
  /** 按会话保存聊天消息。 */
  messages: Ref<Record<string, ChatMessageItem[]>>;
  /** 更新会话发送状态。 */
  setSessionSending: (sessionID: string, sending: boolean) => void;
  /** 流完成后同步会话摘要。 */
  onSession?: (session: AiSession) => void;
  /** 请求或流解析失败后的展示回调。 */
  onError?: (error: Error) => void;
}

/** 当前执行或只读补读任务，取消后拒绝晚到结果。 */
type AiStreamTask = {
  controller: AbortController;
  finished: boolean;
  messageID?: string;
};

/** useAiChatStream 管理 direct stream、活动时间线和多会话取消。 */
export function useAiChatStream(options: AiChatStreamOptions) {
  const pendingDeltaMap = new Map<string, AiStreamPayload>();
  const runningStreamTaskMap = new Map<string, AiStreamTask>();
  let pendingDeltaFrame = 0;

  /** 启动一次会话消息流。 */
  async function run(sessionID: string, payload: SubmitPayload, request: AiChatStreamRequest) {
    const controller = new AbortController();
    const task: AiStreamTask = { controller, finished: false };
    runningStreamTaskMap.set(sessionID, task);
    try {
      const response = await defAiMessageService.StreamAiMessage(
        {
          session_id: sessionID,
          content: payload.text,
          provider_id: request.providerID,
          model_name: request.modelName,
          knowledge_base_ids: request.knowledgeBaseIDs,
          attachments: payload.attachments.map(item => ({
            id: item.id,
            name: item.name,
            size: item.size,
            url: item.url,
            mime_type: item.mime_type
          })),
          action: payload.action
        },
        { signal: controller.signal }
      );
      if (!response.body) throw new Error(t("system.ai.chat.message.empty_stream"));
      await readAiEventStream(response.body, event => handleAiStreamEvent(event, task), controller.signal);
      if (!task.finished && !controller.signal.aborted) throw new Error(t("system.ai.chat.message.incomplete_stream"));
    } catch (cause) {
      if (controller.signal.aborted) return;
      if (task.messageID) {
        flushAiDelta();
        await resume(sessionID, task.messageID, task);
        return;
      }
      const error = cause instanceof Error ? cause : new Error(t("system.ai.chat.message.request_failed"));
      options.messages.value[sessionID] = markThinkingMessageFailed(options.messages.value[sessionID] ?? [], sessionID, error.message);
      options.onError?.(error);
    } finally {
      if (runningStreamTaskMap.get(sessionID) === task) {
        runningStreamTaskMap.delete(sessionID);
        options.setSessionSending(sessionID, false);
      }
    }
  }

  /** 刷新或断流后只读恢复原消息。 */
  async function recover(sessionID: string, messageID: string) {
    if (runningStreamTaskMap.has(sessionID)) return;
    const task: AiStreamTask = { controller: new AbortController(), finished: false, messageID };
    runningStreamTaskMap.set(sessionID, task);
    options.setSessionSending(sessionID, true);
    try {
      await resume(sessionID, messageID, task);
    } finally {
      if (runningStreamTaskMap.get(sessionID) === task) {
        runningStreamTaskMap.delete(sessionID);
        options.setSessionSending(sessionID, false);
      }
    }
  }

  /** 展示恢复进度，失败时保留已收到的正文并允许再次补读。 */
  async function resume(sessionID: string, messageID: string, task: AiStreamTask) {
    const isCancelled = () => task.controller.signal.aborted || runningStreamTaskMap.get(sessionID) !== task;
    const payload = { session_id: sessionID, message_id: messageID };
    const current = ensureStreamingMessage(options.messages.value[sessionID] ?? [], payload);
    options.messages.value[sessionID] = current.map(item =>
      item.role === "ai" && item.id === messageID ? { ...item, recovery: "reconnecting" } : item
    );
    try {
      await readAiMessageEvents(
        cursor => defAiMessageService.ListAiMessageEvent({ session_id: sessionID, message_id: messageID, after_id: cursor }),
        event => handleAiStreamEvent(event, task),
        isCancelled
      );
    } catch (cause) {
      if (!isCancelled()) {
        options.messages.value[sessionID] = (options.messages.value[sessionID] ?? []).map(item =>
          item.role === "ai" && item.id === messageID ? { ...item, recovery: "disconnected" } : item
        );
        options.onError?.(cause instanceof Error ? cause : new Error(String(cause)));
      }
    }
  }

  /** 取消指定会话的流式请求。 */
  function cancelSession(sessionID: string) {
    const task = runningStreamTaskMap.get(sessionID);
    if (!task) return;
    task.finished = true;
    task.controller.abort();
    runningStreamTaskMap.delete(sessionID);
    options.setSessionSending(sessionID, false);
  }

  /** 取消所有会话的流式请求并清理待刷新的分片。 */
  function cancelAll() {
    Array.from(runningStreamTaskMap.keys()).forEach(cancelSession);
    if (pendingDeltaFrame) {
      window.cancelAnimationFrame(pendingDeltaFrame);
      pendingDeltaFrame = 0;
    }
    pendingDeltaMap.clear();
  }

  /** 处理 AI 助手事件，权威终态到达后立即释放旧流以接续审批。 */
  function handleAiStreamEvent(event: AiStreamEvent, task: AiStreamTask) {
    if (task.controller.signal.aborted || runningStreamTaskMap.get(event.payload.session_id) !== task) return;
    task.messageID = event.payload.message_id;
    if (event.event === "delta") {
      if (hasStreamingDelta(event.payload)) queueAiDelta(event.payload);
      return;
    }
    if (event.event === "activity") {
      handleAiActivity(event.payload);
      return;
    }
    if (event.event === "finish" || event.event === "approval_required") {
      handleAiFinish(event.payload, task);
    } else {
      handleAiError(event.payload, task);
    }
    if (event.event !== "approval_required") cancelSession(event.payload.session_id);
  }

  /** 处理流式活动时间线事件。 */
  function handleAiActivity(payload: AiStreamPayload) {
    const sessionID = String(payload.session_id ?? "");
    if (!sessionID || !payload.activity) return;
    const messages = ensureStreamingMessage(options.messages.value[sessionID] ?? [], payload);
    options.messages.value[sessionID] = upsertStreamingActivity(messages, payload);
  }

  /** 收到最终消息后替换本地占位并同步会话。 */
  function handleAiFinish(payload: AiStreamPayload, task: AiStreamTask) {
    const sessionID = String(payload.session_id ?? "");
    if (!sessionID) return;
    task.finished = true;
    flushAiDelta();
    const nextMessages = normalizeMessageList(payload.messages as never[]);
    const current = completeStreamingActivities(options.messages.value[sessionID] ?? [], payload);
    const messageID = String(payload.message_id ?? "");
    const streamKey = messageID ? `${sessionID}:${messageID}` : "";
    const hasLocalStreamingMessages = current.some(item => item.streamKey === streamKey && item.localOnly);
    options.messages.value[sessionID] =
      nextMessages.length || !hasLocalStreamingMessages ? replacePendingMessages(current, nextMessages, payload) : current;
    if (payload.session) options.onSession?.(payload.session as AiSession);
  }

  /** 收到错误事件后优先使用服务端返回的消息，否则标记当前流失败。 */
  function handleAiError(payload: AiStreamPayload, task: AiStreamTask) {
    const sessionID = String(payload.session_id ?? "");
    if (!sessionID || !options.messages.value[sessionID]) return;
    task.finished = true;
    flushAiDelta();
    const nextMessages = normalizeMessageList(payload.messages as never[]);
    if (nextMessages.length) {
      options.messages.value[sessionID] = replacePendingMessages(options.messages.value[sessionID] ?? [], nextMessages, payload);
      return;
    }
    const messages = ensureStreamingMessage(options.messages.value[sessionID] ?? [], payload);
    options.messages.value[sessionID] = markStreamingError(messages, payload);
  }

  /** 合并同一帧内的分片，降低消息列表重排频率。 */
  function queueAiDelta(payload: AiStreamPayload) {
    const sessionID = String(payload.session_id ?? "");
    const messageID = String(payload.message_id ?? "");
    if (!sessionID || !messageID || !options.messages.value[sessionID]) return;
    const key = `${sessionID}:${messageID}`;
    const cachedPayload = pendingDeltaMap.get(key);
    pendingDeltaMap.set(key, { ...payload, delta: `${cachedPayload?.delta ?? ""}${payload.delta ?? ""}` });
    if (pendingDeltaFrame) return;
    pendingDeltaFrame = window.requestAnimationFrame(() => {
      pendingDeltaFrame = 0;
      flushAiDelta();
    });
  }

  /** 立即刷新缓存的流式分片。 */
  function flushAiDelta() {
    if (pendingDeltaFrame) {
      window.cancelAnimationFrame(pendingDeltaFrame);
      pendingDeltaFrame = 0;
    }
    if (!pendingDeltaMap.size) return;
    const payloadList = Array.from(pendingDeltaMap.values());
    pendingDeltaMap.clear();
    for (const payload of payloadList) {
      const sessionID = String(payload.session_id ?? "");
      if (!sessionID || !options.messages.value[sessionID]) continue;
      const messages = ensureStreamingMessage(options.messages.value[sessionID] ?? [], payload);
      options.messages.value[sessionID] = appendStreamingDelta(messages, payload);
    }
  }

  onBeforeUnmount(cancelAll);

  return { run, recover, cancelSession, cancelAll };
}
