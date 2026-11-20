import type { ListAiMessageEventResponse } from "../../../rpc/base/v1/ai_message";
import type { AiMessage, AiSession } from '../../../rpc/base/v1/ai_session'
import type { AiToolCall } from '../../../rpc/base/v1/ai_tool'

/** AI 助手流事件名称。 */
export type AiStreamEventName = 'delta' | 'activity' | 'finish' | 'approval_required' | 'error'

/** AI 助手运行活动。 */
export type AiStreamActivity = {
  id: string
  phase: string
  status: string
  summary?: string
  tool_name?: string
  tool_title?: string
}

/** 流式工具记录附带本轮调用 ID，最终消息仍使用后端返回的工具列表。 */
export type AiStreamTool = AiToolCall & { activityId?: string }

/** 按调用 ID 更新工具活动，推理和回答等阶段不计入工具列表。 */
export function mergeAiToolActivity(tools: AiStreamTool[], activity: AiStreamActivity): AiStreamTool[] {
  if (activity.phase !== 'tool') return tools
  const index = tools.findIndex((tool) => tool.activityId === activity.id)
  const previous = tools[index]
  const tool: AiStreamTool = {
    activityId: activity.id,
    type: 'activity',
    name: activity.tool_name || previous?.name || activity.id,
    title: activity.tool_title || previous?.title || activity.summary || activity.phase,
    status: activity.status,
    input: '',
    output: activity.summary || previous?.output || '',
  }
  if (index === -1) return [...tools, tool]
  const next = tools.slice()
  next[index] = tool
  return next
}

/** AI 助手流事件负载。 */
export type AiStreamPayload = {
  run_id?: string
  event_id?: string
  session_id: string
  message_id: string
  delta?: string
  /** 本次运行活动更新。 */
  activity?: AiStreamActivity
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

const STREAM_EVENT_NAMES = new Set<AiStreamEventName>([
  'delta',
  'activity',
  'finish',
  'approval_required',
  'error',
])

function createSseTextParser(handler: AiStreamEventHandler) {
  handler = createAiEventConsumer(handler)
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
  if (item.id && !payload.event_id) payload.event_id = String(item.id)
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


/** 按十进制事件ID去重乱序通知，避免超过JS安全整数后错误合并；文本分片不参与补读。 */
export function createAiEventConsumer(handler: AiStreamEventHandler): AiStreamEventHandler {
  const cursors = new Map<string, { id: string; run: string }>()
  return (event) => {
    const payload = event.payload
    const key = `${payload.session_id}:${payload.message_id}`
    const previous = cursors.get(key)
    const id = payload.event_id
    if (id) {
      if (!/^[1-9][0-9]*$/.test(id)) return
      if (previous && (id.length < previous.id.length || (id.length === previous.id.length && id <= previous.id))) return
      cursors.set(key, { id, run: payload.run_id || '' })
    } else if (previous?.run && payload.run_id && previous.run !== payload.run_id) return
    handler(event)
  }
}

/** 仅用只读查询补齐关键活动并等待权威终态，不发送提示词、不决定审批、不重放工具。 */
export async function readAiMessageEvents(
  read: (cursor: string) => Promise<ListAiMessageEventResponse>,
  handler: AiStreamEventHandler,
  cancelled: () => boolean,
): Promise<void> {
  let cursor = ''
  const startedAt = Date.now()
  const consume = createAiEventConsumer(handler)
  while (!cancelled()) {
    if (Date.now()-startedAt > 10*60*1000) throw new Error("Execution status is still unknown; refresh the conversation before trying any operation again.")
    const page = await read(cursor)
    if (cancelled()) return
    for (const item of page.events ?? []) {
      if (item.run_id !== page.run_id) continue
      const event = normalizeAiStreamItem({ event: item.event, id: item.id, data: item.payload })
      if (event) {
        event.payload.run_id = item.run_id
        event.payload.event_id = item.id
        consume(event)
      }
    }
    cursor = page.next_cursor || cursor
    if (page.has_more) continue
    const current = page.messages[0]
    if (!current) throw new Error('AI message snapshot is missing')
    if (current.status !== 1) {
      handler({ event: current.status === 4 ? 'approval_required' : current.status === 3 ? 'error' : 'finish', payload: {
        session_id: page.session?.id || '', message_id: current.id, run_id: page.run_id,
        messages: page.messages, session: page.session,
      } })
      return
    }
    await new Promise<void>((resolve) => setTimeout(resolve, 1000))
  }
}
