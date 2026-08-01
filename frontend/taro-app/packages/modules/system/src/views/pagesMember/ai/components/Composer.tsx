import { Button, Text, Textarea, View } from '@tarojs/components'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import type { AiAttachment } from '../../../../rpc/base/v1/ai_session'
import './composer.scss'

type ComposerProps = {
  value: string
  attachments: AiAttachment[]
  placeholder: string
  bottom: string
  recording: boolean
  sending: boolean
  disabled: boolean
  onChange: (value: string) => void
  onAttach: () => void
  onRecord: () => void
  onSend: () => void
  onRemoveAttachment: (attachment: AiAttachment) => void
}

/** AI 助手消息输入器。 */
export default function Composer(props: ComposerProps) {
  return (
    <View className='composer' style={{ paddingBottom: props.bottom }}>
      <View className='composer-main'>
        <Button className='attach-button' hoverClass='none' onClick={props.onAttach}>
          <UniIcon type='plusempty' size={30} color='#111' />
        </Button>
        <View className='composer-card'>
          {props.attachments.length ? (
            <View className='composer-attachments'>
              {props.attachments.map((attachment) => (
                <View
                  key={attachment.id || attachment.url || attachment.name}
                  className='composer-attachment'
                  onClick={() => props.onRemoveAttachment(attachment)}
                >
                  <Text>{attachment.name} ×</Text>
                </View>
              ))}
            </View>
          ) : null}
          <Textarea
            className='composer-input'
            autoHeight
            maxlength={500}
            value={props.value}
            placeholder={props.placeholder}
            placeholderClass='composer-placeholder'
            onInput={(event) => props.onChange(event.detail.value)}
          />
          <Button
            className={`voice-button${props.recording ? ' active' : ''}`}
            hoverClass='none'
            onClick={props.onRecord}
          >
            <UniIcon type='mic' size={28} color={props.recording ? '#00a96b' : '#111'} />
          </Button>
        </View>
        <Button
          className={`send-button${props.disabled ? ' is-disabled' : ''}${props.sending ? ' is-sending' : ''}`}
          disabled={props.disabled}
          hoverClass='none'
          onClick={props.onSend}
        >
          <UniIcon type='paperplane' size={28} color={props.disabled ? '#111' : '#00a96b'} />
        </Button>
      </View>
    </View>
  )
}
