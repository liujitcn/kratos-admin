import { Text, View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import type { ReactNode } from 'react'
import { useI18n } from '../locales'
import { homeTabPage } from '../utils/navigation'
import './KratosPageFrame.scss'

/** KratosPageFrameProps 页面框架属性。 */
export interface KratosPageFrameProps {
  children: ReactNode
  navigationStyle: string
  navigationBarTitleText: string
  navigationBarBackgroundColor: string
  navigationBarTextStyle: string
}

/** KratosPageFrame 补齐 Taro H5 不渲染原生导航栏的差异。 */
export function KratosPageFrame({
  children,
  navigationStyle,
  navigationBarTitleText,
  navigationBarBackgroundColor,
  navigationBarTextStyle,
}: KratosPageFrameProps) {
  const { t } = useI18n()
  if (process.env.TARO_ENV !== 'h5' || navigationStyle === 'custom') return children

  const color = navigationBarTextStyle === 'white' ? '#fff' : '#000'
  const goBack = async () => {
    try {
      await Taro.navigateBack({ delta: 1 })
    } catch {
      await Taro.reLaunch({ url: homeTabPage })
    }
  }

  return (
    <>
      <View
        className='kratos-page-frame__navigation'
        style={{ backgroundColor: navigationBarBackgroundColor, color }}
      >
        <View className='kratos-page-frame__back' aria-label={t('common.action.back')} onClick={() => void goBack()}>
          <View className='kratos-page-frame__back-icon' />
        </View>
        <Text className='kratos-page-frame__title'>{navigationBarTitleText}</Text>
      </View>
      {children}
    </>
  )
}

export default KratosPageFrame
