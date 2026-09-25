import { execFileSync } from 'node:child_process'
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  writeFileSync,
} from 'node:fs'
import { basename, resolve } from 'node:path'
import { cliMessage, resolveCliLocale } from './messages.mjs'

const cliPackage = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
const publicPackageVersion = cliPackage.version
if (typeof publicPackageVersion !== 'string' || !publicPackageVersion) {
  throw new Error(cliMessage('package_version_missing'))
}

/** 创建独立 Kratos uni-app workspace。 */
export function scaffoldKratosApp(targetPath, options = {}) {
  const target = resolve(targetPath)
  if (existsSync(target)) throw new Error(cliMessage('target_exists', { target }))
  const projectName = basename(target)
  const modules = [...new Set(options.modules ?? [])]
  const packages = [...new Set(options.packages ?? [])]
  modules.forEach(validateModuleName)
  packages.forEach(validatePackageName)
  mkdirSync(resolve(target, 'apps/uni-app/src/pages/bootstrap'), { recursive: true })
  modules.forEach((name) =>
    mkdirSync(resolve(target, `packages/modules/${name}/src/views`), { recursive: true }),
  )

  write(target, 'pnpm-workspace.yaml', 'packages:\n  - apps/*\n  - packages/modules/*\n')
  renderWorkspaceTemplates(new URL('../templates/workspace/', import.meta.url), target, {
    __PROJECT_NAME__: projectName,
  })
  writeEnvironmentFiles(target)
  copyFileSync(
    new URL('../assets/favicon.ico', import.meta.url),
    resolve(target, 'apps/uni-app/favicon.ico'),
  )
  write(
    target,
    'README.md',
    renderLocalizedReadme('workspace.md', { __PROJECT_NAME__: projectName }),
  )
  const dependencies = {
    '@liujitcn/kratos-uni-app-core': `^${publicPackageVersion}`,
    '@liujitcn/kratos-uni-app-system': `^${publicPackageVersion}`,
    '@dcloudio/uni-app': '3.0.0-5010520260709002',
    '@dcloudio/uni-components': '3.0.0-5010520260709002',
    '@dcloudio/uni-h5': '3.0.0-5010520260709002',
    '@dcloudio/uni-mp-weixin': '3.0.0-5010520260709002',
    '@dcloudio/uni-ui': '^1.4.28',
    '@dcloudio/vite-plugin-uni': '3.0.0-5010520260709002',
    'asmcrypto.js': '^2.3.2',
    'go-captcha-uni': '1.0.6',
    pinia: '^2.0.27',
    'pinia-plugin-persistedstate': '^3.2.3',
    'qrcode-generator': '^2.0.4',
    sass: '1.77.8',
    vite: '5.2.8',
    vue: '^3.4.21',
    ...Object.fromEntries(modules.map((name) => [`@local/${name}`, 'workspace:*'])),
    ...Object.fromEntries(packages.map((name) => [name, 'latest'])),
  }
  write(
    target,
    'apps/uni-app/package.json',
    json({
      name: '@liujitcn/kratos-uni-app',
      description: `Private uni-app host for ${projectName}.`,
      version: '0.0.1',
      private: true,
      type: 'module',
      scripts: {
        'recover:pages':
          'node -e "import(\'@liujitcn/kratos-uni-app-core/vite\').then(({ recoverStalePageTransaction }) => recoverStalePageTransaction())"',
        'dev:h5': 'pnpm run recover:pages && uni --mode development-h5',
        'dev:mp-weixin': 'pnpm run recover:pages && uni -p mp-weixin --mode development',
        'build:h5': `pnpm run recover:pages && UNI_OUTPUT_DIR=${options.kratosProject ? '../../../../backend/web/uni-app' : 'dist/build/h5'} uni build --mode production-h5`,
        'build:mp-weixin': 'pnpm run recover:pages && uni build -p mp-weixin --mode production',
        tsc: 'vue-tsc --noEmit -p tsconfig.json',
      },
      dependencies,
    }),
  )
  write(
    target,
    'apps/uni-app/README.md',
    renderLocalizedReadme('host.md', { __PROJECT_NAME__: projectName }),
  )
  const imports = [
    "import { coreModule } from '@liujitcn/kratos-uni-app-core/module'",
    "import { systemModule } from '@liujitcn/kratos-uni-app-system/module'",
    ...modules.map((name, index) => `import localModule${index} from '@local/${name}'`),
    ...packages.map((name, index) => `import packageModule${index} from '${name}'`),
  ]
  const members = [
    'coreModule',
    'systemModule',
    ...modules.map((_, index) => `localModule${index}`),
    ...packages.map((_, index) => `packageModule${index}`),
  ]
  write(
    target,
    'apps/uni-app/src/module-manifest.ts',
    `${imports.join('\n')}\n\nexport const moduleManifest = [${members.join(', ')}]\n`,
  )
  write(
    target,
    'apps/uni-app/src/pages.json',
    json({
      pages: [
        {
          path: 'pages/bootstrap/index',
          style: { navigationStyle: 'custom' },
        },
      ],
    }),
  )
  write(
    target,
    'apps/uni-app/src/pages/bootstrap/index.vue',
    `<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { resolveStaticView } from '@liujitcn/kratos-uni-app-core'
onLoad((options) => {
  const route = resolveStaticView('BOOTSTRAP_LOADING') ?? 'pages/status/index'
  const target = options?.route ? decodeURIComponent(options.route) : 'app/home'
  uni.reLaunch({ url: \`/\${route}?state=BOOTSTRAP_LOADING&bootstrap=1&route=\${encodeURIComponent(target)}\` })
})
</script>
<template><view /></template>
`,
  )
  write(target, 'apps/uni-app/src/uni.scss', "@forward '@liujitcn/kratos-uni-app-core/uni.scss';\n")
  write(
    target,
    'apps/uni-app/src/manifest.json',
    json({
      name: projectName,
      appid: '__UNI__KRATOS_APP',
      versionName: '1.0.0',
      versionCode: '100',
      transformPx: false,
      'mp-weixin': { appid: '', setting: { urlCheck: false } },
      h5: { router: { mode: 'hash' } },
    }),
  )
  write(
    target,
    'apps/uni-app/index.html',
    `<!doctype html>
<html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><link rel="icon" href="./favicon.ico"><title>${projectName}</title></head>
<body><div id="app"><!--app-html--></div><script type="module" src="/src/main.ts"></script></body></html>
`,
  )
  write(
    target,
    'apps/uni-app/vite.config.ts',
    `import { dirname, resolve } from 'node:path'
import { existsSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import {
  createKratosUniPlugin,
  defineConfig,
  kratosApp,
  loadEnv,
  viteMessage,
  type ConfigEnv,
  type UserConfig,
} from '@liujitcn/kratos-uni-app-core/vite'
import { moduleManifest } from './src/module-manifest'
const workspaceRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const coreRoot = resolve(workspaceRoot, 'node_modules/@liujitcn/kratos-uni-app-core')
const systemRoot = resolve(workspaceRoot, 'node_modules/@liujitcn/kratos-uni-app-system')

function resolveHttpsOptions(env, root) {
  if (env.VITE_APP_HTTPS !== 'true') return undefined
  const keyPath = resolve(root, env.VITE_APP_HTTPS_KEY || '../../certs/dev-key.pem')
  const certPath = resolve(root, env.VITE_APP_HTTPS_CERT || '../../certs/dev-cert.pem')
  if (!existsSync(keyPath) || !existsSync(certPath)) {
    throw new Error(viteMessage('https_certificate_missing', { keyPath, certPath }))
  }
  return { key: readFileSync(keyPath), cert: readFileSync(certPath) }
}

function resolveEnv(mode) {
  const modeEnv = loadEnv(mode, workspaceRoot, '')
  if (mode === 'development-h5') return { ...loadEnv('development', workspaceRoot, ''), ...modeEnv }
  if (mode === 'production-h5') return { ...loadEnv('production', workspaceRoot, ''), ...modeEnv }
  return modeEnv
}

export default defineConfig(({ mode }) => {
  const env = resolveEnv(mode)
  const devEnv = mode === 'development-h5' ? loadEnv('development', workspaceRoot, '') : env
  const base = env.VITE_APP_BASE_PATH || (mode === 'production-h5' ? '/app/' : '/')
  const httpsOptions = resolveHttpsOptions(env, workspaceRoot)
  const apiProxyOptions = devEnv.VITE_APP_API_URL?.startsWith('https://') ? { secure: false } : {}
  return {
    base,
    envDir: workspaceRoot,
    resolve: {
      preserveSymlinks: true,
      alias: [
        { find: '@', replacement: resolve(coreRoot, 'src') },
        { find: '@system', replacement: resolve(systemRoot, 'src') },
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
      port: Number(env.VITE_APP_PORT || 5004),
      https: httpsOptions,
      proxy: {
        [env.VITE_APP_BASE_API || '/api']: { changeOrigin: true, target: devEnv.VITE_APP_API_URL, ...apiProxyOptions },
        '/events': { changeOrigin: true, target: devEnv.VITE_APP_API_URL, ...apiProxyOptions },
      },
    },
    build: {
      ...(process.env.UNI_OUTPUT_DIR ? { outDir: process.env.UNI_OUTPUT_DIR, emptyOutDir: true } : {}),
      sourcemap: process.env.NODE_ENV === 'development',
    },
    plugins: [kratosApp({ modules: moduleManifest }), createKratosUniPlugin()],
  }
})
`,
  )
  write(
    target,
    'tsconfig.json',
    json({
      extends: '@vue/tsconfig/tsconfig.json',
      compilerOptions: {
        allowJs: true,
        moduleResolution: 'Bundler',
        skipLibCheck: true,
        types: ['@dcloudio/types', 'miniprogram-api-typings'],
      },
      include: ['apps/**/*.ts', 'apps/**/*.vue', 'packages/**/*.ts'],
    }),
  )
  modules.forEach((name) => {
    write(
      target,
      `packages/modules/${name}/package.json`,
      json({
        name: `@local/${name}`,
        description: `Local Kratos uni-app module for ${name}.`,
        version: '0.0.1',
        type: 'module',
        exports: {
          '.': {
            types: './src/index.d.mts',
            import: './src/index.mjs',
            default: './src/index.mjs',
          },
          './views/*': './src/views/*',
          './package.json': './package.json',
        },
        scripts: { build: 'pnpm pack --pack-destination ../../../dist/npm' },
        dependencies: { '@liujitcn/kratos-uni-app-core': `^${publicPackageVersion}` },
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
      `packages/modules/${name}/src/index.mjs`,
      `import { defineKratosAppModule } from '@liujitcn/kratos-uni-app-core/module'
import { LOCALE_MESSAGES } from './locales/generated.mjs'

/** ${name} 业务模块，注册本地页面与语言资源。 */
export default defineKratosAppModule({ name: '@local/${name}', pages: {}, views: {}, messages: LOCALE_MESSAGES })
`,
    )
    write(
      target,
      `packages/modules/${name}/src/index.d.mts`,
      `declare const module: import('@liujitcn/kratos-uni-app-core/module').KratosAppModule
export default module
`,
    )
  })
  modules.forEach((name) => writeModuleLocales(target, name))
  execFileSync(process.execPath, [resolve(target, 'scripts/sync-locales.mjs'), '--write'], {
    stdio: 'inherit',
  })
  return target
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
    `VITE_APP_PORT=5004\nVITE_APP_HTTPS=false\n# VITE_APP_HTTPS_KEY=../../certs/dev-key.pem\n# VITE_APP_HTTPS_CERT=../../certs/dev-cert.pem\n# ${cliMessage('https_environment_hint')}\nVITE_APP_BASE_PATH=/\nVITE_APP_BASE_API=/api\nVITE_APP_API_URL=http://localhost:7001\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=http://localhost:7001\n`,
  )
  write(
    target,
    '.env.production',
    'VITE_APP_BASE_API=/api\nVITE_APP_API_URL=\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=\n',
  )
  write(
    target,
    '.env.production-h5',
    'VITE_APP_BASE_PATH=/app/\nVITE_APP_BASE_API=/api\nVITE_APP_API_URL=\nVITE_APP_STATIC_API=\nVITE_APP_STATIC_URL=\n',
  )
}

/** 校验本地业务模块名称。 */
function validateModuleName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(cliMessage('module_name_invalid', { name }))
}

function validatePackageName(name) {
  if (!/^(?:@[a-z0-9-]+\/)?[a-z0-9-]+$/.test(name))
    throw new Error(cliMessage('package_name_invalid', { name }))
}

function write(root, file, content) {
  const target = resolve(root, file)
  mkdirSync(resolve(target, '..'), { recursive: true })
  writeFileSync(target, content)
}

function json(value) {
  return `${JSON.stringify(value, null, 2)}\n`
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
      const content = Object.entries(tokens).reduce(
        (value, [key, replacement]) => value.replaceAll(key, replacement),
        readFileSync(input, 'utf8'),
      )
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
