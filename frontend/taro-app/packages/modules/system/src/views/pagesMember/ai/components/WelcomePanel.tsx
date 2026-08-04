import { Button, Text, View } from '@tarojs/components'
import { ArrowRight, Refresh } from '@liujitcn/kratos-taro-app-ui'
import { t, useI18n } from '@liujitcn/kratos-taro-app-core'
import type { AiShortcut } from '../../../../rpc/base/v1/ai_tool'
import './welcome-panel.scss'

type WelcomePanelProps = {
  greetingMessage: string
  loading: boolean
  shortcuts: AiShortcut[]
  canRefresh: boolean
  onRefresh: () => void
  onShortcutTap: (shortcut: AiShortcut) => void
}

/** AI 助手空会话欢迎面板。 */
export default function WelcomePanel(props: WelcomePanelProps) {
  useI18n()
  return (
    <View className='welcome-panel'>
      <View className='welcome-row is-hello'>
        <View className='ai-avatar'>
          <View className='ai-avatar__halo' />
          <View className='ai-avatar__hair-back' />
          <View className='ai-avatar__face'>
            <View className='ai-avatar__bang' />
            <View className='ai-avatar__eyes'><View /><View /></View>
            <View className='ai-avatar__blush is-left' />
            <View className='ai-avatar__blush is-right' />
            <View className='ai-avatar__smile' />
          </View>
          <View className='ai-avatar__hair-side is-left' />
          <View className='ai-avatar__hair-side is-right' />
          <View className='ai-avatar__body' />
          <View className='ai-avatar__bow' />
          <View className='ai-avatar__spark' />
        </View>
        <View className='welcome-bubble is-hello'>{t('system.ai.greeting_hello')}</View>
      </View>
      <View className='welcome-bubble is-intro'>{props.greetingMessage}</View>
      <View className='prompt-card'>
        <View className='prompt-card__head'>
          <View>
            <View className='prompt-card__eyebrow'>{t('system.ai.quick_actions')}</View>
            <View className='prompt-card__title'>{t('system.ai.shortcut_question')}</View>
          </View>
          {props.canRefresh ? (
            <Button className='prompt-refresh' hoverClass='none' onClick={props.onRefresh}>
              <Text>{t('system.ai.refresh')}</Text><Refresh size={25} color='#00a96b' />
            </Button>
          ) : null}
        </View>
        {props.loading ? <View className='prompt-loading'>{t('common.message.loading')}</View> : null}
        {!props.loading && !props.shortcuts.length ? (
          <View className='prompt-loading'>{t('system.ai.no_shortcuts')}</View>
        ) : null}
        {!props.loading
          ? props.shortcuts.map((shortcut, index) => (
              <Button
                key={shortcut.key || shortcut.title}
                className='prompt-item'
                hoverClass='none'
                onClick={() => props.onShortcutTap(shortcut)}
              >
                <Text className='prompt-index'>{index + 1}</Text>
                <View className='prompt-content'>
                  <Text className='prompt-text'>{shortcut.title}</Text>
                  <Text className='prompt-meta'>{shortcut.group || t('system.ai.general_assistant')}</Text>
                </View>
                <ArrowRight size={20} color='#9aa0aa' />
              </Button>
            ))
          : null}
      </View>
    </View>
  )
}
