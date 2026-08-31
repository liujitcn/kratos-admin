<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from '../locales'
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from '../utils/passwordCrypto'
import type { PasswordCryptoScene } from '../utils/passwordCrypto'
import type { PasswordCrypto } from '../rpc/common/v1/types'

const { t } = useI18n()

/** 通用密码验证弹窗属性。 */
interface PasswordVerifyDialogProps {
  /** 弹窗显示状态。 */
  modelValue: boolean
  /** 弹窗标题。 */
  title?: string
  /** 密码输入前的可选补充说明。 */
  description?: string
  /** 密码字段标题。 */
  passwordLabel?: string
  /** 密码输入提示。 */
  passwordPlaceholder?: string
  /** 确认按钮文案。 */
  confirmText?: string
  /** 取消按钮文案。 */
  cancelText?: string
  /** 外部业务提交状态。 */
  confirmLoading?: boolean
  /** 密码加密场景。 */
  scene?: PasswordCryptoScene
  /** 是否允许点击遮罩关闭。 */
  closeOnClickModal?: boolean
}

const props = withDefaults(defineProps<PasswordVerifyDialogProps>(), {
  title: '',
  description: '',
  passwordLabel: '',
  passwordPlaceholder: '',
  confirmText: '',
  cancelText: '',
  confirmLoading: false,
  scene: PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
  closeOnClickModal: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: [password: PasswordCrypto]
  cancel: []
  closed: []
}>()

const password = ref('')
const encrypting = ref(false)
const errorMessage = ref('')
const submitting = computed(() => props.confirmLoading || encrypting.value)
const dialogTitle = computed(() => props.title || t('core.password.verify_title'))
const passwordLabel = computed(() => props.passwordLabel || t('core.password.verify_label'))
const passwordPlaceholder = computed(
  () => props.passwordPlaceholder || t('core.crypto.password_required'),
)
const confirmText = computed(() => props.confirmText || t('common.action.confirm'))
const cancelText = computed(() => props.cancelText || t('common.action.cancel'))

/** 同步弹窗显示状态到外部。 */
function handleVisibleChange(value: boolean) {
  emit('update:modelValue', value)
  if (!value) {
    password.value = ''
    errorMessage.value = ''
    emit('closed')
  }
}

/** 点击遮罩时按配置关闭弹窗。 */
function handleMaskTap() {
  if (props.closeOnClickModal && !submitting.value) handleVisibleChange(false)
}

/** 校验密码并向业务方返回加密结果。 */
async function handleConfirm() {
  if (submitting.value) return
  if (!password.value.trim()) {
    errorMessage.value = t('core.crypto.password_required')
    return
  }
  errorMessage.value = ''
  encrypting.value = true
  try {
    const encryptedPassword = await encryptPassword(password.value, props.scene)
    emit('confirm', encryptedPassword)
  } finally {
    encrypting.value = false
  }
}

/** 关闭弹窗并清理输入内容。 */
function handleCancel() {
  if (submitting.value) return
  handleVisibleChange(false)
  emit('cancel')
}

watch(
  () => props.modelValue,
  (value) => {
    if (value) {
      errorMessage.value = ''
      return
    }
    password.value = ''
    errorMessage.value = ''
  },
)
</script>

<template>
  <view v-if="modelValue" class="password-verify-dialog" @tap="handleMaskTap">
    <view class="password-verify-dialog__panel" @tap.stop>
      <view class="password-verify-dialog__header">
        <text class="password-verify-dialog__title">{{ dialogTitle }}</text>
        <text class="password-verify-dialog__close" @tap="handleCancel">×</text>
      </view>
      <view class="password-verify-dialog__body">
        <text v-if="description" class="password-verify-dialog__description">{{
          description
        }}</text>
        <text class="password-verify-dialog__label">{{ passwordLabel }}</text>
        <input
          v-model="password"
          class="password-verify-dialog__input"
          password
          :placeholder="passwordPlaceholder"
          autocomplete="current-password"
          @confirm="handleConfirm"
          @input="errorMessage = ''"
        />
        <text v-if="errorMessage" class="password-verify-dialog__error">{{ errorMessage }}</text>
      </view>
      <view class="password-verify-dialog__footer">
        <button
          class="password-verify-dialog__button password-verify-dialog__button--cancel"
          :disabled="submitting"
          @tap="handleCancel"
        >
          {{ cancelText }}
        </button>
        <button
          class="password-verify-dialog__button password-verify-dialog__button--confirm"
          :loading="submitting"
          @tap="handleConfirm"
        >
          {{ confirmText }}
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.password-verify-dialog {
  position: fixed;
  z-index: 1000;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32rpx;
  background-color: rgb(0 0 0 / 45%);
}

.password-verify-dialog__panel {
  width: min(100%, 640rpx);
  overflow: hidden;
  background-color: #fff;
  border-radius: 16rpx;
  box-shadow: 0 20rpx 64rpx rgb(0 0 0 / 20%);
}

.password-verify-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #eee;
}

.password-verify-dialog__title {
  color: #222;
  font-size: 34rpx;
  font-weight: 600;
}

.password-verify-dialog__close {
  padding: 0 8rpx;
  color: #999;
  font-size: 42rpx;
  line-height: 1;
}

.password-verify-dialog__body {
  padding: 32rpx;
}

.password-verify-dialog__description {
  display: block;
  margin-bottom: 24rpx;
  color: #666;
  font-size: 26rpx;
  line-height: 1.6;
}

.password-verify-dialog__label {
  display: block;
  margin-bottom: 12rpx;
  color: #333;
  font-size: 28rpx;
}

.password-verify-dialog__input {
  box-sizing: border-box;
  width: 100%;
  height: 84rpx;
  padding: 0 24rpx;
  color: #222;
  font-size: 30rpx;
  border: 1rpx solid #ddd;
  border-radius: 8rpx;
}

.password-verify-dialog__input:focus {
  border-color: #27ba9b;
}

.password-verify-dialog__error {
  display: block;
  margin-top: 10rpx;
  color: #e64340;
  font-size: 24rpx;
}

.password-verify-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 20rpx;
  padding: 0 32rpx 32rpx;
}

.password-verify-dialog__button {
  min-width: 160rpx;
  height: 72rpx;
  margin: 0;
  padding: 0 24rpx;
  font-size: 28rpx;
  line-height: 72rpx;
  border: 0;
  border-radius: 8rpx;
}

.password-verify-dialog__button::after {
  border: 0;
}

.password-verify-dialog__button--cancel {
  color: #555;
  background-color: #f3f3f3;
}

.password-verify-dialog__button--confirm {
  color: #fff;
  background-color: #27ba9b;
}

.password-verify-dialog__button[disabled] {
  opacity: 0.6;
}
</style>
