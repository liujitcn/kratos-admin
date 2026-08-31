import { ref } from 'vue'
import { setAppMenuBadge } from '@liujitcn/kratos-uni-app-core/navigation'
import { getRequestAccessToken } from '@liujitcn/kratos-uni-app-core/utils/http'
import { defNotificationService } from './api/base/notification'

/** System 模块共享的站内信未读数。 */
export const notificationUnreadTotal = ref(0)

let notificationTimer: ReturnType<typeof setInterval> | undefined
let stopNotificationSse: (() => void) | undefined
let notificationPaused = false

/** 从服务端刷新未读汇总并同步消息入口 badge。 */
export async function refreshNotificationSummary(): Promise<void> {
  try {
    const summary = await defNotificationService.GetNotificationSummary({})
    notificationUnreadTotal.value = summary.unread_total
    setAppMenuBadge('MESSAGE_INBOX', summary.unread_total)
  } catch {
    // 登录失效或网络不可用时保留上次 badge，后续生命周期事件会再次回源。
  }
}

/** 启动应用端站内信定时回源。 */
export function startNotificationPolling(): void {
  stopNotificationPolling()
  notificationPaused = false
  void refreshNotificationSummary()
  notificationTimer = setInterval(() => void refreshNotificationSummary(), 30_000)
  if (typeof window !== 'undefined' && typeof fetch !== 'undefined')
    stopNotificationSse = startNotificationSse()
}

/** 停止应用端站内信定时回源并清理 badge。 */
export function stopNotificationPolling(): void {
  clearNotificationResources()
  notificationPaused = false
  notificationUnreadTotal.value = 0
  setAppMenuBadge('MESSAGE_INBOX', 0)
}

/** 暂停后台通知资源，保留当前 badge。 */
export function pauseNotificationPolling(): void {
  notificationPaused = true
  clearNotificationResources()
}

/** 恢复前台通知资源并立即对账。 */
export function resumeNotificationPolling(): void {
  if (!notificationPaused) return
  notificationPaused = false
  startNotificationPolling()
}

/** 清理定时器与 SSE 连接。 */
function clearNotificationResources(): void {
  if (notificationTimer) clearInterval(notificationTimer)
  notificationTimer = undefined
  stopNotificationSse?.()
  stopNotificationSse = undefined
}

/** 在 H5 建立用户级通知 SSE，断线时由定时回源兜底。 */
function startNotificationSse(): () => void {
  const controller = new AbortController()
  void (async () => {
    let retryDelay = 1_000
    while (!controller.signal.aborted) {
      try {
        const token = await getRequestAccessToken()
        if (!token || controller.signal.aborted) return
        const response = await fetch(`${window.location.origin}/events?stream=base.notification`, {
          headers: { Accept: 'text/event-stream', Authorization: token },
          signal: controller.signal,
        })
        if (response.status === 401 || response.status === 403) return
        if (!response.ok || !response.body) throw new Error('SSE connection failed')
        retryDelay = 1_000
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (!controller.signal.aborted) {
          const chunk = await reader.read()
          if (chunk.done) break
          buffer += decoder.decode(chunk.value, { stream: true })
          const events = buffer.split('\n\n')
          buffer = events.pop() ?? ''
          for (const event of events) {
            if (event.includes('event: inbox.changed')) void refreshNotificationSummary()
          }
        }
      } catch {
        if (controller.signal.aborted) return
      }
      if (!controller.signal.aborted)
        await new Promise((resolve) => setTimeout(resolve, retryDelay))
      retryDelay = Math.min(retryDelay * 2, 30_000)
    }
  })()
  return () => controller.abort()
}
