import { Button, Image, Text, View } from '@tarojs/components'
import Taro from '@tarojs/taro'
import generateQrCode from 'qrcode-generator'
import { useMemo } from 'react'
import { useI18n } from '../locales'
import './MfaSetupPanel.scss'

/** MFA TOTP 绑定面板属性。 */
export interface MfaSetupPanelProps {
  /** TOTP 绑定地址。 */
  uri: string
}

/** 统一展示 TOTP 二维码、绑定地址和复制操作。 */
export default function MfaSetupPanel({ uri }: MfaSetupPanelProps) {
  const { t } = useI18n()
  const qrDataUrl = useMemo(() => {
    if (!uri) return ''
    const code = generateQrCode(0, 'M')
    code.addData(uri)
    code.make()
    return code.createDataURL(4, 8)
  }, [uri])

  if (!uri) return null

  /** 复制 TOTP 绑定地址。 */
  const copySetupUri = async () => {
    await Taro.setClipboardData({ data: uri })
    await Taro.showToast({ icon: 'none', title: t('core.login.mfa_copy_success') })
  }

  return (
    <View className='mfa-setup-panel'>
      {qrDataUrl ? <Image className='mfa-setup-panel__qr' src={qrDataUrl} mode='aspectFit' /> : null}
      <View className='mfa-setup-panel__uri'>
        <Text className='mfa-setup-panel__uri-text'>{uri}</Text>
        <Button
          className='mfa-setup-panel__copy'
          aria-label={t('core.login.mfa_copy_uri')}
          onClick={() => void copySetupUri()}
        >
          <View className='mfa-setup-panel__copy-icon' aria-hidden='true'>
            <View className='mfa-setup-panel__copy-back' />
            <View className='mfa-setup-panel__copy-front' />
          </View>
        </Button>
      </View>
    </View>
  )
}
