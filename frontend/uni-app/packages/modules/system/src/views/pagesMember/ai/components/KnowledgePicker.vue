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
      </scroll-view>
    </view>
  </view>
</template>

<style lang="scss" scoped>
.knowledge-picker {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-end;
}

.knowledge-picker__mask {
  position: absolute;
  inset: 0;
  background: var(--kratos-color-mask, rgba(2, 6, 12, 0.55));
}

.knowledge-picker__sheet {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-height: 62vh;
  padding: 10px 20px calc(20px + env(safe-area-inset-bottom));
  background: var(--kratos-color-surface-solid, #070d18);
  border-radius: 20px 20px 0 0;
}

.knowledge-picker__handle {
  width: 36px;
  height: 4px;
  margin: 0 auto 10px;
  background: var(--kratos-color-track, rgba(255, 255, 255, 0.06));
  border-radius: 2px;
}

.knowledge-picker__head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.knowledge-picker__title {
  font-size: 17px;
  font-weight: 600;
  color: var(--kratos-color-text, #e9f4f6);
}

.knowledge-picker__count {
  flex: 1;
  font-size: 12px;
  color: var(--kratos-color-text-muted, #8aa0b4);
}

.knowledge-picker__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44rpx;
  height: 44rpx;
  padding: 0;
  margin: 0;
  font-size: 16px;
  line-height: normal;
  color: var(--kratos-color-text-muted, #8aa0b4);
  background: transparent;
  border-radius: 0;
}

.knowledge-picker__close::after {
  border: none;
}

.knowledge-picker__hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--kratos-color-text-muted, #8aa0b4);
}

.knowledge-picker__body {
  margin-top: 10px;
}

.knowledge-picker__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 4px;
  border-bottom: 1rpx solid var(--kratos-color-border, #e3eaf2);
}

.knowledge-picker__row-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.knowledge-picker__row-name {
  font-size: 15px;
  color: var(--kratos-color-text, #e9f4f6);
}

.knowledge-picker__row-meta {
  font-size: 12px;
  color: var(--kratos-color-text-muted, #8aa0b4);
}

.knowledge-picker__check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1rpx solid var(--kratos-color-border, #d0d2d6);
  border-radius: 50%;
  font-size: 13px;
  color: var(--kratos-color-on-primary, #031513);
}

.knowledge-picker__check.is-checked {
  background: var(--kratos-color-primary, #28f0d4);
  border-color: var(--kratos-color-primary, #28f0d4);
}
</style>
