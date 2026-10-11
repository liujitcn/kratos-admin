<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import type { AiKnowledgeOption } from '../../../../rpc/base/v1/ai_knowledge'

const props = defineProps<{
  options: AiKnowledgeOption[]
  selectedIds: number[]
}>()

const emit = defineEmits<{
  toggle: [knowledgeId: number]
  close: []
}>()

const { t } = useI18n()

const selectedCount = computed(() => props.selectedIds.length)

function isSelected(knowledgeId: number) {
  return props.selectedIds.includes(knowledgeId)
}

function handleToggle(knowledgeId: number) {
  emit('toggle', knowledgeId)
}
</script>

<template>
  <view class="knowledge-picker">
    <view class="knowledge-picker__mask" @tap="emit('close')" />
    <view class="knowledge-picker__sheet">
      <view class="knowledge-picker__handle" />
      <view class="knowledge-picker__head">
        <text class="knowledge-picker__title">{{ t('system.ai.knowledge.title') }}</text>
        <text class="knowledge-picker__count">{{
          t('system.ai.knowledge.selected_count', { count: selectedCount })
        }}</text>
        <button class="knowledge-picker__close" hover-class="none" @tap="emit('close')">✕</button>
      </view>
      <text class="knowledge-picker__hint">{{ t('system.ai.knowledge.select_hint') }}</text>
      <scroll-view class="knowledge-picker__body" scroll-y :show-scrollbar="true">
        <view
          v-for="option in options"
          :key="option.id"
          class="knowledge-picker__row"
          :class="{ 'is-selected': isSelected(option.id) }"
          @tap="handleToggle(option.id)"
        >
          <view class="knowledge-picker__row-main">
            <text class="knowledge-picker__row-name">{{ option.name }}</text>
            <text v-if="option.doc_count" class="knowledge-picker__row-meta">{{
              t('system.ai.knowledge.doc_count', { count: option.doc_count })
            }}</text>
          </view>
          <view class="knowledge-picker__check" :class="{ 'is-checked': isSelected(option.id) }">
            <text v-if="isSelected(option.id)">✓</text>
          </view>
        </view>
        <view v-if="!options.length" class="knowledge-picker__empty">{{
          t('system.ai.knowledge.empty')
        }}</view>
      </scroll-view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.knowledge-picker {
  position: fixed;
  inset: 0;
  /* 高于 KratosTabBar 的 z-index:999,避免弹层被底部导航遮挡 */
  z-index: 1000;
}

.knowledge-picker__mask {
  position: absolute;
  inset: 0;
  background: var(--kratos-color-mask, rgba(2, 6, 12, 0.55));
}

.knowledge-picker__sheet {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  max-height: 62vh;
  padding: 40rpx 80rpx calc(80rpx + env(safe-area-inset-bottom));
  overflow: hidden;
  background: var(--kratos-color-surface-solid, #070d18);
  border-radius: 40rpx 40rpx 0 0;
  box-sizing: border-box;
}

.knowledge-picker__handle {
  flex-shrink: 0;
  width: 72rpx;
  height: 8rpx;
  margin: 0 auto 20rpx;
  background: var(--kratos-color-track, rgba(255, 255, 255, 0.06));
  border-radius: 4rpx;
}

.knowledge-picker__head {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.knowledge-picker__title {
  flex-shrink: 0;
  color: var(--kratos-color-text, #e9f4f6);
  font-size: 34rpx;
  font-weight: 600;
}

.knowledge-picker__count {
  flex: 1;
  min-width: 0;
  color: var(--kratos-color-text-muted, #8aa0b4);
  font-size: 24rpx;
}

.knowledge-picker__close {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  padding: 0;
  margin: 0;
  color: var(--kratos-color-text-muted, #8aa0b4);
  font-size: 32rpx;
  line-height: normal;
  background: transparent;
  border-radius: 0;
}

.knowledge-picker__close::after {
  border: none;
}

.knowledge-picker__hint {
  flex-shrink: 0;
  margin-top: 12rpx;
  color: var(--kratos-color-text-muted, #8aa0b4);
  font-size: 24rpx;
}

.knowledge-picker__body {
  flex: 1;
  min-height: 0;
  margin-top: 20rpx;
}

.knowledge-picker__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24rpx;
  min-height: 92rpx;
  padding: 24rpx 8rpx;
  border-bottom: 1rpx solid var(--kratos-color-border, #e3eaf2);
  box-sizing: border-box;
}

.knowledge-picker__row-main {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.knowledge-picker__row-name {
  overflow: hidden;
  color: var(--kratos-color-text, #e9f4f6);
  font-size: 30rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.knowledge-picker__row-meta {
  color: var(--kratos-color-text-muted, #8aa0b4);
  font-size: 24rpx;
}

.knowledge-picker__check {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  border: 1rpx solid var(--kratos-color-border, #d0d2d6);
  border-radius: 50%;
  color: var(--kratos-color-on-primary, #031513);
  font-size: 26rpx;
  box-sizing: border-box;
}

.knowledge-picker__check.is-checked {
  background: var(--kratos-color-primary, #28f0d4);
  border-color: var(--kratos-color-primary, #28f0d4);
}

.knowledge-picker__empty {
  padding: 48rpx 0;
  color: var(--kratos-color-text-muted, #8aa0b4);
  font-size: 26rpx;
  text-align: center;
}
</style>
