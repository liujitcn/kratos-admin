import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('登录页失效请求不重复弹窗，公共接口不携带旧令牌，业务页仍提示重新登录', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-http-'))
  const originalUni = globalThis.uni
  const originalPages = globalThis.getCurrentPages
  const storage = new Map([['access_token', 'Bearer stale']])
  const requests = []
  const modals = []
  const launches = []
  let page = 'pages/login/login'
  let responseStatus = 200
  let requestInterceptor
  try {
    const output = resolve(root, 'http.mjs')
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/utils/http.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
      define: { 'import.meta.env': '{}' },
      plugins: [
        {
          name: 'runtime-stubs',
          setup(builder) {
            builder.onResolve({ filter: /^\.\.\/locales$/ }, () => ({
              path: 'locales',
              namespace: 'stub',
            }))
            builder.onResolve({ filter: /^\.\/navigation$/ }, () => ({
              path: 'navigation',
              namespace: 'stub',
            }))
            builder.onLoad({ filter: /.*/, namespace: 'stub' }, ({ path }) => ({
              contents:
                path === 'locales'
                  ? 'export const t = (key) => key; export const getLocaleRequestHeaders = () => ({})'
                  : 'export const saveCurrentRoute = () => {}',
            }))
          },
        },
      ],
    })
    globalThis.getCurrentPages = () => [{ route: page }]
    globalThis.uni = {
      getStorageSync: (key) => storage.get(key),
      setStorageSync: (key, value) => storage.set(key, value),
      removeStorageSync: (key) => storage.delete(key),
      $emit() {},
      addInterceptor(name, interceptor) {
        if (name === 'request') requestInterceptor = interceptor
      },
      request(options) {
        requestInterceptor.invoke(options)
        requests.push(options)
        options.success({ statusCode: responseStatus, data: {} })
      },
      async showModal(options) {
        modals.push(options)
        return { confirm: true }
      },
      async showToast() {},
      reLaunch(options) {
        launches.push(options)
      },
    }
    const runtime = await import(pathToFileURL(output).href)
    await runtime.http({ url: '/v1/base/config', method: 'GET' })
    assert.equal(requests[0].header.Authorization, undefined, '公共请求不得被拦截器重新附加旧令牌')
    storage.clear()
    for (let index = 0; index < 3; index++) {
      await assert.rejects(runtime.getRequestAccessToken('required'))
    }
    assert.equal(modals.length, 0, '登录页不得弹出会话失效对话框')
    assert.equal(launches.length, 0, '登录页不得反复重启页面栈')
    storage.set('access_token', 'Bearer expired')
    storage.set('refresh_token', 'expired-refresh')
    storage.set('expiresIn', String(Date.now() - 1000))
    responseStatus = 401
    await assert.rejects(runtime.getRequestAccessToken('required'))
    assert.equal(requests.at(-1).header.Authorization, undefined, '刷新接口不能携带旧访问令牌')
    assert.equal(modals.length, 0, '登录页刷新失败也不能触发重新登录弹窗')
    assert.equal(storage.has('access_token'), false, '登录页应清理过期登录态')
    page = 'pages/profile/index'
    await assert.rejects(runtime.getRequestAccessToken('required'))
    assert.equal(modals.length, 1, '业务页的必要鉴权失败仍应提示用户')
    assert.equal(launches.length, 1)
  } finally {
    globalThis.uni = originalUni
    globalThis.getCurrentPages = originalPages
    rmSync(root, { recursive: true, force: true })
  }
})
