import Taro, { getCurrentPages } from '@tarojs/taro'
import { create } from 'zustand'
import { defBaseMenuService } from './api/system/app/v1/base_menu'
import type { BaseMenu, ListBaseMenuResponse } from './rpc/system/app/v1/base_menu'
import { resolveStaticView } from './module'
import { matchLogicalPath, parseLogicalQuery } from './navigation-pattern.mjs'
import { buildMenuTree } from './navigation-tree.mjs'
import { hasValidToken } from './utils/auth'
import { redirectToLogin } from './utils/navigation'
import { getCurrentLocale, t } from './locales'
import { useSettingStore } from './stores'

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
    id: 99050100,
    parentId: 99040000,
    name: 'AppMessageDetail',
    path: 'app/message/detail',
    viewKey: 'MESSAGE_DETAIL',
    title: '',
    access: 'AUTHENTICATED',
    inTabBar: false,
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
  useAppNavigation.setState(createNavigationState(filterUnavailableMenus(cached ?? localizedDefaultAppMenus()), false))
  try {
    const nextMenus = validateMenus(
      filterUnavailableMenus(normalizeMenuResponse(await adapter.list())),
    )
    Taro.setStorageSync(cacheKey, nextMenus)
    useAppNavigation.setState(createNavigationState(nextMenus, true))
  } catch (error) {
    if (!cached) console.warn('navigation fallback to local defaults', error)
    useAppNavigation.setState((state) => ({ ...state, ready: true }))
  }
}

/** 根据运行时能力过滤不可用的移动端入口。 */
function filterUnavailableMenus(nextMenus: AppMenu[]): AppMenu[] {
  if (useSettingStore.getState().aiEnabled) return nextMenus
  return nextMenus.filter((menu) => menu.viewKey !== 'AI')
}

/** 使用已经校验的整份配置进行原子切换。 */
export function installAppNavigation(nextMenus: AppMenu[]): void {
  const snapshot = validateMenus(nextMenus).map((menu) => ({ ...menu }))
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
    if (menu.viewKey === 'AI' && !useSettingStore.getState().aiEnabled) return
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
  // 守卫拦截用替换式跳转：登录页顶掉当前页，返回时不再弹回被拦截的页面形成循环。
  if (resolved.menu.access === 'AUTHENTICATED' && !hasValidToken()) {
    redirectToLogin(`/pages/bootstrap/index?route=${encodeURIComponent(rawRoute)}`)
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

/** 切换自绘 tab：直接替换当前页面，避免微信端页面入栈/出栈动画造成闪烁。 */
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
  if (process.env.TARO_ENV === 'h5') {
    void Taro.redirectTo({ url }).then(release, release)
    return
  }
  void Taro.redirectTo({ url }).then(release, () => {
    navigateTabRouteInStack(url, pages, currentIndex, release)
  })
}

/** 微信端替换路由失败时复用已有页面栈。 */
function navigateTabRouteInStack(
  url: string,
  pages: ReturnType<typeof getCurrentPages>,
  currentIndex: number,
  release: () => void,
): void {
  const targetRoute = url.slice(1).split('?', 1)[0].replace(/^\/+/, '')
  const targetIndex = pages.findIndex((page) => page.route?.replace(/^\/+/, '') === targetRoute)
  if (targetIndex >= 0 && targetIndex < currentIndex) {
    void Taro.navigateBack({ delta: currentIndex - targetIndex }).then(release, release)
    return
  }
  void Taro.navigateTo({ url }).then(release, () => {
    // 小程序页面栈满时只替换栈顶，保留其他 tab 的已加载页面。
    void Taro.redirectTo({ url }).then(release, release)
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
    const cachedMenus = (cached as AppMenu[]).map((menu) => ({
      ...menu,
      title:
        menu.viewKey === 'HOME'
          ? t('core.navigation.home')
          : menu.viewKey === 'MESSAGE_INBOX'
            ? t('core.navigation.message')
            : menu.title,
    }))
    return validateMenus(cachedMenus)
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
    title:
      app?.view_key === 'HOME'
        ? t('core.navigation.home')
        : app?.view_key === 'MESSAGE_INBOX'
          ? t('core.navigation.message')
          : meta?.title ?? '',
    access: (app?.access ?? 'AUTHENTICATED') as AppMenuAccess,
    inTabBar: app?.in_tab_bar ?? false,
    icon: meta?.icon,
    selectedIcon: app?.selected_icon,
  }
}

function validateMenus(nextMenus: AppMenu[]): AppMenu[] {
  // 多端共用后端时菜单可能携带本应用未注册的视图键，过滤跳过而不是让整份菜单失效。
  const knownMenus = nextMenus.filter((menu) => {
    if (resolveStaticView(menu.viewKey)) return true
    console.warn(t('core.navigation.error.view_key', { viewKey: menu.viewKey }))
    return false
  })
  const ids = new Set<number>()
  const names = new Set<string>()
  const paths = new Set<string>()
  for (const menu of knownMenus) {
    if (!Number.isSafeInteger(menu.id) || !menu.name || !menu.path.startsWith('app/')) {
      throw new Error(t('core.navigation.error.identity'))
    }
    if (!menu.name.startsWith('App')) {
      throw new Error(t('core.navigation.error.name_prefix', { name: menu.name }))
    }
    if (!['PUBLIC', 'GUEST_ONLY', 'AUTHENTICATED'].includes(menu.access)) {
      throw new Error(t('core.navigation.error.access', { access: menu.access }))
    }
    if (ids.has(menu.id) || names.has(menu.name) || paths.has(menu.path)) {
      throw new Error(t('core.navigation.error.unique'))
    }
    ids.add(menu.id)
    names.add(menu.name)
    paths.add(menu.path)
  }
  const menuMap = new Map(knownMenus.map((menu) => [menu.id, menu]))
  for (const menu of knownMenus) {
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
  const tabCount = knownMenus.filter((menu) => menu.parentId === APP_MENU_ROOT_ID).length
  if (tabCount === 1 || tabCount > 5) throw new Error(t('core.navigation.error.tab_count'))
  return knownMenus
}
