import type { ListAiMessageEventResponse } from "../../../rpc/base/v1/ai_message";
import type { AiStreamEvent, AiStreamEventName, AiStreamPayload } from "./types";

const STREAM_EVENT_NAMES = new Set<AiStreamEventName>(["delta", "activity", "finish", "approval_required", "error"]);

/** 读取到的原始 SSE 字段结构。 */
type SseOutput = Partial<Record<"data" | "event" | "id" | "retry", unknown>>;

/** AI 助手事件流消费回调。 */
type AiStreamEventHandler = (event: AiStreamEvent) => void;

/** 判断 SSE 事件名称是否为 AI 助手 direct stream 支持的事件。 */
function isAiStreamEventName(event?: unknown): event is AiStreamEventName {
  return STREAM_EVENT_NAMES.has(String(event ?? "").trim() as AiStreamEventName);
}

/** 解析 SSE data 字段，兼容前导空格和空消息。 */
function parseStreamPayload(data?: unknown): AiStreamPayload | null {
  const rawData = String(data ?? "").trimStart();
  if (!rawData) return null;

  try {
    return JSON.parse(rawData) as AiStreamPayload;
  } catch {
    return null;
  }
}

/** 将原始 SSE 项收敛为业务事件，避免页面直接处理字符串 JSON。 */
export function normalizeAiStreamItem(item?: SseOutput): AiStreamEvent | null {
  if (!item || !isAiStreamEventName(item.event)) return null;

  const payload = parseStreamPayload(item.data);
  if (!payload?.session_id || !payload.message_id) return null;
  if (item.id && !payload.event_id) payload.event_id = String(item.id);

  return {
    event: String(item.event).trim() as AiStreamEventName,
    payload
  };
}

/** 读取并解析 AI 助手 direct stream，支持同一页面同时消费多条会话流。 */
export async function readAiEventStream(
  readableStream: ReadableStream<Uint8Array>,
  handler: AiStreamEventHandler,
  signal?: AbortSignal
) {
  handler = createAiEventConsumer(handler);
  const reader = readableStream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let currentItem: SseOutput = {};

  /** 分发当前累计的 SSE 事件。 */
  function dispatchCurrentItem() {
    const event = normalizeAiStreamItem(currentItem);
    currentItem = {};
    if (event) handler(event);
  }

  /** 按 SSE 行协议累积字段，空行代表一条事件结束。 */
  function handleLine(line: string) {
    if (line === "") {
      dispatchCurrentItem();
      return;
    }
    if (line.startsWith(":")) return;

    const separatorIndex = line.indexOf(":");
    const field = separatorIndex >= 0 ? line.slice(0, separatorIndex) : line;
    let value = separatorIndex >= 0 ? line.slice(separatorIndex + 1) : "";
    if (value.startsWith(" ")) value = value.slice(1);

    if (field === "data") {
      currentItem.data = currentItem.data === undefined ? value : `${currentItem.data}\n${value}`;
      return;
    }
    if (field === "event" || field === "id" || field === "retry") {
      currentItem[field] = value;
    }
  }

  /** 消费缓冲区里的完整行，保留最后一个未结束的半行。 */
  function consumeBuffer(flush = false) {
    let lineBreakIndex = buffer.indexOf("\n");
    while (lineBreakIndex >= 0) {
      const line = buffer.slice(0, lineBreakIndex).replace(/\r$/, "");
      buffer = buffer.slice(lineBreakIndex + 1);
      handleLine(line);
      lineBreakIndex = buffer.indexOf("\n");
    }
    if (flush && buffer) {
      handleLine(buffer.replace(/\r$/, ""));
      buffer = "";
    }
  }

  const abortReader = () => {
    void reader.cancel();
  };
  signal?.addEventListener("abort", abortReader, { once: true });
  try {
    while (true) {
      if (signal?.aborted) break;
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      consumeBuffer();
    }
    buffer += decoder.decode();
    consumeBuffer(true);
    dispatchCurrentItem();
  } finally {
    signal?.removeEventListener("abort", abortReader);
    reader.releaseLock();
  }
}

/** 按十进制事件ID去重乱序通知，避免超过JS安全整数后错误合并；文本分片不参与补读。 */
export function createAiEventConsumer(handler: AiStreamEventHandler): AiStreamEventHandler {
  const cursors = new Map<string, { id: string; run: string }>();
  return event => {
    const payload = event.payload;
    const key = `${payload.session_id}:${payload.message_id}`;
    const previous = cursors.get(key);
    const id = payload.event_id;
    if (id) {
      if (!/^[1-9][0-9]*$/.test(id)) return;
      if (previous && (id.length < previous.id.length || (id.length === previous.id.length && id <= previous.id))) return;
      cursors.set(key, { id, run: payload.run_id || "" });
    } else if (previous?.run && payload.run_id && previous.run !== payload.run_id) return;
    handler(event);
  };
}

/** 仅用只读查询补齐关键活动并等待权威终态，不发送提示词、不决定审批、不重放工具。 */
export async function readAiMessageEvents(
  read: (cursor: string) => Promise<ListAiMessageEventResponse>,
  handler: AiStreamEventHandler,
  cancelled: () => boolean
): Promise<void> {
  let cursor = "";
  const startedAt = Date.now();
  const consume = createAiEventConsumer(handler);
  while (!cancelled()) {
    if (Date.now() - startedAt > 10 * 60 * 1000)
      throw new Error("Execution status is still unknown; refresh the conversation before trying any operation again.");
    const page = await read(cursor);
    if (cancelled()) return;
    for (const item of page.events ?? []) {
      if (item.run_id !== page.run_id) continue;
      const event = normalizeAiStreamItem({ event: item.event, id: item.id, data: item.payload });
      if (event) {
        event.payload.run_id = item.run_id;
        event.payload.event_id = item.id;
        consume(event);
      }
    }
    cursor = page.next_cursor || cursor;
    if (page.has_more) continue;
    const current = page.messages[0];
    if (!current) throw new Error("AI message snapshot is missing");
    if (current.status !== 1) {
      handler({
        event: current.status === 4 ? "approval_required" : current.status === 3 ? "error" : "finish",
        payload: {
          session_id: page.session?.id || "",
          message_id: current.id,
          run_id: page.run_id,
          messages: page.messages,
          session: page.session
        }
      });
      return;
    }
    await new Promise<void>(resolve => setTimeout(resolve, 1000));
  }
}
