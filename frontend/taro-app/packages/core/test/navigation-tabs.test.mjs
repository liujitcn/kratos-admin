import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNavigationRuntime(state) {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-tabs-'))
  const output = resolve(root, 'navigation.mjs')
  process.__KRATOS_TARO_TABS_TEST_STATE__ = state
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/navigation.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'taro-tabs-test-runtime',
        setup(buildApi) {
          buildApi.onResolve({ filter: /^@tarojs\/taro$/ }, () => ({
            path: 'taro-tabs-test-runtime',
            namespace: 'test',
          }))
          buildApi.onResolve({ filter: /\/api\/base\/menu(?:\.ts)?$/ }, () => ({
            path: 'menu-test-runtime',
            namespace: 'test',
          }))
          buildApi.onResolve({ filter: /\/utils\/auth(?:\.ts)?$/ }, () => ({
            path: 'auth-test-runtime',
            namespace: 'test',
          }))
          buildApi.onResolve({ filter: /\/utils\/navigation(?:\.ts)?$/ }, () => ({
            path: 'navigation-utils-test-runtime',
            namespace: 'test',
          }))
          buildApi.onResolve({ filter: /\/module(?:\.ts)?$/ }, () => ({
            path: 'module-test-runtime',
            namespace: 'test',
          }))
          buildApi.onResolve({ filter: /\/locales(?:\/index)?(?:\.ts)?$/ }, () => ({
            path: 'locales-test-runtime',
            namespace: 'test',
          }))
          buildApi.onLoad({ filter: /.*/, namespace: 'test' }, (args) => {
            const stateExpression = 'process.__KRATOS_TARO_TABS_TEST_STATE__'
            if (args.path === 'taro-tabs-test-runtime') {
              return {
                loader: 'js',
                contents: `
                  const state = ${stateExpression}
                  const Taro = {
                    getStorageSync() { return undefined },
                    setStorageSync() {},
                    removeStorageSync() {},
                    navigateTo(options) {
                      state.navigateCalls.push(options)
                      return Promise.resolve({ errMsg: 'navigateTo:ok' })
                    },
                    navigateBack(options) {
                      state.navigateBackCalls.push(options)
                      return Promise.resolve({ errMsg: 'navigateBack:ok' })
                    },
                    switchTab(options) {
                      state.switchTabCalls.push(options)
                      return Promise.resolve({ errMsg: 'switchTab:ok' })
                    },
                    reLaunch(options) {
                      state.reLaunchCalls.push(options)
                      return Promise.resolve({ errMsg: 'reLaunch:ok' })
                    },
                  }
                  export default Taro
                  export function getCurrentPages() { return state.pages }
                `,
              }
            }
            if (args.path === 'menu-test-runtime') {
              return { loader: 'js', contents: 'export const defBaseMenuService = { list: async () => [] }' }
            }
            if (args.path === 'auth-test-runtime') {
              return { loader: 'js', contents: 'export function hasValidToken() { return false }' }
            }
            if (args.path === 'navigation-utils-test-runtime') {
              return { loader: 'js', contents: 'export function navigateToLogin() {}' }
            }
            if (args.path === 'module-test-runtime') {
              return {
                loader: 'js',
                contents: `export function resolveStaticView(key) {
                  return { HOME: 'pages/index/index', PROFILE_HOME: 'pages/my/my' }[key]
                }`,
              }
            }
            return {
              loader: 'js',
              contents: `export function getCurrentLocale() { return 'zh-CN' }
                export function t(key) { return key }`,
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

function createState(pages) {
  return {
    pages,
    navigateCalls: [],
    navigateBackCalls: [],
    switchTabCalls: [],
    reLaunchCalls: [],
  }
}

const menus = [
  {
    id: 99901,
    parentId: 999,
    name: 'AppHome',
    path: 'app/home',
    viewKey: 'HOME',
    title: 'Home',
    access: 'PUBLIC',
    inTabBar: true,
  },
  {
    id: 99909,
    parentId: 999,
    name: 'AppMy',
    path: 'app/my',
    viewKey: 'PROFILE_HOME',
    title: 'My',
    access: 'PUBLIC',
    inTabBar: true,
  },
]

test('首次切换固定 tab 使用原生 switchTab，不重建页面栈', async () => {
  const state = createState([{ route: 'pages/index/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')
  await Promise.resolve()

  assert.equal(state.switchTabCalls.length, 1)
  assert.equal(state.switchTabCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
})

test('切换已存在的固定 tab 仍使用原生 switchTab', async () => {
  const state = createState([
    { route: 'pages/index/index' },
    { route: 'pages/my/my' },
    { route: 'pagesMember/profile/profile' },
  ])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')
  await Promise.resolve()

  assert.equal(state.switchTabCalls.length, 1)
  assert.equal(state.switchTabCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
})
