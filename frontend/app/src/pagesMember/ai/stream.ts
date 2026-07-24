import type { AiMessage, AiSession } from '@/rpc/base/v1/ai_session'

/** AI 助手 direct stream SSE 事件名称。 */
export type AiStreamEventName = 'delta' | 'finish' | 'error'

/** AI 助手 direct stream 事件负载。 */
export type AiStreamPayload = {
  /** 会话 ID。 */
  session_id: string
  /** 本轮消息 ID。 */
  message_id: string
  /** 增量文本。 */
  delta?: string
  /** 流式完成后的消息列表。 */
  messages?: AiMessage[]
  /** 流式完成后的会话信息。 */
  session?: AiSession
}

/** AI 助手标准化流式事件。 */
export type AiStreamEvent = {
  /** SSE 事件名称。 */
  event: AiStreamEventName
  /** 已解析的事件负载。 */
  payload: AiStreamPayload
}

type SseOutput = Partial<Record<'data' | 'event' | 'id' | 'retry', unknown>>

/** AI 助手流式事件消费回调。 */
export type AiStreamEventHandler = (event: AiStreamEvent) => void

/** 增量 SSE 文本解析器。 */
export type AiEventStreamTextParser = {
  push: (value: unknown) => void
  flush: () => void
}

const STREAM_EVENT_NAMES = new Set<AiStreamEventName>(['delta', 'finish', 'error'])

function normalizeEventName(value: unknown): AiStreamEventName | undefined {
  const event = String(value ?? '').trim() as AiStreamEventName
  return STREAM_EVENT_NAMES.has(event) ? event : undefined
}

function normalizePayload(data: unknown): AiStreamPayload | null {
  const raw = String(data ?? '').trimStart()
  if (!raw) {
    return null
  }
  try {
    const payload = JSON.parse(raw) as AiStreamPayload
    return payload.session_id && payload.message_id ? payload : null
  } catch {
    return null
  }
}

function normalizeEvent(item: SseOutput): AiStreamEvent | null {
  const event = normalizeEventName(item.event)
  const payload = event ? normalizePayload(item.data) : null
  return event && payload ? { event, payload } : null
}

function createParser(handler: AiStreamEventHandler) {
  let current: SseOutput = {}

  const dispatch = () => {
    const event = normalizeEvent(current)
    current = {}
    if (event) {
      handler(event)
    }
  }

  const handleLine = (line: string) => {
    if (line === '') {
      dispatch()
      return
    }
    if (line.startsWith(':')) {
      return
    }

    const separator = line.indexOf(':')
    const field = separator >= 0 ? line.slice(0, separator) : line
    let value = separator >= 0 ? line.slice(separator + 1) : ''
    if (value.startsWith(' ')) {
      value = value.slice(1)
    }

    if (field === 'data') {
      current.data = current.data === undefined ? value : `${current.data}\n${value}`
      return
    }
    if (field === 'event' || field === 'id' || field === 'retry') {
      current[field] = value
    }
  }

  return { dispatch, handleLine }
}

/** 创建可增量消费的 AI 助手 SSE 文本解析器。 */
export function createAiEventStreamTextParser(
  handler: AiStreamEventHandler,
): AiEventStreamTextParser {
  const parser = createParser(handler)
  let buffer = ''

  const consume = (flush = false) => {
    let lineBreak = buffer.indexOf('\n')
    while (lineBreak >= 0) {
      parser.handleLine(buffer.slice(0, lineBreak).replace(/\r$/, ''))
      buffer = buffer.slice(lineBreak + 1)
      lineBreak = buffer.indexOf('\n')
    }
    if (flush && buffer) {
      parser.handleLine(buffer.replace(/\r$/, ''))
      buffer = ''
    }
  }

  return {
    push(value: unknown) {
      buffer += String(value ?? '')
      consume()
    },
    flush() {
      consume(true)
      parser.dispatch()
    },
  }
}

/** 读取并解析 AI 助手 direct stream。 */
export async function readAiEventStream(
  stream: ReadableStream<Uint8Array>,
  handler: AiStreamEventHandler,
  signal?: AbortSignal,
) {
  const reader = stream.getReader()
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

/** 解析一次性拿到的 SSE 文本。 */
export function parseAiEventStreamText(value: unknown) {
  const events: AiStreamEvent[] = []
  const parser = createAiEventStreamTextParser((event) => events.push(event))
  parser.push(value)
  parser.flush()
  return events
}
