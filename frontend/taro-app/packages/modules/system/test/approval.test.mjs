import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('移动端冻结操作只读展示参数，任何状态均不提供批准和拒绝按钮', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-approval-'))
  const output = resolve(root, 'approval.mjs')
  try {
    await build({
      entryPoints: [
        resolve(import.meta.dirname, '../src/views/pagesMember/ai/components/ApprovalCard.tsx'),
      ],
      bundle: true,
      platform: 'node',
      format: 'esm',
      jsx: 'automatic',
      outfile: output,
      plugins: [
        {
          name: 'approval-ui',
          setup(builder) {
            builder.onResolve(
              { filter: /^(@tarojs\/components|@liujitcn\/kratos-taro-app-core)$/ },
              (args) => ({ path: args.path, namespace: 'test-ui' }),
            )
            builder.onLoad({ filter: /.*/, namespace: 'test-ui' }, ({ path }) => ({
              contents:
                path === '@tarojs/components'
                  ? "export const Button='button', Text='text', View='view'"
                  : 'export const t = key => key',
            }))
          },
        },
      ],
    })
    const { default: ApprovalCard } = await import(pathToFileURL(output).href)
    const elements = (node) =>
      Array.isArray(node)
        ? node.flatMap(elements)
        : node && typeof node === 'object' && node.props
          ? [node, ...elements(node.props.children)]
          : []
    const props = {
      approval: {
        id: 'once',
        calls: [{ name: 'write', title: '更新数据', arguments: '{"id":7}' }],
        expires_at: { seconds: Date.now() / 1000 + 60 },
      },
      status: 4,
    }
    const tree = elements(ApprovalCard(props))
    assert.ok(tree.some((node) => node.type === 'text' && node.props.children === '{"id":7}'))
    assert.equal(tree.filter((node) => node.type === 'button').length, 0)
    for (const status of [1, 2, 3, 4]) {
      assert.equal(
        elements(ApprovalCard({ ...props, status })).filter((node) => node.type === 'button')
          .length,
        0,
      )
    }
    for (const expires_at of [{ seconds: 1 }, '1970-01-01T00:00:01Z']) {
      const expired = elements(
        ApprovalCard({ ...props, approval: { ...props.approval, expires_at } }),
      )
      assert.equal(expired.filter((node) => node.type === 'button').length, 0)
      assert.ok(
        expired.some((node) => node.props.children === 'system.ai.workspace.approval_expired'),
      )
    }
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
