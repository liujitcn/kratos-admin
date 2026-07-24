import type { AiMessage, AiSession } from '@/rpc/base/v1/ai_session'

/** AI 助手 direct stream SSE 事件名称。 */
export type AiStreamEventName = 'delta' | 'finish' | 'error'

/** AI 助手 direct stream 事件负载。 */
export type AiStreamPayload = {
  /** 会话 ID。 */
  session_id: string
  /** 后端单轮消息 ID，用于关联当前轮次。 */
  message_id: string
  /** 本次新增文本分片。 */
  delta?: string
  /** 流式完成后的最终消息列表。 */
  messages?: AiMessage[]
  /** 流式完成后的最新会话。 */
  session?: AiSession
}

/** AI 助手 direct stream 标准化事件。 */
export type AiStreamEvent = {
  /** SSE 事件名称。 */
  event: AiStreamEventName
  /** 已解析的 JSON 负载。 */
  payload: AiStreamPayload
}

/** 读取到的原始 SSE 字段结构。 */
type SseOutput = Partial<Record<'data' | 'event' | 'id' | 'retry', unknown>>

/** AI 助手事件流消费回调。 */
export type AiStreamEventHandler = (event: AiStreamEvent) => void

/** 增量 SSE 文本解析器。 */
export type AiEventStreamTextParser = {
  push: (value: unknown) => void
  flush: () => void
}

const STREAM_EVENT_NAMES = new Set<AiStreamEventName>(['delta', 'finish', 'error'])

function createSseTextParser(handler: AiStreamEventHandler) {
  let currentItem: SseOutput = {}

  const dispatchCurrentItem = () => {
    const event = normalizeAiStreamItem(currentItem)
    currentItem = {}
    if (event) {
      handler(event)
    }
  }

  const handleLine = (line: string) => {
    if (line === '') {
      dispatchCurrentItem()
      return
    }
    if (line.startsWith(':')) {
      return
    }

    const separatorIndex = line.indexOf(':')
    const field = separatorIndex >= 0 ? line.slice(0, separatorIndex) : line
    let value = separatorIndex >= 0 ? line.slice(separatorIndex + 1) : ''
    if (value.startsWith(' ')) {
      value = value.slice(1)
    }

    if (field === 'data') {
      currentItem.data = currentItem.data === undefined ? value : `${currentItem.data}\n${value}`
      return
    }
    if (field === 'event' || field === 'id' || field === 'retry') {
      currentItem[field] = value
    }
  }

  return { dispatchCurrentItem, handleLine }
}

/** 创建可增量消费的 AI 助手 SSE 文本解析器。 */
export function createAiEventStreamTextParser(
  handler: AiStreamEventHandler,
): AiEventStreamTextParser {
  const parser = createSseTextParser(handler)
  let buffer = ''

  const consumeBuffer = (flush = false) => {
    let lineBreakIndex = buffer.indexOf('\n')
    while (lineBreakIndex >= 0) {
      const line = buffer.slice(0, lineBreakIndex).replace(/\r$/, '')
      buffer = buffer.slice(lineBreakIndex + 1)
      parser.handleLine(line)
      lineBreakIndex = buffer.indexOf('\n')
    }
    if (flush && buffer) {
      parser.handleLine(buffer.replace(/\r$/, ''))
      buffer = ''
    }
  }

  return {
    push(value: unknown) {
      buffer += String(value ?? '')
      consumeBuffer()
    },
    flush() {
      consumeBuffer(true)
      parser.dispatchCurrentItem()
    },
  }
}

/** 判断 SSE 事件名称是否为 AI 助手 direct stream 支持的事件。 */
function isAiStreamEventName(event?: unknown): event is AiStreamEventName {
  return STREAM_EVENT_NAMES.has(String(event ?? '').trim() as AiStreamEventName)
}

/** 解析 SSE data 字段，兼容前导空格和空消息。 */
function parseStreamPayload(data?: unknown): AiStreamPayload | null {
  const rawData = String(data ?? '').trimStart()
  if (!rawData) {
    return null
  }

  try {
    return JSON.parse(rawData) as AiStreamPayload
  } catch {
    return null
  }
}

/** 将原始 SSE 项收敛为业务事件，避免页面直接处理字符串 JSON。 */
export function normalizeAiStreamItem(item?: SseOutput): AiStreamEvent | null {
  if (!item || !isAiStreamEventName(item.event)) {
    return null
  }

  const payload = parseStreamPayload(item.data)
  if (!payload?.session_id || !payload.message_id) {
    return null
  }

  return {
    event: String(item.event).trim() as AiStreamEventName,
    payload,
  }
}

/** 解析非流式客户端一次性拿到的 SSE 文本。 */
export function parseAiEventStreamText(value: unknown) {
  const events: AiStreamEvent[] = []
  const parser = createAiEventStreamTextParser((event) => events.push(event))
  parser.push(value)
  parser.flush()
  return events
}

/** 读取并解析 AI 助手 direct stream，支持同一页面同时消费多条会话流。 */
export async function readAiEventStream(
  readableStream: ReadableStream<Uint8Array>,
  handler: AiStreamEventHandler,
  signal?: AbortSignal,
) {
  const reader = readableStream.getReader()
  const decoder = new TextDecoder()
  const parser = createAiEventStreamTextParser(handler)

  const abortReader = () => {
    void reader.cancel()
  }
  signal?.addEventListener('abort', abortReader, { once: true })
  try {
    while (true) {
      if (signal?.aborted) {
        break
      }
      const { value, done } = await reader.read()
      if (done) {
        break
      }
      parser.push(decoder.decode(value, { stream: true }))
    }
    parser.push(decoder.decode())
    parser.flush()
  } finally {
    signal?.removeEventListener('abort', abortReader)
    reader.releaseLock()
  }
}
