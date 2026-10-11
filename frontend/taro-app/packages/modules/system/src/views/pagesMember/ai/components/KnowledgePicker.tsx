import { ScrollView, Text, View } from '@tarojs/components'
import { t } from '@liujitcn/kratos-taro-app-core'
import type { AiKnowledgeOption } from '../../../../rpc/base/v1/ai_knowledge'
import './knowledge-picker.scss'

type KnowledgePickerProps = {
  options: AiKnowledgeOption[]
  selectedIds: number[]
  onToggle: (knowledgeId: number) => void
  onClose: () => void
}

/** AI 知识库多选底部弹层。 */
export default function KnowledgePicker(props: KnowledgePickerProps) {
  return (
    <View className='knowledge-picker' catchMove>
      <View className='knowledge-picker__mask' onClick={props.onClose} />
      <View className='knowledge-picker__sheet'>
        <View className='knowledge-picker__handle' />
        <View className='knowledge-picker__head'>
          <Text className='knowledge-picker__title'>{t('system.ai.knowledge.title')}</Text>
          <Text className='knowledge-picker__count'>
            {t('system.ai.knowledge.selected_count', { count: props.selectedIds.length })}
          </Text>
          <View className='knowledge-picker__close' onClick={props.onClose}>✕</View>
        </View>
        <Text className='knowledge-picker__hint'>{t('system.ai.knowledge.select_hint')}</Text>
        <ScrollView className='knowledge-picker__body' scrollY>
          {props.options.map((option) => {
            const selected = props.selectedIds.includes(option.id)
            return (
              <View
                key={option.id}
                className={`knowledge-picker__row${selected ? ' is-selected' : ''}`}
                onClick={() => props.onToggle(option.id)}
              >
                <View className='knowledge-picker__row-main'>
                  <Text className='knowledge-picker__row-name'>{option.name}</Text>
                  {option.doc_count ? (
                    <Text className='knowledge-picker__row-meta'>
                      {t('system.ai.knowledge.doc_count', { count: option.doc_count })}
                    </Text>
                  ) : null}
                </View>
                <View className={`knowledge-picker__check${selected ? ' is-checked' : ''}`}>
                  {selected ? <Text>✓</Text> : null}
                </View>
              </View>
            )
          })}
          {!props.options.length ? (
            <View className='knowledge-picker__empty'>{t('system.ai.knowledge.empty')}</View>
          ) : null}
        </ScrollView>
      </View>
    </View>
  )
}
