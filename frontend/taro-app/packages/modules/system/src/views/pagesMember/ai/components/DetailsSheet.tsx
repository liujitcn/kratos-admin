import { Button, ScrollView, View } from '@tarojs/components'
import type { ReactNode } from 'react'
import { t } from '@liujitcn/kratos-taro-app-core'

/** DetailsSheetProps 为只读资料与消息信息提供统一底部弹层。 */
type DetailsSheetProps = {
  title: string
  children: ReactNode
  footer?: ReactNode
  onClose: () => void
}

/** DetailsSheet 沿用移动端主题和底部安全区，长内容在弹层内滚动。 */
export default function DetailsSheet({ title, children, footer, onClose }: DetailsSheetProps) {
  return (
    <View className='ai-detail-overlay'>
      <View className='ai-detail-mask' onClick={onClose} />
      <View className='ai-detail-sheet'>
        <View className='ai-detail-handle' />
        <View className='ai-detail-head'>
          <View className='ai-detail-title'>{title}</View>
          <Button
            className='ai-detail-close'
            aria-label={t('system.ai.workspace.close')}
            onClick={onClose}
          >
            ×
          </Button>
        </View>
        <ScrollView className='ai-detail-body' scrollY showScrollbar={false}>
          {children}
        </ScrollView>
        {footer ? <View className='ai-detail-footer'>{footer}</View> : null}
      </View>
    </View>
  )
}
