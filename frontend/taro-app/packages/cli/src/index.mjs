import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, resolve } from 'node:path'

const cliPackage = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
const publicPackageVersion = cliPackage.version
if (typeof publicPackageVersion !== 'string' || !publicPackageVersion) {
  throw new Error('CLI package.json 缺少有效版本')
}

const taroVersion = '4.2.1'
const reactVersion = '18.3.1'

/** 创建独立 Kratos Taro React workspace。 */
export function scaffoldKratosTaroApp(targetPath, options = {}) {
  const target = resolve(targetPath)
  if (existsSync(target)) throw new Error(`目标目录已存在：${target}`)
  const projectName = basename(target)
  validateProjectName(projectName)
  const modules = [...new Set(options.modules ?? [])]
  const packages = [...new Set(options.packages ?? [])]
  modules.forEach(validateModuleName)
  packages.forEach(validatePackageName)
  mkdirSync(resolve(target, 'apps/taro-app/src/pages/bootstrap'), { recursive: true })
  modules.forEach((name) => {
    mkdirSync(resolve(target, `packages/modules/${name}/src/views`), { recursive: true })
  })

  write(
    target,
    '.gitignore',
    'node_modules\ndist\n.kratos-taro-app-pages-state.json\n.kratos-taro-app-pages-state.json.*\napps/taro-app/src/pages/*\n!apps/taro-app/src/pages/bootstrap/\n!apps/taro-app/src/pages/bootstrap/**\napps/taro-app/src/pages?*/\n',
  )
  write(
    target,
    'pnpm-workspace.yaml',
    'packages:\n  - apps/*\n  - packages/modules/*\n',
  )
  write(
    target,
    'package.json',
    json({
      name: projectName,
      description: `Independent pnpm workspace for ${projectName}.`,
      private: true,
      packageManager: 'pnpm@10.13.1',
      scripts: {
        'prepare:modules':
          "pnpm --recursive --filter './packages/modules/**' --if-present run build:entries",
        'dev:h5': 'pnpm prepare:modules && pnpm --filter @local/kratos-taro-app dev:h5',
        'dev:mp-weixin':
          'pnpm prepare:modules && pnpm --filter @local/kratos-taro-app dev:mp-weixin',
        'build:h5': 'pnpm prepare:modules && pnpm --filter @local/kratos-taro-app build:h5',
        'build:mp-weixin':
          'pnpm prepare:modules && pnpm --filter @local/kratos-taro-app build:mp-weixin',
        tsc: 'pnpm --recursive --if-present run tsc',
      },
      devDependencies: {
        '@babel/core': '7.28.4',
        '@babel/preset-react': '7.28.5',
        '@types/node': '20.19.9',
        '@types/react': '18.3.23',
        '@types/react-dom': '18.3.7',
        'babel-preset-taro': taroVersion,
        'cross-env': '7.0.3',
        esbuild: '0.25.8',
        sass: '1.89.2',
        'tsconfig-paths-webpack-plugin': '4.2.0',
        typescript: '5.8.3',
        webpack: '5.91.0',
      },
      pnpm: {
        onlyBuiltDependencies: ['@swc/core', '@tarojs/binding', 'esbuild'],
      },
    }),
  )
  write(
    target,
    'tsconfig.json',
    json({
      compilerOptions: {
        target: 'ES2020',
        module: 'ESNext',
        moduleResolution: 'Bundler',
        allowSyntheticDefaultImports: true,
        esModuleInterop: true,
        forceConsistentCasingInFileNames: true,
        resolveJsonModule: true,
        skipLibCheck: true,
        strict: true,
        jsx: 'react-jsx',
        types: ['node'],
      },
    }),
  )
  writeWorkspaceReadme(target, projectName)
  writeHost(target, projectName, modules, packages)
  modules.forEach((name) => writeLocalModule(target, projectName, name))
  return target
}

/** 解析命令行并运行脚手架。 */
export async function run(args = process.argv.slice(2)) {
  if (!args.length || args.includes('--help') || args.includes('-h')) {
    printHelp()
    return
  }
  if (args[0] !== 'create') throw new Error(`不支持的命令：${args[0]}`)
  const targetPath = args[1]
  if (!targetPath || targetPath.startsWith('--')) {
    throw new Error('用法: kratos-taro-app create <目录> [--module <名称>] [--with <包名>]')
  }
  const modules = readOptions(args.slice(2), '--module')
  const packages = readOptions(args.slice(2), '--with')
  const target = scaffoldKratosTaroApp(targetPath, { modules, packages })
  process.stdout.write(`已创建 Taro workspace：${target}\n`)
}

function writeWorkspaceReadme(target, projectName) {
  write(
    target,
    'README.md',
    `# ${projectName}

\`${projectName}\` 是由 \`@liujitcn/kratos-taro-app-cli\` 创建的独立 pnpm workspace，使用 Taro、React 和 TypeScript，支持 H5 与微信小程序。

## 目录结构

\`\`\`text
${projectName}
├── apps/taro-app             # 私有 Taro 宿主
├── packages/modules          # workspace 内本地业务模块
├── package.json              # 公共命令和开发依赖
├── pnpm-workspace.yaml       # workspace 包范围
└── tsconfig.json             # 共享 TypeScript 配置
\`\`\`

宿主只负责入口、模块清单和平台构建配置。运行时底座、UI 主题和默认业务页分别由 \`@liujitcn/kratos-taro-app-core\`、\`@liujitcn/kratos-taro-app-ui\` 与 \`@liujitcn/kratos-taro-app-system\` 提供。

## 开发

\`\`\`bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm tsc
\`\`\`

模块装配入口是 \`apps/taro-app/src/module-manifest.ts\`。模块顺序决定静态视图覆盖优先级；新增页面时同步维护模块自己的 \`src/pages.ts\` 和视图映射。
`,
  )
}

function writeHost(target, projectName, modules, packages) {
  const localDependencies = Object.fromEntries(modules.map((name) => [`@local/${name}`, 'workspace:*']))
  const publishedDependencies = Object.fromEntries(packages.map((name) => [name, 'latest']))
  write(
    target,
    'apps/taro-app/package.json',
    json({
      name: '@local/kratos-taro-app',
      description: `Private Taro React host for ${projectName}.`,
      version: '0.0.1',
      private: true,
      scripts: {
        'dev:h5':
          'cross-env NODE_ENV=development node scripts/run-taro.mjs --type h5 --watch --mode development',
        'dev:mp-weixin':
          'cross-env NODE_ENV=development node scripts/run-taro.mjs --type weapp --watch --mode development',
        'build:h5':
          'cross-env KRATOS_TARO_OUTPUT_ROOT=dist/build/h5 KRATOS_TARO_PUBLIC_PATH=/ node scripts/run-taro.mjs --type h5 --mode production',
        'build:mp-weixin':
          'cross-env KRATOS_TARO_OUTPUT_ROOT=dist/build/mp-weixin node scripts/run-taro.mjs --type weapp --mode production',
        tsc: 'tsc --noEmit -p tsconfig.json',
      },
      dependencies: {
        '@babel/runtime': '7.28.4',
        '@liujitcn/kratos-taro-app-core': `^${publicPackageVersion}`,
        '@liujitcn/kratos-taro-app-system': `^${publicPackageVersion}`,
        '@liujitcn/kratos-taro-app-ui': `^${publicPackageVersion}`,
        '@tarojs/components': taroVersion,
        '@tarojs/helper': taroVersion,
        '@tarojs/plugin-framework-react': taroVersion,
        '@tarojs/plugin-platform-h5': taroVersion,
        '@tarojs/plugin-platform-weapp': taroVersion,
        '@tarojs/react': taroVersion,
        '@tarojs/runtime': taroVersion,
        '@tarojs/shared': taroVersion,
        '@tarojs/taro': taroVersion,
        react: reactVersion,
        'react-dom': reactVersion,
        ...localDependencies,
        ...publishedDependencies,
      },
      devDependencies: {
        '@pmmmwh/react-refresh-webpack-plugin': '0.5.17',
        '@tarojs/cli': taroVersion,
        '@tarojs/taro-loader': taroVersion,
        '@tarojs/webpack5-runner': taroVersion,
        'react-refresh': '0.14.2',
      },
    }),
  )
  write(
    target,
    'apps/taro-app/README.md',
    `# @local/kratos-taro-app

\`apps/taro-app\` 是 \`${projectName}\` 的私有 Taro React 宿主，负责装配模块并提供 H5、微信小程序构建入口，不承载可复用业务实现。

固定启动页位于 \`src/pages/bootstrap\`。其他模块页面由 core runner 在构建期间临时生成包装器、页面配置与静态资源，构建结束后会自动恢复宿主目录。
`,
  )
  write(target, 'apps/taro-app/scripts/run-taro.mjs', "import '@liujitcn/kratos-taro-app-core/runner'\n")
  write(
    target,
    'apps/taro-app/babel.config.cjs',
    `module.exports = {
  presets: [['taro', { framework: 'react', ts: true, compiler: 'webpack5' }]],
}
`,
  )
  write(
    target,
    'apps/taro-app/tsconfig.json',
    json({
      extends: '../../tsconfig.json',
      compilerOptions: {
        baseUrl: '.',
        paths: { '@/*': ['src/*'] },
        types: ['node', '@tarojs/taro', '@tarojs/components'],
      },
      include: ['src', 'config', 'types'],
    }),
  )
  write(target, 'apps/taro-app/config/dev.ts', platformConfig(true))
  write(target, 'apps/taro-app/config/prod.ts', platformConfig(false))
  const modulePackages = [
    '@liujitcn/kratos-taro-app-core',
    '@liujitcn/kratos-taro-app-ui',
    '@liujitcn/kratos-taro-app-system',
    ...modules.map((name) => `@local/${name}`),
    ...packages,
  ]
  write(target, 'apps/taro-app/config/index.ts', hostConfig(projectName, modulePackages))
  write(
    target,
    'apps/taro-app/project.config.json',
    json({
      miniprogramRoot: './dist/build/mp-weixin',
      projectname: projectName,
      description: `${projectName} Taro React application`,
      appid: '',
      setting: {
        urlCheck: false,
        es6: false,
        enhance: false,
        compileHotReLoad: false,
        postcss: false,
        minified: true,
      },
      compileType: 'miniprogram',
    }),
  )
  write(
    target,
    'apps/taro-app/src/index.html',
    `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1,user-scalable=no,viewport-fit=cover" />
    <meta name="format-detection" content="telephone=no,address=no" />
    <title>${projectName}</title>
    <script><%= htmlWebpackPlugin.options.script %></script>
  </head>
  <body><div id="app"></div></body>
</html>
`,
  )
  write(
    target,
    'apps/taro-app/src/app.config.base.json',
    json({
      pages: ['pages/bootstrap/index'],
      window: {
        backgroundTextStyle: 'light',
        navigationBarBackgroundColor: '#f8f8f8',
        navigationBarTitleText: '',
        navigationBarTextStyle: 'black',
        backgroundColor: '#f8f8f8',
      },
    }),
  )
  write(
    target,
    'apps/taro-app/src/app.config.ts',
    "import config from './app.config.base.json'\n\nexport default defineAppConfig(config)\n",
  )
  write(
    target,
    'apps/taro-app/src/app.scss',
    "@use '@liujitcn/kratos-taro-app-ui/styles/theme.scss';\n@use '@liujitcn/kratos-taro-app-core/styles/base.scss';\n",
  )
  write(
    target,
    'apps/taro-app/src/app.tsx',
    `import type { PropsWithChildren } from 'react'
import { useLaunch } from '@tarojs/taro'
import { bootstrapKratosTaroApp } from '@liujitcn/kratos-taro-app-core'
import { moduleManifest } from './module-manifest'
import './app.scss'

/** Taro 宿主根组件。 */
export default function App({ children }: PropsWithChildren) {
  useLaunch(() => bootstrapKratosTaroApp({ modules: moduleManifest }))
  return children
}
`,
  )
  write(target, 'apps/taro-app/src/module-manifest.ts', moduleManifest(modules, packages))
  write(
    target,
    'apps/taro-app/src/pages/bootstrap/index.tsx',
    `import { View } from '@tarojs/components'
import Taro, { useLoad } from '@tarojs/taro'
import { resolveStaticView } from '@liujitcn/kratos-taro-app-core'

/** 固定启动页，模块注册完成后进入启动状态页。 */
export default function BootstrapPage() {
  useLoad((options) => {
    const route = resolveStaticView('BOOTSTRAP_LOADING') ?? 'pages/status/index'
    const target = options?.route ? decodeURIComponent(options.route) : 'app/home'
    void Taro.reLaunch({
      url: \`/\${route}?state=BOOTSTRAP_LOADING&bootstrap=1&route=\${encodeURIComponent(target)}\`,
    })
  })
  return <View />
}
`,
  )
  write(
    target,
    'apps/taro-app/src/pages/bootstrap/index.config.ts',
    "export default definePageConfig({ navigationStyle: 'custom', navigationBarTitleText: '' })\n",
  )
}

function writeLocalModule(target, projectName, name) {
  const identifier = `${toCamelCase(name)}Module`
  write(
    target,
    `packages/modules/${name}/package.json`,
    json({
      name: `@local/${name}`,
      description: `Local Kratos Taro module for ${name}.`,
      version: '0.0.1',
      type: 'module',
      files: ['dist', 'src', 'README.md'],
      exports: {
        '.': './src/index.ts',
        './build': {
          types: './src/build.ts',
          import: './dist/build.mjs',
          default: './dist/build.mjs',
        },
        './views/*': './src/views/*',
        './package.json': './package.json',
      },
      scripts: {
        'build:entries':
          'esbuild src/build.ts --bundle --platform=node --format=esm --packages=external --outfile=dist/build.mjs',
        tsc: 'tsc --noEmit -p tsconfig.json',
      },
      dependencies: {
        '@liujitcn/kratos-taro-app-core': `^${publicPackageVersion}`,
        '@tarojs/components': taroVersion,
        '@tarojs/taro': taroVersion,
        react: reactVersion,
      },
    }),
  )
  write(
    target,
    `packages/modules/${name}/tsconfig.json`,
    json({
      extends: '../../../tsconfig.json',
      compilerOptions: { types: ['node', '@tarojs/taro', '@tarojs/components'] },
      include: ['src'],
    }),
  )
  write(
    target,
    `packages/modules/${name}/README.md`,
    `# @local/${name}

\`@local/${name}\` 是 \`${projectName}\` workspace 内的本地 Taro 业务模块，通过 core 的公开接口接入宿主。

- 页面放在 \`src/views\`，页面配置登记在 \`src/pages.ts\`。
- 稳定视图键登记在 \`src/index.ts\`，后注册模块具有更高覆盖优先级。
- 请求、认证、导航和状态能力只能通过公开包入口使用。
- \`src/build.ts\` 是 runner 使用的构建期描述，不承载运行时逻辑。
`,
  )
  write(
    target,
    `packages/modules/${name}/src/pages.ts`,
    `import type { KratosTaroPageConfig } from '@liujitcn/kratos-taro-app-core'

/** ${name} 页面编译配置。 */
export const ${toCamelCase(name)}Pages: Record<string, KratosTaroPageConfig> = {}
`,
  )
  write(
    target,
    `packages/modules/${name}/src/index.ts`,
    `import { defineKratosTaroModule } from '@liujitcn/kratos-taro-app-core'
import { ${toCamelCase(name)}Pages } from './pages'

/** ${name} 业务模块。 */
export const ${identifier} = defineKratosTaroModule({
  name: '@local/${name}',
  pages: ${toCamelCase(name)}Pages,
  views: {},
})
`,
  )
  write(
    target,
    `packages/modules/${name}/src/build.ts`,
    `import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineKratosTaroBuildModule } from '@liujitcn/kratos-taro-app-core/build'
import { ${toCamelCase(name)}Pages } from './pages'

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

/** ${name} 构建期模块描述。 */
export const buildModule = defineKratosTaroBuildModule({
  name: '@local/${name}',
  root: packageRoot,
  pages: ${toCamelCase(name)}Pages,
})
`,
  )
}

function moduleManifest(modules, packages) {
  const imports = [
    "import { coreModule } from '@liujitcn/kratos-taro-app-core'",
    "import { systemModule } from '@liujitcn/kratos-taro-app-system'",
    ...modules.map(
      (name) => `import { ${toCamelCase(name)}Module } from '@local/${name}'`,
    ),
    ...packages.map((name, index) => `import packageModule${index} from '${name}'`),
  ]
  const members = [
    'coreModule',
    'systemModule',
    ...modules.map((name) => `${toCamelCase(name)}Module`),
    ...packages.map((_, index) => `packageModule${index}`),
  ]
  return `${imports.join('\n')}\n\n/** 宿主唯一模块清单，顺序决定静态视图覆盖优先级。 */\nexport const moduleManifest = [${members.join(', ')}]\n`
}

function hostConfig(projectName, packageNames) {
  return `import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { defineConfig, type UserConfigExport } from '@tarojs/cli'
import TsconfigPathsPlugin from 'tsconfig-paths-webpack-plugin'
import devConfig from './dev'
import prodConfig from './prod'

const packageNames = ${JSON.stringify(packageNames, null, 2)}
const hostRequire = createRequire(resolve(__dirname, '../package.json'))

export default defineConfig<'webpack5'>(async (merge) => {
  const outputRoot =
    process.env.KRATOS_TARO_OUTPUT_ROOT ||
    (process.env.TARO_ENV === 'h5'
      ? 'dist/h5'
      : process.env.TARO_ENV === 'weapp'
        ? 'dist/mp-weixin'
        : 'dist')
  const publicPath = process.env.KRATOS_TARO_PUBLIC_PATH || '/'
  const apiTargetUrl = process.env.KRATOS_TARO_API_URL || 'http://localhost:7001'
  const packageRoots = Object.fromEntries(
    packageNames.map((name) => [name, dirname(hostRequire.resolve(\`\${name}/package.json\`))]),
  )
  const sourceRoots = Object.values(packageRoots).map((root) => resolve(root, 'src'))
  const aliases = Object.fromEntries([
    ...packageNames.map((name) => [\`\${name}/static\`, resolve(__dirname, '../src/static')]),
    ...Object.entries(packageRoots).map(([name, root]) => [name, resolve(root, 'src')]),
  ])
  const configureWebpack = (chain: any) => {
    chain.resolve.plugin('tsconfig-paths').use(TsconfigPathsPlugin)
    chain.merge({ resolve: { fallback: { crypto: false } } })
    sourceRoots.forEach((root) => chain.module.rule('script').include.add(root))
  }
  const baseConfig: UserConfigExport<'webpack5'> = {
    projectName: ${JSON.stringify(projectName)},
    date: ${JSON.stringify(new Date().toISOString().slice(0, 10))},
    designWidth: 750,
    deviceRatio: { 375: 2, 640: 2.34 / 2, 750: 1, 828: 1.81 / 2 },
    sourceRoot: 'src',
    outputRoot,
    framework: 'react',
    compiler: {
      type: 'webpack5',
      prebundle: { enable: false },
    },
    compile: { include: sourceRoots },
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
      'process.env.KRATOS_TARO_API_BASE': JSON.stringify(process.env.KRATOS_TARO_API_BASE || '/api'),
      'process.env.KRATOS_TARO_API_URL': JSON.stringify(apiTargetUrl),
      'process.env.KRATOS_TARO_PUBLIC_PATH': JSON.stringify(publicPath),
      'process.env.KRATOS_TARO_STATIC_URL': JSON.stringify(process.env.KRATOS_TARO_STATIC_URL || apiTargetUrl),
    },
    alias: aliases,
    mini: {
      postcss: { pxtransform: { enable: true, config: {} }, cssModules: { enable: false } },
      webpackChain: configureWebpack,
    },
    h5: {
      publicPath,
      staticDirectory: 'static',
      router: { mode: 'hash' },
      devServer: {
        port: 5002,
        host: '0.0.0.0',
        proxy: {
          '/api': { target: apiTargetUrl, changeOrigin: true },
          '/events': { target: apiTargetUrl, changeOrigin: true },
        },
      },
      output: {
        filename: 'assets/[name].[contenthash:8].js',
        chunkFilename: 'assets/[name].[contenthash:8].js',
      },
      miniCssExtractPluginOption: {
        ignoreOrder: true,
        filename: 'assets/[name].[contenthash:8].css',
        chunkFilename: 'assets/[name].[contenthash:8].css',
      },
      postcss: { autoprefixer: { enable: true, config: {} }, cssModules: { enable: false } },
      webpackChain: configureWebpack,
    },
  }
  return process.env.NODE_ENV === 'development'
    ? merge({}, baseConfig, devConfig)
    : merge({}, baseConfig, prodConfig)
})
`
}

function platformConfig(development) {
  return `import type { UserConfigExport } from '@tarojs/cli'

export default ${JSON.stringify(
    development ? { logger: { quiet: false, stats: true }, mini: {}, h5: {} } : { mini: {}, h5: {} },
    null,
    2,
  )} satisfies UserConfigExport<'webpack5'>
`
}

function readOptions(args, option) {
  const values = []
  for (let index = 0; index < args.length; index += 1) {
    const argument = args[index]
    if (argument !== '--module' && argument !== '--with') {
      throw new Error(`未知参数：${argument}`)
    }
    const value = args[++index]
    if (!value || value.startsWith('--')) throw new Error(`选项 ${argument} 缺少值`)
    if (argument === option) {
      values.push(...value.split(',').map((item) => item.trim()).filter(Boolean))
    }
  }
  return [...new Set(values)]
}

function validateProjectName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(`项目名必须使用 kebab-case：${name}`)
}

function validateModuleName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(`模块名无效：${name}`)
  if (name === 'system') throw new Error('模块名不能使用保留名称：system')
}

function validatePackageName(name) {
  if (!/^(?:@[a-z0-9-]+\/)?[a-z0-9-]+$/.test(name)) throw new Error(`包名无效：${name}`)
}

function toCamelCase(value) {
  return value.replace(/-([a-z0-9])/g, (_, character) => character.toUpperCase())
}

function write(root, file, content) {
  const target = resolve(root, file)
  mkdirSync(resolve(target, '..'), { recursive: true })
  writeFileSync(target, content)
}

function json(value) {
  return `${JSON.stringify(value, null, 2)}\n`
}

function printHelp() {
  process.stdout.write(
    [
      'kratos-taro-app create <目录> [--module <名称[,名称...]>] [--with <包名>]',
      '',
      '示例:',
      '  kratos-taro-app create customer-app',
      '  kratos-taro-app create shop-app --module shop,order',
      '  kratos-taro-app create customer-app --with @acme/customer-module',
      '',
    ].join('\n'),
  )
}
