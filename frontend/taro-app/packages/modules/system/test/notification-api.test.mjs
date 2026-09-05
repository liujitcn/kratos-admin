import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNotificationApi() {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-notification-api-'))
  const output = resolve(root, 'notification-api.mjs')
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/api/base/v1/notification.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'taro-notification-api-test-runtime',
        setup(buildApi) {
          buildApi.onResolve(
            { filter: /^@liujitcn\/kratos-taro-app-core\/utils\/http$/ },
            () => ({ path: 'http-test-runtime', namespace: 'test' }),
          )
          buildApi.onLoad({ filter: /.*/, namespace: 'test' }, () => ({
            loader: 'js',
            contents: `
              export function http(options) {
                if (options.url.endsWith('/categories')) return Promise.resolve({})
                if (options.url.endsWith('/summary')) return Promise.resolve({})
                return Promise.resolve({ total: 0 })
              }
            `,
          }))
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

test('消息接口对省略的空集合字段提供稳定默认值', async () => {
  const { defNotificationService } = await loadNotificationApi()

  assert.deepEqual(
    await defNotificationService.PageNotification({
      view: 1,
      cursor_id: 0,
      page_num: 1,
      page_size: 20,
    }),
    { notifications: [], total: 0, next_cursor_id: 0, has_more: false },
  )
  assert.deepEqual(await defNotificationService.ListNotificationCategories({}), { categories: [] })
  assert.deepEqual(await defNotificationService.GetNotificationSummary({}), {
    unread_total: 0,
    latest_delivery_id: 0,
    category_unread: [],
  })
})
