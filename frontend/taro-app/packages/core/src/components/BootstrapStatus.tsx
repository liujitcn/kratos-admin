import { Image, Text, View } from '@tarojs/components'
import defaultLogo from '@liujitcn/kratos-taro-app-core/static/images/logo_icon.png'
import './BootstrapStatus.scss'
import { useI18n } from '../locales'
import { useThemeStore } from '../stores'

/** 启动状态组件参数。 */
export interface BootstrapStatusProps {
  title?: string
  detail?: string
}

/** 全屏启动状态。 */
export function BootstrapStatus({ title, detail }: BootstrapStatusProps) {
  const { t } = useI18n()
  const themeStore = useThemeStore()
  return (
    <View className={`bootstrap-status ${themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'}`}>
      <Image
        className='bootstrap-status__logo'
        src={defaultLogo}
        mode='aspectFit'
      />
      <Text className='bootstrap-status__title'>{title || t('core.status.loading')}</Text>
      {detail ? <Text className='bootstrap-status__detail'>{detail}</Text> : null}
    </View>
  )
}

export default BootstrapStatus
