import {
  type AiAttachment,
  type AiInputContent,
  type AiMessage,
  type AiOutputContent,
  type AiSession,
  type AiToken
} from "@/rpc/base/v1/ai_session";
import type { AiAction } from "@/rpc/base/v1/ai_message";
import { AiMessageStatus } from "@/rpc/common/v1/enum";
import { Terminal } from "@/rpc/common/v1/enum";
import type { AiStreamPayload, AIFlowBlock, ChatMessageItem, ReplySourceTag } from "./types";

const THINKING_MESSAGE_ID_PREFIX = "ai-thinking";
const LOCAL_USER_MESSAGE_ID_PREFIX = "ai-user-local";
const PENDING_STREAM_MESSAGE_ID = "pending";
const THINKING_MESSAGE_CONTENT = "正在整理回复";

/** 生成流式消息分组键，确保同一轮回复只更新当前占位气泡。 */
export function buildStreamMessageKey(sessionID: string, messageID: string) {
  return [sessionID, messageID].join(":");
}

/** 生成待绑定真实消息编号前的本地流式分组键。 */
export function buildPendingStreamMessageKey(sessionID: string) {
  return buildStreamMessageKey(sessionID, PENDING_STREAM_MESSAGE_ID);
}

/** 计算消息角色排序权重，确保同一轮对话里用户问题先于助手回答。 */
function resolveRoleOrder(role?: string) {
  return role === "user" ? 0 : 1;
}

/** 解析 protobuf 时间戳为毫秒。 */
export function resolveTimestamp(timestamp?: { seconds?: number; nanos?: number }) {
  if (!timestamp) return 0;
  const seconds = Number(timestamp.seconds ?? 0);
  const nanos = Number(timestamp.nanos ?? 0);
  return seconds * 1000 + Math.floor(nanos / 1_000_000);
}

/** 归一化会话对象，避免空值影响渲染。 */
export function normalizeSession(session?: Partial<AiSession> | null): AiSession {
  return {
    id: String(session?.id ?? ""),
    title: String(session?.title ?? "新对话"),
    summary: String(session?.summary ?? ""),
    updated_at: session?.updated_at,
    terminal: Number(session?.terminal ?? Terminal.TERMINAL_ADMIN)
  };
}

/** 将会话列表收敛成可安全渲染的数组。 */
export function normalizeSessionList(list?: AiSession[] | null) {
  if (!Array.isArray(list)) return [];
  return list.map(item => normalizeSession(item)).filter(item => item.id);
}

/** 生成回复来源标签。 */
export function resolveReplySourceTag(item: Pick<ChatMessageItem, "role" | "fallback" | "reply_source" | "model">): ReplySourceTag | undefined {
  if (item.role === "user") return undefined;
  if (item.fallback) return { text: "降级回复", tone: "warning" };
  switch (String(item.reply_source ?? "")) {
    case "network":
      return { text: "网络数据", tone: "success" };
    case "llm":
      return { text: "模型回答", tone: "primary" };
    case "fallback":
      return { text: "降级回复", tone: "warning" };
    default:
      return item.model ? { text: "模型回答", tone: "primary" } : undefined;
  }
}

/** 归一化输入内容，避免后端空对象影响渲染。 */
function normalizeInputContent(content?: AiInputContent): AiInputContent {
  return {
    kind: String(content?.kind || "text"),
    content: String(content?.content ?? "")
  };
}

/** 归一化输出内容，避免后端空对象影响渲染。 */
function normalizeOutputContent(content?: AiOutputContent): AiOutputContent {
  return {
    kind: String(content?.kind || "text"),
    content: String(content?.content ?? ""),
    reply_source: String(content?.reply_source ?? ""),
    model: String(content?.model ?? ""),
    fallback: Boolean(content?.fallback),
    fallback_reason: String(content?.fallback_reason ?? ""),
    flow: String(content?.flow ?? ""),
    step: String(content?.step ?? ""),
    blocks_json: String(content?.blocks_json ?? "")
  };
}

/** 解析结构化 Flow 卡片，非法 JSON 直接降级为空数组。 */
export function parseFlowBlocks(raw?: string) {
  if (!raw) return [];
  try {
    const blocks = JSON.parse(raw) as unknown;
    if (!Array.isArray(blocks)) return [];
    return blocks.filter((item): item is AIFlowBlock => Boolean(item && typeof item === "object" && String(item.type ?? "")));
  } catch {
    return [];
  }
}

/** 根据消息状态禁用过期结构化动作，减少用户误点旧步骤。 */
export function markFlowBlocksDisabled(blocks: AIFlowBlock[], messageID: string) {
  return blocks.map(block => markFlowValueDisabled(block, messageID)) as AIFlowBlock[];
}

/** 递归标记结构化卡片中的 action 是否仍属于当前消息。 */
function markFlowValueDisabled(value: unknown, messageID: string): unknown {
  if (Array.isArray(value)) {
    value.forEach(item => markFlowValueDisabled(item, messageID));
    return value;
  }
  if (!value || typeof value !== "object") return value;

  const current = value as Record<string, any>;
  if (current.type && current.flow) {
    current.disabled = !isFlowActionFromMessage(current, messageID);
  }
  if (current.action?.type) {
    current.disabled = !isFlowActionFromMessage(current.action, messageID);
  }
  Object.values(current).forEach(item => markFlowValueDisabled(item, messageID));
  return current;
}

/** 判断结构化动作是否来自指定助手消息。 */
export function isFlowActionFromMessage(action: Partial<AiAction> | undefined, messageID: string) {
  if (!messageID || !action?.source_message_id || !action.action_id) return false;
  return action.source_message_id === messageID && String(action.flow_version || "") === messageID;
}

/** 归一化 token 统计，保证用量展示字段都有默认值。 */
function normalizeToken(token?: AiToken): AiToken {
  return {
    input: Number(token?.input ?? 0),
    output: Number(token?.output ?? 0),
    cache: Number(token?.cache ?? 0),
    total: Number(token?.total ?? 0)
  };
}

/** 将后端一轮消息映射为单个聊天气泡结构。 */
export function mapMessageItem(message: AiMessage, role: "user" | "ai"): ChatMessageItem {
  const inputContent = normalizeInputContent(message.input_content);
  const outputContent = normalizeOutputContent(message.output_content);
  const content = role === "user" ? inputContent : outputContent;
  const item: ChatMessageItem = {
    ...message,
    role,
    key: `${message.id}:${role}`,
    content: content.content,
    kind: content.kind,
    placement: role === "user" ? "end" : "start",
    reply_source: role === "ai" ? outputContent.reply_source : "",
    model: role === "ai" ? outputContent.model : "",
    fallback: role === "ai" && outputContent.fallback,
    fallback_reason: role === "ai" ? outputContent.fallback_reason : "",
    flow: role === "ai" ? outputContent.flow : "",
    step: role === "ai" ? outputContent.step : "",
    blocksJson: role === "ai" ? outputContent.blocks_json : "",
    blocks:
      role === "ai"
        ? markFlowBlocksDisabled(parseFlowBlocks(outputContent.blocks_json), String(message.id ?? ""))
        : [],
    status: Number(message.status ?? AiMessageStatus.SUCCESS_AAMS),
    token: normalizeToken(message.token),
    tools: Array.isArray(message.tools) ? message.tools : [],
    variant: role === "user" ? "filled" : "borderless",
    shape: "corner",
    progressState:
      message.status === AiMessageStatus.GENERATING_AAMS
        ? role === "user"
          ? "idle"
          : "streaming"
        : message.status === AiMessageStatus.FAILED_AAMS
          ? "failed"
          : "idle",
    maxWidth: role === "user" ? "380px" : "100%"
  };
  item.replySourceTag = resolveReplySourceTag(item);
  return item;
}

/** 收敛消息数组，并把每轮消息拆成用户气泡和助手气泡。 */
export function normalizeMessageList(list?: AiMessage[] | null) {
  if (!Array.isArray(list)) return [];
  return list.filter(Boolean).flatMap(item => [mapMessageItem(item, "user"), mapMessageItem(item, "ai")]);
}

/** 创建本地用户回显消息。 */
export function createLocalUserMessage(payload: {
  text: string;
  attachments: AiAttachment[];
}) {
  const now = new Date();
  const message = mapMessageItem(
    {
      id: `${LOCAL_USER_MESSAGE_ID_PREFIX}-${now.getTime()}`,
      input_content: {
        kind: "text",
        content: payload.text
      },
      output_content: undefined,
      attachments: payload.attachments,
      created_at: {
        seconds: Math.floor(now.getTime() / 1000),
        nanos: (now.getTime() % 1000) * 1_000_000
      },
      status: AiMessageStatus.GENERATING_AAMS,
      token: normalizeToken(),
      tools: [],
      first_token_ms: 0,
      duration_ms: 0
    },
    "user"
  );
  // 本地回显消息只用于等待服务端响应，收到正式消息后需要被替换掉，避免同一问题展示两遍。
  message.localOnly = true;
  // 用户消息只是本地回显，不参与助手流式动画，避免用户气泡出现思考中的省略点。
  message.progressState = "idle";
  return message;
}

/** 创建助手思考中的占位消息。 */
export function createThinkingMessage(options?: { sessionID?: string; messageID?: string }) {
  const now = new Date();
  const streamKey = options?.sessionID
    ? buildStreamMessageKey(options.sessionID, options.messageID || PENDING_STREAM_MESSAGE_ID)
    : undefined;
  const message = mapMessageItem(
    {
      id: streamKey || `${THINKING_MESSAGE_ID_PREFIX}-${now.getTime()}`,
      input_content: undefined,
      output_content: {
        kind: "text",
        content: THINKING_MESSAGE_CONTENT,
        reply_source: "",
        model: "",
        fallback: false,
        fallback_reason: "",
        flow: "",
        step: "",
        blocks_json: ""
      },
      attachments: [],
      created_at: {
        seconds: Math.floor(now.getTime() / 1000),
        nanos: (now.getTime() % 1000) * 1_000_000
      },
      status: AiMessageStatus.GENERATING_AAMS,
      token: normalizeToken(),
      tools: [],
      first_token_ms: 0,
      duration_ms: 0
    },
    "ai"
  );
  message.progressState = "streaming";
  message.localOnly = true;
  message.streamKey = streamKey;
  message.replySourceTag = { text: "思考中", tone: "info" };
  return message;
}

/** 确保当前轮次存在流式占位消息。 */
export function ensureStreamingMessage(current: ChatMessageItem[], payload: AiStreamPayload) {
  const sessionID = String(payload.session_id ?? "");
  const messageID = String(payload.message_id ?? "");
  if (!sessionID || !messageID) return current;

  const streamKey = buildStreamMessageKey(sessionID, messageID);
  if (current.some(item => item.streamKey === streamKey)) return current;

  const pendingStreamKey = buildPendingStreamMessageKey(sessionID);
  const next = current.map<ChatMessageItem>(item =>
    item.streamKey === pendingStreamKey ? { ...item, id: messageID, key: `${messageID}:ai`, streamKey } : item
  );
  if (next.some(item => item.streamKey === streamKey)) return next;

  return sortMessages([...next, createThinkingMessage({ sessionID, messageID })]);
}

/** 将已有助手消息清空并标记为重新生成中，保留消息 ID 与原位置。 */
export function markAIMessageRegenerating(current: ChatMessageItem[], sessionID: string, messageID: string) {
  const streamKey = buildStreamMessageKey(sessionID, messageID);
  return current.map<ChatMessageItem>(item => {
    if (String(item.id) !== messageID || item.role === "user") return item;
    return {
      ...item,
      content: THINKING_MESSAGE_CONTENT,
      fallback: false,
      fallback_reason: "",
      token: normalizeToken(),
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
      progressState: "streaming",
      replySourceTag: { text: "思考中", tone: "info" },
      status: AiMessageStatus.GENERATING_AAMS,
      streamKey
    };
  });
}

/** 将消息按创建时间排序。 */
export function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const leftTime = resolveTimestamp(left.created_at);
    const rightTime = resolveTimestamp(right.created_at);
    if (leftTime === rightTime) {
      const roleOrder = resolveRoleOrder(left.role) - resolveRoleOrder(right.role);
      if (roleOrder !== 0) return roleOrder;
      return String(left.id).localeCompare(String(right.id), "zh-Hans-CN", { numeric: true });
    }
    return leftTime - rightTime;
  });
}

/** 去掉当前轮次的本地占位消息，并拼入服务端返回。 */
export function replacePendingMessages(
  current: ChatMessageItem[],
  nextMessages: ChatMessageItem[],
  payload?: AiStreamPayload
) {
  const sessionID = String(payload?.session_id ?? "");
  const streamKey = payload && buildStreamMessageKey(sessionID, String(payload.message_id ?? ""));
  const pendingStreamKey = payload?.message_id && sessionID ? buildPendingStreamMessageKey(sessionID) : "";
  const stableMessages = current.filter(item => {
    if (!item.localOnly) return true;
    if (payload?.message_id && item.role === "user") {
      return !nextMessages.some(message => message.role === "user" && String(message.id) === String(payload.message_id));
    }
    if (!streamKey) return false;
    return item.streamKey !== streamKey && item.streamKey !== pendingStreamKey;
  });
  const messageMap = new Map<string, ChatMessageItem>();

  for (const item of stableMessages) {
    messageMap.set(String(item.key ?? `${item.id}:${item.role}`), item);
  }
  for (const item of nextMessages) {
    messageMap.set(String(item.key ?? `${item.id}:${item.role}`), item);
  }

  return sortMessages(Array.from(messageMap.values()));
}

/** 标记思考中消息失败，可按当前轮次限定失败范围。 */
export function markThinkingMessageFailed(
  current: ChatMessageItem[],
  options?: { sessionID?: string; messageID?: string }
) {
  const streamKey =
    options?.sessionID && options.messageID ? buildStreamMessageKey(options.sessionID, options.messageID) : "";
  return current.map<ChatMessageItem>(item => {
    if (!item.localOnly) return item;
    if (streamKey && item.streamKey !== streamKey) return item;
    if (item.role === "user" && !streamKey) {
      return {
        ...item,
        progressState: "failed",
        status: AiMessageStatus.FAILED_AAMS
      };
    }
    if (item.progressState !== "streaming") return item;
    return {
      ...item,
      progressState: "failed",
      status: AiMessageStatus.FAILED_AAMS,
      content: "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}

/** 判断流式增量是否包含可追加内容。 */
export function hasStreamingDelta(payload: Pick<AiStreamPayload, "delta">) {
  return payload.delta !== undefined && payload.delta !== "";
}

/** 将流式文本增量追加到本地占位消息。 */
export function appendStreamingDelta(current: ChatMessageItem[], payload: AiStreamPayload) {
  if (!hasStreamingDelta(payload)) return current;
  const streamKey = buildStreamMessageKey(String(payload.session_id ?? ""), String(payload.message_id ?? ""));
  return current.map<ChatMessageItem>(item => {
    if (item.streamKey !== streamKey || (!item.localOnly && item.role === "user")) return item;
    const baseContent = item.content === THINKING_MESSAGE_CONTENT ? "" : item.content;
    const nextContent = `${baseContent}${payload.delta ?? ""}`;
    return {
      ...item,
      content: nextContent || item.content,
      progressState: "streaming",
      replySourceTag: { text: "回答中", tone: "info" }
    };
  });
}

/** 根据流式异常事件更新本地占位消息。 */
export function markStreamingError(current: ChatMessageItem[], payload: AiStreamPayload) {
  const messageID = String(payload.message_id ?? "");
  const streamKey = messageID ? buildStreamMessageKey(String(payload.session_id ?? ""), messageID) : "";
  return current.map<ChatMessageItem>(item => {
    if (!item.localOnly) return item;
    if (streamKey && item.streamKey !== streamKey) return item;
    return {
      ...item,
      progressState: "failed",
      status: AiMessageStatus.FAILED_AAMS,
      content: "这次回复没有成功返回，你可以直接重试刚才的问题。",
      replySourceTag: { text: "发送失败", tone: "warning" }
    };
  });
}

/** 标记正在朗读的消息，保证全局只有一个气泡处于朗读态。 */
export function markSpeakingMessage(current: ChatMessageItem[], messageID?: string) {
  return current.map<ChatMessageItem>(item => ({
    ...item,
    speaking: Boolean(messageID && String(item.key ?? `${item.id}:${item.role}`) === messageID)
  }));
}
