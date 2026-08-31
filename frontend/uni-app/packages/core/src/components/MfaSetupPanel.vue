<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import qrcode from 'qrcode-generator'
import { useI18n } from '../locales'
import { copyText } from '../utils/clipboard'

/** MFA TOTP 绑定面板属性。 */
interface MfaSetupPanelProps {
  /** TOTP 绑定地址。 */
  uri: string
}

const props = defineProps<MfaSetupPanelProps>()
const { t } = useI18n()
const copyState = ref<'idle' | 'copied' | 'failed'>('idle')
let resetCopyStateTimer: ReturnType<typeof setTimeout> | undefined

const qrDataUrl = computed(() => {
  if (!props.uri) return ''
  const code = qrcode(0, 'M')
  code.addData(props.uri)
  code.make()
  return code.createDataURL(4, 8)
})

/** 复制 TOTP 绑定地址。 */
async function copySetupUri() {
  if (!props.uri) return
  try {
    await copyText(props.uri)
    copyState.value = 'copied'
  } catch {
    copyState.value = 'failed'
  }
  if (resetCopyStateTimer) clearTimeout(resetCopyStateTimer)
  resetCopyStateTimer = setTimeout(() => {
    copyState.value = 'idle'
  }, 2000)
}

onBeforeUnmount(() => {
  if (resetCopyStateTimer) clearTimeout(resetCopyStateTimer)
})
</script>

<template>
  <view v-if="uri" class="mfa-setup-panel">
    <image v-if="qrDataUrl" class="mfa-setup-panel__qr" :src="qrDataUrl" mode="aspectFit" />
    <view class="mfa-setup-panel__uri">
      <text class="mfa-setup-panel__uri-text">{{ uri }}</text>
      <button
        class="mfa-setup-panel__copy"
        :class="{ 'is-copied': copyState === 'copied', 'is-failed': copyState === 'failed' }"
        :aria-label="t('core.login.mfa_copy_uri')"
        :title="t('core.login.mfa_copy_uri')"
        @tap="copySetupUri"
      >
        <text v-if="copyState === 'copied'" class="mfa-setup-panel__status" aria-hidden="true"
          >✓</text
        >
        <text v-else-if="copyState === 'failed'" class="mfa-setup-panel__status" aria-hidden="true"
          >!</text
        >
        <view v-else class="mfa-setup-panel__copy-icon" aria-hidden="true">
          <view class="mfa-setup-panel__copy-back" />
          <view class="mfa-setup-panel__copy-front" />
        </view>
      </button>
    </view>
  </view>
</template>

<style scoped lang="scss">
.mfa-setup-panel__qr {
  display: block;
  width: min(360rpx, 100%);
  aspect-ratio: 1;
  margin: 0 auto 24rpx;
}

.mfa-setup-panel__uri {
  display: flex;
  gap: 16rpx;
  align-items: flex-start;
  margin-bottom: 20rpx;
}

.mfa-setup-panel__uri-text {
  flex: 1;
  min-width: 0;
  color: #666;
  font-size: 24rpx;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.mfa-setup-panel__copy {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 58rpx;
  min-width: 58rpx;
  height: 58rpx;
  margin: 0;
  padding: 0;
  color: #27ba9b;
  background: #f0fbf8;
  border: 0;
  border-radius: 8rpx;
}

.mfa-setup-panel__copy::after {
  border: 0;
}

.mfa-setup-panel__copy.is-copied {
  color: #fff;
  background: #27ba9b;
}

.mfa-setup-panel__copy.is-failed {
  color: #e64340;
  background: #fff1f0;
}

.mfa-setup-panel__status {
  font-size: 30rpx;
  font-weight: 700;
  line-height: 1;
}

.mfa-setup-panel__copy-icon {
  position: relative;
  display: block;
  width: 24rpx;
  height: 24rpx;
  color: currentColor;
}

.mfa-setup-panel__copy-back,
.mfa-setup-panel__copy-front {
  position: absolute;
  width: 15rpx;
  height: 17rpx;
  border: 2rpx solid currentColor;
  border-radius: 3rpx;
  box-sizing: border-box;
}

.mfa-setup-panel__copy-back {
  top: 0;
  left: 0;
}

.mfa-setup-panel__copy-front {
  right: 0;
  bottom: 0;
  background-color: #f0fbf8;
}
</style>
