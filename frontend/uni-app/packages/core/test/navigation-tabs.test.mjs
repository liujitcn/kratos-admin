import assert from 'node:assert/strict'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadNavigationRuntime(state) {
  const manifest = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../../../apps/uni-app/src/pages.json'), 'utf8'),
  )
  const nativeTabRoutes = new Set(manifest.tabBar?.list.map((item) => `/${item.pagePath}`))
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-tabs-'))
  const output = resolve(root, 'navigation.mjs')
  const originalWindow = globalThis.window
  if (state.isH5) globalThis.window = {}
  else Reflect.deleteProperty(globalThis, 'window')
  globalThis.uni = {
    getStorageSync() {
      return undefined
    },
    setStorageSync() {},
    removeStorageSync() {},
    navigateTo(options) {
      state.navigateCalls.push(options)
      if (state.pages.length >= 10) {
        options.fail?.({ errMsg: 'navigateTo:fail exceed max webview count' })
        return
      }
      if (state.deferNavigation) return
      state.pages.push({ route: options.url.slice(1).split('?', 1)[0] })
      options.success?.({ errMsg: 'navigateTo:ok' })
      options.complete?.({ errMsg: 'navigateTo:ok' })
    },
    redirectTo(options) {
      state.redirectCalls.push(options)
      if (nativeTabRoutes.has(options.url.split('?', 1)[0])) {
        const error = { errMsg: 'redirectTo:fail can not redirectTo a tabbar page' }
        state.navigationErrors.push(error.errMsg)
        options.fail?.(error)
        options.complete?.(error)
        return
      }
      if (state.deferNavigation) return
      state.pages.splice(state.pages.length - 1, 1, {
        route: options.url.slice(1).split('?', 1)[0],
      })
      options.success?.({ errMsg: 'redirectTo:ok' })
      options.complete?.({ errMsg: 'redirectTo:ok' })
    },
    navigateBack(options) {
      state.navigateBackCalls.push(options)
      state.pages.splice(state.pages.length - options.delta)
      options.success?.({ errMsg: 'navigateBack:ok' })
      options.complete?.({ errMsg: 'navigateBack:ok' })
    },
    switchTab(options) {
      state.switchTabCalls.push(options)
      if (!nativeTabRoutes.has(options.url.split('?', 1)[0])) {
        const error = { errMsg: 'switchTab:fail can not switch to no-tabBar page' }
        state.navigationErrors.push(error.errMsg)
        options.fail?.(error)
        options.complete?.(error)
        return
      }
      options.success?.({ errMsg: 'switchTab:ok' })
    },
    reLaunch(options) {
      state.reLaunchCalls.push(options)
      options.success?.({ errMsg: 'reLaunch:ok' })
      options.complete?.({ errMsg: 'reLaunch:ok' })
    },
  }
  globalThis.getCurrentPages = () => state.pages
  globalThis.__menuResponse = state.menuResponse ?? { items: [] }
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/navigation.ts')],
    bundle: true,
    platform: 'node',
    format: 'esm',
    outfile: output,
    plugins: [
      {
        name: 'uni-tabs-test-runtime',
        setup(buildApi) {
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
          buildApi.onResolve({ filter: /^vue$/ }, () => ({
            path: 'vue-test-runtime',
            namespace: 'test',
          }))
          buildApi.onLoad({ filter: /.*/, namespace: 'test' }, (args) => {
            if (args.path === 'vue-test-runtime') {
              return {
                loader: 'js',
                contents: `
                  export function ref(value) { return { value } }
                  export function readonly(value) { return value }
                  export function computed(getter) { return { get value() { return getter() } } }
                `,
              }
            }
            if (args.path === 'stores-test-runtime') {
              return {
                loader: 'js',
                contents: 'export function useSettingStore() { return { aiEnabled: true } }',
              }
            }
            if (args.path === 'menu-test-runtime') {
              return {
                loader: 'js',
                contents:
                  'export const defBaseMenuService = { ListBaseMenu: async () => globalThis.__menuResponse }',
              }
            }
            if (args.path === 'auth-test-runtime') {
              return { loader: 'js', contents: 'export function hasValidToken() { return false }' }
            }
            if (args.path === 'navigation-utils-test-runtime') {
              return {
                loader: 'js',
                contents:
                  'export function navigateToLogin() {}\nexport function redirectToLogin() {}',
              }
            }
            if (args.path === 'module-test-runtime') {
              return {
                loader: 'js',
                contents: `export function resolveStaticView(key) {
                  return {
                    HOME: 'pages/index/index',
                    SHOP_CATEGORY: 'pages/category/category',
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
  let runtime
  try {
    runtime = await import(`${pathToFileURL(output).href}?test=${Date.now()}`)
  } finally {
    if (originalWindow === undefined) Reflect.deleteProperty(globalThis, 'window')
    else globalThis.window = originalWindow
    rmSync(root, { recursive: true, force: true })
  }
  return runtime
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
  }
}

test('H5 自绘 tab 替换当前路由，避免重建应用产生闪烁', async () => {
  const state = { ...createState([{ route: 'pages/index/index' }]), isH5: true }
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.deepEqual(state.navigationErrors, [])
  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
})

test('微信端 tab 切换替换当前页面，避免页面入栈动画闪烁', async () => {
  const state = createState([{ route: 'pagesMember/message/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.pages.length, 1)
  assert.equal(state.pages.at(-1).route, 'pages/my/my')
})

test('自绘 tab 页面不声明缺少 custom-tab-bar 组件的原生 tabBar', () => {
  const manifest = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../../../apps/uni-app/src/pages.json'), 'utf8'),
  )

  assert.equal(manifest.tabBar, undefined)
})

test('动态菜单缺失标题时仅按公共 viewKey 回退本地化标题', async () => {
  const rootId = 99000000
  const state = {
    ...createState([{ route: 'pages/index/index' }]),
    menuResponse: {
      items: [
        {
          id: 99010000,
          parent_id: rootId,
          name: 'AppHome',
          path: 'app/home',
          meta: { app: { view_key: 'HOME', access: 'PUBLIC', in_tab_bar: true } },
        },
        {
          id: 99020000,
          parent_id: rootId,
          name: 'AppCategory',
          path: 'app/category',
          meta: { app: { view_key: 'SHOP_CATEGORY', access: 'PUBLIC', in_tab_bar: true } },
        },
        {
          id: 99040000,
          parent_id: rootId,
          name: 'AppMessage',
          path: 'app/message',
          meta: { app: { view_key: 'MESSAGE_INBOX', access: 'PUBLIC', in_tab_bar: true } },
        },
        {
          id: 99090000,
          parent_id: rootId,
          name: 'AppMy',
          path: 'app/my',
          meta: { app: { view_key: 'PROFILE_HOME', access: 'PUBLIC', in_tab_bar: true } },
        },
      ],
    },
  }
  const runtime = await loadNavigationRuntime(state)
  await runtime.initializeAppNavigation()

  assert.deepEqual(
    runtime.useAppNavigation().tabBar.value.map((item) => item.title),
    ['core.navigation.home', '', 'core.navigation.message', 'core.navigation.my'],
  )
})

test('H5 从消息 tab 切换到我的仍替换当前路由，不触发页面滑动栈', async () => {
  const state = { ...createState([{ route: 'pagesMember/message/index' }]), isH5: true }
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/my')

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
})

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

  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pages/my/my')
  assert.equal(state.navigateCalls.length, 0)
  assert.equal(state.navigateBackCalls.length, 0)
  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.pages.at(-1).route, 'pages/my/my')
})

test('微信端首页、消息、我的反复切换不重建页面栈', async () => {
  const state = createState([{ route: 'pages/index/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  for (let index = 0; index < 20; index++) {
    runtime.navigateAppRoute('app/message')
    assert.equal(state.pages.at(-1).route, 'pagesMember/message/index')
    runtime.navigateAppRoute('app/my')
    assert.equal(state.pages.at(-1).route, 'pages/my/my')
    runtime.navigateAppRoute('app/message')
    assert.equal(state.pages.at(-1).route, 'pagesMember/message/index')
    runtime.navigateAppRoute('app/home')
    assert.equal(state.pages.at(-1).route, 'pages/index/index')
  }

  assert.equal(state.pages.length, 1)
  assert.equal(state.reLaunchCalls.length, 0)
  assert.equal(state.switchTabCalls.length, 0)
  assert.equal(state.redirectCalls.length, 80)
  assert.deepEqual(state.navigationErrors, [])
})

test('外部业务模块覆盖首页并扩展到五个 tab 时，两端保持同一导航策略', async () => {
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

  for (const isH5 of [true, false]) {
    const state = { ...createState([{ route: viewRoutes.HOME }]), isH5, viewRoutes }
    const runtime = await loadNavigationRuntime(state)
    runtime.installAppNavigation(businessMenus)

    for (const menu of businessMenus) {
      runtime.navigateAppRoute(menu.path)
      assert.equal(state.pages.at(-1).route, viewRoutes[menu.viewKey])
    }
    runtime.navigateAppRoute('app/home')

    assert.equal(state.pages.at(-1).route, 'pages/workspace/index')
    assert.equal(state.pages.length, 1)
    assert.equal(state.reLaunchCalls.length, 0)
    assert.equal(state.switchTabCalls.length, 0)
    assert.deepEqual(state.navigationErrors, [])
  }
})

test('微信端页面栈满时替换栈顶，保留之前的页面', async () => {
  const pages = Array.from({ length: 10 }, (_, index) => ({ route: `pages/detail/${index}` }))
  const state = createState([...pages])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/message')

  assert.equal(state.redirectCalls.length, 1)
  assert.equal(state.redirectCalls[0].url, '/pagesMember/message/index')
  assert.deepEqual(state.pages.slice(0, 9), pages.slice(0, 9))
  assert.equal(state.pages.at(-1).route, 'pagesMember/message/index')
  assert.equal(state.pages.length, 10)
  assert.equal(state.reLaunchCalls.length, 0)
})

test('两端重复点击当前 tab 不发起跳转', async () => {
  for (const isH5 of [true, false]) {
    const state = { ...createState([{ route: 'pages/index/index' }]), isH5 }
    const runtime = await loadNavigationRuntime(state)
    runtime.installAppNavigation(menus)

    runtime.navigateAppRoute('app/home')

    assert.equal(state.redirectCalls.length, 0)
    assert.equal(state.navigateCalls.length, 0)
    assert.equal(state.navigateBackCalls.length, 0)
    assert.equal(state.switchTabCalls.length, 0)
    assert.equal(state.reLaunchCalls.length, 0)
  }
})

test('两端跳转未结束时忽略同一 tab 的重复点击，失败后允许重试', async () => {
  for (const isH5 of [true, false]) {
    const state = {
      ...createState([{ route: 'pages/index/index' }]),
      isH5,
      deferNavigation: true,
    }
    const runtime = await loadNavigationRuntime(state)
    runtime.installAppNavigation(menus)

    runtime.navigateAppRoute('app/my')
    runtime.navigateAppRoute('app/my')

    const calls = state.redirectCalls
    assert.equal(calls.length, 1)
    calls[0].fail({ errMsg: 'navigation:fail' })
    if (!isH5) {
      state.navigateCalls[0].fail({ errMsg: 'navigateTo:fail' })
      state.redirectCalls[1].fail({ errMsg: 'redirectTo:fail' })
    }
    runtime.navigateAppRoute('app/my')

    assert.equal(calls.length, isH5 ? 2 : 3)
    assert.equal(state.reLaunchCalls.length, 0)
  }
})

test('启动或身份变化要求 replace 时保留整栈替换行为', async () => {
  const state = createState([{ route: 'pages/bootstrap/index' }])
  const runtime = await loadNavigationRuntime(state)
  runtime.installAppNavigation(menus)

  runtime.navigateAppRoute('app/home', { replace: true })

  assert.equal(state.reLaunchCalls.length, 1)
  assert.equal(state.reLaunchCalls[0].url, '/pages/index/index')
})
