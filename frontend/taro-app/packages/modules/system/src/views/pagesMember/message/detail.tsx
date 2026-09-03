import { useLoad } from '@tarojs/taro'
import { Button, RichText, Text, View } from '@tarojs/components'
import { useState } from 'react'
import { navigateAppView, t } from '@liujitcn/kratos-taro-app-core'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { MessageActionType, MessageContentFormat } from '../../../rpc/base/v1/notification'
import './message.scss'

/** Taro 站内信详情页面。 */
export default function MessageDetailPage() {
  const [detail, setDetail] = useState<Notification>()

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

  /** 执行通知携带的稳定 viewKey 动作。 */
  function openAction() {
    if (!detail || detail.action_type !== MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY) return
    let params: Record<string, string> = {}
    if (detail.action_params) {
      try {
        const value = JSON.parse(detail.action_params) as Record<string, unknown>
        if (value && typeof value === 'object' && !Array.isArray(value)) {
          params = Object.fromEntries(Object.entries(value).filter(([, item]) => typeof item === 'string' || typeof item === 'number').map(([key, item]) => [key, String(item)]))
        }
      } catch {
        params = {}
      }
    }
    navigateAppView(detail.action_target, params)
  }

  if (!detail) return <View className='empty'>{t('common.status.loading')}</View>
  return (
    <View className='detail-page'>
      <Text className='detail-title'>{detail.title}</Text>
      <View className='detail-meta'>
        <Text>{detail.category_name}</Text><Text>{detail.sender_name}</Text><Text>{detail.received_at}</Text>
      </View>
      {detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT
        ? <RichText className='detail-content-rich' nodes={detail.content} />
        : <Text className='detail-content'>{detail.content}</Text>}
      {detail.action_type === MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY && <Button className='detail-action' onClick={openAction}>{t('common.action.view')}</Button>}
    </View>
  )
}
