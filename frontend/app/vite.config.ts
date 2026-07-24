import { defineConfig, loadEnv, type ConfigEnv, type UserConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

/**
 * 合并基础环境与 H5 覆盖环境
 */
const resolveEnv = (mode: string) => {
  const modeEnv = loadEnv(mode, process.cwd(), '')
  if (mode === 'development-h5') {
    const baseEnv = loadEnv('development', process.cwd(), '')
    return {
      env: { ...baseEnv, ...modeEnv },
    }
  }
  if (mode === 'production-h5') {
    const baseEnv = loadEnv('production', process.cwd(), '')
    return {
      env: { ...baseEnv, ...modeEnv },
    }
  }
  return {
    env: modeEnv,
  }
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const { env } = resolveEnv(mode)
  const devH5ProxyEnv = mode === 'development-h5' ? loadEnv('development', process.cwd(), '') : env
  /**
   * H5 生产环境默认发布到 /app/ 路径，开发环境保持根路径访问。
   */
  const appBasePath = env.VITE_APP_BASE_PATH || (mode === 'production-h5' ? '/app/' : '/')

  return {
    base: appBasePath,
    define: {
      // 兼容 uni-h5 运行时代码对全局 process/global 的访问，避免浏览器环境报错。
      process: JSON.stringify({ env: {} }),
      global: 'globalThis',
      'import.meta.env.VITE_APP_PORT': JSON.stringify(env.VITE_APP_PORT || ''),
      'import.meta.env.VITE_APP_BASE_PATH': JSON.stringify(appBasePath),
      'import.meta.env.VITE_APP_BASE_API': JSON.stringify(env.VITE_APP_BASE_API || ''),
      'import.meta.env.VITE_APP_API_URL': JSON.stringify(env.VITE_APP_API_URL || ''),
      'import.meta.env.VITE_APP_STATIC_API': JSON.stringify(env.VITE_APP_STATIC_API || ''),
      'import.meta.env.VITE_APP_STATIC_URL': JSON.stringify(env.VITE_APP_STATIC_URL || ''),
    },
    server: {
      host: '0.0.0.0',
      port: +env.VITE_APP_PORT,
      proxy: {
        [env.VITE_APP_BASE_API]: {
          changeOrigin: true,
          target: devH5ProxyEnv.VITE_APP_API_URL,
          rewrite: (proxyPath) =>
            proxyPath.replace(new RegExp('^' + env.VITE_APP_BASE_API), env.VITE_APP_BASE_API),
        },
        ...(env.VITE_APP_STATIC_API
          ? {
              [env.VITE_APP_STATIC_API]: {
                changeOrigin: true,
                target: devH5ProxyEnv.VITE_APP_STATIC_URL,
                rewrite: (proxyPath) =>
                  proxyPath.replace(
                    new RegExp('^' + env.VITE_APP_STATIC_API),
                    env.VITE_APP_STATIC_API,
                  ),
              },
            }
          : {}),
      },
    },
    build: {
      // 仅在指定环境变量时覆盖输出目录，避免影响其它平台构建命令
      ...(process.env.UNI_OUTPUT_DIR
        ? {
            outDir: process.env.UNI_OUTPUT_DIR,
            emptyOutDir: true,
          }
        : {}),
      // 开发阶段启用源码映射：https://uniapp.dcloud.net.cn/tutorial/migration-to-vue3.html#需主动开启-sourcemap
      sourcemap: process.env.NODE_ENV === 'development',
    },
    plugins: [uni()],
  }
})
