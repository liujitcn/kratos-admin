import { readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import uniPlugin from '@dcloudio/vite-plugin-uni'
import { defineConfig, loadEnv } from 'vite'
import type { ConfigEnv, Plugin, UserConfig } from 'vite'
import { viteMessage } from './vite-messages.js'

export { defineConfig, loadEnv, viteMessage }
export type { ConfigEnv, UserConfig } from 'vite'

/**
 * 给微信小程序产物里的全部自定义组件注入 styleIsolation。
 *
 * uni-app x 编译后页面内容都位于自定义组件内，微信默认 isolated 隔离会让
 * app.wxss 的全局 .k-* 视觉类和 .theme-* 主题变量无法作用于组件内部，
 * 表现为小程序端整页样式崩坏；apply-shared 与 H5/Taro 端行为对齐。
 */
function createStyleIsolationPlugin(): Plugin | undefined {
  const outDir = process.env.UNI_OUTPUT_DIR
  // 只处理微信小程序产物；H5/App 的输出目录不走该后处理。
  if (!outDir || !outDir.includes('mp-weixin')) {
    return undefined
  }
  const patchJson = (file: string): boolean => {
    let data: Record<string, unknown>
    try {
      data = JSON.parse(readFileSync(file, 'utf8'))
    } catch {
      return false
    }
    if (data.component !== true || data.styleIsolation) {
      return false
    }
    // 保持键顺序：styleIsolation 紧跟 component 声明。
    const ordered: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(data)) {
      ordered[key] = value
      if (key === 'component') {
        ordered.styleIsolation = 'apply-shared'
      }
    }
    writeFileSync(file, `${JSON.stringify(ordered, null, 2)}\n`)
    return true
  }
  const walk = (dir: string, patched: string[]): void => {
    for (const entry of readdirSync(dir)) {
      const full = join(dir, entry)
      if (statSync(full).isDirectory()) {
        walk(full, patched)
      } else if (entry.endsWith('.json') && patchJson(full)) {
        patched.push(full)
      }
    }
  }
  return {
    name: 'kratos-uni-style-isolation',
    apply: 'build',
    closeBundle() {
      const patched: string[] = []
      walk(outDir, patched)
      if (patched.length) {
        console.log(`[kratos-uni-style-isolation] patched ${patched.length} component json files`)
      }
    },
  }
}

/** 创建 uni-app x 官方 Vite 插件。 */
export function createKratosUniPlugin(): Plugin | Plugin[] {
  const createPlugin =
    typeof uniPlugin === 'function'
      ? uniPlugin
      : (uniPlugin as unknown as { default: typeof uniPlugin }).default
  const uni = createPlugin()
  const styleIsolation = createStyleIsolationPlugin()
  if (!styleIsolation) {
    return uni
  }
  // uni 官方插件可能返回插件数组，统一拍平后追加产物后处理插件。
  const list = Array.isArray(uni) ? uni : [uni]
  return [...list, styleIsolation]
}
