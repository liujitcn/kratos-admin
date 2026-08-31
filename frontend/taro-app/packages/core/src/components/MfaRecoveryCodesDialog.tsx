import { Button, Text, View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import { useI18n } from '../locales'
import './MfaRecoveryCodesDialog.scss'

/** MFA 恢复码弹窗属性。 */
export interface MfaRecoveryCodesDialogProps {
  /** 弹窗显示状态。 */
  open: boolean
  /** 本次生成的一次性恢复码。 */
  codes: string[]
  /** 用户确认已保存恢复码。 */
  onConfirm: () => void | Promise<void>
}

/** 统一展示、复制并确认保存 MFA 恢复码。 */
export default function MfaRecoveryCodesDialog({
  open,
  codes,
  onConfirm,
}: MfaRecoveryCodesDialogProps) {
  const { t } = useI18n()
  if (!open) return null

  const codesText = codes.join('\n')

  /** 复制当前显示的恢复码。 */
  const copyRecoveryCodes = async () => {
    if (!codesText) return
    await Taro.setClipboardData({ data: codesText })
    await Taro.showToast({ icon: 'none', title: t('core.login.mfa_recovery_codes_copied') })
  }

  return (
    <View className='mfa-recovery-dialog'>
      <View className='mfa-recovery-dialog__panel' onClick={(event) => event.stopPropagation()}>
        <View className='mfa-recovery-dialog__header'>
          <Text className='mfa-recovery-dialog__title'>
            {t('core.login.mfa_recovery_codes_title')}
          </Text>
        </View>
        <View className='mfa-recovery-dialog__body'>
          <View className='mfa-recovery-dialog__codes'>
            <Text>{codesText}</Text>
          </View>
          <Button className='mfa-recovery-dialog__copy' onClick={() => void copyRecoveryCodes()}>
            {t('core.login.mfa_copy_recovery_codes')}
          </Button>
          <Text className='mfa-recovery-dialog__warning'>
            {t('core.login.mfa_recovery_codes_warning')}
          </Text>
        </View>
        <View className='mfa-recovery-dialog__footer'>
          <Button className='mfa-recovery-dialog__confirm' onClick={() => void onConfirm()}>
            {t('core.login.mfa_recovery_codes_confirm')}
          </Button>
        </View>
      </View>
    </View>
  )
}
