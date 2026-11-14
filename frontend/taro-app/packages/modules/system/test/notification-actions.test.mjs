import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import ts from 'typescript'

/** loadActions 加载页面实际操作方法，隔离弹窗和接口以验证确认前后的副作用。 */
async function loadActions() {
  const source = await readFile(new URL('../src/views/pagesMember/message/index.tsx', import.meta.url), 'utf8')
  const file = ts.createSourceFile('message.tsx', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
  const methods = []
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && ['toggleArchive', 'deleteItem'].includes(node.name?.text)) methods.push(node.getText(file))
    ts.forEachChild(node, visit)
  }
  visit(file)
  assert.equal(methods.length, 2)
  const { outputText } = ts.transpileModule(methods.join('\n'), { compilerOptions: { target: ts.ScriptTarget.ES2022 } })
  const calls = []
  const dialogs = []
  let resolveModal
  const actions = runInNewContext(`${outputText}\n({ toggleArchive, deleteItem })`, {
    Taro: { showModal: (options) => {
      dialogs.push(options)
      return new Promise((resolve) => { resolveModal = resolve })
    } },
    t: (key) => key,
    themeStore: { colors: {} },
    defNotificationService: Object.fromEntries(['ArchiveNotification', 'RestoreNotification', 'DeleteNotification'].map(name => [name, async ({ id }) => { calls.push(`${name}:${id}`) }])),
    refresh: async () => { calls.push('refresh') },
    refreshNotificationSummary: async () => { calls.push('summary'); return 0 },
    setUnreadTotal: () => {},
  })
  return { ...actions, calls, dialogs, respond: (confirm) => resolveModal({ confirm }) }
}

for (const [method, api] of [['toggleArchive', 'ArchiveNotification'], ['deleteItem', 'DeleteNotification']]) {
  for (const confirm of [false, true]) {
    test(`${method} 确认前不请求接口，${confirm ? '确认后执行' : '取消后不执行'}`, async () => {
      const runtime = await loadActions()
      const pending = runtime[method]({ id: 7, allow_archive: true, allow_delete: true })
      assert.deepEqual(runtime.calls, [])
      assert.equal(runtime.dialogs.length, 1)
      runtime.respond(confirm)
      await pending
      assert.deepEqual(runtime.calls, confirm ? [`${api}:7`, 'refresh', 'summary'] : [])
    })
  }
  test(`${method} 无操作权限时不弹窗或请求接口`, async () => {
    const runtime = await loadActions()
    await runtime[method]({ id: 7, allow_archive: false, allow_delete: false })
    assert.deepEqual(runtime.calls, [])
    assert.equal(runtime.dialogs.length, 0)
  })
}

test('恢复消息沿用现有流程', async () => {
  const runtime = await loadActions()
  await runtime.toggleArchive({ id: 7, archived_at: '2026-11-13 10:00:00' })
  assert.deepEqual(runtime.calls, ['RestoreNotification:7', 'refresh', 'summary'])
  assert.equal(runtime.dialogs.length, 0)
})
