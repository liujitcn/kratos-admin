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
    void initializeAppNavigation()
      .then(() => {
        navigateAppRoute(options?.route ? decodeURIComponent(options.route) : 'app/home', {
          replace: true,
        })
      })
      .catch((error: unknown) => {
        launchAppStatus('CONFIG_ERROR', error instanceof Error ? error.message : String(error))
      })
  })

  const title = state === 'BOOTSTRAP_LOADING' ? t('core.status.loading') : t(`core.status.${state}`)
  return <BootstrapStatus title={title} detail={detail} />
}
