import { Image, Text, View } from '@tarojs/components'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { APP_MENU_ROOT_ID, navigateAppRoute, useAppMenuBadges, useAppNavigation } from '../navigation'
import { useI18n } from '../locales'
import { useThemeStore } from '../stores'
import { resolveRootMenuId } from '../navigation-tree.mjs'
import './KratosTabBar.scss'

/** 自绘 TabBar 参数。 */
export interface KratosTabBarProps {
  route: string
}

/** 设计稿导航坞线稿图标（viewKey → SVG path）。 */
const TAB_ICON_PATHS: Record<string, string> = {
  HOME: 'M3 11 12 3l9 8M5 10v10h14V10',
  APP_ANALYTIC: 'M4 20V10M10 20V4M16 20v-7M21 20H3',
  APP_REPORT: 'M6 4h9l5 5v11H6zM14 4v5h5',
  MESSAGE_INBOX: 'M4 5h16v12H8l-4 4z',
  PROFILE_HOME: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 21c0-4 3.5-6 8-6s8 2 8 6',
}

/** 按设计稿线稿生成随主题与激活态取色的图标地址，未知 viewKey 返回空串走后端图标兜底。 */
const tabIconSrc = (viewKey: string, active: boolean, theme: string): string => {
  const path = TAB_ICON_PATHS[viewKey]
  if (!path) return ''
  const color = active
    ? theme === 'light'
      ? '#039e8c'
      : '#28f0d4'
    : theme === 'light'
      ? '#9aabbb'
      : '#4e6378'
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="${path}"/></svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** 根据动态菜单显示的自绘 TabBar。 */
export function KratosTabBar({ route }: KratosTabBarProps) {
  const menus = useAppNavigation((state) => state.menus)
  const tabBar = useAppNavigation((state) => state.tabBar)
  const { t } = useI18n()
  const themeStore = useThemeStore()

  /** 按稳定 viewKey 优先取界面语言标签，未配置的菜单回退后端标题。 */
  const tabBarTitleKeys: Record<string, string> = {
    HOME: 'core.navigation.home',
    APP_ANALYTIC: 'core.navigation.analytics',
    APP_REPORT: 'core.navigation.report',
    MESSAGE_INBOX: 'core.navigation.message',
    PROFILE_HOME: 'core.navigation.my',
  }
  const tabBarTitle = (item: { viewKey: string; title: string }): string =>
    (tabBarTitleKeys[item.viewKey] ? t(tabBarTitleKeys[item.viewKey]) : '') || item.title
  const messageBadge = useAppMenuBadges((state) => state.badges.MESSAGE_INBOX ?? 0)
  const routeMenu = menus.find((menu) => resolveStaticView(menu.viewKey) === route)
  const tabMenuId = routeMenu
    ? resolveRootMenuId(menus, routeMenu.id, APP_MENU_ROOT_ID)
    : undefined
  const activeMenu = tabBar.find((menu) => menu.id === tabMenuId)
  if (!activeMenu || !tabBar.length) return null

  return (
    <View
      className={`kratos-tab-bar ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      {tabBar.map((item) => {
        const active = activeMenu.id === item.id
        const designIcon = tabIconSrc(item.viewKey, active, themeStore.theme)
        const icon = designIcon || resolveModuleIcon(active && item.selectedIcon ? item.selectedIcon : item.icon)
        return (
          <View
            key={item.id}
            className={`kratos-tab-bar__item${active ? ' kratos-tab-bar__item--active' : ''}`}
            onClick={() => navigateAppRoute(item.path)}
          >
            {icon ? (
              <Image className='kratos-tab-bar__icon' src={icon} mode='aspectFit' />
            ) : null}
            {!icon && item.viewKey === 'MESSAGE_INBOX' ? (
              <View className={`kratos-tab-bar__message-icon${active ? ' kratos-tab-bar__message-icon--active' : ''}`} />
            ) : null}
            {item.viewKey === 'MESSAGE_INBOX' && messageBadge ? (
              <View className='kratos-tab-bar__badge' aria-hidden='true' />
            ) : null}
            <Text className={active ? 'kratos-tab-bar__text--active' : ''}>{tabBarTitle(item)}</Text>
          </View>
        )
      })}
    </View>
  )
}

export default KratosTabBar
