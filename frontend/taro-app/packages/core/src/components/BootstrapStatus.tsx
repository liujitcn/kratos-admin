import { Image, Text, View } from '@tarojs/components'
import './BootstrapStatus.scss'
import { useI18n } from '../locales'

/** 启动状态组件参数。 */
export interface BootstrapStatusProps {
  title?: string
  detail?: string
}

/** 全屏启动状态。 */
export function BootstrapStatus({ title, detail }: BootstrapStatusProps) {
  const { t } = useI18n()
  return (
    <View className='bootstrap-status'>
      <Image
        className='bootstrap-status__logo'
        src='/static/images/logo_icon.png'
        mode='aspectFit'
      />
      <Text className='bootstrap-status__title'>{title || t('core.status.loading')}</Text>
      {detail ? <Text className='bootstrap-status__detail'>{detail}</Text> : null}
    </View>
  )
}

export default BootstrapStatus
