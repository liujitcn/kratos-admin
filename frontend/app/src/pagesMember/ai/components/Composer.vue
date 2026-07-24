<script setup lang="ts">
import type { AiAttachment } from '@/rpc/base/v1/ai_session'

type InputEventValue = {
  detail: {
    value: string
  }
}

defineProps<{
  modelValue: string
  attachments: AiAttachment[]
  placeholder: string
  bottom: string
  recording: boolean
  sending: boolean
  disabled: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  attach: []
  record: []
  send: []
  'remove-attachment': [attachment: AiAttachment]
}>()

function handleInput(event: Event) {
  emit('update:modelValue', ((event as unknown as InputEventValue).detail?.value || '').toString())
}
</script>

<template>
  <view class="composer" :style="{ paddingBottom: bottom }">
    <view class="composer-main">
      <button class="attach-button" hover-class="none" @tap="emit('attach')">
        <uni-icons type="plusempty" size="30" color="#111" />
      </button>
      <view class="composer-card">
        <view v-if="attachments.length" class="composer-attachments">
          <view
            v-for="attachment in attachments"
            :key="attachment.id || attachment.url || attachment.name"
            class="composer-attachment"
            @tap="emit('remove-attachment', attachment)"
          >
            {{ attachment.name }} ×
          </view>
        </view>
        <textarea
          class="composer-input"
          auto-height
          :maxlength="500"
          :value="modelValue"
          :placeholder="placeholder"
          placeholder-class="composer-placeholder"
          @input="handleInput"
        />
        <button
          class="voice-button"
          :class="{ active: recording }"
          hover-class="none"
          @tap="emit('record')"
        >
          <uni-icons type="mic" size="28" :color="recording ? '#00a96b' : '#111'" />
        </button>
      </view>
      <button
        class="send-button"
        :class="{ 'is-disabled': disabled, 'is-sending': sending }"
        :disabled="disabled"
        hover-class="none"
        @tap="emit('send')"
      >
        <uni-icons type="paperplane" size="28" :color="disabled ? '#111' : '#00a96b'" />
      </button>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.attach-button,
.voice-button,
.send-button {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.composer {
  flex-shrink: 0;
  width: 100%;
  padding: 14rpx 24rpx 18rpx;
  overflow: hidden;
  background-color: transparent;
  box-sizing: border-box;
}

.composer-main {
  display: flex;
  align-items: center;
  gap: 16rpx;
  min-height: 124rpx;
  padding: 22rpx 18rpx;
  border-radius: 24rpx;
  background-color: #fff;
  box-shadow: 0 12rpx 34rpx rgba(15, 23, 42, 0.08);
  box-sizing: border-box;
}

.attach-button {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72rpx;
  height: 72rpx;
  border: 2rpx solid #d6dae2;
  border-radius: 50%;
  background-color: #fff;
  box-shadow: none;
  box-sizing: border-box;
}

.composer-card {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10rpx;
  min-height: 72rpx;
  padding: 0 12rpx 0 28rpx;
  border: 2rpx solid #d6dae2;
  border-radius: 38rpx;
  background-color: #fff;
  box-shadow: none;
  box-sizing: border-box;
}

.composer-attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  width: 100%;
  padding-top: 4rpx;
}

.composer-attachment {
  max-width: 240rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 6rpx 12rpx;
  border-radius: 8rpx;
  color: #16806d;
  font-size: 20rpx;
  line-height: 28rpx;
  background-color: #e8f8f4;
}

.composer-input {
  flex: 1;
  min-width: 0;
  min-height: 42rpx;
  max-height: 126rpx;
  padding: 14rpx 0;
  box-sizing: border-box;
  color: #333;
  font-size: 29rpx;
  line-height: 42rpx;
  overflow-y: auto;
  background-color: transparent;
}

.composer-placeholder {
  color: #8d929c;
  font-size: 29rpx;
}

.voice-button,
.send-button {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.voice-button {
  width: 60rpx;
  height: 60rpx;
  background-color: transparent;
}

.voice-button.active {
  background-color: #e8f8f4;
}

.send-button {
  width: 74rpx;
  height: 74rpx;
  background-color: #f2f2f2;
}

.send-button.is-disabled {
  background-color: #f2f2f2;
}

.send-button:not(.is-disabled) {
  background-color: #e7f7f2;
}
</style>
