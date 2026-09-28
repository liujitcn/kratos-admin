<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import type { AiProviderModelOption } from '../../../../rpc/base/v1/ai_provider'
import { fallbackModelDisplayName } from '../modelDisplay'

type PickerModel = {
  name: string
  displayName: string
}

type PickerProvider = {
  providerId: number
  label: string
  models: PickerModel[]
}

const props = defineProps<{
  providers: AiProviderModelOption[]
  providerId: number
  modelName: string
}>()

const emit = defineEmits<{
  select: [providerId: number, modelName: string]
  close: []
}>()

const { t } = useI18n()

const pickerProviders = computed<PickerProvider[]>(() =>
  props.providers.map((provider) => ({
    providerId: provider.provider_id,
    label: provider.name,
    models: provider.models.map((model) => ({
      name: model.model_name,
      displayName: model.display_name || fallbackModelDisplayName(model.model_name),
    })),
  })),
)

const modelCount = computed(() =>
  pickerProviders.value.reduce((total, provider) => total + provider.models.length, 0),
)

function isSelected(providerId: number, modelName: string) {
  return props.providerId === providerId && props.modelName === modelName
}

function handleSelect(providerId: number, modelName: string) {
  emit('select', providerId, modelName)
}
</script>

<template>
  <view class="model-picker">
    <view class="model-picker__mask" @tap="emit('close')" />
    <view class="model-picker__sheet">
      <view class="model-picker__handle" />
      <view class="model-picker__head">
        <text class="model-picker__title">{{ t('system.ai.model.title') }}</text>
        <text class="model-picker__count">{{
          t('system.ai.model.provider_model_count', {
            providers: pickerProviders.length,
            models: modelCount,
          })
        }}</text>
        <button class="model-picker__close" hover-class="none" @tap="emit('close')">✕</button>
      </view>
      <scroll-view class="model-picker__body" scroll-y :show-scrollbar="true">
        <block v-for="provider in pickerProviders" :key="provider.providerId">
          <view class="model-picker__group">{{ provider.label }}</view>
          <view
            v-for="model in provider.models"
            :key="model.name"
            class="model-picker__row"
            :class="{ 'is-selected': isSelected(provider.providerId, model.name) }"
            @tap="handleSelect(provider.providerId, model.name)"
          >
            <view class="model-picker__row-main">
              <text class="model-picker__row-name">{{ model.displayName }}</text>
              <text v-if="model.displayName !== model.name" class="model-picker__row-raw">{{
                model.name
              }}</text>
            </view>
            <text v-if="isSelected(provider.providerId, model.name)" class="model-picker__row-check"
              >✓</text
            >
          </view>
        </block>
        <view v-if="!pickerProviders.length" class="model-picker__empty">{{
          t('system.ai.model.empty')
        }}</view>
      </scroll-view>
      <view class="model-picker__safe" />
    </view>
  </view>
</template>

<style lang="scss" scoped>
.model-picker {
  position: fixed;
  inset: 0;
  /* 高于 KratosTabBar 的 z-index:999,避免弹层被底部导航遮挡 */
  z-index: 1000;
}

.model-picker__mask {
  position: absolute;
  inset: 0;
  background-color: rgba(15, 23, 42, 0.45);
}

.model-picker__sheet {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  max-height: 68vh;
  overflow: hidden;
  background-color: #fff;
  border-radius: 32rpx 32rpx 0 0;
  box-shadow: 0 -24rpx 80rpx rgba(15, 23, 42, 0.16);
  box-sizing: border-box;
}

.model-picker__handle {
  flex-shrink: 0;
  width: 72rpx;
  height: 8rpx;
  margin: 16rpx auto 8rpx;
  background-color: #e2e5ea;
  border-radius: 4rpx;
}

.model-picker__head {
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 12rpx;
  padding: 0 96rpx 16rpx;
}

.model-picker__title {
  color: #111;
  font-size: 32rpx;
  font-weight: 600;
}

.model-picker__count {
  color: #8a8f99;
  font-size: 20rpx;
}

.model-picker__close {
  position: absolute;
  top: -6rpx;
  right: 24rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56rpx;
  height: 56rpx;
  padding: 0;
  margin: 0;
  color: #8a8f99;
  font-size: 28rpx;
  line-height: normal;
  background: transparent;
  border-radius: 0;
}

.model-picker__close::after {
  border: 0;
}

.model-picker__body {
  flex: 1;
  min-height: 0;
  max-height: calc(68vh - 130rpx - env(safe-area-inset-bottom));
  padding: 0 32rpx;
  box-sizing: border-box;
}

/* #ifdef H5 */
.model-picker__body :deep(::-webkit-scrollbar) {
  width: 8rpx;
  height: 8rpx;
}

.model-picker__body :deep(::-webkit-scrollbar-thumb) {
  background-color: #c9ced6;
  border-radius: 4rpx;
}

.model-picker__body :deep(::-webkit-scrollbar-track) {
  background-color: transparent;
}
/* #endif */

.model-picker__group {
  padding: 24rpx 8rpx 12rpx;
  color: #8a8f99;
  font-size: 22rpx;
  line-height: 32rpx;
}

.model-picker__row {
  display: flex;
  align-items: center;
  min-height: 88rpx;
  padding: 16rpx 28rpx;
  margin-bottom: 12rpx;
  border: 2rpx solid #eef0f4;
  border-radius: 24rpx;
  box-sizing: border-box;
}

.model-picker__row.is-selected {
  color: #0f7a62;
  font-weight: 500;
  background-color: #e8f8f4;
  border-color: #d5efe7;
}

.model-picker__row-main {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.model-picker__row-name {
  color: inherit;
  font-size: 28rpx;
  line-height: 40rpx;
}

.model-picker__row-raw {
  color: #a7acb5;
  font-size: 22rpx;
  font-weight: 400;
  line-height: 30rpx;
}

.model-picker__row-check {
  margin-left: auto;
  color: #00a96b;
  font-size: 32rpx;
}

.model-picker__empty {
  padding: 64rpx 0;
  color: #8a8f99;
  font-size: 26rpx;
  text-align: center;
}

.model-picker__safe {
  flex-shrink: 0;
  height: calc(18rpx + constant(safe-area-inset-bottom));
  height: calc(18rpx + env(safe-area-inset-bottom));
}
</style>
