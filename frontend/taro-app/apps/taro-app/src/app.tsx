import type { PropsWithChildren } from 'react'
import { useLaunch } from '@tarojs/taro'
import { bootstrapKratosTaroApp } from '@liujitcn/kratos-taro-app-core'
import { moduleManifest } from './module-manifest'
import './app.scss'

/** Taro 宿主根组件。 */
export default function App({ children }: PropsWithChildren) {
  useLaunch(() => {
    bootstrapKratosTaroApp({ modules: moduleManifest })
  })
  return children
}
