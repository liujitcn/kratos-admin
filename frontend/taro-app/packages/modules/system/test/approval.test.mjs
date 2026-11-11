import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('共享对话展示冻结参数，只有有效待确认操作提供批准和拒绝按钮', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-approval-'))
  const output = resolve(root, 'approval.mjs')
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/views/pagesMember/ai/components/ApprovalCard.tsx')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      jsx: 'automatic',
      outfile: output,
      plugins: [{
        name: 'approval-ui',
        setup(builder) {
          builder.onResolve({ filter: /^(@tarojs\/components|@liujitcn\/kratos-taro-app-core)$/ }, args => ({ path: args.path, namespace: 'test-ui' }))
          builder.onLoad({ filter: /.*/, namespace: 'test-ui' }, ({ path }) => ({
            contents: path === '@tarojs/components'
              ? "export const Button='button', Text='text', View='view'"
              : 'export const t = key => key',
          }))
        },
      }],
    })
    const { default: ApprovalCard } = await import(pathToFileURL(output).href)
    const elements = node => Array.isArray(node)
      ? node.flatMap(elements)
      : node && typeof node === 'object' && node.props
        ? [node, ...elements(node.props.children)]
        : []
    const decisions = []
    const props = {
      approval: { id: 'once', calls: [{ name: 'write', title: '更新数据', arguments: '{"id":7}' }], expires_at: { seconds: Date.now() / 1000 + 60 } },
      status: 4,
      busy: false,
      onDecision: approved => decisions.push(approved),
    }
    const tree = elements(ApprovalCard(props))
    assert.ok(tree.some(node => node.type === 'text' && node.props.children === '{"id":7}'))
    const buttons = tree.filter(node => node.type === 'button')
    assert.equal(buttons.length, 2)
    buttons[0].props.onClick()
    buttons[1].props.onClick()
    assert.deepEqual(decisions, [true, false])
    assert.ok(elements(ApprovalCard({ ...props, busy: true })).filter(node => node.type === 'button').every(node => node.props.disabled))
    assert.equal(elements(ApprovalCard({ ...props, status: 2 })).filter(node => node.type === 'button').length, 0)
    assert.equal(elements(ApprovalCard({ ...props, approval: { ...props.approval, expires_at: { seconds: 1 } } })).filter(node => node.type === 'button').length, 0)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
