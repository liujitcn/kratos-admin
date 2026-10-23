import { Image, Text, View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import type { CSSProperties } from 'react'
import { useI18n } from '../locales'
import { useThemeStore } from '../stores'
import { getMpCapsuleRightGap } from '../utils'
import { homeTabPage } from '../utils/navigation'
import './KratosNavBar.scss'

/** KratosNavBarAction 导航栏操作按钮，统一渲染尺寸、图标与无障碍文案。 */
export interface KratosNavBarAction {
  /** 按钮名称，同时用作可见文字与无障碍文案。 */
  label: string
  /** 内置 settings、read-all、history 图标名称，或自定义图片地址。 */
  icon?: string
  /** 显示方式；默认有图标时显示图标，否则显示文字。 */
  display?: 'icon' | 'text' | 'both'
  /** 点击按钮后执行的业务操作。 */
  onClick: () => void
}

/** KratosNavBarProps 普通页面与 tabBar 页面共用的顶部导航属性。 */
export interface KratosNavBarProps {
  /** 居中显示的页面标题。 */
  title: string
  /** 普通页面默认显示返回按钮，tabBar 页面默认隐藏返回按钮。 */
  variant?: 'page' | 'tab'
  /** 显式覆盖返回按钮的显隐。 */
  back?: boolean
  /** 可选的页面操作按钮。 */
  action?: KratosNavBarAction
  /** 自定义返回操作；不传时返回上一页，失败后回到首页。 */
  onBack?: () => void
}

const iconPaths: Record<string, string> = {
  settings:
    '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8 2 2 0 1 1-2.8 2.8 1.6 1.6 0 0 0-2.7 1.1 2 2 0 1 1-4 0 1.6 1.6 0 0 0-2.6-1.1 2 2 0 1 1-2.8-2.8A1.6 1.6 0 0 0 3 13.9a2 2 0 1 1 0-4 1.6 1.6 0 0 0 1.5-2.7 2 2 0 1 1 2.8-2.8A1.6 1.6 0 0 0 10 4.6a2 2 0 1 1 4 0 1.6 1.6 0 0 0 2.7.6 2 2 0 1 1 2.8 2.8 1.6 1.6 0 0 0-.1 2.6a2.2 2.2 0 1 1 0 4.4Z"/>',
  'read-all': '<path d="m3 12 4 4L17 6m-5 10 9-9"/>',
  history: '<path d="M4 6h16M4 12h16M4 18h16"/>',
}

/** KratosNavBar 统一页面标题、返回与操作按钮，并处理安全区和微信胶囊避让。 */
export default function KratosNavBar({ title, variant = 'page', back, action, onBack }: KratosNavBarProps) {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  const capsuleGap = getMpCapsuleRightGap()
  const navStyle = { '--kratos-nav-bar-capsule-gap': `${capsuleGap}px` } as CSSProperties
  const showBack = back ?? variant === 'page'
  const iconPath = action?.icon ? iconPaths[action.icon] : undefined
  const color = themeStore.colors.text
  const actionIcon = iconPath
    ? `data:image/svg+xml;charset=utf-8,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${iconPath}</svg>`)}`
    : action?.icon
  const actionDisplay = action?.display ?? (actionIcon ? 'icon' : 'text')
  const showActionIcon = !!actionIcon && actionDisplay !== 'text'
  const showActionLabel = !actionIcon || actionDisplay !== 'icon'

  /** goBack 优先执行页面返回操作，否则返回上一页并在失败时回到首页。 */
  function goBack() {
    if (onBack) {
      onBack()
      return
    }
    void Taro.navigateBack().catch(() => Taro.reLaunch({ url: homeTabPage }))
  }

  return (
    <View
      className={`kratos-nav-bar${variant === 'tab' ? ' kratos-nav-bar--tab' : ''}${process.env.TARO_ENV === 'weapp' ? ' kratos-nav-bar--weapp' : ''}`}
      style={navStyle}
    >
      <View className='kratos-nav-bar__row'>
        {showBack && (
          <View className='kratos-nav-bar__button kratos-nav-bar__back' aria-label={t('common.action.back')} onClick={goBack} />
        )}
        <View className='kratos-nav-bar__title'>{title}</View>
        {action && (
          <View className={`kratos-nav-bar__button kratos-nav-bar__action${showActionLabel ? ' kratos-nav-bar__action--label' : ''}`} aria-label={action.label} onClick={action.onClick}>
            {showActionIcon && <Image className='kratos-nav-bar__icon' src={actionIcon} mode='aspectFit' aria-hidden='true' />}
            {showActionLabel && <Text className='kratos-nav-bar__label'>{action.label}</Text>}
          </View>
        )}
      </View>
    </View>
  )
}
