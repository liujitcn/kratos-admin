import { getRequestAccessToken, handleAuthExpired, http, requestBaseURL } from '@/utils/http'
import type { ListAiMessageRequest, ListAiMessageResponse } from '@/rpc/base/v1/ai_session'
import type {
  AiMessageService,
  DeleteAiMessageRequest,
  DeleteAiMessageResponse,
  RegenerateAiMessageRequest,
  RetryAiUserMessageRequest,
  SendAiMessageRequest,
  SendAiMessageResponse,
  UpdateAiMessageRequest,
} from '@/rpc/base/v1/ai_message'

const AI_SESSION_URL = '/v1/base/ai/session'

/** 微信小程序 chunked stream 分片处理选项。 */
export type AiMessageChunkedStreamOptions = {
  onChunk: (chunkText: string) => void
}

/** 微信小程序 chunked stream 请求任务。 */
export type AiMessageChunkedStreamTask = {
  promise: Promise<void>
  abort: () => void
}

/** direct stream 请求控制选项。 */
export type AiMessageStreamOptions = {
  /** 外部取消信号，用于页面卸载或会话删除时终止流式请求。 */
  signal?: AbortSignal
}

type ChunkReceivedResult = {
  data?: ArrayBuffer
}

type ChunkedRequestTask = UniNamespace.RequestTask & {
  onChunkReceived?: (listener: (result: ChunkReceivedResult) => void) => number
}

/** 从 direct stream 错误响应中提取后端业务提示。 */
async function resolveStreamErrorMessage(response: Response): Promise<string> {
  const fallbackMessage = `AI 助手请求失败（${response.status}）`
  const contentType = response.headers.get('Content-Type') ?? ''
  if (contentType.includes('application/json')) {
    try {
      const payload = await response.json()
      return String(payload?.message || payload?.error || fallbackMessage)
    } catch {
      return fallbackMessage
    }
  }

  try {
    const text = (await response.text()).trim()
    return text || fallbackMessage
  } catch {
    return fallbackMessage
  }
}

/** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供调用方消费。 */
export async function SendAiMessageStream(
  request: SendAiMessageRequest,
  options?: AiMessageStreamOptions,
): Promise<Response> {
  const accessToken = await getRequestAccessToken()
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Content-Type': 'application/json;charset=utf-8',
    'source-client': 'miniapp',
  }
  if (accessToken) {
    headers.Authorization = accessToken
  }

  const response = await fetch(`${requestBaseURL}${AI_SESSION_URL}/${request.session_id}/message`, {
    method: 'POST',
    headers,
    body: JSON.stringify(request),
    signal: options?.signal,
  })

  // direct stream 不经过 uni.request 拦截器，需要在这里补齐登录失效处理。
  if (response.status === 401 || response.status === 403) {
    handleAuthExpired('required')
    throw new Error('登录状态已失效，请重新登录')
  }
  if (!response.ok) {
    throw new Error(await resolveStreamErrorMessage(response))
  }

  return response
}

/** 使用微信小程序 chunked request 发送 AI 助手消息并增量返回 SSE 文本。 */
export function StreamAiMessageByChunkedRequest(
  request: SendAiMessageRequest,
  options: AiMessageChunkedStreamOptions,
): AiMessageChunkedStreamTask {
  let requestTask: ChunkedRequestTask | undefined
  let aborted = false
  let receivedChunk = false
  const decoder = createChunkTextDecoder()

  const promise = (async () => {
    const accessToken = await getRequestAccessToken()
    if (aborted) {
      return
    }

    await new Promise<void>((resolve, reject) => {
      requestTask = uni.request({
        url: `${requestBaseURL}${AI_SESSION_URL}/${request.session_id}/message`,
        method: 'POST',
        data: request,
        dataType: 'text',
        responseType: 'arraybuffer',
        enableChunked: true,
        timeout: 120000,
        header: {
          Accept: 'text/event-stream',
          'Content-Type': 'application/json;charset=utf-8',
          'source-client': 'miniapp',
          ...(accessToken ? { Authorization: accessToken } : {}),
        },
        success(res) {
          if (aborted) {
            resolve()
            return
          }
          if (res.statusCode === 401 || res.statusCode === 403) {
            handleAuthExpired('required')
            reject(new Error('登录状态已失效，请重新登录'))
            return
          }
          if (res.statusCode < 200 || res.statusCode >= 300) {
            reject(new Error(resolveChunkedStreamErrorMessage(res)))
            return
          }

          const tailText = decoder.flush()
          if (tailText) {
            options.onChunk(tailText)
          }
          if (!receivedChunk) {
            const fallbackText = decodeChunkedResponseData(res.data)
            if (fallbackText) {
              options.onChunk(fallbackText)
            }
          }
          resolve()
        },
        fail(error) {
          if (aborted) {
            resolve()
            return
          }
          reject(error)
        },
      }) as ChunkedRequestTask

      if (typeof requestTask.onChunkReceived !== 'function') {
        requestTask.abort()
        reject(new Error('当前端不支持流式回复'))
        return
      }

      requestTask.onChunkReceived((result) => {
        if (aborted || !result.data) {
          return
        }
        receivedChunk = true
        const chunkText = decoder.decode(result.data)
        if (chunkText) {
          options.onChunk(chunkText)
        }
      })
    })
  })()

  return {
    promise,
    abort() {
      aborted = true
      requestTask?.abort()
    },
  }
}

/** AI 助手消息服务。 */
export class AiMessageServiceImpl implements AiMessageService {
  /** 查询 AI 助手消息列表。 */
  ListAiMessage(request: ListAiMessageRequest): Promise<ListAiMessageResponse> {
    return http<ListAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }

  /** 发送 AI 助手消息并等待完整响应。 */
  SendAiMessage(request: SendAiMessageRequest): Promise<SendAiMessageResponse> {
    return http<SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 删除 AI 助手消息。 */
  DeleteAiMessage(request: DeleteAiMessageRequest): Promise<DeleteAiMessageResponse> {
    return http<DeleteAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}`,
      method: 'DELETE',
      authMode: 'required',
      data: request,
    })
  }

  /** 更新 AI 助手消息文本并重新生成输出。 */
  UpdateAiMessage(request: UpdateAiMessageRequest): Promise<SendAiMessageResponse> {
    return http<SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }

  /** 重试失败的 AI 助手消息。 */
  RetryAiUserMessage(request: RetryAiUserMessageRequest): Promise<SendAiMessageResponse> {
    return http<SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}/retry`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 重新生成 AI 助手输出。 */
  RegenerateAiMessage(request: RegenerateAiMessageRequest): Promise<SendAiMessageResponse> {
    return http<SendAiMessageResponse>({
      url: `${AI_SESSION_URL}/${request.session_id}/message/${request.message_id}/regeneration`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }

  /** 使用 direct stream 发送 AI 助手消息，并返回原始 Fetch Response 供调用方消费。 */
  StreamAiMessage(
    request: SendAiMessageRequest,
    options?: AiMessageStreamOptions,
  ): Promise<Response> {
    return SendAiMessageStream(request, options)
  }
}

export const defAiMessageService = new AiMessageServiceImpl()

function resolveChunkedStreamErrorMessage(response: UniApp.RequestSuccessCallbackResult) {
  const fallbackMessage = `AI 助手请求失败（${response.statusCode}）`
  const text = decodeChunkedResponseData(response.data).trim()
  if (!text) {
    return fallbackMessage
  }

  try {
    const payload = JSON.parse(text) as { message?: string; error?: string }
    return payload.message || payload.error || fallbackMessage
  } catch {
    return text || fallbackMessage
  }
}

function decodeChunkedResponseData(data: unknown) {
  if (typeof data === 'string') {
    return data
  }
  if (data instanceof ArrayBuffer) {
    return createChunkTextDecoder().decode(data)
  }
  if (data && typeof data === 'object') {
    return JSON.stringify(data)
  }
  return ''
}

function createChunkTextDecoder() {
  if (typeof TextDecoder !== 'undefined') {
    const decoder = new TextDecoder('utf-8')
    return {
      decode(data: ArrayBuffer) {
        return decoder.decode(data, { stream: true })
      },
      flush() {
        return decoder.decode()
      },
    }
  }

  let pendingBytes: number[] = []
  return {
    decode(data: ArrayBuffer) {
      const bytes = [...pendingBytes, ...Array.from(new Uint8Array(data))]
      const boundary = resolveUtf8Boundary(bytes)
      pendingBytes = bytes.slice(boundary)
      return decodeUtf8Bytes(bytes.slice(0, boundary))
    },
    flush() {
      const text = decodeUtf8Bytes(pendingBytes)
      pendingBytes = []
      return text
    },
  }
}

function resolveUtf8Boundary(bytes: number[]) {
  let index = bytes.length - 1
  while (index >= 0 && (bytes[index] & 0xc0) === 0x80) {
    index--
  }
  if (index < 0) {
    return 0
  }

  const length = resolveUtf8SequenceLength(bytes[index])
  if (length > 1 && bytes.length - index < length) {
    return index
  }
  return bytes.length
}

function resolveUtf8SequenceLength(lead: number) {
  if ((lead & 0x80) === 0) {
    return 1
  }
  if ((lead & 0xe0) === 0xc0) {
    return 2
  }
  if ((lead & 0xf0) === 0xe0) {
    return 3
  }
  if ((lead & 0xf8) === 0xf0) {
    return 4
  }
  return 1
}

function decodeUtf8Bytes(bytes: number[]) {
  if (!bytes.length) {
    return ''
  }

  let encoded = ''
  for (const byte of bytes) {
    encoded += `%${byte.toString(16).padStart(2, '0')}`
  }

  try {
    return decodeURIComponent(encoded)
  } catch {
    return String.fromCharCode(...bytes)
  }
}
