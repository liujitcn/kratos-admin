import Taro, { useLoad } from '@tarojs/taro'
import { Button, RichText, Text, View } from '@tarojs/components'
import { useState } from 'react'
import { homeTabPage, navigateAppView, t, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { MessageActionType, MessageContentFormat } from '../../../rpc/base/v1/notification'
import { resolveMessageCategoryIcon } from './icons'
import './message.scss'

/** Taro 站内信详情页面。 */
export default function MessageDetailPage() {
  const themeStore = useThemeStore()
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

  /** 将系统通知发送者映射到当前语言。 */
  function resolveSenderName(value?: string) {
    if (value === '系统' || value === '__I18N__:system.notification.sender.system') {
      return t('system.notification.sender.system')
    }
    return value || ''
  }

  /** 执行通知携带的稳定 viewKey 动作。 */
  function openAction() {
    if (!detail || detail.action_type !== MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY) return
    let params: Record<string, string> = {}
    if (detail.action_params) {
      try {
        const value = JSON.parse(detail.action_params) as Record<string, unknown>
        if (value && typeof value === 'object' && !Array.isArray(value)) {
          params = Object.fromEntries(
            Object.entries(value)
              .filter(([, item]) => typeof item === 'string' || typeof item === 'number')
              .map(([key, item]) => [key, String(item)]),
          )
        }
      } catch {
        params = {}
      }
    }
    navigateAppView(detail.action_target, params)
  }

  /** 返回消息列表，页面栈为空时回到首页。 */
  function onBack() {
    void Taro.navigateBack({ fail: () => void Taro.reLaunch({ url: homeTabPage }) })
  }

  return (
    <View
      className={`detail-page ${themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'}`}
    >
      <View className={`k-page-nav ${process.env.TARO_ENV === 'weapp' ? 'k-page-nav--weapp' : ''}`}>
        <View className='k-page-nav__back' aria-label={t('common.action.back')} onClick={onBack} />
        <View className='k-page-nav__title'>{t('system.notification.title')}</View>
        <View className='k-page-nav__side' aria-hidden />
      </View>
      {detail ? (
        <>
          <Text className='detail-title'>{detail.title}</Text>
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
            <Text>{resolveSenderName(detail.sender_name)}</Text>
            <Text>{detail.received_at}</Text>
          </View>
          {detail.content_format === MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT ? (
            <RichText className='detail-content-rich' nodes={detail.content} />
          ) : (
            <Text className='detail-content'>{detail.content}</Text>
          )}
          {detail.action_type === MessageActionType.MESSAGE_ACTION_TYPE_VIEW_KEY && (
            <Button className='detail-action' onClick={openAction}>
              {t('common.action.view')}
            </Button>
          )}
        </>
      ) : (
        <View className='detail-loading'>{t('core.status.loading')}</View>
      )}
    </View>
  )
}
