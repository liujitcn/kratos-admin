import { useDidShow } from '@tarojs/taro'
import { Button, Input, Text, View } from '@tarojs/components'
import { useState } from 'react'
import { navigateAppRoute, t } from '@liujitcn/kratos-taro-app-core'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { refreshNotificationSummary } from '../../../notification'
import './message.scss'

/** Taro 站内信收件箱页面。 */
export default function MessageInboxPage() {
  const [items, setItems] = useState<Notification[]>([])
  const [cursorId, setCursorId] = useState(0)
  const [loading, setLoading] = useState(false)
  const [finished, setFinished] = useState(false)
  const [unreadTotal, setUnreadTotal] = useState(0)
  const [selectedView, setSelectedView] = useState(NotificationView.NOTIFICATION_VIEW_INBOX)
  const [categoryInput, setCategoryInput] = useState('')
  const [categoryId, setCategoryId] = useState<number>()
  const viewOptions = [
    { value: NotificationView.NOTIFICATION_VIEW_INBOX, key: 'system.notification.view.inbox' },
    { value: NotificationView.NOTIFICATION_VIEW_UNREAD, key: 'system.notification.view.unread' },
    {
      value: NotificationView.NOTIFICATION_VIEW_ARCHIVED,
      key: 'system.notification.view.archived',
    },
  ]

  useDidShow(() => {
    void refresh().catch(() => undefined)
    void refreshNotificationSummary().then(setUnreadTotal).catch(() => undefined)
  })

  /** 刷新站内信列表。 */
  async function refresh(view = selectedView, category = categoryId) {
    setLoading(true)
    try {
      const result = await defNotificationService.PageNotification({
        view,
        category_id: category,
        priority: undefined,
        cursor_id: 0,
        page_num: 1,
        page_size: 20,
      })
      setItems(result.notifications)
      setCursorId(result.next_cursor_id)
      setFinished(!result.has_more)
    } finally {
      setLoading(false)
    }
  }

  /** 加载下一页。 */
  async function loadMore() {
    if (loading || finished) return
    setLoading(true)
    try {
      const result = await defNotificationService.PageNotification({
        view: selectedView,
        category_id: categoryId,
        priority: undefined,
        cursor_id: cursorId,
        page_num: 1,
        page_size: 20,
      })
      const next = [...items, ...result.notifications]
      setItems(next)
      setCursorId(result.next_cursor_id)
      setFinished(!result.has_more)
    } finally {
      setLoading(false)
    }
  }

  /** 切换收件箱视图。 */
  function changeView(view: NotificationView) {
    setSelectedView(view)
    void refresh(view, categoryId).catch(() => undefined)
  }

  /** 应用分类筛选。 */
  function applyCategoryFilter() {
    const value = Number(categoryInput)
    const next = Number.isInteger(value) && value > 0 ? value : undefined
    setCategoryId(next)
    void refresh(selectedView, next).catch(() => undefined)
  }

  /** 标记当前水位线之前的消息为已读。 */
  async function markAllRead() {
    const beforeDeliveryId = items.reduce((max, item) => Math.max(max, item.id), 0)
    if (beforeDeliveryId <= 0) return
    await defNotificationService.MarkAllNotificationRead({ before_delivery_id: beforeDeliveryId })
    await refresh()
    setUnreadTotal(await refreshNotificationSummary())
  }

  /** 归档或恢复一条消息。 */
  async function toggleArchive(item: Notification) {
    if (item.archived_at) await defNotificationService.RestoreNotification({ id: item.id })
    else if (item.allow_archive) await defNotificationService.ArchiveNotification({ id: item.id })
    await refresh()
    setUnreadTotal(await refreshNotificationSummary())
  }

  /** 删除一条消息。 */
  async function deleteItem(item: Notification) {
    if (!item.allow_delete) return
    await defNotificationService.DeleteNotification({ id: item.id })
    await refresh()
    setUnreadTotal(await refreshNotificationSummary())
  }

  /** 打开站内信详情。 */
  async function openDetail(item: Notification) {
    if (!item.read_at) await defNotificationService.MarkNotificationRead({ ids: [item.id] })
    navigateAppRoute(`app/message/detail?id=${item.id}`)
  }

  return (
    <View className='message-page'>
      <View className='message-header'>
        <Text className='message-title'>{t('system.notification.title')}</Text>
        {unreadTotal > 0 && (
          <Text className='message-unread'>{unreadTotal > 99 ? '99+' : unreadTotal}</Text>
        )}
      </View>
      <View className='message-controls'>
        <View className='message-tabs'>
          {viewOptions.map((option) => (
            <Button
              key={option.value}
              className={selectedView === option.value ? 'message-tab active' : 'message-tab'}
              onClick={() => changeView(option.value)}
            >
              {t(option.key)}
            </Button>
          ))}
        </View>
        <View className='message-filter'>
          <Input
            type='number'
            value={categoryInput}
            placeholder={t('system.notification.category_filter')}
            onInput={(event) => setCategoryInput(event.detail.value)}
            onConfirm={applyCategoryFilter}
          />
          <Button size='mini' onClick={applyCategoryFilter}>
            {t('common.action.confirm')}
          </Button>
          {selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && unreadTotal > 0 && (
            <Button size='mini' onClick={() => void markAllRead().catch(() => undefined)}>
              {t('system.notification.mark_all_read')}
            </Button>
          )}
        </View>
      </View>
      {items.map((item) => (
        <View
          className='message-item'
          key={item.id}
          onClick={() => void openDetail(item).catch(() => undefined)}
        >
          <View className='message-item__main'>
            <Text className={`message-item__title ${item.read_at ? '' : 'unread'}`}>
              {item.title}
            </Text>
            <Text className='message-item__time'>{item.received_at}</Text>
          </View>
          <Text className='message-item__category'>{item.category_name}</Text>
          <View className='message-item__actions'>
            {selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && item.allow_archive && (
              <Button
                size='mini'
                onClick={(event) => {
                  event.stopPropagation()
                  void toggleArchive(item).catch(() => undefined)
                }}
              >
                {t('system.notification.action.archive')}
              </Button>
            )}
            {selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED && (
              <Button
                size='mini'
                onClick={(event) => {
                  event.stopPropagation()
                  void toggleArchive(item).catch(() => undefined)
                }}
              >
                {t('system.notification.action.restore')}
              </Button>
            )}
            {item.allow_delete && (
              <Button
                size='mini'
                onClick={(event) => {
                  event.stopPropagation()
                  void deleteItem(item).catch(() => undefined)
                }}
              >
                {t('system.notification.action.delete')}
              </Button>
            )}
          </View>
        </View>
      ))}
      {!finished && (
        <Button
          className='load-more'
          loading={loading}
          onClick={() => void loadMore().catch(() => undefined)}
        >
          {t('common.action.load_more')}
        </Button>
      )}
      {finished && items.length === 0 && (
        <View className='empty'>{t('system.notification.empty')}</View>
      )}
    </View>
  )
}
