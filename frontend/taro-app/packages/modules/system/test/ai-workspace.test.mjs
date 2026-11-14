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

test('只读工作区时间兼容真实JSON时间、空值和计划时区', async () => {
  const { workspaceTimestamp, formatWorkspaceTime, formatWorkspaceSchedule } = await loadRules()
  assert.equal(workspaceTimestamp(undefined), 0)
  assert.equal(workspaceTimestamp({ seconds: 2, nanos: 500000000 }), 2500)
  assert.equal(workspaceTimestamp('2026-11-13T01:00:00Z'), Date.parse('2026-11-13T01:00:00Z'))
  assert.match(formatWorkspaceTime('2026-11-13T01:00:00Z', 'zh-CN', 'Asia/Shanghai'), /09:00/)
  assert.equal(formatWorkspaceTime(undefined, 'zh-CN'), '—')
  const messages = {
    'system.ai.workspace.weekly': '每周',
    'system.ai.workspace.weekday_1': '一',
    'system.ai.workspace.weekday_5': '五',
  }
  assert.equal(
    formatWorkspaceSchedule(
      { frequency: 'weekly', weekdays: [1, 5], time: '09:00', timezone: 'Asia/Shanghai' },
      (key) => messages[key] || key,
    ),
    '每周一、五 09:00',
  )
})

test('插件查看按当前终端和服务端实际可用状态显示原因', async () => {
  const { workspacePluginIssue } = await loadRules()
  const plugin = {
    enabled: true,
    assembled: true,
    installation: { enabled: true, terminals: ['app'] },
  }
  assert.equal(workspacePluginIssue(plugin, [{ effective_enabled: true }]), '')
  assert.equal(workspacePluginIssue(plugin, []), 'no_tools')
  assert.equal(
    workspacePluginIssue({ ...plugin, installation: { enabled: true, terminals: ['admin'] } }, [
      { effective_enabled: true },
    ]),
    'terminal_unavailable',
  )
  assert.equal(
    workspacePluginIssue({ ...plugin, unavailable_reason: '授权已撤销' }, [
      { effective_enabled: true },
    ]),
    '授权已撤销',
  )
})

test('任务待处理状态覆盖等待确认和依赖错误，归档状态保持明确', async () => {
  const { workspaceAutomationStatus } = await loadRules()
  assert.equal(workspaceAutomationStatus({ status: 'active' }), 'active')
  assert.equal(
    workspaceAutomationStatus({ status: 'active', last_run: { status: 'waiting_approval' } }),
    'attention',
  )
  assert.equal(workspaceAutomationStatus({ status: 'paused', problem: '依赖不可用' }), 'attention')
  assert.equal(workspaceAutomationStatus({ status: 'archived', problem: '旧记录' }), 'archived')
})
