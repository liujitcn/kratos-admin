import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('刷新令牌失效后在重新登录弹窗确认前清理本地认证', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-http-'))
  const output = resolve(root, 'http.mjs')
  const state = {
    events: [],
    modals: [],
    launches: [],
    storage: new Map([
      ['access_token', 'Bearer expired'],
      ['refresh_token', 'expired-refresh'],
      ['expiresIn', String(Date.now() - 1000)],
    ]),
  }
  process.__KRATOS_TARO_HTTP_TEST_STATE__ = state
  let resolveModal
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/utils/http.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
      define: {
        'process.env.TARO_ENV': JSON.stringify('h5'),
        'process.env.VITE_APP_BASE_API': JSON.stringify('/api'),
        'process.env.VITE_APP_API_URL': JSON.stringify(''),
      },
      plugins: [
        {
          name: 'runtime-stubs',
          setup(builder) {
            builder.onResolve({ filter: /^@tarojs\/taro$/ }, () => ({
              path: 'taro-runtime',
              namespace: 'stub',
            }))
            builder.onResolve({ filter: /^\.\.\/locales$/ }, () => ({
              path: 'locales',
              namespace: 'stub',
            }))
            builder.onResolve({ filter: /^\.\/navigation$/ }, () => ({
              path: 'navigation',
              namespace: 'stub',
            }))
            builder.onLoad({ filter: /.*/, namespace: 'stub' }, ({ path }) => {
              if (path === 'locales') {
                return {
                  contents:
                    'export const t = (key) => key; export const getLocaleRequestHeaders = () => ({})',
                }
              }
              if (path === 'navigation') {
                return { contents: 'export const saveCurrentRoute = () => {}' }
              }
              return {
                contents: `
                  const state = process.__KRATOS_TARO_HTTP_TEST_STATE__
                  const Taro = {
                    eventCenter: { trigger(event) { state.events.push(event) } },
                    getStorageSync(key) { return state.storage.get(key) },
                    setStorageSync(key, value) { state.storage.set(key, value) },
                    removeStorageSync(key) { state.storage.delete(key) },
                    request() {
                      return Promise.resolve({ statusCode: 401, data: { reason: 'UNAUTHENTICATED' } })
                    },
                    showModal(options) {
                      state.modals.push(options)
                      return new Promise((done) => { process.__KRATOS_TARO_HTTP_MODAL_RESOLVE__ = done })
                    },
                    showToast() { return Promise.resolve() },
                    reLaunch(options) {
                      state.launches.push(options)
                      return Promise.resolve()
                    },
                  }
                  export default Taro
                `,
              }
            })
          },
        },
      ],
    })
    const runtime = await import(pathToFileURL(output).href)
    const refreshAttempt = runtime.getRequestAccessToken('required')
    await new Promise((done) => setTimeout(done, 0))
    resolveModal = process.__KRATOS_TARO_HTTP_MODAL_RESOLVE__

    assert.equal(state.modals.length, 1, '刷新失败后仍应提示用户重新登录')
    assert.equal(state.storage.has('access_token'), false, '弹窗确认前应清理访问令牌')
    assert.equal(state.storage.has('refresh_token'), false, '弹窗确认前应清理刷新令牌')
    assert.deepEqual(state.events, ['auth:silent-logout'])

    resolveModal({ confirm: true })
    await assert.rejects(refreshAttempt)
    assert.deepEqual(state.launches, [{ url: '/pages/login/login' }])
  } finally {
    delete process.__KRATOS_TARO_HTTP_TEST_STATE__
    delete process.__KRATOS_TARO_HTTP_MODAL_RESOLVE__
    rmSync(root, { recursive: true, force: true })
  }
})
