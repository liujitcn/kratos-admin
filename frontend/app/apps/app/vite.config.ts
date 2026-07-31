import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  createKratosUniPlugin,
  defineConfig,
  kratosApp,
  loadEnv,
  type ConfigEnv,
  type UserConfig,
} from '@liujitcn/kratos-app-core/vite'
import { moduleManifest } from './src/module-manifest'

const workspaceRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')

function resolveEnv(mode: string) {
  const modeEnv = loadEnv(mode, workspaceRoot, '')
  if (mode === 'development-h5') {
    return { ...loadEnv('development', workspaceRoot, ''), ...modeEnv }
  }
  if (mode === 'production-h5') {
    return { ...loadEnv('production', workspaceRoot, ''), ...modeEnv }
  }
  return modeEnv
}

export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = resolveEnv(mode)
  const devEnv = mode === 'development-h5' ? loadEnv('development', workspaceRoot, '') : env
  const base = env.VITE_APP_BASE_PATH || (mode === 'production-h5' ? '/app/' : '/')
  return {
    base,
    envDir: workspaceRoot,
    resolve: {
      preserveSymlinks: true,
      alias: [
        { find: '@', replacement: resolve(workspaceRoot, 'packages/core/src') },
        { find: '@system', replacement: resolve(workspaceRoot, 'packages/modules/system/src') },
      ],
    },
    define: {
      process: JSON.stringify({ env: {} }),
      global: 'globalThis',
      'import.meta.env.VITE_APP_PORT': JSON.stringify(env.VITE_APP_PORT || ''),
      'import.meta.env.VITE_APP_BASE_PATH': JSON.stringify(base),
      'import.meta.env.VITE_APP_BASE_API': JSON.stringify(env.VITE_APP_BASE_API || ''),
      'import.meta.env.VITE_APP_API_URL': JSON.stringify(env.VITE_APP_API_URL || ''),
      'import.meta.env.VITE_APP_STATIC_API': JSON.stringify(env.VITE_APP_STATIC_API || ''),
      'import.meta.env.VITE_APP_STATIC_URL': JSON.stringify(env.VITE_APP_STATIC_URL || ''),
    },
    server: {
      host: '0.0.0.0',
      port: Number(env.VITE_APP_PORT || 5002),
      proxy: {
        [env.VITE_APP_BASE_API || '/api']: {
          changeOrigin: true,
          target: devEnv.VITE_APP_API_URL,
        },
      },
    },
    build: {
      ...(process.env.UNI_OUTPUT_DIR
        ? { outDir: process.env.UNI_OUTPUT_DIR, emptyOutDir: true }
        : {}),
      sourcemap: process.env.NODE_ENV === 'development',
    },
    plugins: [kratosApp({ modules: moduleManifest }), createKratosUniPlugin()],
  }
})
