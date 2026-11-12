import uniPlugin from '@dcloudio/vite-plugin-uni'
import { defineConfig, loadEnv } from 'vite'
import type { ConfigEnv, Plugin, UserConfig } from 'vite'
import { viteMessage } from './vite-messages.js'

export { defineConfig, loadEnv, viteMessage }
export type { ConfigEnv, UserConfig } from 'vite'

/** 创建 uni-app x 官方 Vite 插件。 */
export function createKratosUniPlugin(): Plugin | Plugin[] {
  const createPlugin =
    typeof uniPlugin === 'function'
      ? uniPlugin
      : (uniPlugin as unknown as { default: typeof uniPlugin }).default
  return createPlugin()
}
