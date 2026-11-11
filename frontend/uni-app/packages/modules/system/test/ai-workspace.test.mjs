import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

async function loadRules() {
  const source = await readFile(
    new URL('../src/views/pagesMember/ai/workspace.ts', import.meta.url),
    'utf8',
  )
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext },
  })
  return import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
}

test('插件选用保留具体账号并拒绝失效或无有效工具的连接', async () => {
  const { pluginSelectionIssue } = await loadRules()
  const plugin = {
    enabled: true,
    source: 'MCP',
    installation: { enabled: true, terminals: ['app'] },
  }
  const connection = { id: '2', state: 'connected', expires_at: 0 }
  const tools = [{ connection_id: '2', effective_enabled: true }]
  assert.equal(pluginSelectionIssue(plugin, tools, connection), '')
  assert.equal(
    pluginSelectionIssue(plugin, tools, { ...connection, state: 'disconnected' }),
    'connection_unavailable',
  )
  assert.equal(pluginSelectionIssue(plugin, tools, { ...connection, id: '3' }), 'no_tools')
  assert.equal(
    pluginSelectionIssue({ ...plugin, unavailable_reason: '管理员停用' }, tools, connection),
    '管理员停用',
  )
  assert.equal(
    pluginSelectionIssue({ ...plugin, source: 'LOCAL', assembled: false }, tools),
    'not_assembled',
  )
})

test('停止本次只适用于活动记录，结果未知不能被当作已取消', async () => {
  const { canStopAutomationRun } = await loadRules()
  for (const status of ['queued', 'running', 'waiting_approval'])
    assert.equal(canStopAutomationRun({ status }), true)
  for (const status of ['unknown', 'canceled', 'succeeded', 'failed', 'expired', 'skipped'])
    assert.equal(canStopAutomationRun({ status }), false)
  assert.equal(canStopAutomationRun({ status: 'running', cancel_requested: true }), false)
})

test('跨端继续对话时插件选择沿用会话的工具能力配置', async () => {
  const { pluginSelectionIssue } = await loadRules()
  const plugin = {
    enabled: true,
    source: 'LOCAL',
    assembled: true,
    installation: { enabled: true, terminals: ['admin'] },
  }
  const tools = [{ connection_id: '', effective_enabled: true }]
  assert.equal(pluginSelectionIssue(plugin, tools, undefined, 'admin'), '')
  assert.equal(pluginSelectionIssue(plugin, tools, undefined, 'app'), 'terminal_unavailable')
})

test('提醒链接只提取正整数资源编号，不接受角色、来源或只读标记', async () => {
  const source = await readFile(
    new URL('../src/views/pagesMember/ai/resultLink.ts', import.meta.url),
    'utf8',
  )
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext },
  })
  const { parseAiResultParams } = await import(
    `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
  )
  assert.deepEqual(
    parseAiResultParams({ run_id: '1', session_id: '2', read_only: false, terminal: 'admin' }),
    { run_id: '1', session_id: '2' },
  )
  assert.equal(parseAiResultParams({ run_id: '../3', session_id: '-2' }), undefined)
  assert.equal(parseAiResultParams({ run_id: 3 }), undefined)
})
