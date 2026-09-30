import { Image, Text, View } from '@tarojs/components'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { APP_MENU_ROOT_ID, navigateAppRoute, useAppMenuBadges, useAppNavigation } from '../navigation'
import { useI18n } from '../locales'
import { resolveRootMenuId } from '../navigation-tree.mjs'
import './KratosTabBar.scss'

/** 自绘 TabBar 参数。 */
export interface KratosTabBarProps {
  route: string
}

/** 根据动态菜单显示的自绘 TabBar。 */
export function KratosTabBar({ route }: KratosTabBarProps) {
  const menus = useAppNavigation((state) => state.menus)
  const tabBar = useAppNavigation((state) => state.tabBar)
  const { t } = useI18n()

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
    <View className='kratos-tab-bar'>
      {tabBar.map((item) => {
        const active = activeMenu.id === item.id
        const icon = resolveModuleIcon(active && item.selectedIcon ? item.selectedIcon : item.icon)
        return (
          <View
            key={item.id}
            className='kratos-tab-bar__item'
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
