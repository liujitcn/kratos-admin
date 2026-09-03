import Taro, { getCurrentPages } from '@tarojs/taro'
import { create } from 'zustand'
import { defBaseMenuService } from './api/system/app/v1/base_menu'
import type { BaseMenu, ListBaseMenuResponse } from './rpc/system/app/v1/base_menu'
import { resolveStaticView } from './module'
import { matchLogicalPath, parseLogicalQuery } from './navigation-pattern.mjs'
import { buildMenuTree } from './navigation-tree.mjs'
import { hasValidToken } from './utils/auth'
import { navigateToLogin } from './utils/navigation'
import { getCurrentLocale, t } from './locales'

/** 移动菜单访问模式。 */
export type AppMenuAccess = 'PUBLIC' | 'GUEST_ONLY' | 'AUTHENTICATED'

/** Taro 运行时使用的扁平菜单。 */
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

/** Taro 运行时使用的树形菜单节点。 */
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

/** 导航响应式状态。 */
export interface AppNavigationState {
  menus: AppMenu[]
  ready: boolean
  menuTree: AppMenuNode[]
  tabBar: AppMenuNode[]
}

/** 移动端菜单未读 badge 状态。 */
interface AppMenuBadgeState {
  badges: Record<string, number>
}

/** 移动端菜单 badge 状态容器。 */
export const useAppMenuBadges = create<AppMenuBadgeState>(() => ({ badges: {} }))

/** 设置移动端菜单 badge 数量，供业务模块注册未读状态。 */
export function setAppMenuBadge(viewKey: string, count: number): void {
  useAppMenuBadges.setState((state) => {
    const badges = { ...state.badges }
    if (count > 0) badges[viewKey] = Math.min(Math.floor(count), 99)
    else delete badges[viewKey]
    return { badges }
  })
}

const ANONYMOUS_CACHE_KEY = 'kratos-taro-app:navigation:anonymous'
const AUTHENTICATED_CACHE_KEY = 'kratos-taro-app:navigation:authenticated'
/** 固定移动端菜单根目录编号。 */
export const APP_MENU_ROOT_ID = 99000000
/** 本地兜底移动菜单。 */
export const defaultAppMenus: AppMenu[] = [
  {
    id: 99010000,
    parentId: APP_MENU_ROOT_ID,
    name: 'AppHome',
    path: 'app/home',
    viewKey: 'HOME',
    title: '',
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
    title: '',
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
    title: '',
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
    title: '',
    access: 'GUEST_ONLY',
    inTabBar: false,
  },
  {
    id: 99010101,
    parentId: 99010100,
    name: 'AppProtocol',
    path: 'app/protocol/:type',
    viewKey: 'PROTOCOL',
    title: '',
    access: 'PUBLIC',
    inTabBar: false,
  },
  {
    id: 99090100,
    parentId: 99090000,
    name: 'AppProfile',
    path: 'app/profile',
    viewKey: 'PROFILE',
    title: '',
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 99090200,
    parentId: 99090000,
    name: 'AppSettings',
    path: 'app/settings',
    viewKey: 'SETTINGS',
    title: '',
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 99090300,
    parentId: 99090000,
    name: 'AppAi',
    path: 'app/ai',
    viewKey: 'AI',
    title: '',
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

const defaultMenuTitleKeys: Record<string, string> = {
  AppHome: 'core.navigation.home',
  AppMessage: 'core.navigation.message',
  AppMy: 'core.navigation.my',
  AppLogin: 'common.action.login',
  AppProtocol: 'core.navigation.protocol',
  AppProfile: 'system.profile.title',
  AppSettings: 'system.settings.title',
  AppAi: 'system.ai.chat_title',
}

let tabNavigationTarget: string | undefined
const nativeTabViewKeys = ['HOME', 'PROFILE_HOME']
const resetTabViewKeys = ['MESSAGE_INBOX']

function localizedDefaultAppMenus(): AppMenu[] {
  return defaultAppMenus.map((menu) => ({
    ...menu,
    title: defaultMenuTitleKeys[menu.name] ? t(defaultMenuTitleKeys[menu.name]) : menu.title,
  }))
}

function createNavigationState(menus: AppMenu[], ready: boolean): AppNavigationState {
  const menuTree = buildMenuTree(menus, APP_MENU_ROOT_ID) as AppMenuNode[]
  return { menus, ready, menuTree, tabBar: menuTree.filter((menu) => menu.inTabBar) }
}

/** 应用导航 Zustand Store。 */
export const useAppNavigation = create<AppNavigationState>(() =>
  createNavigationState(defaultAppMenus, false),
)

let adapter: AppNavigationAdapter = {
  list: () => defBaseMenuService.ListBaseMenu(),
}

/** 替换导航远端适配器，供宿主接入自定义契约。 */
export function setAppNavigationAdapter(nextAdapter: AppNavigationAdapter): void {
  adapter = nextAdapter
}

/** 初始化导航，远端失败时保留当前身份最后成功配置。 */
export async function initializeAppNavigation(): Promise<void> {
  const cacheKey = resolveCacheKey()
  const cached = readCachedMenus(cacheKey)
  useAppNavigation.setState(createNavigationState(cached ?? localizedDefaultAppMenus(), false))
  try {
    const nextMenus = normalizeMenuResponse(await adapter.list())
    validateMenus(nextMenus)
    Taro.setStorageSync(cacheKey, nextMenus)
    useAppNavigation.setState(createNavigationState(nextMenus, true))
  } catch (error) {
    if (!cached) console.warn('navigation fallback to local defaults', error)
    useAppNavigation.setState((state) => ({ ...state, ready: true }))
  }
}

/** 使用已经校验的整份配置进行原子切换。 */
export function installAppNavigation(nextMenus: AppMenu[]): void {
  validateMenus(nextMenus)
  const snapshot = nextMenus.map((menu) => ({ ...menu }))
  Taro.setStorageSync(resolveCacheKey(), snapshot)
  useAppNavigation.setState(createNavigationState(snapshot, true))
}

/** 解析带路径参数和 query 的逻辑路由。 */
export function resolveAppRoute(rawRoute: string): ResolvedAppRoute | undefined {
  const [rawPath, rawQuery = ''] = rawRoute.replace(/^\/+/, '').split('?', 2)
  for (const menu of useAppNavigation.getState().menus) {
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

/** 跳转逻辑路由，并执行登录访问控制。 */
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
    const home = useAppNavigation.getState().menus.find((menu) => menu.viewKey === 'HOME')
    if (home) navigateAppRoute(home.path, { replace: true })
    return
  }
  const query = Object.entries(resolved.query)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&')
  const url = `/${resolved.physicalRoute}${query ? `?${query}` : ''}`
  if (options.replace) {
    void Taro.reLaunch({ url })
    return
  }
  if (resolved.menu.inTabBar) {
    navigateTabRoute(url)
    return
  }
  void Taro.navigateTo({ url, fail: () => void Taro.reLaunch({ url }) })
}

/** 按稳定 viewKey 跳转，并将动作参数作为查询参数传递。 */
export function navigateAppView(viewKey: string, params: Record<string, string> = {}): void {
  const menu = useAppNavigation.getState().menus.find((item) => item.viewKey === viewKey)
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

/** 切换 tab 页面，优先复用已有页面，避免重建页面栈产生空白帧。 */
function navigateTabRoute(url: string): void {
  const targetRoute = url.slice(1).split('?', 1)[0].replace(/^\/+/, '')
  const pages = getCurrentPages()
  const currentIndex = pages.length - 1
  const currentRoute = pages[currentIndex]?.route?.replace(/^\/+/, '')
  if (currentRoute === targetRoute || tabNavigationTarget) return

  tabNavigationTarget = targetRoute
  const release = () => {
    if (tabNavigationTarget === targetRoute) tabNavigationTarget = undefined
  }
  if (nativeTabViewKeys.some((viewKey) => resolveStaticView(viewKey) === targetRoute)) {
    void Taro.switchTab({ url })
      .then(release)
      .catch(() => {
        navigateTabRouteInStack(url, targetRoute, pages, currentIndex, release)
      })
    return
  }
  if (resetTabViewKeys.some((viewKey) => resolveStaticView(viewKey) === targetRoute)) {
    void Taro.reLaunch({ url }).then(release, release)
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
  const navigation =
    targetIndex >= 0 && targetIndex < currentIndex
      ? Taro.navigateBack({ delta: currentIndex - targetIndex })
      : Taro.navigateTo({ url })
  void navigation.then(release).catch(() => {
    void Taro.reLaunch({ url }).then(release, release)
  })
}

/** 打开可替换的状态视图。 */
export function launchAppStatus(state: import('./module').BootstrapViewKey, detail = ''): void {
  const route = resolveStaticView(state) ?? 'pages/status/index'
  const query = `state=${encodeURIComponent(state)}${detail ? `&detail=${encodeURIComponent(detail)}` : ''}`
  void Taro.reLaunch({ url: `/${route}?${query}` })
}

function resolveCacheKey(): string {
  const identity = hasValidToken() ? AUTHENTICATED_CACHE_KEY : ANONYMOUS_CACHE_KEY
  return `${identity}:${getCurrentLocale()}`
}

function readCachedMenus(cacheKey: string): AppMenu[] | undefined {
  const cached = Taro.getStorageSync<unknown>(cacheKey)
  if (!Array.isArray(cached)) return
  try {
    const cachedMenus = cached as AppMenu[]
    validateMenus(cachedMenus)
    return cachedMenus
  } catch {
    Taro.removeStorageSync(cacheKey)
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
      if (!parent?.parentId)
        throw new Error(t('core.navigation.error.parent_not_found', { name: menu.name }))
      parentId = parent.parentId
    }
  }
  const tabCount = nextMenus.filter((menu) => menu.parentId === APP_MENU_ROOT_ID).length
  if (tabCount === 1 || tabCount > 5) throw new Error(t('core.navigation.error.tab_count'))
}
