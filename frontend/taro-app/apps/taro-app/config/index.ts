import { resolve } from 'node:path'
import { defineConfig, type UserConfigExport } from '@tarojs/cli'
import TsconfigPathsPlugin from 'tsconfig-paths-webpack-plugin'
import devConfig from './dev'
import prodConfig from './prod'

const h5RootFontScript = `!function(n){function f(){var e=n.document.documentElement,w=e.clientWidth||n.innerWidth||375,x=w>960?375:w;e.style.fontSize=20*x/375+"px"}n.addEventListener("resize",function(){f();setTimeout(f,500)}),f()}(window);`

export default defineConfig<'webpack5'>(async (merge) => {
  const outputRoot = process.env.KRATOS_TARO_OUTPUT_ROOT || 'dist'
  const publicPath = process.env.KRATOS_TARO_PUBLIC_PATH || '/'
  const apiBasePath = process.env.KRATOS_TARO_API_BASE || '/api'
  const apiTargetUrl = process.env.KRATOS_TARO_API_URL || 'http://192.168.60.52:7001'
  const staticUrl = process.env.KRATOS_TARO_STATIC_URL || apiTargetUrl
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
      'process.env.KRATOS_TARO_API_BASE': JSON.stringify(apiBasePath),
      'process.env.KRATOS_TARO_API_URL': JSON.stringify(apiTargetUrl),
      'process.env.KRATOS_TARO_PUBLIC_PATH': JSON.stringify(publicPath),
      'process.env.KRATOS_TARO_STATIC_URL': JSON.stringify(staticUrl),
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
        port: 5002,
        host: '0.0.0.0',
        proxy: {
          '/api': {
            target: process.env.KRATOS_TARO_API_URL || 'http://localhost:7001',
            changeOrigin: true,
          },
          '/events': {
            target: process.env.KRATOS_TARO_API_URL || 'http://localhost:7001',
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
