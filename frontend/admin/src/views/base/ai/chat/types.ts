import type { AiAction } from "@/rpc/base/v1/ai_message";
import type { AiAttachment, AiMessage, AiSession } from "@/rpc/base/v1/ai_session";

/** 会话菜单动作类型。 */
export type SessionAction = "rename" | "delete";

/** 聊天消息操作类型。 */
export type ChatMessageAction = "retry" | "copy" | "delete" | "branch" | "speak" | "edit";

/** 聊天消息文本编辑提交内容。 */
export type ChatMessageEditPayload = {
  /** 被编辑的用户消息。 */
  item: ChatMessageItem;
  /** 更新后的纯文本内容。 */
  content: string;
};

/** 聊天输入提交内容。 */
export type SubmitPayload = {
  /** 用户输入文本。 */
  text: string;
  /** 已上传附件列表。 */
  attachments: AiAttachment[];
  /** 点击结构化卡片或快捷入口时携带的流程动作。 */
  action?: AiAction;
};

/** AI 助手结构化流程卡片。 */
export type AIFlowBlock = {
  /** 卡片类型，由后端 Flow 输出。 */
  type: string;
  /** 卡片标题。 */
  title?: string;
  /** 卡片说明。 */
  desc?: string;
  /** 单一可执行动作。 */
  action?: AiAction;
  /** 多个可执行动作。 */
  actions?: AiAction[];
  /** 前端是否禁用当前块内动作。 */
  disabled?: boolean;
  /** 兼容后端按不同业务卡片返回的扩展字段。 */
  [key: string]: any;
};

/** 消息进度状态，用于控制气泡加载、失败和常规展示。 */
export type MessageProgressState = "idle" | "streaming" | "failed";

/** 回复来源标签配置。 */
export type ReplySourceTag = {
  /** 标签文本。 */
  text: string;
  /** 标签视觉语义。 */
  tone: "primary" | "success" | "warning" | "info";
};

/** 聊天气泡展示项，在后端消息基础上补充 UI 状态。 */
export type ChatMessageItem = AiMessage & {
  /** 当前气泡角色，由一轮消息拆分得到。 */
  role: "user" | "ai";
  /** 当前气泡正文。 */
  content: string;
  /** 当前气泡内容类型。 */
  kind: string;
  /** 回复来源。 */
  reply_source?: string;
  /** 回复模型。 */
  model?: string;
  /** 是否降级回复。 */
  fallback?: boolean;
  /** 降级原因。 */
  fallback_reason?: string;
  /** 固定流程标识。 */
  flow?: string;
  /** 固定流程步骤。 */
  step?: string;
  /** 原始结构化卡片 JSON。 */
  blocksJson?: string;
  /** 已解析的结构化卡片。 */
  blocks?: AIFlowBlock[];
  /** BubbleList 稳定渲染键。 */
  key: string;
  /** 气泡左右位置。 */
  placement: "start" | "end";
  /** 气泡视觉样式。 */
  variant?: "filled" | "borderless" | "outlined" | "shadow";
  /** 气泡形状。 */
  shape?: "round" | "corner";
  /** 气泡最大宽度。 */
  maxWidth?: string;
  /** 本地消息进度状态。 */
  progressState?: MessageProgressState;
  /** 是否为前端临时消息，收到服务端最终消息后会被替换。 */
  localOnly?: boolean;
  /** 回复来源标签。 */
  replySourceTag?: ReplySourceTag;
  /** 本地流式消息键，按会话和单轮消息拆分。 */
  streamKey?: string;
  /** 是否正在朗读当前消息。 */
  speaking?: boolean;
};

/** AI 助手 direct stream SSE 事件名称。 */
export type AiStreamEventName = "delta" | "finish" | "error";

/** AI 助手 direct stream 事件负载。 */
export type AiStreamPayload = {
  /** 会话 ID。 */
  session_id: string;
  /** 后端单轮消息 ID，用于关联当前轮次。 */
  message_id: string;
  /** 本次新增文本分片。 */
  delta?: string;
  /** 流式完成后的最终消息列表。 */
  messages?: AiMessage[];
  /** 流式完成后的最新会话。 */
  session?: AiSession;
};

/** AI 助手 direct stream 标准化事件。 */
export type AiStreamEvent = {
  /** SSE 事件名称。 */
  event: AiStreamEventName;
  /** 已解析的 JSON 负载。 */
  payload: AiStreamPayload;
};
