import { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import BootstrapStatus from '../../../components/BootstrapStatus'
import {
  initializeAppNavigation,
  launchAppStatus,
  navigateAppRoute,
} from '../../../navigation'
import type { BootstrapViewKey } from '../../../module'
import { useI18n } from '../../../locales'
import { useSettingStore } from '../../../stores'

/** 启动与错误状态页。 */
export default function StatusPage() {
  const { t } = useI18n()
  const [state, setState] = useState<BootstrapViewKey>('BOOTSTRAP_LOADING')
  const [detail, setDetail] = useState('')

  useLoad((options) => {
    const nextState = (options?.state as BootstrapViewKey | undefined) ?? 'BOOTSTRAP_LOADING'
    setState(nextState)
    setDetail(options?.detail ? decodeURIComponent(options.detail) : '')
    if (options?.bootstrap !== '1') return
    void useSettingStore
      .getState()
      .loadData()
      .catch(() => undefined)
      .then(() => initializeAppNavigation())
      .then(() => {
        navigateAppRoute(options?.route ? decodeURIComponent(options.route) : 'app/home', {
          replace: true,
        })
      })
      .catch((error: unknown) => {
        launchAppStatus('CONFIG_ERROR', resolveErrorMessage(error))
      })
  })

  const title = state === 'BOOTSTRAP_LOADING' ? t('core.status.loading') : t(`core.status.${state}`)
  return <BootstrapStatus title={title} detail={detail} />
}

/** 将请求错误转换为可展示的文本，避免直接显示 Object。 */
function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message || '')
  }
  return ''
}
