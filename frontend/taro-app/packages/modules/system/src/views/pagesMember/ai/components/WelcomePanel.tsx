import { Button, Text, View } from '@tarojs/components'
import { ArrowRight, Refresh } from '@liujitcn/kratos-taro-app-ui'
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
        <View className='welcome-bubble is-hello'>您好，AI助手为您服务！</View>
      </View>
      <View className='welcome-bubble is-intro'>{props.greetingMessage}</View>
      <View className='prompt-card'>
        <View className='prompt-card__head'>
          <View>
            <View className='prompt-card__eyebrow'>快捷操作</View>
            <View className='prompt-card__title'>您可以这样问</View>
          </View>
          {props.canRefresh ? (
            <Button className='prompt-refresh' hoverClass='none' onClick={props.onRefresh}>
              <Text>换一换</Text><Refresh size={25} color='#00a96b' />
            </Button>
          ) : null}
        </View>
        {props.loading ? <View className='prompt-loading'>正在加载...</View> : null}
        {!props.loading && !props.shortcuts.length ? (
          <View className='prompt-loading'>暂无可用快捷助手</View>
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
                  <Text className='prompt-meta'>{shortcut.group || '通用助手'}</Text>
                </View>
                <ArrowRight size={20} color='#9aa0aa' />
              </Button>
            ))
          : null}
      </View>
    </View>
  )
}
