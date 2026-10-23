import assert from 'node:assert/strict'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNavigationRuntime(state, environment = 'weapp') {
  const manifest = JSON.parse(readFileSync(
    resolve(import.meta.dirname, '../../../apps/taro-app/src/app.config.base.json'), 'utf8',
  ))
  state.nativeTabRoutes = manifest.tabBar?.list.map((item) => `/${item.pagePath}`) ?? []
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-tabs-'))
  const output = resolve(root, 'navigation.mjs')
  process.__KRATOS_TARO_TABS_TEST_STATE__ = state
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/navigation.ts')],
    bundle: true,
    platform: 'node',
    define: { 'process.env.TARO_ENV': JSON.stringify(environment) },
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
          buildApi.onResolve({ filter: /\/api\/system\/app\/v1\/base_menu(?:\.ts)?$/ }, () => ({
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
          buildApi.onResolve({ filter: /\/stores(?:\/index)?(?:\.ts)?$/ }, () => ({
            path: 'stores-test-runtime',
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
                      if (${JSON.stringify(environment)} === 'weapp' && state.nativeTabRoutes.includes(options.url.split('?', 1)[0])) {
                        state.navigationErrors.push('navigateTo:fail can not navigateTo a tabbar page')
                        return Promise.reject(new Error(state.navigationErrors.at(-1)))
                      }
                      if (state.pages.length >= 10) return Promise.reject(new Error('navigateTo:fail exceed max webview count'))
                      if (state.deferNavigation) {
                        return new Promise((resolve) => state.pendingNavigations.push(resolve))
                      }
                      state.pages.push({ route: options.url.slice(1).split('?', 1)[0] })
                      return Promise.resolve({ errMsg: 'navigateTo:ok' })
                    },
                    redirectTo(options) {
                      state.redirectCalls.push(options)
                      if (state.redirectError) return Promise.reject(state.redirectError)
                      if (state.deferNavigation) {
                        return new Promise((resolve) => state.pendingNavigations.push(resolve))
                      }
                      state.pages.splice(state.pages.length - 1, 1, { route: options.url.slice(1).split('?', 1)[0] })
                      return Promise.resolve({ errMsg: 'redirectTo:ok' })
                    },
                    navigateBack(options) {
                      state.navigateBackCalls.push(options)
                      state.pages.splice(state.pages.length - options.delta)
                      return Promise.resolve({ errMsg: 'navigateBack:ok' })
                    },
                    switchTab(options) {
                      state.switchTabCalls.push(options)
                      if (!state.nativeTabRoutes.includes(options.url.split('?', 1)[0])) {
                        state.navigationErrors.push('switchTab:fail can not switch to no-tabBar page')
                        return Promise.reject(new Error(state.navigationErrors.at(-1)))
                      }
                      state.pages = [{ route: options.url.slice(1).split('?', 1)[0] }]
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
            if (args.path === 'stores-test-runtime') {
              return {
                loader: 'js',
                contents: `export const useSettingStore = {
                  getState() { return { aiEnabled: true } }
                }`,
              }
            }
            if (args.path === 'menu-test-runtime') {
              return {
                loader: 'js',
                contents: 'export const defBaseMenuService = { list: async () => [] }',
              }
            }
            if (args.path === 'auth-test-runtime') {
              return { loader: 'js', contents: 'export function hasValidToken() { return false }' }
            }
            if (args.path === 'navigation-utils-test-runtime') {
              return {
                loader: 'js',
                contents: 'export function navigateToLogin() {}\nexport function redirectToLogin() {}',
              }
            }
            if (args.path === 'module-test-runtime') {
              return {
                loader: 'js',
                contents: `export function resolveStaticView(key) {
                  return {
                    HOME: 'pages/index/index',
                    MESSAGE_INBOX: 'pagesMember/message/index',
                    PROFILE_HOME: 'pages/my/my',
                    ...${JSON.stringify(state.viewRoutes ?? {})},
                  }[key]
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
    redirectCalls: [],
    navigateBackCalls: [],
    switchTabCalls: [],
    reLaunchCalls: [],
    navigationErrors: [],
    pendingNavigations: [],
  }
}

const menus = [
  {
    id: 99010000,
    parentId: 99000000,
    name: 'AppHome',
    path: 'app/home',
    viewKey: 'HOME',
    title: 'Home',
    access: 'PUBLIC',
    inTabBar: true,
  },
  {
    id: 99090000,
    parentId: 99000000,
    name: 'AppMy',
    path: 'app/my',
    viewKey: 'PROFILE_HOME',
    title: 'My',
    access: 'PUBLIC',
    inTabBar: true,
  },
  {
    id: 99040000,
    parentId: 99000000,
    name: 'AppMessage',
    path: 'app/message',
    viewKey: 'MESSAGE_INBOX',
    title: 'Messages',
    access: 'PUBLIC',
    inTabBar: true,
  },
]

test('微信端首次切换我的替换当前页面，避免页面入栈动画', async () => {
  const state = createState([{ route: 'pages/index/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')
  await Promise.resolve()

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
  assert.deepEqual(state.navigationErrors, [])
  assert.equal(state.pages.at(-1).route, 'pages/my/my')
})

test('微信端从子页面切换 tab 仍替换当前页面', async () => {
  const state = createState([
    { route: 'pages/index/index' },
    { route: 'pages/my/my' },
    { route: 'pagesMember/profile/profile' },
  ])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')
  await Promise.resolve()

  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.pages.at(-1).route, 'pages/my/my')
})

test('微信端切换消息 tab 只替换当前页面', async () => {
  const state = createState([{ route: 'pages/index/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/message')
  await new Promise((done) => setImmediate(done))

  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.pages.at(-1).route, 'pagesMember/message/index')
})

test('H5 tab 跳转失败只解除导航锁，不重建页面栈', async () => {
  const state = { ...createState([{ route: 'pages/index/index' }]), redirectError: new Error('redirectTo:fail') }
  const runtime = await loadNavigationRuntime(state, 'h5')
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')
  await new Promise((done) => setImmediate(done))

  assert.equal(state.reLaunchCalls.length, 0)
  runtime.navigateAppRoute('app/my')
  await new Promise((done) => setImmediate(done))
  assert.equal(state.redirectCalls.length, 2)
})

test('H5 自绘 tab 替换当前路由，避免重建应用产生闪烁', async () => {
  const state = createState([{ route: 'pages/index/index' }])
  const navigation = await loadNavigationRuntime(state, 'h5')
  navigation.installAppNavigation(menus)

  navigation.navigateAppRoute('app/my')
  await new Promise((done) => setTimeout(done, 0))

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
})

test('微信端 tab 切换替换当前页面，避免页面入栈动画闪烁', async () => {
  const state = createState([{ route: 'pagesMember/message/index' }])
  const navigation = await loadNavigationRuntime(state, 'weapp')
  navigation.installAppNavigation(menus)

  navigation.navigateAppRoute('app/my')
  await Promise.resolve()

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.pages.length, 1)
  assert.equal(state.pages.at(-1).route, 'pages/my/my')
})

test('H5 从消息 tab 切换到我的仍替换当前路由，不触发页面滑动栈', async () => {
  const state = createState([{ route: 'pagesMember/message/index' }])
  const navigation = await loadNavigationRuntime(state, 'h5')
  navigation.installAppNavigation(menus)

  navigation.navigateAppRoute('app/my')
  await new Promise((done) => setTimeout(done, 0))

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
})

test('微信端页面栈满时替换栈顶，保留之前的页面', async () => {
  const pages = Array.from({ length: 10 }, (_, index) => ({ route: `pages/detail/${index}` }))
  const state = createState([...pages])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/message')
  await new Promise((done) => setImmediate(done))

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pagesMember/message/index')
  assert.deepEqual(state.pages.slice(0, 9), pages.slice(0, 9))
  assert.equal(state.pages.at(-1).route, 'pagesMember/message/index')
  assert.equal(state.pages.length, 10)
  assert.equal(state.reLaunchCalls.length, 0)
})

test('两端重复点击当前 tab 和跳转未结束时点击其他 tab 均不重复跳转', async () => {
  for (const environment of ['h5', 'weapp']) {
    const state = { ...createState([{ route: 'pages/index/index' }]), deferNavigation: true }
    const runtime = await loadNavigationRuntime(state, environment)
    runtime.installAppNavigation(menus)

    runtime.navigateAppRoute('app/home')
    assert.equal(state.navigateCalls.length, 0)
    assert.equal(state.redirectCalls.length, 0)
    runtime.navigateAppRoute('app/my')
    runtime.navigateAppRoute('app/my')
    runtime.navigateAppRoute('app/message')

    const calls = state.redirectCalls
    assert.equal(calls.length, 1)
    state.pendingNavigations[0]({ errMsg: 'navigation:ok' })
    await new Promise((done) => setImmediate(done))
    runtime.navigateAppRoute('app/message')
    assert.equal(calls.length, 2)
    state.pendingNavigations[1]({ errMsg: 'navigation:ok' })
    await new Promise((done) => setImmediate(done))
    assert.equal(state.reLaunchCalls.length, 0)
  }
})

test('外部业务模块覆盖首页并扩展到五个 tab 时，两端反复切换不重建页面栈', async () => {
  const businessMenus = [
    ...menus,
    {
      id: 99020000,
      parentId: 99000000,
      name: 'AppAnalytic',
      path: 'app/analytic',
      viewKey: 'APP_ANALYTIC',
      title: 'Analytics',
      access: 'PUBLIC',
      inTabBar: true,
    },
    {
      id: 99030000,
      parentId: 99000000,
      name: 'AppReport',
      path: 'app/report',
      viewKey: 'APP_REPORT',
      title: 'Reports',
      access: 'PUBLIC',
      inTabBar: true,
    },
  ]
  const viewRoutes = {
    HOME: 'pages/workspace/index',
    APP_ANALYTIC: 'pages/analytic/index',
    APP_REPORT: 'pages/report/index',
    MESSAGE_INBOX: 'pagesMember/message/index',
    PROFILE_HOME: 'pages/my/my',
  }

  for (const environment of ['h5', 'weapp']) {
    const state = { ...createState([{ route: viewRoutes.HOME }]), viewRoutes }
    const runtime = await loadNavigationRuntime(state, environment)
    runtime.installAppNavigation(businessMenus)

    for (let index = 0; index < 20; index++) {
      for (const menu of businessMenus) {
        runtime.navigateAppRoute(menu.path)
        await new Promise((done) => setImmediate(done))
        assert.equal(state.pages.at(-1).route, viewRoutes[menu.viewKey])
      }
      runtime.navigateAppRoute('app/home')
      await new Promise((done) => setImmediate(done))
    }

    assert.equal(state.pages.at(-1).route, 'pages/workspace/index')
    assert.equal(state.pages.length, 1)
    assert.equal(state.reLaunchCalls.length, 0)
    assert.equal(state.switchTabCalls.length, 0)
    assert.deepEqual(state.navigationErrors, [])
  }
})
