import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNotificationRuntime(state) {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-notification-'))
  const output = resolve(root, 'notification.mjs')
  process.__KRATOS_TARO_NOTIFICATION_TEST_STATE__ = state
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/notification.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'taro-notification-test-runtime',
        setup(buildApi) {
          buildApi.onResolve(
            { filter: /^@liujitcn\/kratos-taro-app-core\/navigation$/ },
            () => ({ path: 'navigation-test-runtime', namespace: 'test' }),
          )
          buildApi.onResolve(
            { filter: /^@liujitcn\/kratos-taro-app-core\/utils\/auth$/ },
            () => ({ path: 'auth-test-runtime', namespace: 'test' }),
          )
          buildApi.onResolve(
            { filter: /^@liujitcn\/kratos-taro-app-core\/utils\/http$/ },
            () => ({ path: 'http-test-runtime', namespace: 'test' }),
          )
          buildApi.onLoad({ filter: /.*/, namespace: 'test' }, (args) => {
            const stateExpression = 'process.__KRATOS_TARO_NOTIFICATION_TEST_STATE__'
            if (args.path === 'navigation-test-runtime') {
              return {
                loader: 'js',
                contents: `
                  const state = ${stateExpression}
                  export function setAppMenuBadge(viewKey, count) {
                    state.badges.push({ viewKey, count })
                  }
                `,
              }
            }
            if (args.path === 'auth-test-runtime') {
              return {
                loader: 'js',
                contents: `
                  const state = ${stateExpression}
                  export function getToken() { return state.token }
                `,
              }
            }
            return {
              loader: 'js',
              contents: `
                const state = ${stateExpression}
                export function getRequestAccessToken() { return Promise.resolve(state.token) }
                export function http() {
                  state.summaryCalls += 1
                  return Promise.resolve({ unread_total: 3 })
                }
              `,
            }
          })
        },
      },
    ],
  })
  try {
    return await import(`${pathToFileURL(output).href}?test=${Date.now()}`)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
}

test('匿名用户前后台恢复不会请求站内信或触发登录失效处理', async () => {
  const state = { token: '', summaryCalls: 0, badges: [] }
  const runtime = await loadNotificationRuntime(state)

  runtime.pauseNotificationPolling()
  runtime.resumeNotificationPolling()
  await Promise.resolve()

  assert.equal(state.summaryCalls, 0)
  runtime.stopNotificationPolling()
})

test('登录后存在令牌时启动站内信轮询', async () => {
  const state = { token: 'Bearer token', summaryCalls: 0, badges: [] }
  const runtime = await loadNotificationRuntime(state)

  runtime.startNotificationPolling()
  await Promise.resolve()

  assert.equal(state.summaryCalls, 1)
  runtime.stopNotificationPolling()
})
