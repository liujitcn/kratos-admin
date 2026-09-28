import { Button, Picker, Text, Textarea, View } from '@tarojs/components'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import type { AiAttachment } from '../../../../rpc/base/v1/ai_session'
import type { AiProviderModelOption } from '../../../../rpc/base/v1/ai_provider'
import './composer.scss'

/** AI 助手输入框与模型选择器参数。 */
type ComposerProps = {
  value: string
  attachments: AiAttachment[]
  /** 当前可供选择的供应商与模型。 */
  modelProviders: AiProviderModelOption[]
  /** 当前选中的供应商编号和模型名称。 */
  providerId: number
  modelName: string
  /** 选择器未配置时显示的本地化占位文字。 */
  providerPlaceholder: string
  modelPlaceholder: string
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
  onModelChange: (providerId: number, modelName: string) => void
}

/** AI 助手消息输入器。 */
export default function Composer(props: ComposerProps) {
  const selectedProvider = props.modelProviders.find((provider) => provider.provider_id === props.providerId)
  const modelNames = selectedProvider?.models ?? []
  const providerIndex = Math.max(props.modelProviders.findIndex((provider) => provider.provider_id === props.providerId), 0)
  const modelIndex = Math.max(modelNames.indexOf(props.modelName), 0)

  return (
    <View className='composer' style={{ paddingBottom: props.bottom }}>
      <View className='composer-model-selectors'>
        <Picker
          mode='selector'
          range={props.modelProviders.map((provider) => provider.name)}
          value={providerIndex}
          disabled={props.sending || !props.modelProviders.length}
          onChange={(event) => {
            const provider = props.modelProviders[Number(event.detail.value)]
            props.onModelChange(provider?.provider_id ?? 0, provider?.models[0] ?? '')
          }}
        >
          <View className='composer-model-select'><Text>{selectedProvider?.name || props.providerPlaceholder}</Text></View>
        </Picker>
        <Picker
          mode='selector'
          range={modelNames}
          value={modelIndex}
          disabled={props.sending || !modelNames.length}
          onChange={(event) => props.onModelChange(props.providerId, modelNames[Number(event.detail.value)] ?? '')}
        >
          <View className='composer-model-select'><Text>{props.modelName || props.modelPlaceholder}</Text></View>
        </Picker>
      </View>
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
