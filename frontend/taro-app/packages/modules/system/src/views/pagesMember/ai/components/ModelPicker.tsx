import { ScrollView, Text, View } from '@tarojs/components'
import { useMemo } from 'react'
import { t } from '@liujitcn/kratos-taro-app-core'
import type { AiProviderModelOption } from '../../../../rpc/base/v1/ai_provider'
import { fallbackModelDisplayName } from '../modelDisplay'
import './model-picker.scss'

type PickerModel = {
  name: string
  displayName: string
}

type PickerProvider = {
  providerId: number
  label: string
  models: PickerModel[]
}

type ModelPickerProps = {
  providers: AiProviderModelOption[]
  providerId: number
  modelName: string
  onSelect: (providerId: number, modelName: string) => void
  onClose: () => void
}

/** AI 模型选择底部弹层。 */
export default function ModelPicker(props: ModelPickerProps) {
  const pickerProviders = useMemo<PickerProvider[]>(
    () =>
      props.providers.map((provider) => ({
        providerId: provider.provider_id,
        label: provider.name,
        models: provider.models.map((model) => ({
          name: model.model_name,
          displayName: model.display_name || fallbackModelDisplayName(model.model_name),
        })),
      })),
    [props.providers],
  )

  const modelCount = pickerProviders.reduce((total, provider) => total + provider.models.length, 0)

  const isSelected = (providerId: number, modelName: string) =>
    props.providerId === providerId && props.modelName === modelName

  return (
    <View className='model-picker' catchMove>
      <View className='model-picker__mask' onClick={props.onClose} />
      <View className='model-picker__sheet'>
        <View className='model-picker__handle' />
        <View className='model-picker__head'>
          <Text className='model-picker__title'>{t('system.ai.model.title')}</Text>
          <Text className='model-picker__count'>
            {t('system.ai.model.provider_model_count', {
              providers: pickerProviders.length,
              models: modelCount,
            })}
          </Text>
          <View className='model-picker__close' onClick={props.onClose}>
            ✕
          </View>
        </View>
        <ScrollView className='model-picker__body' scrollY>
          {pickerProviders.map((provider) => (
            <View key={provider.providerId}>
              <View className='model-picker__group'>{provider.label}</View>
              {provider.models.map((model) => (
                <View
                  key={model.name}
                  className={`model-picker__row${isSelected(provider.providerId, model.name) ? ' is-selected' : ''}`}
                  onClick={() => props.onSelect(provider.providerId, model.name)}
                >
                  <View className='model-picker__row-main'>
                    <Text className='model-picker__row-name'>{model.displayName}</Text>
                    {model.displayName !== model.name ? (
                      <Text className='model-picker__row-raw'>{model.name}</Text>
                    ) : null}
                  </View>
                  {isSelected(provider.providerId, model.name) ? (
                    <Text className='model-picker__row-check'>✓</Text>
                  ) : null}
                </View>
              ))}
            </View>
          ))}
          {!pickerProviders.length ? (
            <View className='model-picker__empty'>{t('system.ai.model.empty')}</View>
          ) : null}
        </ScrollView>
        <View className='model-picker__safe' />
      </View>
    </View>
  )
}
