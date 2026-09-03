import { Button, Input, Text, View } from '@tarojs/components'
import { useEffect, useState } from 'react'
import { useI18n } from '@liujitcn/kratos-taro-app-core'
import MfaRecoveryCodesDialog from '@liujitcn/kratos-taro-app-core/components/MfaRecoveryCodesDialog'
import MfaSetupPanel from '@liujitcn/kratos-taro-app-core/components/MfaSetupPanel'
import { defMfaService } from '@liujitcn/kratos-taro-app-core/api/base/v1/mfa'
import {
  createWebAuthnCredential,
  getWebAuthnAssertion,
} from '@liujitcn/kratos-taro-app-core/utils/webauthn'
import {
  PASSWORD_CRYPTO_SCENE,
  encryptPassword,
} from '@liujitcn/kratos-taro-app-core/utils/passwordCrypto'
import './PasswordVerifyDialog.scss'

/** MFA 通用操作弹窗属性。 */
export interface PasswordVerifyDialogProps {
  /** 弹窗显示状态。 */
  open: boolean
  /** MFA 操作模式，默认执行绑定。 */
  mode?: 'setup' | 'disable'
  /** 当前已启用的 MFA 方式，禁用模式使用。 */
  method?: string
  /** 绑定成功回调。 */
  onSuccess?: (method: string) => void | Promise<void>
  /** 禁用成功回调。 */
  onDisabled?: () => void | Promise<void>
  /** 关闭弹窗。 */
  onClose: () => void
}

/** 应用端 MFA 绑定与禁用流程弹窗。 */
export default function PasswordVerifyDialog({
  open,
  mode = 'setup',
  method = 'totp',
  onSuccess,
  onDisabled,
  onClose,
}: PasswordVerifyDialogProps) {
  const { t } = useI18n()
  const [setupDialogOpen, setSetupDialogOpen] = useState(false)
  const [recoveryDialogOpen, setRecoveryDialogOpen] = useState(false)
  const [disableDialogOpen, setDisableDialogOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [setupTicket, setSetupTicket] = useState('')
  const [setupMethod, setSetupMethod] = useState('totp')
  const [setupUri, setSetupUri] = useState('')
  const [setupWebAuthnOptionsJson, setSetupWebAuthnOptionsJson] = useState('')
  const [setupPasswordInput, setSetupPasswordInput] = useState('')
  const [setupCode, setSetupCode] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [disablePasswordInput, setDisablePasswordInput] = useState('')
  const [disableCode, setDisableCode] = useState('')
  const [disableRecoveryCode, setDisableRecoveryCode] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (open) {
      setSetupDialogOpen(mode === 'setup')
      setRecoveryDialogOpen(false)
      setDisableDialogOpen(mode === 'disable')
      setErrorMessage('')
      return
    }
    setSetupDialogOpen(false)
    setRecoveryDialogOpen(false)
    setDisableDialogOpen(false)
    setSetupTicket('')
    setSetupMethod('totp')
    setSetupUri('')
    setSetupWebAuthnOptionsJson('')
    setSetupPasswordInput('')
    setSetupCode('')
    setRecoveryCodes([])
    setDisablePasswordInput('')
    setDisableCode('')
    setDisableRecoveryCode('')
    setErrorMessage('')
  }, [open, mode])

  /** 关闭 MFA 操作流程并清理临时状态。 */
  const closeDialog = () => {
    if (loading) return
    onClose()
  }

  /** 在同一弹窗内校验密码并开始 MFA 绑定。 */
  const beginMfaSetup = async () => {
    if (!setupPasswordInput.trim()) {
      setErrorMessage(t('core.crypto.password_required'))
      return
    }
    setLoading(true)
    setErrorMessage('')
    try {
      const password = await encryptPassword(
        setupPasswordInput,
        PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
      )
      const result = await defMfaService.BeginMfaSetup({ password, setup_ticket: '' })
      setSetupTicket(result.setup_ticket)
      setSetupMethod(result.method || 'totp')
      setSetupUri(result.otpauth_uri || '')
      setSetupWebAuthnOptionsJson(result.webauthn_options_json || '')
      setSetupCode('')
      setSetupPasswordInput('')
    } finally {
      setLoading(false)
    }
  }

  /** 根据绑定阶段处理密码校验或 MFA 确认。 */
  const handleSetupAction = async () => {
    if (loading) return
    if (!setupTicket) {
      await beginMfaSetup()
      return
    }
    await handleSetupConfirm()
  }

  /** 校验 MFA 因素并禁用当前用户的多因素认证。 */
  const handleDisableConfirm = async () => {
    if (loading) return
    if (!disablePasswordInput.trim()) {
      setErrorMessage(t('core.crypto.password_required'))
      return
    }
    if (method !== 'webauthn' && !disableCode.trim() && !disableRecoveryCode.trim()) {
      setErrorMessage(t('system.settings.mfa_disable_factor_required'))
      return
    }
    setLoading(true)
    setErrorMessage('')
    try {
      const password = await encryptPassword(
        disablePasswordInput,
        PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
      )
      if (disableRecoveryCode.trim()) {
        await defMfaService.DisableMfa({
          password,
          code: '',
          webauthn_challenge_id: '',
          webauthn_response_json: '',
          recovery_code: disableRecoveryCode,
        })
      } else if (method === 'webauthn') {
        const challenge = await defMfaService.BeginMfaDisable({})
        const webauthnResponseJson = await getWebAuthnAssertion(challenge.webauthn_options_json)
        await defMfaService.DisableMfa({
          password,
          code: '',
          webauthn_challenge_id: challenge.challenge_id,
          webauthn_response_json: webauthnResponseJson,
          recovery_code: '',
        })
      } else {
        await defMfaService.DisableMfa({
          password,
          code: disableCode,
          webauthn_challenge_id: '',
          webauthn_response_json: '',
          recovery_code: '',
        })
      }
      setDisableDialogOpen(false)
      await onDisabled?.()
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('common.message.request_error'))
    } finally {
      setLoading(false)
    }
  }

  /** 确认绑定 MFA 并展示一次性恢复码。 */
  const handleSetupConfirm = async () => {
    if (loading) return
    if (setupMethod !== 'webauthn' && !setupCode.trim()) {
      setErrorMessage(t('core.login.mfa_code'))
      return
    }
    setLoading(true)
    setErrorMessage('')
    try {
      const webauthnResponseJson =
        setupMethod === 'webauthn' ? await createWebAuthnCredential(setupWebAuthnOptionsJson) : ''
      const result = await defMfaService.ConfirmMfaSetup({
        setup_ticket: setupTicket,
        code: setupCode,
        webauthn_response_json: webauthnResponseJson,
      })
      setRecoveryCodes(result.recovery_codes || [])
      setSetupDialogOpen(false)
      setRecoveryDialogOpen(true)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('common.message.request_error'))
    } finally {
      setLoading(false)
    }
  }

  /** 确认已保存恢复码。 */
  const finishBinding = async () => {
    await onSuccess?.(setupMethod)
  }

  return (
    <>
      {setupDialogOpen ? (
        <View className='mfa-bind-dialog' onClick={closeDialog}>
          <View className='mfa-bind-dialog__panel' onClick={(event) => event.stopPropagation()}>
            <View className='mfa-bind-dialog__header'>
              <Text className='mfa-bind-dialog__title'>
                {setupTicket ? t('core.login.mfa_setup_title') : t('core.password.verify_title')}
              </Text>
              <Text className='mfa-bind-dialog__close' onClick={closeDialog}>
                ×
              </Text>
            </View>
            <View className='mfa-bind-dialog__body'>
              {!setupTicket ? (
                <>
                  <Text className='mfa-bind-dialog__label'>{t('core.password.verify_label')}</Text>
                  <Input
                    className='mfa-bind-dialog__input'
                    password
                    value={setupPasswordInput}
                    placeholder={t('core.crypto.password_required')}
                    onInput={(event) => {
                      setSetupPasswordInput(event.detail.value)
                      setErrorMessage('')
                    }}
                    onConfirm={() => void handleSetupAction()}
                  />
                </>
              ) : (
                <>
                  <MfaSetupPanel uri={setupUri} />
                  {setupMethod !== 'webauthn' ? (
                    <Input
                      className='mfa-bind-dialog__input'
                      type='number'
                      maxlength={8}
                      value={setupCode}
                      placeholder={t('core.login.mfa_code')}
                      onInput={(event) => setSetupCode(event.detail.value)}
                      onConfirm={() => void handleSetupAction()}
                    />
                  ) : null}
                </>
              )}
              {errorMessage ? <Text className='mfa-bind-dialog__error'>{errorMessage}</Text> : null}
            </View>
            <View className='mfa-bind-dialog__footer'>
              <Button
                className='mfa-bind-dialog__button mfa-bind-dialog__button--cancel'
                disabled={loading}
                onClick={closeDialog}
              >
                {t('common.action.cancel')}
              </Button>
              <Button
                className='mfa-bind-dialog__button mfa-bind-dialog__button--confirm'
                loading={loading}
                onClick={() => void handleSetupAction()}
              >
                {!setupTicket
                  ? t('common.action.confirm')
                  : setupMethod === 'webauthn'
                  ? t('core.login.mfa_webauthn_action')
                  : t('common.action.confirm')}
              </Button>
            </View>
          </View>
        </View>
      ) : null}

      {disableDialogOpen ? (
        <View className='mfa-bind-dialog' onClick={closeDialog}>
          <View className='mfa-bind-dialog__panel' onClick={(event) => event.stopPropagation()}>
            <View className='mfa-bind-dialog__header'>
              <Text className='mfa-bind-dialog__title'>
                {t('system.settings.mfa_disable_title')}
              </Text>
              <Text className='mfa-bind-dialog__close' onClick={closeDialog}>
                ×
              </Text>
            </View>
            <View className='mfa-bind-dialog__body'>
              <Text className='mfa-bind-dialog__label'>{t('core.password.verify_label')}</Text>
              <Input
                className='mfa-bind-dialog__input'
                password
                value={disablePasswordInput}
                placeholder={t('core.crypto.password_required')}
                onInput={(event) => {
                  setDisablePasswordInput(event.detail.value)
                  setErrorMessage('')
                }}
                onConfirm={() => void handleDisableConfirm()}
              />
              {method !== 'webauthn' ? (
                <Input
                  className='mfa-bind-dialog__input'
                  type='number'
                  maxlength={8}
                  value={disableCode}
                  placeholder={t('core.login.mfa_code')}
                  onInput={(event) => {
                    setDisableCode(event.detail.value)
                    setErrorMessage('')
                  }}
                  onConfirm={() => void handleDisableConfirm()}
                />
              ) : null}
              <Input
                className='mfa-bind-dialog__input'
                value={disableRecoveryCode}
                placeholder={t('core.login.mfa_recovery_code')}
                onInput={(event) => {
                  setDisableRecoveryCode(event.detail.value)
                  setErrorMessage('')
                }}
                onConfirm={() => void handleDisableConfirm()}
              />
              {errorMessage ? (
                <Text className='mfa-bind-dialog__error'>{errorMessage}</Text>
              ) : null}
            </View>
            <View className='mfa-bind-dialog__footer'>
              <Button
                className='mfa-bind-dialog__button mfa-bind-dialog__button--cancel'
                disabled={loading}
                onClick={closeDialog}
              >
                {t('common.action.cancel')}
              </Button>
              <Button
                className='mfa-bind-dialog__button mfa-bind-dialog__button--confirm mfa-bind-dialog__button--danger'
                loading={loading}
                onClick={() => void handleDisableConfirm()}
              >
                {t('system.settings.mfa_disable_action')}
              </Button>
            </View>
          </View>
        </View>
      ) : null}

      <MfaRecoveryCodesDialog
        open={recoveryDialogOpen}
        codes={recoveryCodes}
        onConfirm={finishBinding}
      />
    </>
  )
}
