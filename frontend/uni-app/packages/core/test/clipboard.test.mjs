import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('H5 Clipboard API 失败时降级到同步选区复制', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-clipboard-'))
  const output = resolve(root, 'clipboard.mjs')
  const appended = []
  const textarea = {
    setAttribute() {},
    select() {},
    style: {},
    value: '',
  }
  const originalNavigator = globalThis.navigator
  const originalDocument = globalThis.document
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/utils/clipboard.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
    })
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: { clipboard: { writeText: async () => Promise.reject(new Error('insecure')) } },
    })
    globalThis.document = {
      body: {
        appendChild(node) {
          appended.push(node)
        },
        removeChild(node) {
          assert.equal(appended.pop(), node)
        },
      },
      createElement() {
        return textarea
      },
      execCommand(command) {
        assert.equal(command, 'copy')
        return true
      },
    }
    const runtime = await import(`${pathToFileURL(output).href}?test=${Date.now()}`)
    await runtime.copyText('RECOVERY-CODE')
    assert.equal(textarea.value, 'RECOVERY-CODE')
    assert.equal(appended.length, 0)
  } finally {
    Object.defineProperty(globalThis, 'navigator', {
      configurable: true,
      value: originalNavigator,
    })
    globalThis.document = originalDocument
    rmSync(root, { recursive: true, force: true })
  }
})
