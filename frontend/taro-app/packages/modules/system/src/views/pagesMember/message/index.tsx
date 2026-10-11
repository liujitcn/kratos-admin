import type { CSSProperties } from 'react'
import { useDidShow } from '@tarojs/taro'
import { Button, Text, View } from '@tarojs/components'
import { useRef, useState } from 'react'
import { getMpCapsuleRightGap, navigateAppView, t, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { defNotificationService } from '../../../api/base/v1/notification'
import type { Notification, NotificationCategory } from '../../../rpc/base/v1/notification'
import { NotificationView } from '../../../rpc/base/v1/notification'
import { refreshNotificationSummary } from '../../../notification'
import { resolveMessageCategoryIcon } from './icons'
import './message.scss'

/** Taro 站内信收件箱页面。 */
export default function MessageInboxPage() {
  const themeStore = useThemeStore()
  /** MP-WEIXIN 原生胶囊按钮右侧避让宽度，写入 --mp-capsule-gap 供 .k-top 预留空间。 */
  const capsuleGap = getMpCapsuleRightGap()
  const topNavStyle = { '--mp-capsule-gap': `${capsuleGap}px` } as CSSProperties
  const [items, setItems] = useState<Notification[]>([])
  const [cursorId, setCursorId] = useState(0)
  const [loading, setLoading] = useState(false)
  const [finished, setFinished] = useState(false)
  const [unreadTotal, setUnreadTotal] = useState(0)
  const [selectedView, setSelectedView] = useState(NotificationView.NOTIFICATION_VIEW_INBOX)
  const [categoryId, setCategoryId] = useState<number>()
  const [categoryOptions, setCategoryOptions] = useState<
    NotificationCategory[]
  >([])
  const requestSeqRef = useRef(0)
  const loadingRef = useRef(false)
  const finishedRef = useRef(false)
  const viewOptions = [
    { value: NotificationView.NOTIFICATION_VIEW_INBOX, key: 'system.notification.view.inbox' },
    { value: NotificationView.NOTIFICATION_VIEW_UNREAD, key: 'system.notification.view.unread' },
    {
      value: NotificationView.NOTIFICATION_VIEW_ARCHIVED,
      key: 'system.notification.view.archived',
    },
  ]

  useDidShow(() => {
    void loadCategories().catch(() => undefined)
    void refresh().catch(() => undefined)
    void refreshNotificationSummary()
      .then(setUnreadTotal)
      .catch(() => undefined)
  })

  /** 加载管理端维护的消息分类。 */
  async function loadCategories() {
    const result = await defNotificationService.ListNotificationCategories({})
    setCategoryOptions(result.categories ?? [])
  }

  /** 刷新站内信列表。 */
  async function refresh(view = selectedView, category = categoryId) {
    const seq = ++requestSeqRef.current
    loadingRef.current = true
    finishedRef.current = false
    setItems([])
    setCursorId(0)
    setFinished(false)
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
      if (seq !== requestSeqRef.current) return
      setItems(result.notifications)
      setCursorId(result.next_cursor_id)
      finishedRef.current = !result.has_more
      setFinished(!result.has_more)
    } finally {
      if (seq === requestSeqRef.current) {
        loadingRef.current = false
        setLoading(false)
      }
    }
  }

  /** 加载下一页。 */
  async function loadMore() {
    if (loadingRef.current || finishedRef.current) return
    const seq = requestSeqRef.current
    loadingRef.current = true
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
      if (seq !== requestSeqRef.current) return
      setItems((current) => [...current, ...result.notifications])
      setCursorId(result.next_cursor_id)
      finishedRef.current = !result.has_more
      setFinished(!result.has_more)
    } finally {
      if (seq === requestSeqRef.current) {
        loadingRef.current = false
        setLoading(false)
      }
    }
  }

  /** 切换收件箱视图。 */
  function changeView(view: NotificationView) {
    setSelectedView(view)
    void refresh(view, categoryId).catch(() => undefined)
  }

  /** 切换消息分类。 */
  function changeCategory(id?: number) {
    setCategoryId(id)
    void refresh(selectedView, id).catch(() => undefined)
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
    navigateAppView('MESSAGE_DETAIL', { id: String(item.id) })
  }

  return (
    <View
      className={`message-page ${
        themeStore.theme === 'light' ? 'theme-light' : 'theme-dark'
      }`}
    >
      <View className='k-top message-header' style={topNavStyle}>
        <View className='message-header__title'>
          <Text className='k-top__t2'>{t('system.notification.title')}</Text>
          {unreadTotal > 0 && <View className='message-unread-dot' aria-hidden='true' />}
        </View>
        {selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && unreadTotal > 0 && (
          <View
            className='k-ch k-ch--pri'
            onClick={() => void markAllRead().catch(() => undefined)}
          >
            {t('system.notification.mark_all_read')}
          </View>
        )}
      </View>
      <View className='k-chips'>
        <View
          className={`k-ch${categoryId === undefined ? ' k-ch--pri' : ''}`}
          onClick={() => changeCategory()}
        >
          {t('system.notification.view.inbox')}
        </View>
        {categoryOptions.map((category) => (
          <View
            key={category.id}
            className={`k-ch${categoryId === category.id ? ' k-ch--pri' : ''}`}
            onClick={() => changeCategory(category.id)}
          >
            <UniIcon
              type={resolveMessageCategoryIcon(category.icon)}
              size={16}
              color={category.color || '#64748b'}
            />
            {category.name}
          </View>
        ))}
      </View>
      <View className='k-pills message-tabs'>
        {viewOptions.map((option) => (
          <View
            key={option.value}
            className={`k-pills__item message-tab${
              selectedView === option.value ? ' k-pills__item--on' : ''
            }`}
            onClick={() => changeView(option.value)}
          >
            {t(option.key)}
          </View>
        ))}
      </View>
      <View className='k-msg-list'>
        {items.map((item) => (
          <View
            className={`k-msg-item${item.read_at ? '' : ' k-msg-item--unread'}`}
            key={item.id}
            onClick={() => void openDetail(item).catch(() => undefined)}
          >
            <View className='k-msg-item__head'>
              <View className='k-msg-item__icon'>
                <UniIcon
                  type={resolveMessageCategoryIcon(item.category_icon)}
                  size={18}
                  color={item.category_color || '#64748b'}
                />
              </View>
              <View className='message-item__body'>
                <Text className='k-msg-item__title'>{item.title}</Text>
                <Text className='k-msg-item__meta'>{item.category_name}</Text>
              </View>
              <Text className='k-msg-item__time'>{item.received_at}</Text>
            </View>
            {(selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED ||
              item.allow_archive ||
              item.allow_delete) && (
              <View className='k-msg-item__actions'>
                {selectedView !== NotificationView.NOTIFICATION_VIEW_ARCHIVED && item.allow_archive && (
                  <Text
                    className='k-msg-item__action'
                    onClick={(event) => {
                      event.stopPropagation()
                      void toggleArchive(item).catch(() => undefined)
                    }}
                  >
                    {t('system.notification.action.archive')}
                  </Text>
                )}
                {selectedView === NotificationView.NOTIFICATION_VIEW_ARCHIVED && (
                  <Text
                    className='k-msg-item__action'
                    onClick={(event) => {
                      event.stopPropagation()
                      void toggleArchive(item).catch(() => undefined)
                    }}
                  >
                    {t('system.notification.action.restore')}
                  </Text>
                )}
                {item.allow_delete && (
                  <Text
                    className='k-msg-item__action'
                    onClick={(event) => {
                      event.stopPropagation()
                      void deleteItem(item).catch(() => undefined)
                    }}
                  >
                    {t('system.notification.action.delete')}
                  </Text>
                )}
              </View>
            )}
          </View>
        ))}
      </View>
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
