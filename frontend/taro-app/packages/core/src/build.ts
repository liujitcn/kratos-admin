import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { corePages } from './pages'

/** 构建期模块描述。 */
export interface KratosTaroBuildModule {
  name: string
  root: string
  pages: Record<string, { route?: string; style?: Record<string, unknown> }>
}

/** 声明构建期模块描述。 */
export function defineKratosTaroBuildModule<T extends KratosTaroBuildModule>(module: T): T {
  return module
}

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

/** core 构建期模块描述。 */
export const buildModule = defineKratosTaroBuildModule({
  name: '@liujitcn/kratos-taro-app-core',
  root: packageRoot,
  pages: corePages,
})
