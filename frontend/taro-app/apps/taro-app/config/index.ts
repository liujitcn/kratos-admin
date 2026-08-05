import { resolve } from 'node:path'
import { defineConfig, type UserConfigExport } from '@tarojs/cli'
import { dotenvParse } from '@tarojs/helper'
import TsconfigPathsPlugin from 'tsconfig-paths-webpack-plugin'
import devConfig from './dev'
import prodConfig from './prod'

const workspaceRoot = resolve(__dirname, '../../..')
const h5RootFontScript = `!function(n){function f(){var e=n.document.documentElement,w=e.clientWidth||n.innerWidth||375,x=w>960?375:w;e.style.fontSize=20*x/375+"px"}n.addEventListener("resize",function(){f();setTimeout(f,500)}),f()}(window);`

function resolveEnv(mode: string, platform: string): Record<string, string> {
  const baseEnv = dotenvParse(workspaceRoot, 'VITE_APP_', mode)
  const platformEnv =
    platform === 'h5' ? dotenvParse(workspaceRoot, 'VITE_APP_', `${mode}-${platform}`) : {}
  const shellEnv = Object.fromEntries(
    Object.entries(process.env).filter(([key]) => key.startsWith('VITE_APP_')),
  ) as Record<string, string>
  return { ...baseEnv, ...platformEnv, ...shellEnv }
}

export default defineConfig<'webpack5'>(async (merge) => {
  const mode = process.env.NODE_ENV || 'production'
  const platform = process.env.TARO_ENV || ''
  const env = resolveEnv(mode, platform)
  const outputRoot =
    process.env.KRATOS_TARO_OUTPUT_ROOT ||
    (platform === 'h5'
      ? 'dist/h5'
      : platform === 'weapp'
        ? 'dist/mp-weixin'
        : 'dist')
  const publicPath = env.VITE_APP_BASE_PATH ?? '/'
  const apiBasePath = env.VITE_APP_BASE_API ?? '/api'
  const apiTargetUrl = env.VITE_APP_API_URL ?? 'http://127.0.0.1:7001'
  const staticApi = env.VITE_APP_STATIC_API ?? ''
  const staticUrl = env.VITE_APP_STATIC_URL ?? apiTargetUrl
  const packageRoots = [
    resolve(__dirname, '../../../packages/core/src'),
    resolve(__dirname, '../../../packages/ui/src'),
    resolve(__dirname, '../../../packages/modules/system/src'),
  ]
  const baseConfig: UserConfigExport<'webpack5'> = {
    projectName: 'kratos-taro-app',
    date: '2026-07-31',
    designWidth: 750,
    deviceRatio: {
      375: 2,
      640: 2.34 / 2,
      750: 1,
      828: 1.81 / 2,
    },
    sourceRoot: 'src',
    outputRoot,
    framework: 'react',
    compiler: {
      type: 'webpack5',
      prebundle: { enable: false },
    },
    compile: {
      include: packageRoots,
    },
    cache: { enable: true },
    copy: {
      patterns: [
        {
          from: resolve(__dirname, '../src/static'),
          to: resolve(__dirname, '..', outputRoot, 'static'),
        },
      ],
      options: {},
    },
    defineConstants: {
      'process.env.VITE_APP_PORT': JSON.stringify(env.VITE_APP_PORT ?? ''),
      'process.env.VITE_APP_BASE_PATH': JSON.stringify(publicPath),
      'process.env.VITE_APP_BASE_API': JSON.stringify(apiBasePath),
      'process.env.VITE_APP_API_URL': JSON.stringify(apiTargetUrl),
      'process.env.VITE_APP_STATIC_API': JSON.stringify(staticApi),
      'process.env.VITE_APP_STATIC_URL': JSON.stringify(staticUrl),
    },
    alias: {
      '@liujitcn/kratos-taro-app-core/static': resolve(__dirname, '../src/static'),
      '@liujitcn/kratos-taro-app-system/static': resolve(__dirname, '../src/static'),
      '@liujitcn/kratos-taro-app-core': packageRoots[0],
      '@liujitcn/kratos-taro-app-ui': packageRoots[1],
      '@liujitcn/kratos-taro-app-system': packageRoots[2],
    },
    mini: {
      postcss: {
        pxtransform: {
          enable: true,
          config: {},
        },
        cssModules: {
          enable: false,
          config: {
            namingPattern: 'module',
            generateScopedName: '[name]__[local]___[hash:base64:5]',
          },
        },
      },
      fontUrlLoaderOption: {
        name: 'static/fonts/uniicons.ttf',
      },
      webpackChain(chain) {
        chain.resolve.plugin('tsconfig-paths').use(TsconfigPathsPlugin)
        chain.merge({ resolve: { fallback: { crypto: false } } })
        packageRoots.forEach((root) => chain.module.rule('script').include.add(root))
      },
    },
    h5: {
      publicPath,
      staticDirectory: 'static',
      // 与 uni-app H5 的 rpx 规则一致：宽屏使用 375px 基准，移动端随页面宽度缩放。
      htmlPluginOption: {
        script: h5RootFontScript,
      },
      router: {
        mode: 'hash',
      },
      devServer: {
        port: Number(env.VITE_APP_PORT || 5002),
        host: '0.0.0.0',
        proxy: {
          [apiBasePath || '/api']: {
            target: apiTargetUrl || 'http://localhost:7001',
            changeOrigin: true,
          },
          '/events': {
            target: apiTargetUrl || 'http://localhost:7001',
            changeOrigin: true,
          },
        },
      },
      output: {
        filename: 'assets/[name].[contenthash:8].js',
        chunkFilename: 'assets/[name].[contenthash:8].js',
      },
      fontUrlLoaderOption: {
        name: 'static/fonts/uniicons.ttf',
      },
      miniCssExtractPluginOption: {
        ignoreOrder: true,
        filename: 'assets/[name].[contenthash:8].css',
        chunkFilename: 'assets/[name].[contenthash:8].css',
      },
      postcss: {
        autoprefixer: {
          enable: true,
          config: {},
        },
        cssModules: {
          enable: false,
          config: {
            namingPattern: 'module',
            generateScopedName: '[name]__[local]___[hash:base64:5]',
          },
        },
      },
      webpackChain(chain) {
        chain.resolve.plugin('tsconfig-paths').use(TsconfigPathsPlugin)
        chain.merge({ resolve: { fallback: { crypto: false } } })
        packageRoots.forEach((root) => chain.module.rule('script').include.add(root))
      },
    },
  }

  return process.env.NODE_ENV === 'development'
    ? merge({}, baseConfig, devConfig)
    : merge({}, baseConfig, prodConfig)
})
