<script setup lang="ts">
import type { AiSession } from '@/rpc/base/v1/ai_session'

type PressPoint = {
  clientX?: number
  clientY?: number
  pageX?: number
  pageY?: number
  x?: number
  y?: number
}

type PressEvent = {
  detail?: {
    x?: number
    y?: number
  }
  touches?: PressPoint[]
  changedTouches?: PressPoint[]
}

type InputEventValue = {
  detail: {
    value: string
  }
}

defineProps<{
  open: boolean
  topPadding: string
  keyword: string
  loading: boolean
  sessions: AiSession[]
  activeSessionId: string
}>()

const emit = defineEmits<{
  close: []
  create: []
  select: [sessionID: string]
  action: [session: AiSession, event?: PressEvent]
  'update:keyword': [value: string]
}>()

function handleKeywordInput(event: Event) {
  emit('update:keyword', ((event as unknown as InputEventValue).detail?.value || '').toString())
}

function formatSessionTime(session: AiSession) {
  const seconds = Number(session.updated_at?.seconds ?? 0)
  const nanos = Number(session.updated_at?.nanos ?? 0)
  const timestamp = seconds * 1000 + Math.floor(nanos / 1_000_000)
  if (!timestamp) {
    return ''
  }
  const date = new Date(timestamp)
  const now = new Date()
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return isToday ? `${hour}:${minute}` : `${month}-${day}`
}
</script>

<template>
  <view v-if="open" class="session-mask" @tap="emit('close')"></view>
  <view class="session-drawer" :class="{ 'is-open': open }" :style="{ paddingTop: topPadding }">
    <view class="session-drawer__head">
      <view class="session-drawer__title">历史会话</view>
      <button class="session-create" hover-class="none" @tap="emit('create')">
        <uni-icons type="plusempty" size="16" color="#27ba9b" />
        <text>新建</text>
      </button>
    </view>
    <view class="session-search">
      <uni-icons type="search" size="16" color="#898b94" />
      <input
        class="session-search-input"
        confirm-type="search"
        placeholder="搜索会话"
        placeholder-class="session-search-placeholder"
        :value="keyword"
        @input="handleKeywordInput"
      />
    </view>
    <scroll-view class="session-list" scroll-y :show-scrollbar="false">
      <view v-if="loading" class="session-empty">正在加载会话...</view>
      <view
        v-for="session in sessions"
        :key="session.id"
        class="session-item"
        :class="{ 'is-active': session.id === activeSessionId }"
        @tap="emit('select', session.id)"
        @longpress="emit('action', session, $event)"
      >
        <view class="session-content">
          <view class="session-row">
            <view class="session-title">{{ session.title }}</view>
            <view class="session-time">{{ formatSessionTime(session) }}</view>
          </view>
          <view class="session-summary">{{ session.summary || '暂无摘要' }}</view>
        </view>
        <button class="session-more" hover-class="none" @tap.stop="emit('action', session, $event)">
          <view></view>
          <view></view>
          <view></view>
        </button>
      </view>
      <view v-if="!loading && !sessions.length" class="session-empty">没有匹配的会话</view>
    </scroll-view>
  </view>
</template>

<style lang="scss" scoped>
.session-create,
.session-more {
  padding: 0;
  margin: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;

  &::after {
    border: none;
  }
}

.session-mask {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 560rpx;
  z-index: 20;
  background-color: rgba(0, 0, 0, 0.18);
}

.session-drawer {
  display: flex;
  flex-direction: column;
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 21;
  width: 560rpx;
  padding: 0 24rpx 36rpx;
  background-color: #fff;
  box-shadow: none;
  box-sizing: border-box;
  transform: translateX(-100%);
  transition: transform 0.2s ease;
}

.session-drawer.is-open {
  box-shadow: 24rpx 0 60rpx rgba(0, 0, 0, 0.12);
  transform: translateX(0);
}

.session-drawer__head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 76rpx;
  margin-bottom: 18rpx;
}

.session-drawer__title {
  color: #333;
  font-size: 32rpx;
  font-weight: 700;
  line-height: 40rpx;
}

.session-create {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4rpx;
  width: 112rpx;
  height: 52rpx;
  border-radius: 8rpx;
  color: #27ba9b;
  font-size: 24rpx;
  line-height: 52rpx;
  background-color: #e8f8f4;
}

.session-search {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10rpx;
  height: 64rpx;
  padding: 0 18rpx;
  margin-bottom: 18rpx;
  border-radius: 10rpx;
  background-color: #f6f7f9;
  box-sizing: border-box;
}

.session-search-input {
  flex: 1;
  min-width: 0;
  color: #333;
  font-size: 24rpx;
}

.session-search-placeholder {
  color: #b8bcc5;
}

.session-list {
  flex: 1;
  min-height: 0;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 12rpx;
  padding: 22rpx 16rpx 22rpx 22rpx;
  border: 1rpx solid #eee;
  border-radius: 10rpx;
  background-color: #fff;
}

.session-item + .session-item {
  margin-top: 18rpx;
}

.session-item.is-active {
  border-color: #c7eee4;
  background-color: #e8f8f4;
}

.session-content {
  flex: 1;
  min-width: 0;
}

.session-row {
  display: flex;
  align-items: center;
  gap: 14rpx;
}

.session-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
  font-size: 26rpx;
  font-weight: 600;
  line-height: 34rpx;
}

.session-time {
  flex-shrink: 0;
  color: #a6abb3;
  font-size: 20rpx;
  line-height: 30rpx;
}

.session-summary {
  margin-top: 10rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #777;
  font-size: 22rpx;
  line-height: 32rpx;
}

.session-more {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5rpx;
  width: 56rpx;
  height: 56rpx;
  border-radius: 50%;
  background-color: transparent;
}

.session-more view {
  width: 6rpx;
  height: 6rpx;
  border-radius: 50%;
  background-color: #9ca3af;
}

.session-empty {
  padding: 60rpx 0;
  color: #999;
  font-size: 24rpx;
  text-align: center;
}
</style>
