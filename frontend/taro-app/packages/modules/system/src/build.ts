import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineKratosTaroBuildModule } from '@liujitcn/kratos-taro-app-core/build'
import { systemPages } from './pages'

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

/** system 构建期模块描述。 */
export const buildModule = defineKratosTaroBuildModule({
  name: '@liujitcn/kratos-taro-app-system',
  root: packageRoot,
  pages: systemPages,
})
