import type { AiPlugin, AiPluginConnection, AiPluginTool } from '../../../rpc/base/v1/ai_plugin'
import type { AiAutomationRun } from '../../../rpc/base/v1/ai_automation'

/** 按会话工具能力显示插件及指定账号不可用的原因，不自动替换账号。 */
export function pluginSelectionIssue(
  plugin: AiPlugin,
  tools: AiPluginTool[],
  connection?: AiPluginConnection,
  terminal: 'admin' | 'app' = 'app',
): string {
  if (plugin.unavailable_reason) return plugin.unavailable_reason
  if (plugin.source !== 'MCP' && !plugin.assembled) return 'not_assembled'
  if (!plugin.enabled || !plugin.installation?.enabled) return 'disabled'
  if (!plugin.installation.terminals.includes(terminal)) return 'terminal_unavailable'
  if (
    plugin.source === 'MCP' &&
    (!connection ||
      connection.state !== 'connected' ||
      (connection.expires_at > 0 && connection.expires_at * 1000 <= Date.now()))
  )
    return connection?.last_error || 'connection_unavailable'
  if (
    !tools.some((tool) => tool.connection_id === (connection?.id || '') && tool.effective_enabled)
  )
    return 'no_tools'
  return ''
}
/** 终态和已请求停止的记录不重复提交停止操作。 */
export function canStopAutomationRun(
  run: Pick<AiAutomationRun, 'status' | 'cancel_requested'>,
): boolean {
  return !run.cancel_requested && ['queued', 'running', 'waiting_approval'].includes(run.status)
}
