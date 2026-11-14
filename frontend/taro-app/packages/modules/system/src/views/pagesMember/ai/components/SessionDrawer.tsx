import { Button, Input, ScrollView, Text, View } from '@tarojs/components'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { t, useI18n, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import type { AiSession } from '../../../../rpc/base/v1/ai_session'
import { workspaceTimestamp, type AiWorkspaceView } from '../workspace'
import './SessionDrawer.scss'

/** 带只读工作区入口的会话抽屉参数。 */
type SessionDrawerProps = {
  open: boolean
  topPadding: string
  keyword: string
  loading: boolean
  sessions: AiSession[]
  activeSessionId: string
  view: AiWorkspaceView
  pending: number
  onWorkspace: (view: AiWorkspaceView) => void
  onClose: () => void
  onCreate: () => void
  onSelect: (sessionID: string) => void
  onKeywordChange: (value: string) => void
}

/** 按真实会话时间显示今天的时分或历史日期。 */
function formatSessionTime(session: AiSession) {
  const timestamp = workspaceTimestamp(session.updated_at)
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
  useI18n()
  const themeStore = useThemeStore()
  const primaryIconColor = themeStore.colors.aiPrimary
  const mutedIconColor = themeStore.colors.aiTextMuted
  return (
    <>
      {props.open ? <View className='session-mask' onClick={props.onClose} /> : null}
      <View
        className={`session-drawer${props.open ? ' is-open' : ''}`}
        style={{ paddingTop: props.topPadding }}
      >
        <View className='session-workspaces'>
          <Button
            className={`session-workspace${props.view === 'automations' ? ' is-active' : ''}`}
            onClick={() => props.onWorkspace('automations')}
          >
            <UniIcon type='calendar-filled' size={22} color={mutedIconColor} />
            <Text className='session-workspace__label'>{t('system.ai.workspace.automations')}</Text>
            {props.pending ? (
              <Text className='session-workspace__count'>{props.pending}</Text>
            ) : null}
          </Button>
          <Button
            className={`session-workspace${props.view === 'files' ? ' is-active' : ''}`}
            onClick={() => props.onWorkspace('files')}
          >
            <UniIcon type='list' size={22} color={mutedIconColor} />
            <Text className='session-workspace__label'>{t('system.ai.workspace.files')}</Text>
          </Button>
          <Button
            className={`session-workspace${props.view === 'plugins' ? ' is-active' : ''}`}
            onClick={() => props.onWorkspace('plugins')}
          >
            <UniIcon type='settings-filled' size={22} color={mutedIconColor} />
            <Text className='session-workspace__label'>{t('system.ai.workspace.plugins')}</Text>
          </Button>
        </View>
        <View className='session-drawer__head'>
          <View className='session-drawer__title'>{t('system.ai.history')}</View>
          <Button className='session-create' hoverClass='none' onClick={props.onCreate}>
            <UniIcon type='plusempty' size={16} color={primaryIconColor} />
            <Text>{t('common.action.new')}</Text>
          </Button>
        </View>
        <View className='session-search'>
          <UniIcon type='search' size={16} color={mutedIconColor} />
          <Input
            className='session-search-input'
            confirmType='search'
            placeholder={t('system.ai.search_session')}
            placeholderClass='session-search-placeholder'
            value={props.keyword}
            onInput={(event) => props.onKeywordChange(event.detail.value)}
          />
        </View>
        <ScrollView className='session-list' scrollY showScrollbar={false}>
          {props.loading ? (
            <View className='session-empty'>{t('system.ai.load_sessions')}</View>
          ) : null}
          {props.sessions.map((session) => (
            <View
              key={session.id}
              className={`session-item${session.id === props.activeSessionId ? ' is-active' : ''}`}
              onClick={() => props.onSelect(session.id)}
            >
              <View className='session-content'>
                <View className='session-row'>
                  <View className='session-title'>{session.title}</View>
                  <View className='session-time'>{formatSessionTime(session)}</View>
                </View>
                <View className='session-summary'>
                  {session.summary || t('system.ai.no_summary')}
                </View>
              </View>
            </View>
          ))}
          {!props.loading && !props.sessions.length ? (
            <View className='session-empty'>{t('system.ai.no_matching_sessions')}</View>
          ) : null}
        </ScrollView>
      </View>
    </>
  )
}
