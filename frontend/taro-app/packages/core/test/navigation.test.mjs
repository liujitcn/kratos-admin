import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNavigationRuntime(state) {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-navigation-'))
  const output = resolve(root, 'navigation.mjs')
  process.__KRATOS_TARO_NAVIGATION_TEST_STATE__ = state
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/utils/navigation.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'taro-navigation-test-runtime',
        setup(buildApi) {
          buildApi.onResolve({ filter: /^@tarojs\/taro$/ }, () => ({
            path: 'taro-navigation-test-runtime',
            namespace: 'test',
          }))
          buildApi.onLoad({ filter: /.*/, namespace: 'test' }, () => ({
            loader: 'js',
            contents: `
              const state = process.__KRATOS_TARO_NAVIGATION_TEST_STATE__
              const Taro = {
                getStorageSync(key) { return state.storage.get(key) },
                setStorageSync(key, value) { state.storage.set(key, value) },
                removeStorageSync(key) { state.storage.delete(key) },
                navigateTo(options) { state.navigateCalls.push(options); return Promise.resolve({ errMsg: 'navigateTo:ok' }) },
                reLaunch(options) {
                  state.reLaunchCalls.push(options)
                  return state.navigationError
                    ? Promise.reject(state.navigationError)
                    : Promise.resolve({ errMsg: 'reLaunch:ok' })
                },
              }
              export default Taro
              export function getCurrentPages() { return state.pages }
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

function createState() {
  return {
    storage: new Map(),
    pages: [],
    navigateCalls: [],
    reLaunchCalls: [],
    navigationError: undefined,
  }
}

test('保存当前 H5 页面时规范化路径并保留查询参数', async () => {
  const state = createState()
  state.pages = [
    {
      route: '/pagesMember/ai/index',
      options: {},
      $taroParams: { prompt: '你好 world', source: 'menu' },
    },
  ]
  const runtime = await loadNavigationRuntime(state)

  runtime.saveCurrentRoute()

  assert.equal(
    state.storage.get('lastRoute'),
    '/pagesMember/ai/index?prompt=%E4%BD%A0%E5%A5%BD%20world&source=menu',
  )
})

test('保存微信页面时保留 options 并覆盖同名 H5 参数', async () => {
  const state = createState()
  state.pages = [
    {
      route: 'pagesMember/profile/profile',
      options: { id: 8, source: 'weapp' },
      $taroParams: { source: 'h5' },
    },
  ]
  const runtime = await loadNavigationRuntime(state)

  runtime.saveCurrentRoute()

  assert.equal(state.storage.get('lastRoute'), '/pagesMember/profile/profile?source=weapp&id=8')
})

test('登录页不会清理已经保存的回跳地址', async () => {
  const state = createState()
  state.storage.set('lastRoute', '/pagesMember/profile/profile?id=8')
  state.pages = [{ route: 'pages/login/login', options: {} }]
  const runtime = await loadNavigationRuntime(state)

  runtime.saveCurrentRoute()

  assert.equal(state.storage.get('lastRoute'), '/pagesMember/profile/profile?id=8')
})

test('回跳失败时保留地址，成功后才清理', async () => {
  const state = createState()
  state.storage.set('lastRoute', '//pagesMember/ai/index?prompt=test')
  state.navigationError = new Error('navigation failed')
  const runtime = await loadNavigationRuntime(state)

  await assert.rejects(runtime.restoreLoginRedirect(), /navigation failed/)
  assert.equal(state.storage.get('lastRoute'), '//pagesMember/ai/index?prompt=test')
  assert.equal(state.reLaunchCalls[0].url, '/pagesMember/ai/index?prompt=test')

  state.navigationError = undefined
  await runtime.restoreLoginRedirect()
  assert.equal(state.storage.has('lastRoute'), false)
})
