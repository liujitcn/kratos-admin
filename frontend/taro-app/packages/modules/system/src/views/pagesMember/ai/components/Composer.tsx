import { Button, Text, Textarea, View } from '@tarojs/components'
import { useMemo, useState } from 'react'
import { t, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import type { AiAttachment } from '../../../../rpc/base/v1/ai_session'
import type { AiProviderModelOption } from '../../../../rpc/base/v1/ai_provider'
import type { AiKnowledgeOption } from '../../../../rpc/base/v1/ai_knowledge'
import { fallbackModelDisplayName } from '../modelDisplay'
import KnowledgePicker from './KnowledgePicker'
import ModelPicker from './ModelPicker'
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
  /** 当前可供选择的知识库及已选择编号。 */
  knowledgeOptions: AiKnowledgeOption[]
  knowledgeIds: number[]
  /** 选择器未配置时显示的本地化占位文字。 */
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
  onKnowledgeChange: (knowledgeIds: number[]) => void
}

/** AI 助手消息输入器。 */
export default function Composer(props: ComposerProps) {
  const [showModelPicker, setShowModelPicker] = useState(false)
  const [showKnowledgePicker, setShowKnowledgePicker] = useState(false)
  const themeStore = useThemeStore()
  const iconColor = themeStore.colors.aiTextMuted
  const primaryIconColor = themeStore.colors.aiPrimary

  const selectedModelDisplayName = useMemo(() => {
    const provider = props.modelProviders.find((item) => item.provider_id === props.providerId)
    const matched = provider?.models.find((model) => model.model_name === props.modelName)
    return matched?.display_name || fallbackModelDisplayName(props.modelName)
  }, [props.modelProviders, props.providerId, props.modelName])

  const openModelPicker = () => {
    if (props.sending || !props.modelProviders.length) {
      return
    }
    setShowModelPicker(true)
  }

  const handleModelSelect = (providerId: number, modelName: string) => {
    setShowModelPicker(false)
    props.onModelChange(providerId, modelName)
  }

  /** 打开可用知识库选择器。 */
  const openKnowledgePicker = () => {
    if (props.sending || !props.knowledgeOptions.length) return
    setShowKnowledgePicker(true)
  }

  /** 切换知识库选择并通知页面。 */
  const handleKnowledgeToggle = (knowledgeId: number) => {
    const current = new Set(props.knowledgeIds)
    if (current.has(knowledgeId)) current.delete(knowledgeId)
    else current.add(knowledgeId)
    props.onKnowledgeChange([...current])
  }

  return (
    <View className='composer' style={{ paddingBottom: props.bottom }}>
      <View
        className={`composer-model-chip${props.sending || !props.modelProviders.length ? ' is-disabled' : ''}`}
        onClick={openModelPicker}
      >
        <Text className='composer-model-chip__spark'>✦</Text>
        <Text className='composer-model-chip__label'>
          {selectedModelDisplayName || props.modelPlaceholder}
        </Text>
        <Text className='composer-model-chip__chevron'>▾</Text>
      </View>
      <View
        className={`composer-model-chip${props.sending || !props.knowledgeOptions.length ? ' is-disabled' : ''}${props.knowledgeIds.length ? ' is-selected' : ''}`}
        onClick={openKnowledgePicker}
      >
        <Text className='composer-model-chip__spark'>📚</Text>
        <Text className='composer-model-chip__label'>
          {props.knowledgeIds.length
            ? t('system.ai.knowledge.selected_count', { count: props.knowledgeIds.length })
            : t('system.ai.knowledge.title')}
        </Text>
        <Text className='composer-model-chip__chevron'>▾</Text>
      </View>
      <View className='composer-main'>
        <Button className='attach-button' hoverClass='none' onClick={props.onAttach}>
          <UniIcon type='plusempty' size={30} color={iconColor} />
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
            <UniIcon type='mic' size={28} color={props.recording ? primaryIconColor : iconColor} />
          </Button>
        </View>
        <Button
          className={`send-button${props.disabled ? ' is-disabled' : ''}${props.sending ? ' is-sending' : ''}`}
          disabled={props.disabled}
          hoverClass='none'
          onClick={props.onSend}
        >
          <UniIcon
            type='paperplane'
            size={28}
            color={props.disabled ? iconColor : primaryIconColor}
          />
        </Button>
      </View>

      {showModelPicker ? (
        <ModelPicker
          providers={props.modelProviders}
          providerId={props.providerId}
          modelName={props.modelName}
          onSelect={handleModelSelect}
          onClose={() => setShowModelPicker(false)}
        />
      ) : null}
      {showKnowledgePicker ? (
        <KnowledgePicker
          options={props.knowledgeOptions}
          selectedIds={props.knowledgeIds}
          onToggle={handleKnowledgeToggle}
          onClose={() => setShowKnowledgePicker(false)}
        />
      ) : null}
    </View>
  )
}
