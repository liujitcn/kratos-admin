import { useLoad } from '@tarojs/taro'
import { useState } from 'react'
import BootstrapStatus from '../../../components/BootstrapStatus'
import {
  initializeAppNavigation,
  launchAppStatus,
  navigateAppRoute,
} from '../../../navigation'
import type { BootstrapViewKey } from '../../../module'

const stateTitles: Record<BootstrapViewKey, string> = {
  BOOTSTRAP_LOADING: '正在加载',
  FORBIDDEN: '暂无访问权限',
  NOT_FOUND: '页面不存在',
  OFFLINE: '网络不可用',
  CONFIG_ERROR: '导航配置无效',
  PAGE_UNAVAILABLE: '页面暂不可用',
}

/** 启动与错误状态页。 */
export default function StatusPage() {
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

  return <BootstrapStatus title={stateTitles[state]} detail={detail} />
}
