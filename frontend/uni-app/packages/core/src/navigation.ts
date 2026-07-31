import { computed, readonly, ref } from 'vue'
import { defBaseMenuService } from './api/base/menu'
import { hasValidToken } from './utils/auth'
import { navigateToLogin } from './utils/navigation'
import { resolveStaticView } from './module'
import { matchLogicalPath, parseLogicalQuery } from './navigation-pattern.mjs'
import { buildMenuTree } from './navigation-tree.mjs'

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
const defaultMenus: AppMenu[] = [
  {
    id: 99901,
    parentId: APP_MENU_ROOT_ID,
    name: 'AppHome',
    path: 'app/home',
    viewKey: 'HOME',
    title: '首页',
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
    title: '我的',
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
    title: '登录',
    access: 'GUEST_ONLY',
    inTabBar: false,
  },
  {
    id: 999010101,
    parentId: 9990101,
    name: 'AppProtocol',
    path: 'app/protocol/:type',
    viewKey: 'PROTOCOL',
    title: '协议详情',
    access: 'PUBLIC',
    inTabBar: false,
  },
  {
    id: 9990901,
    parentId: 99909,
    name: 'AppProfile',
    path: 'app/profile',
    viewKey: 'PROFILE',
    title: '个人信息',
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 9990902,
    parentId: 99909,
    name: 'AppSettings',
    path: 'app/settings',
    viewKey: 'SETTINGS',
    title: '设置',
    access: 'AUTHENTICATED',
    inTabBar: false,
  },
  {
    id: 9990903,
    parentId: 99909,
    name: 'AppAi',
    path: 'app/ai',
    viewKey: 'AI',
    title: 'AI 助手',
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

const menus = ref<AppMenu[]>(defaultMenus)
const ready = ref(false)
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
  menus.value = cached ?? defaultMenus
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
  if (options.replace || resolved.menu.inTabBar) {
    uni.reLaunch({ url })
    return
  }
  uni.navigateTo({ url, fail: () => uni.reLaunch({ url }) })
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
  return hasValidToken() ? AUTHENTICATED_CACHE_KEY : ANONYMOUS_CACHE_KEY
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
  if (!response || typeof response !== 'object') throw new Error('菜单响应不是对象')
  const record = response as Record<string, unknown>
  const list = record.items ?? record.list ?? record.data
  if (!Array.isArray(list)) throw new Error('菜单响应缺少扁平列表')
  return list.map(normalizeMenu)
}

function normalizeMenu(value: unknown): AppMenu {
  if (!value || typeof value !== 'object') throw new Error('菜单项不是对象')
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
      throw new Error('菜单 id、name 或 app/ 路径无效')
    }
    if (!menu.name.startsWith('App')) throw new Error(`菜单名称必须使用 App 前缀：${menu.name}`)
    if (!['PUBLIC', 'GUEST_ONLY', 'AUTHENTICATED'].includes(menu.access)) {
      throw new Error(`菜单访问模式无效：${menu.access}`)
    }
    if (!resolveStaticView(menu.viewKey)) throw new Error(`未注册 viewKey：${menu.viewKey}`)
    if (ids.has(menu.id) || names.has(menu.name) || paths.has(menu.path)) {
      throw new Error('菜单 id、name、path 必须唯一')
    }
    ids.add(menu.id)
    names.add(menu.name)
    paths.add(menu.path)
  }
  const menuMap = new Map(nextMenus.map((menu) => [menu.id, menu]))
  for (const menu of nextMenus) {
    if (menu.parentId === undefined) throw new Error(`菜单缺少父级编号：${menu.name}`)
    if (menu.parentId === APP_MENU_ROOT_ID) {
      if (!menu.inTabBar) throw new Error(`移动端二级菜单必须作为 tab：${menu.name}`)
      continue
    }
    if (menu.inTabBar) throw new Error(`只有移动端二级菜单可以作为 tab：${menu.name}`)
    const visited = new Set<number>([menu.id])
    let parentId = menu.parentId
    while (parentId !== APP_MENU_ROOT_ID) {
      if (visited.has(parentId)) throw new Error(`移动端菜单存在循环父级：${menu.name}`)
      visited.add(parentId)
      const parent = menuMap.get(parentId)
      if (!parent) throw new Error(`移动端菜单父级不存在：${menu.name}`)
      if (parent.parentId === undefined) throw new Error(`移动端菜单父级编号无效：${menu.name}`)
      parentId = parent.parentId
    }
  }
  const tabCount = nextMenus.filter((menu) => menu.parentId === APP_MENU_ROOT_ID).length
  if (tabCount === 1 || tabCount > 5) throw new Error('tabBar 只能配置 0 或 2-5 项')
}
