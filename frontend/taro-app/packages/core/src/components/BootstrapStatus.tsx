import { Image, Text, View } from '@tarojs/components'
import './BootstrapStatus.scss'

/** 启动状态组件参数。 */
export interface BootstrapStatusProps {
  title?: string
  detail?: string
}

/** 全屏启动状态。 */
export function BootstrapStatus({ title, detail }: BootstrapStatusProps) {
  return (
    <View className='bootstrap-status'>
      <Image
        className='bootstrap-status__logo'
        src='/static/images/logo_icon.png'
        mode='aspectFit'
      />
      <Text className='bootstrap-status__title'>{title || '正在加载'}</Text>
      {detail ? <Text className='bootstrap-status__detail'>{detail}</Text> : null}
    </View>
  )
}

export default BootstrapStatus
