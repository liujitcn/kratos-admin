import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

/** 使用替换式历史记录验证真实导航工具，并模拟平台 API。 */
async function loadRuntime(t, { authenticated = false, pages = ['pages/login/login'] } = {}) {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-login-navigation-'))
  const output = resolve(root, 'navigation.mjs')
  const storage = new Map()
  const history = [...pages]
  const calls = []
  let navigationError
  if (authenticated) {
    storage.set('access_token', 'Bearer valid')
    storage.set('expiresIn', String(Date.now() + 60_000))
  }
  const api = {
    getStorageSync: (key) => storage.get(key),
    setStorageSync: (key, value) => storage.set(key, value),
    removeStorageSync: (key) => storage.delete(key),
    getCurrentPages: () => history.map((route) => ({ route })),
    navigateTo(options) {
      calls.push(['navigateTo', options.url])
      history.push(options.url)
      return Promise.resolve()
    },
    redirectTo(options) {
      calls.push(['redirectTo', options.url])
      history[history.length - 1] = options.url
      return Promise.resolve()
    },
    reLaunch(options) {
      calls.push(['reLaunch', options.url])
      if (navigationError) return Promise.reject(navigationError)
      history[history.length - 1] = options.url
      return Promise.resolve()
    },
  }
  const originalUni = globalThis.uni
  const originalPages = globalThis.getCurrentPages
  const originalRuntime = globalThis.__KRATOS_LOGIN_NAVIGATION_TEST__
  globalThis.uni = api
  globalThis.getCurrentPages = api.getCurrentPages
  globalThis.__KRATOS_LOGIN_NAVIGATION_TEST__ = api
  t.after(() => {
    globalThis.uni = originalUni
    globalThis.getCurrentPages = originalPages
    globalThis.__KRATOS_LOGIN_NAVIGATION_TEST__ = originalRuntime
    rmSync(root, { recursive: true, force: true })
  })
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/utils/navigation.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'login-navigation-runtime',
        setup(builder) {
          builder.onResolve({ filter: /^@tarojs\/taro$/ }, () => ({
            path: 'platform',
            namespace: 'stub',
          }))
          builder.onLoad({ filter: /.*/, namespace: 'stub' }, () => ({
            contents: `const api = globalThis.__KRATOS_LOGIN_NAVIGATION_TEST__;
            export default api;
            export const getCurrentPages = api.getCurrentPages;`,
          }))
        },
      },
    ],
  })
  const runtime = await import(pathToFileURL(output).href)
  return {
    runtime,
    storage,
    history,
    calls,
    failNavigation(error) {
      navigationError = error
    },
  }
}

test('直接打开登录页清理旧目标，登录成功替换到首页，历史中不残留登录页', async (t) => {
  const { runtime, storage, history } = await loadRuntime(t)
  storage.set('lastRoute', '/pagesMember/profile/profile?id=8')
  assert.equal(runtime.prepareLoginPage(), true)
  assert.equal(storage.has('lastRoute'), false)
  await runtime.restoreLoginRedirect()
  assert.deepEqual(history, ['/pages/index/index'])
  assert.equal(storage.get('SwitchTabIndex'), true)
})

test('登录页返回替换成首页，清理旧目标且不退入受保护页面', async (t) => {
  const { runtime, storage, history, calls } = await loadRuntime(t, {
    pages: ['pagesMember/profile/profile', 'pages/login/login'],
  })
  storage.set('lastRoute', '/pagesMember/profile/profile')
  await runtime.returnFromLogin()
  assert.deepEqual(calls, [['reLaunch', '/pages/index/index']])
  assert.equal(history.at(-1), '/pages/index/index')
  assert.equal(
    history.some((route) => route.includes('pages/login/login')),
    false,
  )
  assert.equal(storage.has('lastRoute'), false)
})

test('本次会话过期重登保留目标与参数，成功后清理目标', async (t) => {
  const { runtime, storage, history } = await loadRuntime(t)
  const target = '/pagesMember/profile/profile?id=8&source=expired'
  runtime.saveLoginRedirectUrl(target)
  assert.equal(runtime.prepareLoginPage('1'), true)
  await runtime.restoreLoginRedirect()
  assert.equal(history.at(-1), target)
  assert.equal(storage.has('lastRoute'), false)
})

test('已登录再次显示登录页直接替换回首页，不继续登录表单初始化', async (t) => {
  const { runtime, storage, history } = await loadRuntime(t, { authenticated: true })
  storage.set('lastRoute', '/pagesMember/profile/profile')
  assert.equal(runtime.prepareLoginPage('1'), false)
  assert.equal(history.at(-1), '/pages/index/index')
  assert.equal(storage.has('lastRoute'), false)
})

test('非法登录页回跳目标退回首页，避免自跳循环', async (t) => {
  const { runtime, storage, history } = await loadRuntime(t)
  storage.set('lastRoute', '//pages/login/login?redirect=1')
  await runtime.restoreLoginRedirect()
  assert.equal(history.at(-1), '/pages/index/index')
  assert.equal(storage.has('lastRoute'), false)
})

test('回跳失败保留业务目标，重试成功后才清理', async (t) => {
  const { runtime, storage, history, failNavigation } = await loadRuntime(t)
  storage.set('lastRoute', '/pagesMember/profile/profile?id=8')
  failNavigation(new Error('navigation failed'))
  await assert.rejects(runtime.restoreLoginRedirect(), /navigation failed/)
  assert.equal(storage.get('lastRoute'), '/pagesMember/profile/profile?id=8')
  failNavigation(undefined)
  await runtime.restoreLoginRedirect()
  assert.equal(history.at(-1), '/pagesMember/profile/profile?id=8')
  assert.equal(storage.has('lastRoute'), false)
})

test('主动登录和守卫重定向均标明本次回跳，避免被当作直接打开', async (t) => {
  const { runtime, storage, calls } = await loadRuntime(t)
  runtime.navigateToLogin('/pagesMember/profile/profile?id=8')
  assert.deepEqual(calls.at(-1), ['navigateTo', '/pages/login/login?redirect=1'])
  assert.equal(storage.get('lastRoute'), '/pagesMember/profile/profile?id=8')
  runtime.redirectToLogin('/pages/bootstrap/index?route=app%2Fmy')
  assert.deepEqual(calls.at(-1), ['redirectTo', '/pages/login/login?redirect=1'])
  assert.equal(storage.get('lastRoute'), '/pages/bootstrap/index?route=app%2Fmy')
})
