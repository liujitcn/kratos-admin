<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from '../locales'
import { copyText } from '../utils/clipboard'

/** MFA 恢复码弹窗属性。 */
interface MfaRecoveryCodesDialogProps {
  /** 弹窗显示状态。 */
  modelValue: boolean
  /** 本次生成的一次性恢复码。 */
  codes: string[]
}

const props = withDefaults(defineProps<MfaRecoveryCodesDialogProps>(), {
  modelValue: false,
  codes: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
}>()

const { t } = useI18n()
const codesText = computed(() => props.codes.join('\n'))
const copyState = ref<'idle' | 'copied' | 'failed'>('idle')
const copyTextLabel = computed(() => {
  if (copyState.value === 'copied') return t('core.login.mfa_recovery_codes_copied')
  if (copyState.value === 'failed') return t('common.message.request_error')
  return t('core.login.mfa_copy_recovery_codes')
})
let resetCopyStateTimer: ReturnType<typeof setTimeout> | undefined

/** 复制当前显示的恢复码。 */
async function copyRecoveryCodes() {
  if (!codesText.value) return
  try {
    await copyText(codesText.value)
    copyState.value = 'copied'
  } catch {
    copyState.value = 'failed'
  }
  if (resetCopyStateTimer) clearTimeout(resetCopyStateTimer)
  resetCopyStateTimer = setTimeout(() => {
    copyState.value = 'idle'
  }, 2000)
}

/** 关闭弹窗并通知调用方继续完成绑定流程。 */
function handleConfirm() {
  emit('update:modelValue', false)
  emit('confirm')
}

onBeforeUnmount(() => {
  if (resetCopyStateTimer) clearTimeout(resetCopyStateTimer)
})
</script>

<template>
  <view v-if="modelValue" class="mfa-recovery-dialog">
    <view class="mfa-recovery-dialog__panel" @tap.stop>
      <view class="mfa-recovery-dialog__header">
        <text class="mfa-recovery-dialog__title">{{
          t('core.login.mfa_recovery_codes_title')
        }}</text>
      </view>
      <view class="mfa-recovery-dialog__body">
        <view class="mfa-recovery-dialog__codes">
          <text>{{ codesText }}</text>
        </view>
        <button
          class="mfa-recovery-dialog__copy"
          :class="{ 'is-failed': copyState === 'failed' }"
          @tap.stop="copyRecoveryCodes"
        >
          {{ copyTextLabel }}
        </button>
        <text class="mfa-recovery-dialog__warning">{{
          t('core.login.mfa_recovery_codes_warning')
        }}</text>
      </view>
      <view class="mfa-recovery-dialog__footer">
        <button class="mfa-recovery-dialog__confirm" @tap="handleConfirm">
          {{ t('core.login.mfa_recovery_codes_confirm') }}
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.mfa-recovery-dialog {
  position: fixed;
  z-index: 1001;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32rpx;
  background-color: rgb(0 0 0 / 45%);
}

.mfa-recovery-dialog__panel {
  width: min(100%, 640rpx);
  overflow: hidden;
  background-color: #fff;
  border-radius: 16rpx;
  box-shadow: 0 20rpx 64rpx rgb(0 0 0 / 20%);
}

.mfa-recovery-dialog__header {
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #eee;
}

.mfa-recovery-dialog__title {
  color: #222;
  font-size: 34rpx;
  font-weight: 600;
}

.mfa-recovery-dialog__body {
  padding: 32rpx;
}

.mfa-recovery-dialog__codes {
  box-sizing: border-box;
  width: 100%;
  min-height: 220rpx;
  margin-bottom: 16rpx;
  padding: 20rpx 24rpx;
  color: #222;
  font-size: 30rpx;
  line-height: 1.6;
  border: 1rpx solid #ddd;
  border-radius: 8rpx;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.mfa-recovery-dialog__copy {
  width: auto;
  min-width: 140rpx;
  height: 58rpx;
  margin: 0;
  padding: 0 18rpx;
  color: #27ba9b;
  font-size: 24rpx;
  line-height: 58rpx;
  background-color: #f0fbf8;
  border: 0;
  border-radius: 8rpx;
}

.mfa-recovery-dialog__copy::after,
.mfa-recovery-dialog__confirm::after {
  border: 0;
}

.mfa-recovery-dialog__copy.is-failed {
  color: #e64340;
  background-color: #fff1f0;
}

.mfa-recovery-dialog__warning {
  display: block;
  margin-top: 12rpx;
  color: #c58b00;
  font-size: 24rpx;
  line-height: 1.5;
}

.mfa-recovery-dialog__footer {
  display: flex;
  justify-content: flex-end;
  padding: 0 32rpx 32rpx;
}

.mfa-recovery-dialog__confirm {
  min-width: 160rpx;
  height: 72rpx;
  margin: 0;
  padding: 0 24rpx;
  color: #fff;
  font-size: 28rpx;
  line-height: 72rpx;
  background-color: #27ba9b;
  border: 0;
  border-radius: 8rpx;
}
</style>
