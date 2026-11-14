import type { Timestamp } from '../../../rpc/google/protobuf/timestamp'
import type { AiAutomation, AiAutomationSchedule } from '../../../rpc/base/v1/ai_automation'
import type { AiKnowledgeOption } from '../../../rpc/base/v1/ai_knowledge'
import type { AiPlugin, AiPluginTool } from '../../../rpc/base/v1/ai_plugin'

/** AI 助手内部工作区，不切换或销毁当前对话。 */
export type AiWorkspaceView = 'chat' | 'automations' | 'files' | 'plugins'
/** 只读工作区向导航栏暴露的返回能力。 */
export interface AiWorkspaceHandle {
  goBack: () => boolean
}
/** 真实 RPC 时间在 JSON 中可为 RFC3339 或 protobuf Timestamp。 */
export type AiWorkspaceTime = Timestamp | string | undefined

/** 将服务端时间转换为可排序毫秒，不把空值当成当前时间。 */
export function workspaceTimestamp(value: AiWorkspaceTime): number {
  return typeof value === 'string'
    ? Date.parse(value) || 0
    : Number(value?.seconds || 0) * 1000 + Number(value?.nanos || 0) / 1000000
}

/** 按计划时区展示时间；不支持该时区的平台明确回退 UTC。 */
export function formatWorkspaceTime(
  value: AiWorkspaceTime,
  locale: string,
  timezone?: string,
): string {
  const timestamp = workspaceTimestamp(value)
  if (!timestamp) return '—'
  try {
    return new Intl.DateTimeFormat(locale, {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
      ...(timezone ? { timeZone: timezone } : {}),
    }).format(timestamp)
  } catch {
    return new Date(timestamp).toISOString().replace('T', ' ').slice(0, 16) + ' UTC'
  }
}

/** 展示计划频率与本地时间，不在客户端重新计算服务端调度。 */
export function formatWorkspaceSchedule(
  value: AiAutomationSchedule | undefined,
  translate: (key: string) => string,
): string {
  if (!value) return '—'
  const days =
    value.frequency === 'weekly'
      ? (value.weekdays || [])
          .map((day) => translate(`system.ai.workspace.weekday_${day}`))
          .join('、')
      : ''
  return `${translate(`system.ai.workspace.${value.frequency || 'once'}`)}${days} ${value.frequency === 'once' ? (value.once_at || '').replace('T', ' ') : value.time || ''}`
}

/** 使用可读文件大小单位。 */
export function formatWorkspaceFileSize(bytes: number): string {
  const value = Number(bytes || 0)
  return value < 1024
    ? `${value} B`
    : value < 1048576
      ? `${(value / 1024).toFixed(1)} KB`
      : `${(value / 1048576).toFixed(1)} MB`
}

/** 插件详情使用服务端审核状态，不把管理权限当成可调用权限。 */
export function workspacePluginIssue(plugin: AiPlugin, tools: AiPluginTool[]): string {
  if (plugin.unavailable_reason) return plugin.unavailable_reason
  if (!plugin.enabled || !plugin.installation?.enabled) return 'disabled'
  if (!plugin.installation.terminals.includes('app')) return 'terminal_unavailable'
  if (!plugin.assembled) return 'not_assembled'
  if (!tools.some((tool) => tool.effective_enabled)) return 'no_tools'
  return ''
}

/** 待确认执行和依赖问题统一显示为需处理，归档任务保留归档状态。 */
export function workspaceAutomationStatus(task: AiAutomation): string {
  return task.status !== 'archived' &&
    (task.problem || task.last_run?.status === 'waiting_approval')
    ? 'attention'
    : task.status
}

/** 知识库按当前可见名称展示，已不可见引用仍保留其编号。 */
export function formatWorkspaceKnowledge(
  ids: number[] | undefined,
  options: AiKnowledgeOption[],
): string {
  return (
    (ids || [])
      .map((id) => options.find((item) => item.id === id)?.name || String(id))
      .join('、') || '—'
  )
}
