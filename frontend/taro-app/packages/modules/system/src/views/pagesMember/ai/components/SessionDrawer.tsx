import { Button, Input, ScrollView, Text, View } from '@tarojs/components'
import { More, Plus, Search } from '@liujitcn/kratos-taro-app-ui'
import type { AiSession } from '../../../../rpc/base/v1/ai_session'
import './session-drawer.scss'

type SessionDrawerProps = {
  open: boolean
  topPadding: string
  keyword: string
  loading: boolean
  sessions: AiSession[]
  activeSessionId: string
  onClose: () => void
  onCreate: () => void
  onSelect: (sessionID: string) => void
  onAction: (session: AiSession) => void
  onKeywordChange: (value: string) => void
}

function formatSessionTime(session: AiSession) {
  const seconds = Number(session.updated_at?.seconds ?? 0)
  const nanos = Number(session.updated_at?.nanos ?? 0)
  const timestamp = seconds * 1000 + Math.floor(nanos / 1_000_000)
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return isToday ? `${hour}:${minute}` : `${month}-${day}`
}

/** AI 助手历史会话抽屉。 */
export default function SessionDrawer(props: SessionDrawerProps) {
  return (
    <>
      {props.open ? <View className='session-mask' onClick={props.onClose} /> : null}
      <View
        className={`session-drawer${props.open ? ' is-open' : ''}`}
        style={{ paddingTop: props.topPadding }}
      >
        <View className='session-drawer__head'>
          <View className='session-drawer__title'>历史会话</View>
          <Button className='session-create' hoverClass='none' onClick={props.onCreate}>
            <Plus size={16} color='#27ba9b' />
            <Text>新建</Text>
          </Button>
        </View>
        <View className='session-search'>
          <Search size={16} color='#898b94' />
          <Input
            className='session-search-input'
            confirmType='search'
            placeholder='搜索会话'
            placeholderClass='session-search-placeholder'
            value={props.keyword}
            onInput={(event) => props.onKeywordChange(event.detail.value)}
          />
        </View>
        <ScrollView className='session-list' scrollY showScrollbar={false}>
          {props.loading ? <View className='session-empty'>正在加载会话...</View> : null}
          {props.sessions.map((session) => (
            <View
              key={session.id}
              className={`session-item${session.id === props.activeSessionId ? ' is-active' : ''}`}
              onClick={() => props.onSelect(session.id)}
              onLongPress={() => props.onAction(session)}
            >
              <View className='session-content'>
                <View className='session-row'>
                  <View className='session-title'>{session.title}</View>
                  <View className='session-time'>{formatSessionTime(session)}</View>
                </View>
                <View className='session-summary'>{session.summary || '暂无摘要'}</View>
              </View>
              <Button
                className='session-more'
                hoverClass='none'
                onClick={(event) => {
                  event.stopPropagation()
                  props.onAction(session)
                }}
              >
                <More size={22} color='#9ca3af' />
              </Button>
            </View>
          ))}
          {!props.loading && !props.sessions.length ? (
            <View className='session-empty'>没有匹配的会话</View>
          ) : null}
        </ScrollView>
      </View>
    </>
  )
}
