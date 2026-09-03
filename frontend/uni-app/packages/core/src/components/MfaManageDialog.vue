<script setup lang="ts">
import { ref, watch } from 'vue'
import MfaRecoveryCodesDialog from './MfaRecoveryCodesDialog.vue'
import MfaSetupPanel from './MfaSetupPanel.vue'
import type {
  BeginMfaSetupRequest,
  ConfirmMfaSetupRequest,
  DisableMfaRequest,
} from '../rpc/base/v1/mfa'
import { createWebAuthnCredential, getWebAuthnAssertion } from '../utils/webauthn'
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from '../utils/passwordCrypto'
import { useI18n } from '../locales'
import { defMfaService } from '../api/base/v1/mfa'

const props = withDefaults(
  defineProps<{
    /** 弹窗显示状态。 */
    modelValue: boolean
    /** MFA 操作模式，默认执行绑定。 */
    mode?: 'setup' | 'disable'
    /** 当前已启用的 MFA 方式，禁用模式使用。 */
    method?: string
  }>(),
  { modelValue: false, mode: 'setup', method: 'totp' },
)

defineOptions({ name: 'MfaManageDialog' })

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: [method: string]
  disabled: []
}>()

const { t } = useI18n()
const setupDialogVisible = ref(false)
const recoveryDialogVisible = ref(false)
const disableDialogVisible = ref(false)
const loading = ref(false)
const setupTicket = ref('')
const setupMethod = ref('totp')
const setupUri = ref('')
const setupWebAuthnOptionsJson = ref('')
const setupPasswordInput = ref('')
const setupCode = ref('')
const recoveryCodes = ref<string[]>([])
const disablePasswordInput = ref('')
const disableCode = ref('')
const disableRecoveryCode = ref('')
const errorMessage = ref('')

/** 根据操作模式打开 MFA 流程。 */
function openDialog() {
  errorMessage.value = ''
  if (props.mode === 'disable') {
    setupDialogVisible.value = false
    recoveryDialogVisible.value = false
    disableDialogVisible.value = true
    return
  }
  setupDialogVisible.value = true
  recoveryDialogVisible.value = false
  disableDialogVisible.value = false
}

/** 关闭所有 MFA 弹窗并清理临时状态。 */
function closeDialog() {
  if (loading.value) return
  setupDialogVisible.value = false
  recoveryDialogVisible.value = false
  disableDialogVisible.value = false
  setupTicket.value = ''
  setupMethod.value = 'totp'
  setupUri.value = ''
  setupWebAuthnOptionsJson.value = ''
  setupPasswordInput.value = ''
  setupCode.value = ''
  recoveryCodes.value = []
  disablePasswordInput.value = ''
  disableCode.value = ''
  disableRecoveryCode.value = ''
  errorMessage.value = ''
  emit('update:modelValue', false)
}

/** 在同一弹窗内校验密码并开始 MFA 绑定。 */
async function beginMfaSetup() {
  if (!setupPasswordInput.value.trim()) {
    errorMessage.value = t('core.crypto.password_required')
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const password = await encryptPassword(
      setupPasswordInput.value,
      PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
    )
    const request: BeginMfaSetupRequest = { password, setup_ticket: '' }
    const result = await defMfaService.BeginMfaSetup(request)
    setupTicket.value = result.setup_ticket
    setupMethod.value = result.method || 'totp'
    setupUri.value = result.otpauth_uri || ''
    setupWebAuthnOptionsJson.value = result.webauthn_options_json || ''
    setupCode.value = ''
    setupPasswordInput.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('common.message.request_error')
  } finally {
    loading.value = false
  }
}

/** 根据绑定阶段处理密码校验或 MFA 确认。 */
async function handleSetupAction() {
  if (loading.value) return
  if (!setupTicket.value) {
    await beginMfaSetup()
    return
  }
  await handleSetupConfirm()
}

/** 校验 MFA 因素并禁用当前用户的多因素认证。 */
async function handleDisableConfirm() {
  if (loading.value) return
  if (!disablePasswordInput.value.trim()) {
    errorMessage.value = t('core.crypto.password_required')
    return
  }
  const method = props.method === 'webauthn' ? 'webauthn' : 'totp'
  if (method === 'totp' && !disableCode.value.trim() && !disableRecoveryCode.value.trim()) {
    errorMessage.value = t('core.settings.mfa_disable_factor_required')
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const password = await encryptPassword(
      disablePasswordInput.value,
      PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
    )
    let request: DisableMfaRequest
    if (disableRecoveryCode.value.trim()) {
      request = {
        password,
        code: '',
        webauthn_challenge_id: '',
        webauthn_response_json: '',
        recovery_code: disableRecoveryCode.value,
      }
    } else if (method === 'webauthn') {
      const challenge = await defMfaService.BeginMfaDisable({})
      const webauthnResponseJson = await getWebAuthnAssertion(challenge.webauthn_options_json)
      request = {
        password,
        code: '',
        webauthn_challenge_id: challenge.challenge_id,
        webauthn_response_json: webauthnResponseJson,
        recovery_code: '',
      }
    } else {
      request = {
        password,
        code: disableCode.value,
        webauthn_challenge_id: '',
        webauthn_response_json: '',
        recovery_code: '',
      }
    }
    await defMfaService.DisableMfa(request)
    loading.value = false
    closeDialog()
    emit('disabled')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('common.message.request_error')
  } finally {
    loading.value = false
  }
}

/** 确认 MFA 绑定并展示一次性恢复码。 */
async function handleSetupConfirm() {
  if (loading.value) return
  if (setupMethod.value !== 'webauthn' && !setupCode.value.trim()) {
    errorMessage.value = t('core.login.mfa_code')
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const webauthnResponseJson =
      setupMethod.value === 'webauthn'
        ? await createWebAuthnCredential(setupWebAuthnOptionsJson.value)
        : ''
    const request: ConfirmMfaSetupRequest = {
      setup_ticket: setupTicket.value,
      code: setupCode.value,
      webauthn_response_json: webauthnResponseJson,
    }
    const result = await defMfaService.ConfirmMfaSetup(request)
    recoveryCodes.value = result.recovery_codes || []
    setupDialogVisible.value = false
    recoveryDialogVisible.value = true
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : t('common.message.request_error')
  } finally {
    loading.value = false
  }
}

/** 确认已保存恢复码并通知业务方刷新状态。 */
function finishBinding() {
  const method = setupMethod.value
  closeDialog()
  emit('success', method)
}

watch(
  () => props.modelValue,
  (value) => {
    if (value) {
      openDialog()
      return
    }
    closeDialog()
  },
)
</script>

<template>
  <view v-if="setupDialogVisible" class="mfa-bind-dialog" @tap="closeDialog">
    <view class="mfa-bind-dialog__panel" @tap.stop>
      <view class="mfa-bind-dialog__header">
        <text class="mfa-bind-dialog__title">
          {{ setupTicket ? t('core.login.mfa_setup_title') : t('core.password.verify_title') }}
        </text>
        <text class="mfa-bind-dialog__close" @tap="closeDialog">×</text>
      </view>
      <view class="mfa-bind-dialog__body">
        <template v-if="!setupTicket">
          <text class="mfa-bind-dialog__label">{{ t('core.password.verify_label') }}</text>
          <input
            v-model="setupPasswordInput"
            class="mfa-bind-dialog__input"
            password
            autocomplete="current-password"
            :placeholder="t('core.crypto.password_required')"
            @input="errorMessage = ''"
            @confirm="handleSetupAction"
          />
        </template>
        <template v-else>
          <MfaSetupPanel :uri="setupUri" />
          <input
            v-if="setupMethod !== 'webauthn'"
            v-model="setupCode"
            class="mfa-bind-dialog__input"
            type="number"
            maxlength="8"
            :placeholder="t('core.login.mfa_code')"
            @confirm="handleSetupAction"
          />
        </template>
        <text v-if="errorMessage" class="mfa-bind-dialog__error">{{ errorMessage }}</text>
      </view>
      <view class="mfa-bind-dialog__footer">
        <button
          class="mfa-bind-dialog__button mfa-bind-dialog__button--cancel"
          :disabled="loading"
          @tap="closeDialog"
        >
          {{ t('common.action.cancel') }}
        </button>
        <button
          class="mfa-bind-dialog__button mfa-bind-dialog__button--confirm"
          :loading="loading"
          @tap="handleSetupAction"
        >
          {{
            !setupTicket
              ? t('common.action.confirm')
              : setupMethod === 'webauthn'
                ? t('core.login.mfa_webauthn_action')
                : t('common.action.confirm')
          }}
        </button>
      </view>
    </view>
  </view>

  <view v-if="disableDialogVisible" class="mfa-bind-dialog" @tap="closeDialog">
    <view class="mfa-bind-dialog__panel" @tap.stop>
      <view class="mfa-bind-dialog__header">
        <text class="mfa-bind-dialog__title">{{ t('core.settings.mfa_disable_title') }}</text>
        <text class="mfa-bind-dialog__close" @tap="closeDialog">×</text>
      </view>
      <view class="mfa-bind-dialog__body">
        <text class="mfa-bind-dialog__label">{{ t('core.password.verify_label') }}</text>
        <input
          v-model="disablePasswordInput"
          class="mfa-bind-dialog__input"
          password
          :placeholder="t('core.crypto.password_required')"
          autocomplete="current-password"
          @input="errorMessage = ''"
          @confirm="handleDisableConfirm"
        />
        <input
          v-if="props.method !== 'webauthn'"
          v-model="disableCode"
          class="mfa-bind-dialog__input"
          type="number"
          maxlength="8"
          :placeholder="t('core.login.mfa_code')"
          @input="errorMessage = ''"
          @confirm="handleDisableConfirm"
        />
        <input
          v-model="disableRecoveryCode"
          class="mfa-bind-dialog__input"
          :placeholder="t('core.login.mfa_recovery_code')"
          @input="errorMessage = ''"
          @confirm="handleDisableConfirm"
        />
        <text v-if="errorMessage" class="mfa-bind-dialog__error">{{ errorMessage }}</text>
      </view>
      <view class="mfa-bind-dialog__footer">
        <button
          class="mfa-bind-dialog__button mfa-bind-dialog__button--cancel"
          :disabled="loading"
          @tap="closeDialog"
        >
          {{ t('common.action.cancel') }}
        </button>
        <button
          class="mfa-bind-dialog__button mfa-bind-dialog__button--confirm mfa-bind-dialog__button--danger"
          :loading="loading"
          @tap="handleDisableConfirm"
        >
          {{ t('core.settings.mfa_disable_action') }}
        </button>
      </view>
    </view>
  </view>

  <MfaRecoveryCodesDialog
    v-model="recoveryDialogVisible"
    :codes="recoveryCodes"
    @confirm="finishBinding"
  />
</template>

<style scoped lang="scss">
.mfa-bind-dialog {
  position: fixed;
  z-index: 1000;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32rpx;
  background-color: rgb(0 0 0 / 45%);
}

.mfa-bind-dialog__panel {
  width: min(100%, 680rpx);
  overflow: hidden;
  background-color: #fff;
  border-radius: 16rpx;
  box-shadow: 0 20rpx 64rpx rgb(0 0 0 / 20%);
}

.mfa-bind-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #eee;
}

.mfa-bind-dialog__title {
  color: #222;
  font-size: 34rpx;
  font-weight: 600;
}

.mfa-bind-dialog__close {
  color: #999;
  font-size: 42rpx;
  line-height: 1;
}

.mfa-bind-dialog__body {
  padding: 32rpx;
}

.mfa-bind-dialog__label {
  display: block;
  margin-bottom: 12rpx;
  color: #333;
  font-size: 28rpx;
}

.mfa-bind-dialog__input {
  box-sizing: border-box;
  width: 100%;
  padding: 20rpx 24rpx;
  color: #222;
  font-size: 30rpx;
  border: 1rpx solid #ddd;
  border-radius: 8rpx;
}

.mfa-bind-dialog__input {
  height: 84rpx;
  margin-bottom: 16rpx;
}

.mfa-bind-dialog__error {
  display: block;
  margin-top: 12rpx;
  font-size: 24rpx;
  line-height: 1.5;
}

.mfa-bind-dialog__error {
  color: #e64340;
}

.mfa-bind-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 20rpx;
  padding: 0 32rpx 32rpx;
}

.mfa-bind-dialog__button {
  min-width: 160rpx;
  height: 72rpx;
  margin: 0;
  padding: 0 24rpx;
  font-size: 28rpx;
  line-height: 72rpx;
  border: 0;
  border-radius: 8rpx;
}

.mfa-bind-dialog__button::after {
  border: 0;
}

.mfa-bind-dialog__button--cancel {
  color: #555;
  background-color: #f3f3f3;
}

.mfa-bind-dialog__button--confirm {
  color: #fff;
  background-color: #27ba9b;
}

.mfa-bind-dialog__button--danger {
  background-color: #e64340;
}
</style>
