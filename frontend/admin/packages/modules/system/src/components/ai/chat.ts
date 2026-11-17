/** AI 聊天公开协议，供外部布局组合 ChatPanel 时复用。 */
export type {
  AIFlowBlock,
  AiActivityItem,
  AiActivityPhase,
  AiActivityStatus,
  AiStreamEvent,
  AiStreamEventName,
  AiStreamPayload,
  ChatMessageAction,
  ChatMessageEditPayload,
  ChatMessageItem,
  MessageProgressState,
  ReplySourceTag,
  SessionAction,
  SubmitPayload
} from "../../views/ai/chat/types";

/** AI 聊天消息归一化与流式状态辅助函数。 */
export {
  appendStreamingDelta,
  buildPendingStreamMessageKey,
  buildStreamMessageKey,
  completeStreamingActivities,
  createLocalUserMessage,
  createThinkingMessage,
  ensureStreamingMessage,
  hasStreamingDelta,
  markAIMessageRegenerating,
  markFlowBlocksDisabled,
  markSpeakingMessage,
  markStreamingError,
  markThinkingMessageFailed,
  mapMessageItem,
  normalizeMessageList,
  normalizeSession,
  normalizeSessionList,
  parseFlowBlocks,
  replacePendingMessages,
  resolveReplySourceTag,
  resolveTimestamp,
  sortMessages,
  upsertStreamingActivity
} from "../../views/ai/chat/message";

/** 读取管理端 AI direct stream 事件。 */
export { readAiEventStream } from "../../views/ai/chat/stream";

/** AI 聊天状态容器，供自定义布局复用会话和消息状态。 */
export { useAiChatState } from "./useAiChatState";

/** AI 聊天 direct stream 状态容器，供自定义布局复用流式处理和取消逻辑。 */
export { useAiChatStream } from "./useAiChatStream";
