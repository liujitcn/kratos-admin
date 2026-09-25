import { execFileSync } from 'node:child_process'
import { copyFileSync, existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, resolve } from 'node:path'
import { cliMessage, resolveCliLocale } from './messages.mjs'

const cliPackage = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
const publicPackageVersion = cliPackage.version
if (typeof publicPackageVersion !== 'string' || !publicPackageVersion) {
  throw new Error(cliMessage('package_version_missing'))
}

const taroVersion = '4.2.1'
const reactVersion = '18.3.1'
const h5RootFontSource =
  '!function(n){function f(){var e=n.document.documentElement,w=e.clientWidth||n.innerWidth||375,x=w>960?375:w;e.style.fontSize=20*x/375+"px"}n.addEventListener("resize",function(){f();setTimeout(f,500)}),f()}(window);'

/** 创建独立 Kratos Taro React workspace。 */
export function scaffoldKratosTaroApp(targetPath, options = {}) {
  const target = resolve(targetPath)
  if (existsSync(target)) throw new Error(cliMessage('target_exists', { target }))
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
  write(target, 'pnpm-workspace.yaml', 'packages:\n  - apps/*\n  - packages/modules/*\n')
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
  renderWorkspaceTemplates(new URL('../templates/workspace/', import.meta.url), target, { __PROJECT_NAME__: projectName })
  writeEnvironmentFiles(target)
  writeHost(target, projectName, modules, packages, options)
  write(target, 'apps/taro-app/src/static/h5-root-font.js', `${h5RootFontSource}\n`)
  copyFileSync(
    new URL('../assets/favicon.ico', import.meta.url),
    resolve(target, 'apps/taro-app/src/static/favicon.ico'),
  )
  modules.forEach((name) => writeLocalModule(target, projectName, name))
  modules.forEach((name) => writeModuleLocales(target, name))
  execFileSync(process.execPath, [resolve(target, 'scripts/sync-locales.mjs'), '--write'], { stdio: 'inherit' })
  return target
}

/** 解析命令行并运行脚手架。 */
export async function run(args = process.argv.slice(2)) {
  if (!args.length || args.includes('--help') || args.includes('-h')) {
    printHelp()
    return
  }
  if (args[0] !== 'create') throw new Error(cliMessage('unsupported_command', { command: args[0] }))
  const targetPath = args[1]
  if (!targetPath || targetPath.startsWith('--')) {
    throw new Error(cliMessage('usage'))
  }
  const modules = readOptions(args.slice(2), '--module')
  const packages = readOptions(args.slice(2), '--with')
  const target = scaffoldKratosTaroApp(targetPath, { modules, packages, kratosProject: args.includes('--kratos-project') })
  process.stdout.write(`${cliMessage('workspace_created', { target })}\n`)
}

function writeWorkspaceReadme(target, projectName) {
  write(target, 'README.md', renderLocalizedReadme('workspace.md', { __PROJECT_NAME__: projectName }))
}

function writeEnvironmentFiles(target) {
  write(
    target,
    '.env.development',
    'VITE_APP_BASE_API=/api\nVITE_APP_API_URL=http://localhost:7001\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=http://localhost:7001\n',
  )
  write(
    target,
    '.env.development-h5',
    `VITE_APP_PORT=5002\nVITE_APP_HTTPS=false\n# VITE_APP_HTTPS_KEY=../../certs/dev-key.pem\n# VITE_APP_HTTPS_CERT=../../certs/dev-cert.pem\n# ${cliMessage('https_environment_hint')}\nVITE_APP_BASE_PATH=/\nVITE_APP_BASE_API=/api\nVITE_APP_API_URL=http://localhost:7001\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=http://localhost:7001\n`,
  )
  write(
    target,
    '.env.production',
    'VITE_APP_BASE_API=/api\nVITE_APP_API_URL=http://localhost:7001\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=http://localhost:7001\n',
  )
  write(
    target,
    '.env.production-h5',
    'VITE_APP_BASE_PATH=/app/\nVITE_APP_BASE_API=/api\nVITE_APP_API_URL=\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=\n',
  )
}

/** 生成 Taro 宿主与平台构建配置。 */
function writeHost(target, projectName, modules, packages, options) {
  const localDependencies = Object.fromEntries(
    modules.map((name) => [`@local/${name}`, 'workspace:*']),
  )
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
          `cross-env KRATOS_TARO_OUTPUT_ROOT=${options.kratosProject ? '../../../../backend/web/taro-app' : 'dist/build/h5'} node scripts/run-taro.mjs --type h5 --mode production`,
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
    renderLocalizedReadme('host.md', { __PROJECT_NAME__: projectName }),
  )
  write(
    target,
    'apps/taro-app/scripts/run-taro.mjs',
    "import '@liujitcn/kratos-taro-app-core/runner'\n",
  )
  write(
    target,
    'apps/taro-app/babel.config.cjs',
    `module.exports = {
  presets: [['taro', { framework: 'react', ts: true, compiler: 'webpack5' }]],
}
`,
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
      miniprogramRoot: './dist/dev/mp-weixin',
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
    <link rel="icon" type="image/x-icon" href="./static/favicon.ico" />
    <title>${projectName}</title>
    <script src="./static/h5-root-font.js" defer></script>
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

/** 生成本地 Taro 模块与独立构建入口。 */
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
        build: 'pnpm build:entries && pnpm pack --pack-destination ../../../dist/npm',
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
    renderLocalizedReadme('module.md', {
      __PROJECT_NAME__: projectName,
      __MODULE_NAME__: name,
    }),
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
import { LOCALE_MESSAGES } from './locales/generated'

/** ${name} 业务模块。 */
export const ${identifier} = defineKratosTaroModule({
  name: '@local/${name}',
  pages: ${toCamelCase(name)}Pages,
  views: {},
  messages: LOCALE_MESSAGES,
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

/** 生成模块清单，导入别名避免本地 system 与内置模块重名。 */
function moduleManifest(modules, packages) {
  const imports = [
    "import { coreModule } from '@liujitcn/kratos-taro-app-core'",
    "import { systemModule } from '@liujitcn/kratos-taro-app-system'",
    ...modules.map((name, index) => `import { ${toCamelCase(name)}Module as localModule${index} } from '@local/${name}'`),
    ...packages.map((name, index) => `import packageModule${index} from '${name}'`),
  ]
  const members = [
    'coreModule',
    'systemModule',
    ...modules.map((_, index) => `localModule${index}`),
    ...packages.map((_, index) => `packageModule${index}`),
  ]
  return `${imports.join('\n')}\n\n/** 宿主唯一模块清单，顺序决定静态视图覆盖优先级。 */\nexport const moduleManifest = [${members.join(', ')}]\n`
}

/** 生成宿主构建配置，让已装配 npm 源码包参与脚本和样式编译。 */
function hostConfig(projectName, packageNames) {
  return `import { createRequire } from 'node:module'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { defineConfig, type UserConfigExport } from '@tarojs/cli'
import { dotenvParse } from '@tarojs/helper'
import TsconfigPathsPlugin from 'tsconfig-paths-webpack-plugin'
import { runnerMessage } from '@liujitcn/kratos-taro-app-core/build'
import devConfig from './dev'
import prodConfig from './prod'

const packageNames = ${JSON.stringify(packageNames, null, 2)}
const hostRequire = createRequire(resolve(__dirname, '../package.json'))
const workspaceRoot = resolve(__dirname, '../../..')

function resolveHttpsOptions(env: Record<string, string>, root: string) {
  if (env.VITE_APP_HTTPS !== 'true') return undefined
  const keyPath = resolve(root, env.VITE_APP_HTTPS_KEY || '../../certs/dev-key.pem')
  const certPath = resolve(root, env.VITE_APP_HTTPS_CERT || '../../certs/dev-cert.pem')
  if (!existsSync(keyPath) || !existsSync(certPath)) {
    throw new Error(runnerMessage('https_certificate_missing', { keyPath, certPath }))
  }
  return { key: readFileSync(keyPath), cert: readFileSync(certPath) }
}

function resolveEnv(mode: string, platform: string): Record<string, string> {
  const shellEnv = Object.fromEntries(
    Object.entries(process.env).filter(([key]) => key.startsWith('VITE_APP_')),
  ) as Record<string, string>
  const parseEnv = (envMode: string) => {
    const savedEnv = Object.fromEntries(
      Object.entries(process.env).filter(([key]) => key.startsWith('VITE_APP_')),
    ) as Record<string, string>
    try {
      Object.keys(process.env)
        .filter((key) => key.startsWith('VITE_APP_'))
        .forEach((key) => delete process.env[key])
      return dotenvParse(workspaceRoot, 'VITE_APP_', envMode)
    } finally {
      Object.keys(process.env)
        .filter((key) => key.startsWith('VITE_APP_'))
        .forEach((key) => delete process.env[key])
      Object.assign(process.env, savedEnv)
    }
  }
  const baseEnv = parseEnv(mode)
  const platformEnv =
    platform === 'h5' ? parseEnv(\`\${mode}-\${platform}\`) : {}
  return { ...baseEnv, ...platformEnv, ...shellEnv }
}

export default defineConfig<'webpack5'>(async (merge) => {
  const mode = process.env.NODE_ENV || 'production'
  const platform = process.env.TARO_ENV || ''
  const env = resolveEnv(mode, platform)
  const outputMode = mode === 'development' ? 'dev' : 'build'
  const outputRoot =
    process.env.KRATOS_TARO_OUTPUT_ROOT ||
    'dist/' + outputMode + '/' + (platform === 'weapp' ? 'mp-weixin' : platform || 'h5')
  const publicPath = env.VITE_APP_BASE_PATH ?? '/'
  const apiBasePath = env.VITE_APP_BASE_API ?? '/api'
  const apiTargetUrl = env.VITE_APP_API_URL ?? 'http://localhost:7001'
  const apiProxyOptions = apiTargetUrl.startsWith('https://') ? { secure: false } : {}
  const staticApi = env.VITE_APP_STATIC_API ?? ''
  const staticUrl = env.VITE_APP_STATIC_URL ?? apiTargetUrl
  const httpsOptions = resolveHttpsOptions(env, workspaceRoot)
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
      'process.env.VITE_APP_PORT': JSON.stringify(env.VITE_APP_PORT ?? ''),
      'process.env.VITE_APP_BASE_PATH': JSON.stringify(publicPath),
      'process.env.VITE_APP_BASE_API': JSON.stringify(apiBasePath),
      'process.env.VITE_APP_API_URL': JSON.stringify(apiTargetUrl),
      'process.env.VITE_APP_STATIC_API': JSON.stringify(staticApi),
      'process.env.VITE_APP_STATIC_URL': JSON.stringify(staticUrl),
    },
    alias: aliases,
    mini: {
      postcss: { pxtransform: { enable: true, config: {} }, cssModules: { enable: false } },
      webpackChain: configureWebpack,
    },
    h5: {
      // Taro 默认跳过 node_modules 样式，已装配源码包需要进行 px 到 rem 转换。
      esnextModules: sourceRoots,
      publicPath,
      staticDirectory: 'static',
      router: { mode: 'hash' },
      devServer: {
        port: Number(env.VITE_APP_PORT || 5002),
        host: '0.0.0.0',
        https: httpsOptions,
        proxy: {
          [apiBasePath || '/api']: { target: apiTargetUrl || 'http://localhost:7001', changeOrigin: true, ...apiProxyOptions },
          '/events': { target: apiTargetUrl || 'http://localhost:7001', changeOrigin: true, ...apiProxyOptions },
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
    development
      ? { logger: { quiet: false, stats: true }, mini: {}, h5: {} }
      : { mini: {}, h5: {} },
    null,
    2,
  )} satisfies UserConfigExport<'webpack5'>
`
}

/** 读取重复模块参数并识别项目集成开关。 */
function readOptions(args, option) {
  const values = []
  for (let index = 0; index < args.length; index += 1) {
    const argument = args[index]
    if (argument === '--kratos-project') continue
    if (argument !== '--module' && argument !== '--with') {
      throw new Error(cliMessage('unknown_argument', { argument }))
    }
    const value = args[++index]
    if (!value || value.startsWith('--')) throw new Error(cliMessage('option_missing_value', { option: argument }))
    if (argument === option) {
      values.push(
        ...value
          .split(',')
          .map((item) => item.trim())
          .filter(Boolean),
      )
    }
  }
  return [...new Set(values)]
}

function validateProjectName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(cliMessage('project_name_invalid', { name }))
}

/** 校验本地业务模块名称。 */
function validateModuleName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(cliMessage('module_name_invalid', { name }))
}

function validatePackageName(name) {
  if (!/^(?:@[a-z0-9-]+\/)?[a-z0-9-]+$/.test(name)) throw new Error(cliMessage('package_name_invalid', { name }))
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
      cliMessage('usage'),
      '',
      cliMessage('examples_heading'),
      cliMessage('example_create'),
      cliMessage('example_modules'),
      cliMessage('example_package'),
      '',
    ].join('\n'),
  )
}

/** 渲染随 CLI 发布的宿主与工具链模板。 */
function renderWorkspaceTemplates(source, target, tokens) {
  for (const entry of readdirSync(source, { withFileTypes: true })) {
    const input = new URL(entry.name + (entry.isDirectory() ? '/' : ''), source)
    const output = resolve(target, entry.name)
    if (entry.isDirectory()) {
      mkdirSync(output, { recursive: true })
      renderWorkspaceTemplates(input, output, tokens)
    } else {
      const content = Object.entries(tokens).reduce((value, [key, replacement]) => value.replaceAll(key, replacement), readFileSync(input, 'utf8'))
      writeFileSync(output, content)
    }
  }
}

/** 按 CLI 语言加载并替换 workspace README 模板。 */
function renderLocalizedReadme(name, tokens) {
  const template = readFileSync(
    new URL(`../templates/readmes/${resolveCliLocale()}/${name}`, import.meta.url),
    'utf8',
  )
  return Object.entries(tokens).reduce(
    (content, [key, value]) => content.replaceAll(key, () => value),
    template,
  )
}

/** 创建业务模块语言源文件及 API、RPC、测试目录，注册产物由同步命令生成。 */
function writeModuleLocales(target, name) {
  for (const directory of ['src/api', 'src/rpc', 'test']) {
    write(target, `packages/modules/${name}/${directory}/.gitkeep`, '')
  }
  for (const locale of ['zh-CN', 'en-US', 'zh-TW', 'ja-JP']) {
    write(target, `packages/modules/${name}/src/locales/${locale}.json`, '{}\n')
  }
}
