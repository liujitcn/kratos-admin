import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('游客或过期登录态恢复前台时不启动通知请求，运行中失效时静默停止', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-notification-'))
  const originalWindow = globalThis.window
  const originalFetch = globalThis.fetch
  const originalSetInterval = globalThis.setInterval
  let validToken = false
  let intervals = 0
  let intervalHandler
  try {
    const output = resolve(root, 'notification.mjs')
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/notification.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
      plugins: [
        {
          name: 'runtime-stubs',
          setup(builder) {
            builder.onResolve({ filter: /^vue$/ }, () => ({ path: 'vue', namespace: 'stub' }))
            builder.onResolve({ filter: /kratos-uni-app-core\/navigation$/ }, () => ({
              path: 'navigation',
              namespace: 'stub',
            }))
            builder.onResolve({ filter: /kratos-uni-app-core\/utils\/auth$/ }, () => ({
              path: 'auth',
              namespace: 'stub',
            }))
            builder.onResolve({ filter: /kratos-uni-app-core\/utils\/http$/ }, () => ({
              path: 'http',
              namespace: 'stub',
            }))
            builder.onLoad({ filter: /.*/, namespace: 'stub' }, ({ path }) => {
              if (path === 'vue') return { contents: 'export const ref = (value) => ({ value })' }
              if (path === 'navigation')
                return { contents: 'export const setAppMenuBadge = () => {}' }
              if (path === 'auth') {
                return { contents: 'export const hasValidToken = () => globalThis.__validToken' }
              }
              if (path === 'http') {
                return {
                  contents:
                    "export const requestBaseURL = '/api'; export const getRequestAccessToken = async (mode = 'required') => { globalThis.__authModes.push(`stream:${mode}`); if (!globalThis.__validToken) { if (mode === 'required') globalThis.__reloginPrompts += 1; throw new Error('expired'); } return 'Bearer valid'; }; export const http = async (options) => { globalThis.__authModes.push(`request:${options.authMode}`); if (!globalThis.__validToken) { if (options.authMode === 'required') globalThis.__reloginPrompts += 1; throw new Error('expired'); } return { unread_total: 0 }; }",
                }
              }
            })
          },
        },
      ],
    })
    globalThis.__validToken = validToken
    globalThis.__authModes = []
    globalThis.__reloginPrompts = 0
    globalThis.window = undefined
    globalThis.fetch = undefined
    globalThis.setInterval = (handler) => {
      intervals += 1
      intervalHandler = handler
      return 1
    }
    const runtime = await import(pathToFileURL(output).href)

    runtime.pauseNotificationPolling()
    runtime.resumeNotificationPolling()
    await Promise.resolve()
    assert.deepEqual(globalThis.__authModes, [])
    assert.equal(intervals, 0)

    validToken = true
    globalThis.__validToken = validToken
    runtime.resumeNotificationPolling()
    await Promise.resolve()
    assert.deepEqual(globalThis.__authModes, ['request:optional'])
    assert.equal(intervals, 1)

    runtime.stopNotificationPolling()
    globalThis.__authModes = []
    globalThis.window = { location: { origin: 'http://localhost:5004' } }
    globalThis.fetch = () => new Promise(() => {})
    runtime.startNotificationPolling()
    await Promise.resolve()
    assert.deepEqual(globalThis.__authModes, ['request:optional', 'stream:optional'])

    validToken = false
    globalThis.__validToken = validToken
    intervalHandler()
    await Promise.resolve()
    assert.equal(globalThis.__reloginPrompts, 0, '后台轮询不得弹出重新登录对话框')
    assert.equal(globalThis.__authModes.at(-1), 'request:optional')
  } finally {
    globalThis.window = originalWindow
    globalThis.fetch = originalFetch
    globalThis.setInterval = originalSetInterval
    delete globalThis.__validToken
    delete globalThis.__authModes
    delete globalThis.__reloginPrompts
    rmSync(root, { recursive: true, force: true })
  }
})
