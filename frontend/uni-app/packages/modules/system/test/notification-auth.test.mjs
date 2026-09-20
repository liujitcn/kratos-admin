import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('游客或过期登录态恢复前台时不启动通知请求', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-notification-'))
  const originalWindow = globalThis.window
  const originalFetch = globalThis.fetch
  const originalSetInterval = globalThis.setInterval
  let validToken = false
  let summaryRequests = 0
  let intervals = 0
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
            builder.onResolve({ filter: /^\.\/api\/base\/v1\/notification$/ }, () => ({
              path: 'notification-api',
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
                    "export const requestBaseURL = '/api'; export const getRequestAccessToken = async () => 'Bearer valid'",
                }
              }
              return {
                contents:
                  'export const defNotificationService = { GetNotificationSummary: async () => { globalThis.__summaryRequests += 1; return { unread_total: 0 } } }',
              }
            })
          },
        },
      ],
    })
    globalThis.__validToken = validToken
    globalThis.__summaryRequests = summaryRequests
    globalThis.window = undefined
    globalThis.fetch = undefined
    globalThis.setInterval = () => {
      intervals += 1
      return 1
    }
    const runtime = await import(pathToFileURL(output).href)

    runtime.pauseNotificationPolling()
    runtime.resumeNotificationPolling()
    await Promise.resolve()
    assert.equal(globalThis.__summaryRequests, 0)
    assert.equal(intervals, 0)

    validToken = true
    globalThis.__validToken = validToken
    runtime.resumeNotificationPolling()
    await Promise.resolve()
    assert.equal(globalThis.__summaryRequests, 1)
    assert.equal(intervals, 1)
  } finally {
    globalThis.window = originalWindow
    globalThis.fetch = originalFetch
    globalThis.setInterval = originalSetInterval
    delete globalThis.__validToken
    delete globalThis.__summaryRequests
    rmSync(root, { recursive: true, force: true })
  }
})
