import { computed, readonly, ref } from 'vue'
import { defBaseMenuService } from './api/system/app/v1/base_menu'
import type { BaseMenu, ListBaseMenuResponse } from './rpc/system/app/v1/base_menu'
import { hasValidToken } from './utils/auth'
import { navigateToLogin } from './utils/navigation'
import { resolveStaticView } from './module'
import { matchLogicalPath, parseLogicalQuery } from './navigation-pattern.mjs'
import { buildMenuTree } from './navigation-tree.mjs'
import { getCurrentLocale, t } from './locales'

/** 移动菜单访问模式。 */
export type AppMenuAccess = 'PUBLIC' | 'GUEST_ONLY' | 'AUTHENTICATED'

/** uni-app 运行时使用的扁平菜单。 */
export interface AppMenu {
  id: number
  parentId?: number
  name: string
  path: string
  viewKey: string
  title: string
  access: AppMenuAccess
  inTabBar: boolean
  icon?: string
  selectedIcon?: string
}

/** uni-app 运行时使用的树形菜单节点。 */
export interface AppMenuNode extends AppMenu {
  children: AppMenuNode[]
}

/** 菜单获取适配器。 */
export interface AppNavigationAdapter {
  /** 获取扁平移动菜单配置。 */
  list(): Promise<ListBaseMenuResponse>
}

/** 逻辑路由解析结果。 */
export interface ResolvedAppRoute {
  menu: AppMenu
  physicalRoute: string
  query: Record<string, string>
}

const ANONYMOUS_CACHE_KEY = 'kratos-uni-app:navigation:anonymous'
const AUTHENTICATED_CACHE_KEY = 'kratos-uni-app:navigation:authenticated'
/** 固定移动端菜单根目录编号。 */
export const APP_MENU_ROOT_ID = 99000000
const createDefaultMenus = (): AppMenu[] => [
  {
    id: 99010000,
    parentId: APP_MENU_ROOT_ID,
    name: 'AppHome',
    path: 'app/home',
    viewKey: 'HOME',
    title: t('core.navigation.home'),
    access: 'PUBLIC',
    inTabBar: true,
    icon: 'HOME_DEFAULT',
    selectedIcon: 'HOME_SELECTED',
  },
  {
    id: 99040000,
    parentId: APP_MENU_ROOT_ID,
    name: 'AppMessage',
    path: 'app/message',
    viewKey: 'MESSAGE_INBOX',
    title: t('core.navigation.message'),
    access: 'AUTHENTICATED',
    inTabBar: true,
    icon: 'MESSAGE_DEFAULT',
    selectedIcon: 'MESSAGE_SELECTED',
  },
  {
    id: 99090000,
    parentId: APP_MENU_ROOT_ID,
    name: 'AppMy',
    path: 'app/my',
    viewKey: 'PROFILE_HOME',
    title: t('core.navigation.my'),
    access: 'PUBLIC',
    inTabBar: true,
    icon: 'USER_DEFAULT',
    selectedIcon: 'USER_SELECTED',
  },
  {
    id: 99010100,
    parentId: 99010000,
    name: 'AppLogin',
    path: 'app/login',
    viewKey: 'LOGIN',
    title: t('common.action.login'),
    access: 'GUEST_ONLY',
    inTabBar: false,
  },
  {
    id: 99010101,
    parentId: 99010100,
    name: 'AppProtocol',
    path: 'app/protocol/:type',
    viewKey: 'PROTOCOL',
    title: t('core.navigation.protocol'),
    access: 'PUBLIC',
    inTabBar: false,
  },
  {
    id: 99090100,
    parentId: 99090000,
    name: 'AppProfile',
    path: 'app/profile',
    viewKey: 'PROFILE',
    title: t('system.profile.title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 99090200,
    parentId: 99090000,
    name: 'AppSettings',
    path: 'app/settings',
    viewKey: 'SETTINGS',
    title: t('core.settings.title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 99090300,
    parentId: 99090000,
    name: 'AppAi',
    path: 'app/ai',
    viewKey: 'AI',
    title: t('system.ai.chat_title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 99010200,
    parentId: 99010000,
    name: 'AppWebView',
    path: 'app/webview',
    viewKey: 'WEBVIEW',
    title: '',
    access: 'PUBLIC',
    inTabBar: false,
  },
]

const menus = ref<AppMenu[]>([])
const ready = ref(false)
const appMenuBadges = ref<Record<string, number>>({})
const isH5Runtime = typeof window !== 'undefined'
let tabNavigationTarget: string | undefined
let adapter: AppNavigationAdapter = {
  list: () => defBaseMenuService.ListBaseMenu(),
}

const nativeTabViewKeys = ['HOME', 'PROFILE_HOME']
const pageTitleKeys: Record<string, string> = {
  'pages/index/index': 'core.home.main_title',
  'pages/login/login': 'common.action.login',
  'pages/login/protocal': 'core.navigation.protocol',
  'pages/my/my': 'core.navigation.my',
  'pagesMember/ai/index': 'system.ai.chat_title',
  'pagesMember/message/detail': 'system.notification.title',
  'pagesMember/message/index': 'system.notification.title',
  'pagesMember/profile/profile': 'system.profile.title',
  'pagesMember/settings/settings': 'core.settings.title',
}
const resetTabViewKeys = ['MESSAGE_INBOX']

/** 替换导航远端适配器，供宿主接入自定义契约。 */
export function setAppNavigationAdapter(nextAdapter: AppNavigationAdapter): void {
  adapter = nextAdapter
}

/** 初始化导航，远端失败时保留当前身份最后成功配置。 */
export async function initializeAppNavigation(): Promise<void> {
  const cacheKey = resolveCacheKey()
  const cached = readCachedMenus(cacheKey)
  menus.value = cached ?? createDefaultMenus()
  try {
    const nextMenus = normalizeMenuResponse(await adapter.list())
    validateMenus(nextMenus)
    uni.setStorageSync(cacheKey, nextMenus)
    menus.value = nextMenus
  } catch (error) {
    if (!cached) {
      console.warn('navigation fallback to local defaults', error)
    }
  } finally {
    ready.value = true
  }
}

/** 使用已经校验的整份配置进行原子切换。 */
export function installAppNavigation(nextMenus: AppMenu[]): void {
  validateMenus(nextMenus)
  const snapshot = nextMenus.map((menu) => ({ ...menu }))
  uni.setStorageSync(resolveCacheKey(), snapshot)
  menus.value = snapshot
  ready.value = true
}

/** 解析带路径参数和 query 的逻辑路由。 */
export function resolveAppRoute(rawRoute: string): ResolvedAppRoute | undefined {
  const [rawPath, rawQuery = ''] = rawRoute.replace(/^\/+/, '').split('?', 2)
  for (const menu of menus.value) {
    const params = matchLogicalPath(menu.path, rawPath)
    if (!params) continue
    const physicalRoute = resolveStaticView(menu.viewKey)
    if (!physicalRoute) return
    return {
      menu,
      physicalRoute,
      query: { ...params, ...parseLogicalQuery(rawQuery) },
    }
  }
}

/** 跳转逻辑路由，并执行与管理端一致的登录访问控制。 */
export function navigateAppRoute(rawRoute: string, options: { replace?: boolean } = {}): void {
  const resolved = resolveAppRoute(rawRoute)
  if (!resolved) {
    launchAppStatus('NOT_FOUND')
    return
  }
  if (resolved.menu.access === 'AUTHENTICATED' && !hasValidToken()) {
    navigateToLogin(`/pages/bootstrap/index?route=${encodeURIComponent(rawRoute)}`)
    return
  }
  if (resolved.menu.access === 'GUEST_ONLY' && hasValidToken()) {
    const home = menus.value.find((menu) => menu.viewKey === 'HOME')
    if (home) navigateAppRoute(home.path, { replace: true })
    return
  }
  const query = Object.entries(resolved.query)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&')
  const url = `/${resolved.physicalRoute}${query ? `?${query}` : ''}`
  if (options.replace) {
    uni.reLaunch({ url })
    return
  }
  if (resolved.menu.inTabBar) {
    navigateTabRoute(url)
    return
  }
  uni.navigateTo({ url, fail: () => uni.reLaunch({ url }) })
}

/** 切换 tab 页面，优先复用已有页面，避免重建页面栈产生空白帧。 */
function navigateTabRoute(url: string): void {
  const targetRoute = url.slice(1).split('?', 1)[0].replace(/^\/+/, '')
  const pages = getCurrentPages()
  const currentIndex = pages.length - 1
  const currentRoute = pages[currentIndex]?.route?.replace(/^\/+/, '')
  if (currentRoute === targetRoute || tabNavigationTarget === targetRoute) return

  tabNavigationTarget = targetRoute
  const release = () => {
    if (tabNavigationTarget === targetRoute) tabNavigationTarget = undefined
  }
  if (isH5Runtime) {
    uni.reLaunch({ url, success: release, fail: release, complete: release })
    return
  }
  if (nativeTabViewKeys.some((viewKey) => resolveStaticView(viewKey) === targetRoute)) {
    uni.switchTab({
      url,
      success: release,
      fail: () => navigateTabRouteInStack(url, targetRoute, pages, currentIndex, release),
      complete: release,
    })
    return
  }
  if (resetTabViewKeys.some((viewKey) => resolveStaticView(viewKey) === targetRoute)) {
    uni.reLaunch({ url, success: release, fail: release, complete: release })
    return
  }
  navigateTabRouteInStack(url, targetRoute, pages, currentIndex, release)
}

/** 在没有原生 tab 路由的动态菜单上复用页面栈。 */
function navigateTabRouteInStack(
  url: string,
  targetRoute: string,
  pages: ReturnType<typeof getCurrentPages>,
  currentIndex: number,
  release: () => void,
): void {
  const targetIndex = pages.findIndex((page) => page.route?.replace(/^\/+/, '') === targetRoute)
  if (targetIndex >= 0 && targetIndex < currentIndex) {
    uni.navigateBack({
      delta: currentIndex - targetIndex,
      success: release,
      fail: release,
      complete: release,
    })
    return
  }
  uni.navigateTo({
    url,
    success: release,
    fail: () => {
      uni.reLaunch({ url, success: release, fail: release, complete: release })
    },
  })
}

/** 按稳定 viewKey 跳转，并将动作参数作为查询参数传递。 */
export function navigateAppView(viewKey: string, params: Record<string, string> = {}): void {
  const menu = menus.value.find((item) => item.viewKey === viewKey)
  if (!menu) {
    launchAppStatus('NOT_FOUND')
    return
  }
  const remaining = { ...params }
  const path = menu.path.replace(/:([A-Za-z0-9_]+)/g, (placeholder, key: string) => {
    if (!(key in remaining)) return placeholder
    const value = remaining[key]
    delete remaining[key]
    return encodeURIComponent(value)
  })
  const query = Object.entries(remaining)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&')
  navigateAppRoute(`${path}${query ? `?${query}` : ''}`)
}

/** 设置移动端菜单 badge 数量，供业务模块注册未读状态。 */
export function setAppMenuBadge(viewKey: string, count: number): void {
  const next = { ...appMenuBadges.value }
  if (count > 0) next[viewKey] = Math.min(Math.floor(count), 99)
  else delete next[viewKey]
  appMenuBadges.value = next
}

/** 读取移动端菜单 badge 数量。 */
export function useAppMenuBadge(viewKey: string) {
  return computed(() => appMenuBadges.value[viewKey] ?? 0)
}

/** 根据物理页面路由更新 H5 和原生导航栏标题。 */
export function setAppPageTitle(route: string): void {
  const titleKey = pageTitleKeys[route]
  const menuTitle = menus.value.find((menu) => resolveStaticView(menu.viewKey) === route)?.title
  const title = titleKey ? t(titleKey) : menuTitle
  if (!title) return
  void uni.setNavigationBarTitle({ title })
}

/** 获取导航响应式状态。 */
export function useAppNavigation() {
  const menuTree = computed<AppMenuNode[]>(() => buildMenuTree(menus.value, APP_MENU_ROOT_ID))
  const tabBar = computed(() => menuTree.value.filter((menu) => menu.inTabBar))
  return {
    menus: readonly(menus),
    menuTree,
    ready: readonly(ready),
    tabBar,
    navigate: navigateAppRoute,
  }
}

/** 打开可替换的状态视图。 */
export function launchAppStatus(state: import('./module').BootstrapViewKey, detail = ''): void {
  const route = resolveStaticView(state) ?? 'pages/status/index'
  const query = `state=${encodeURIComponent(state)}${detail ? `&detail=${encodeURIComponent(detail)}` : ''}`
  uni.reLaunch({ url: `/${route}?${query}` })
}

function resolveCacheKey(): string {
  const identity = hasValidToken() ? AUTHENTICATED_CACHE_KEY : ANONYMOUS_CACHE_KEY
  return `${identity}:${getCurrentLocale()}`
}

function readCachedMenus(cacheKey: string): AppMenu[] | undefined {
  const cached = uni.getStorageSync(cacheKey) as unknown
  if (!Array.isArray(cached)) return
  try {
    const cachedMenus = cached as AppMenu[]
    validateMenus(cachedMenus)
    return cachedMenus
  } catch {
    uni.removeStorageSync(cacheKey)
  }
}

function normalizeMenuResponse(response: ListBaseMenuResponse): AppMenu[] {
  return response.items.map(normalizeMenu)
}

function normalizeMenu(item: BaseMenu): AppMenu {
  const meta = item.meta
  const app = meta?.app
  return {
    id: item.id,
    parentId: item.parent_id,
    name: item.name,
    path: item.path,
    viewKey: app?.view_key ?? '',
    title: meta?.title ?? '',
    access: (app?.access ?? 'AUTHENTICATED') as AppMenuAccess,
    inTabBar: app?.in_tab_bar ?? false,
    icon: meta?.icon,
    selectedIcon: app?.selected_icon,
  }
}

function validateMenus(nextMenus: AppMenu[]): void {
  const ids = new Set<number>()
  const names = new Set<string>()
  const paths = new Set<string>()
  for (const menu of nextMenus) {
    if (!Number.isSafeInteger(menu.id) || !menu.name || !menu.path.startsWith('app/')) {
      throw new Error(t('core.navigation.error.identity'))
    }
    if (!menu.name.startsWith('App')) {
      throw new Error(t('core.navigation.error.name_prefix', { name: menu.name }))
    }
    if (!['PUBLIC', 'GUEST_ONLY', 'AUTHENTICATED'].includes(menu.access)) {
      throw new Error(t('core.navigation.error.access', { access: menu.access }))
    }
    if (!resolveStaticView(menu.viewKey)) {
      throw new Error(t('core.navigation.error.view_key', { viewKey: menu.viewKey }))
    }
    if (ids.has(menu.id) || names.has(menu.name) || paths.has(menu.path)) {
      throw new Error(t('core.navigation.error.unique'))
    }
    ids.add(menu.id)
    names.add(menu.name)
    paths.add(menu.path)
  }
  const menuMap = new Map(nextMenus.map((menu) => [menu.id, menu]))
  for (const menu of nextMenus) {
    if (menu.parentId === undefined) {
      throw new Error(t('core.navigation.error.parent_missing', { name: menu.name }))
    }
    if (menu.parentId === APP_MENU_ROOT_ID) {
      if (!menu.inTabBar) throw new Error(t('core.navigation.error.root_tab', { name: menu.name }))
      continue
    }
    if (menu.inTabBar) throw new Error(t('core.navigation.error.child_tab', { name: menu.name }))
    const visited = new Set<number>([menu.id])
    let parentId = menu.parentId
    while (parentId !== APP_MENU_ROOT_ID) {
      if (visited.has(parentId))
        throw new Error(t('core.navigation.error.cycle', { name: menu.name }))
      visited.add(parentId)
      const parent = menuMap.get(parentId)
      if (!parent) throw new Error(t('core.navigation.error.parent_not_found', { name: menu.name }))
      if (parent.parentId === undefined) {
        throw new Error(t('core.navigation.error.parent_invalid', { name: menu.name }))
      }
      parentId = parent.parentId
    }
  }
  const tabCount = nextMenus.filter((menu) => menu.parentId === APP_MENU_ROOT_ID).length
  if (tabCount === 1 || tabCount > 5) throw new Error(t('core.navigation.error.tab_count'))
}
