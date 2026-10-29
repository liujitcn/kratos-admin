import { useLoad } from '@tarojs/taro'
import { RichText, Text, View } from '@tarojs/components'
import { useState } from 'react'
import { KratosNavBar, t, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { NotificationDetail } from '../../../rpc/base/v1/notification'
import { MessageContentFormat } from '../../../rpc/base/v1/notification'
import { resolvePersistedMessage } from '../../../utils/persisted-message'
import { resolveMessageCategoryIcon } from './icons'
import './message.scss'

/** MessageDetailPage 展示站内信详情与通知操作。 */
export default function MessageDetailPage() {
  const themeStore = useThemeStore()
  const [detail, setDetail] = useState<NotificationDetail>()

  useLoad((options) => {
    const id = Number(options?.id)
    if (id > 0) void loadDetail(id).catch(() => undefined)
  })

  /** 加载站内信详情。 */
  async function loadDetail(id: number) {
    const value = await defNotificationService.GetNotification({ id })
    setDetail(value)
    if (!value.read_at) await defNotificationService.MarkNotificationRead({ ids: [id] })
  }

  return (
    <View
      className={`detail-page ${themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'}`}
    >
      <KratosNavBar title={t('system.notification.title')} />
      {detail ? (
        <>
          <Text className='detail-title'>{resolvePersistedMessage(detail.title)}</Text>
          <View className='detail-meta'>
            <View className='detail-category' style={{ color: detail.category_color || undefined }}>
              <UniIcon
                className='detail-category__icon'
                type={resolveMessageCategoryIcon(detail.category_icon)}
                size={18}
                color={detail.category_color || '#64748b'}
              />
              <Text>{detail.category_name}</Text>
            </View>
            <Text>{detail.received_at}</Text>
          </View>
          {detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT ? (
            <RichText className='detail-content-rich' nodes={resolvePersistedMessage(detail.content)} />
          ) : (
            <Text className='detail-content'>{resolvePersistedMessage(detail.content)}</Text>
          )}
        </>
      ) : (
        <View className='detail-loading'>{t('core.status.loading')}</View>
      )}
    </View>
  )
}
