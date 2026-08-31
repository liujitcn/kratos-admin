import { Button, Input, Text, View } from '@tarojs/components'
import { useEffect, useState } from 'react'
import { useI18n } from '../locales'
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from '../utils/passwordCrypto'
import type { PasswordCryptoScene } from '../utils/passwordCrypto'
import type { PasswordCrypto } from '../rpc/common/v1/types'
import './PasswordVerifyDialog.scss'

/** Taro 通用密码验证弹窗属性。 */
export interface PasswordVerifyDialogProps {
  /** 弹窗显示状态。 */
  open: boolean
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
  /** 是否允许点击遮罩关闭弹窗。 */
  closeOnClickModal?: boolean
  /** 确认并返回加密密码。 */
  onConfirm: (password: PasswordCrypto) => void | Promise<void>
  /** 取消验证。 */
  onCancel: () => void
}

/** 通用密码验证弹窗，负责收集并加密当前密码。 */
export default function PasswordVerifyDialog({
  open,
  title,
  description,
  passwordLabel,
  passwordPlaceholder,
  confirmText,
  cancelText,
  confirmLoading = false,
  scene = PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
  closeOnClickModal = false,
  onConfirm,
  onCancel,
}: PasswordVerifyDialogProps) {
  const { t } = useI18n()
  const [password, setPassword] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [encrypting, setEncrypting] = useState(false)
  const submitting = confirmLoading || encrypting

  useEffect(() => {
    if (!open) {
      setPassword('')
      setErrorMessage('')
    }
  }, [open])

  if (!open) return null

  const handleConfirm = async () => {
    if (submitting) return
    if (!password.trim()) {
      setErrorMessage(t('core.crypto.password_required'))
      return
    }
    setErrorMessage('')
    setEncrypting(true)
    try {
      const encryptedPassword = await encryptPassword(password, scene)
      await onConfirm(encryptedPassword)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('common.message.request_error'))
    } finally {
      setEncrypting(false)
    }
  }

  const handleCancel = () => {
    if (submitting) return
    onCancel()
  }

  const handleMaskClick = () => {
    if (closeOnClickModal) handleCancel()
  }

  return (
    <View className='password-verify-dialog' onClick={handleMaskClick}>
      <View className='password-verify-dialog__panel' onClick={(event) => event.stopPropagation()}>
        <View className='password-verify-dialog__header'>
          <Text className='password-verify-dialog__title'>
            {title || t('core.password.verify_title')}
          </Text>
          <Text className='password-verify-dialog__close' onClick={handleCancel}>
            ×
          </Text>
        </View>
        <View className='password-verify-dialog__body'>
          {description ? (
            <Text className='password-verify-dialog__description'>{description}</Text>
          ) : null}
          <Text className='password-verify-dialog__label'>
            {passwordLabel || t('core.password.verify_label')}
          </Text>
          <Input
            className='password-verify-dialog__input'
            password
            value={password}
            placeholder={passwordPlaceholder || t('core.crypto.password_required')}
            onInput={(event) => {
              setPassword(event.detail.value)
              setErrorMessage('')
            }}
            onConfirm={() => void handleConfirm()}
          />
          {errorMessage ? (
            <Text className='password-verify-dialog__error'>{errorMessage}</Text>
          ) : null}
        </View>
        <View className='password-verify-dialog__footer'>
          <Button
            className='password-verify-dialog__button password-verify-dialog__button--cancel'
            disabled={submitting}
            onClick={handleCancel}
          >
            {cancelText || t('common.action.cancel')}
          </Button>
          <Button
            className='password-verify-dialog__button password-verify-dialog__button--confirm'
            loading={submitting}
            onClick={() => void handleConfirm()}
          >
            {confirmText || t('common.action.confirm')}
          </Button>
        </View>
      </View>
    </View>
  )
}
