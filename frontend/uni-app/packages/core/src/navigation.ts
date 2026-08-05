import { computed, readonly, ref } from 'vue'
import { defBaseMenuService } from './api/base/menu'
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
  list(): Promise<unknown>
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
export const APP_MENU_ROOT_ID = 999
const createDefaultMenus = (): AppMenu[] => [
  {
    id: 99901,
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
    id: 99909,
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
    id: 9990101,
    parentId: 99901,
    name: 'AppLogin',
    path: 'app/login',
    viewKey: 'LOGIN',
    title: t('common.action.login'),
    access: 'GUEST_ONLY',
    inTabBar: false,
  },
  {
    id: 999010101,
    parentId: 9990101,
    name: 'AppProtocol',
    path: 'app/protocol/:type',
    viewKey: 'PROTOCOL',
    title: t('core.navigation.protocol'),
    access: 'PUBLIC',
    inTabBar: false,
  },
  {
    id: 9990901,
    parentId: 99909,
    name: 'AppProfile',
    path: 'app/profile',
    viewKey: 'PROFILE',
    title: t('system.profile.title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 9990902,
    parentId: 99909,
    name: 'AppSettings',
    path: 'app/settings',
    viewKey: 'SETTINGS',
    title: t('system.settings.title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 9990903,
    parentId: 99909,
    name: 'AppAi',
    path: 'app/ai',
    viewKey: 'AI',
    title: t('system.ai.chat_title'),
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 9990102,
    parentId: 99901,
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
let tabNavigationTarget: string | undefined
let adapter: AppNavigationAdapter = {
  list: () => defBaseMenuService.ListBaseMenu(),
}

const nativeTabViewKeys = ['HOME', 'PROFILE_HOME']

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
  if (currentRoute === targetRoute || tabNavigationTarget) return

  tabNavigationTarget = targetRoute
  const release = () => {
    if (tabNavigationTarget === targetRoute) tabNavigationTarget = undefined
  }
  if (nativeTabViewKeys.some((viewKey) => resolveStaticView(viewKey) === targetRoute)) {
    uni.switchTab({
      url,
      success: release,
      fail: () => navigateTabRouteInStack(url, targetRoute, pages, currentIndex, release),
    })
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
    const normalized = cached.map(normalizeMenu)
    validateMenus(normalized)
    return normalized
  } catch {
    uni.removeStorageSync(cacheKey)
  }
}

function normalizeMenuResponse(response: unknown): AppMenu[] {
  if (Array.isArray(response)) return response.map(normalizeMenu)
  if (!response || typeof response !== 'object')
    throw new Error(t('core.navigation.error.response_object'))
  const record = response as Record<string, unknown>
  const list = record.items ?? record.list ?? record.data
  if (!Array.isArray(list)) throw new Error(t('core.navigation.error.list_missing'))
  return list.map(normalizeMenu)
}

function normalizeMenu(value: unknown): AppMenu {
  if (!value || typeof value !== 'object') throw new Error(t('core.navigation.error.item_object'))
  const item = value as Record<string, unknown>
  const meta = (item.meta ?? {}) as Record<string, unknown>
  const app = (meta.app ?? item.app ?? {}) as Record<string, unknown>
  return {
    id: Number(item.id),
    parentId:
      item.parent_id === undefined && item.parentId === undefined
        ? undefined
        : Number(item.parent_id ?? item.parentId),
    name: String(item.name ?? ''),
    path: String(item.path ?? ''),
    viewKey: String(app.view_key ?? app.viewKey ?? item.view_key ?? item.viewKey ?? ''),
    title: String(meta.title ?? item.title ?? ''),
    access: String(app.access ?? 'AUTHENTICATED') as AppMenuAccess,
    inTabBar: Boolean(app.in_tab_bar ?? app.inTabBar),
    icon: typeof meta.icon === 'string' ? meta.icon : undefined,
    selectedIcon:
      typeof app.selected_icon === 'string'
        ? app.selected_icon
        : typeof app.selectedIcon === 'string'
          ? app.selectedIcon
          : undefined,
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
