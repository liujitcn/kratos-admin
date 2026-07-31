import type { AiMessage, AiSession } from '../../../rpc/base/v1/ai_session'

/** AI 助手流事件名称。 */
export type AiStreamEventName = 'delta' | 'finish' | 'error'

/** AI 助手流事件负载。 */
export type AiStreamPayload = {
  session_id: string
  message_id: string
  delta?: string
  messages?: AiMessage[]
  session?: AiSession
}

/** AI 助手标准化流事件。 */
export type AiStreamEvent = {
  event: AiStreamEventName
  payload: AiStreamPayload
}

type SseOutput = Partial<Record<'data' | 'event' | 'id' | 'retry', unknown>>

/** AI 助手流事件消费回调。 */
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
    if (event) handler(event)
  }

  const handleLine = (line: string) => {
    if (line === '') {
      dispatchCurrentItem()
      return
    }
    if (line.startsWith(':')) return

    const separatorIndex = line.indexOf(':')
    const field = separatorIndex >= 0 ? line.slice(0, separatorIndex) : line
    let value = separatorIndex >= 0 ? line.slice(separatorIndex + 1) : ''
    if (value.startsWith(' ')) value = value.slice(1)

    if (field === 'data') {
      currentItem.data = currentItem.data === undefined ? value : `${currentItem.data}\n${value}`
      return
    }
    if (field === 'event' || field === 'id' || field === 'retry') currentItem[field] = value
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
    push(value) {
      buffer += String(value ?? '')
      consumeBuffer()
    },
    flush() {
      consumeBuffer(true)
      parser.dispatchCurrentItem()
    },
  }
}

function isAiStreamEventName(event?: unknown): event is AiStreamEventName {
  return STREAM_EVENT_NAMES.has(String(event ?? '').trim() as AiStreamEventName)
}

function parseStreamPayload(data?: unknown): AiStreamPayload | null {
  const rawData = String(data ?? '').trimStart()
  if (!rawData) return null
  try {
    return JSON.parse(rawData) as AiStreamPayload
  } catch {
    return null
  }
}

/** 将原始 SSE 项收敛为业务事件。 */
export function normalizeAiStreamItem(item?: SseOutput): AiStreamEvent | null {
  if (!item || !isAiStreamEventName(item.event)) return null
  const payload = parseStreamPayload(item.data)
  if (!payload?.session_id || !payload.message_id) return null
  return { event: String(item.event).trim() as AiStreamEventName, payload }
}

/** 解析一次性取得的 SSE 文本。 */
export function parseAiEventStreamText(value: unknown) {
  const events: AiStreamEvent[] = []
  const parser = createAiEventStreamTextParser((event) => events.push(event))
  parser.push(value)
  parser.flush()
  return events
}

/** 读取并解析 AI 助手流式响应。 */
export async function readAiEventStream(
  readableStream: ReadableStream<Uint8Array>,
  handler: AiStreamEventHandler,
  signal?: AbortSignal,
) {
  const reader = readableStream.getReader()
  const decoder = new TextDecoder()
  const parser = createAiEventStreamTextParser(handler)
  const abortReader = () => void reader.cancel()
  signal?.addEventListener('abort', abortReader, { once: true })
  try {
    while (!signal?.aborted) {
      const { value, done } = await reader.read()
      if (done) break
      parser.push(decoder.decode(value, { stream: true }))
    }
    parser.push(decoder.decode())
    parser.flush()
  } finally {
    signal?.removeEventListener('abort', abortReader)
    reader.releaseLock()
  }
}
