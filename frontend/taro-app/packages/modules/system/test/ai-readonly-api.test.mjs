import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

/** 使用真实服务适配器，替换传输边界以检查方法、身份模式和资源编号。 */
async function loadService(name) {
  const source = await readFile(new URL(`../src/api/base/v1/${name}.ts`, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ESNext },
  })
  const transport = `const http = async request => request; const t = key => key; const getCurrentLocale = () => 'zh-CN'; const getRequestAccessToken = async () => ''; const requestBaseURL = 'http://127.0.0.1/api'; const sourceClient = 'test'; const Taro = {};`
  return import(
    `data:text/javascript;base64,${Buffer.from(transport + outputText.replace(/^import[^\n]+\n/gm, '')).toString('base64')}`
  )
}

test('任务适配器只提供鉴权查询，并准确传递任务和执行编号', async () => {
  const { defAiAutomationService: api, getAiAutomationRun } = await loadService('ai_automation')
  assert.deepEqual(
    Object.getOwnPropertyNames(Object.getPrototypeOf(api))
      .filter((name) => name !== 'constructor')
      .sort(),
    ['GetAiAutomation', 'GetAiAutomationRun', 'ListAiAutomation', 'ListAiAutomationRun'].sort(),
  )
  for (const request of [
    await api.ListAiAutomation({ status: 'paused', keyword: '项目' }),
    await api.GetAiAutomation({ id: '11' }),
    await api.ListAiAutomationRun({ automation_id: '11', limit: 100 }),
    await api.GetAiAutomationRun({ id: '22' }),
    await getAiAutomationRun({ id: '22' }),
  ]) {
    assert.equal(request.method, 'GET')
    assert.equal(request.authMode, 'required')
  }
  assert.equal((await api.GetAiAutomation({ id: '11' })).url, '/v1/base/ai/automation/11')
  assert.equal((await api.GetAiAutomationRun({ id: '22' })).url, '/v1/base/ai/automation-run/22')
  assert.equal(
    (await api.ListAiAutomationRun({ automation_id: '11', limit: 100 })).data.automation_id,
    '11',
  )
})

test('文件管理只暴露分页查询，回收站仍使用只读视图', async () => {
  const { defAiFileService: api, isAiWorkspaceTextFile } = await loadService('ai_file')
  assert.deepEqual(Object.getOwnPropertyNames(Object.getPrototypeOf(api)), [
    'constructor',
    'PageAiFile',
  ])
  const request = await api.PageAiFile({ view: 4, keyword: '报告', page_num: 2, page_size: 20 })
  assert.equal(request.method, 'GET')
  assert.equal(request.authMode, 'required')
  assert.deepEqual(request.data, { view: 4, keyword: '报告', page_num: 2, page_size: 20 })
  assert.equal(
    isAiWorkspaceTextFile({ file_name: '说明.html', extension: 'html', mime_type: 'text/html' }),
    true,
  )
  assert.equal(
    isAiWorkspaceTextFile({ file_name: '图表.svg', extension: 'svg', mime_type: 'image/svg+xml' }),
    true,
  )
  assert.equal(
    isAiWorkspaceTextFile({
      file_name: '数据.JSON',
      extension: '.JSON',
      mime_type: 'application/json',
    }),
    true,
  )
  assert.equal(
    isAiWorkspaceTextFile({ file_name: '图表.svg', extension: '.svg', mime_type: 'image/svg+xml' }),
    true,
  )
  assert.equal(
    isAiWorkspaceTextFile({
      file_name: '报告.pdf',
      extension: 'pdf',
      mime_type: 'application/pdf',
    }),
    false,
  )
})

test('插件目录和账号仅使用查询，保留确定插件编号与应用终端', async () => {
  const { defAiPluginService: api } = await loadService('ai_plugin')
  assert.deepEqual(
    Object.getOwnPropertyNames(Object.getPrototypeOf(api))
      .filter((name) => name !== 'constructor')
      .sort(),
    ['ListAiPlugin', 'ListAiPluginConnection', 'ListAiPluginTool'].sort(),
  )
  const request = await api.ListAiPluginTool({ plugin_id: '33', terminal: 2 })
  assert.equal(request.url, '/v1/base/ai/plugin/33/tool')
  assert.equal(request.method, 'GET')
  assert.equal(request.authMode, 'required')
  assert.equal(request.data.terminal, 2)
  assert.equal((await api.ListAiPluginConnection({ plugin_id: '33' })).data.plugin_id, '33')
})
