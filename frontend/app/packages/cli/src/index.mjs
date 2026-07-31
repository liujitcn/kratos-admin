import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { basename, resolve } from 'node:path'

/** 创建独立 kratos app workspace。 */
export function scaffoldKratosApp(targetPath, options = {}) {
  const target = resolve(targetPath)
  if (existsSync(target)) throw new Error(`目标目录已存在：${target}`)
  const projectName = basename(target)
  const modules = [...new Set(options.modules ?? [])]
  const packages = [...new Set(options.packages ?? [])]
  modules.forEach(validateModuleName)
  packages.forEach(validatePackageName)
  mkdirSync(resolve(target, 'apps/app/src/pages/bootstrap'), { recursive: true })
  modules.forEach((name) =>
    mkdirSync(resolve(target, `packages/modules/${name}/src/views`), { recursive: true }),
  )

  write(target, 'pnpm-workspace.yaml', 'packages:\n  - apps/*\n  - packages/modules/*\n')
  write(
    target,
    'package.json',
    json({
      name: projectName,
      description: `Independent pnpm workspace for ${projectName}.`,
      private: true,
      scripts: {
        'dev:h5': 'pnpm --filter @liujitcn/kratos-app dev:h5',
        'dev:mp-weixin': 'pnpm --filter @liujitcn/kratos-app dev:mp-weixin',
        'build:h5': 'pnpm --filter @liujitcn/kratos-app build:h5',
        'build:mp-weixin': 'pnpm --filter @liujitcn/kratos-app build:mp-weixin',
      },
      devDependencies: {
        '@dcloudio/types': 'latest',
        '@vue/compiler-sfc': '^3.4.21',
        '@vue/tsconfig': 'latest',
        'miniprogram-api-typings': 'latest',
        typescript: 'latest',
        'vue-tsc': 'latest',
      },
    }),
  )
  write(
    target,
    'README.md',
    `# ${projectName}

\`${projectName}\` 是由 \`@liujitcn/kratos-app-cli\` 创建的独立 pnpm workspace，支持 H5 和微信小程序。

## 目录结构

\`\`\`text
${projectName}
├── apps/app                 # 私有 uni-app 宿主
├── packages/modules        # workspace 内本地业务模块
├── package.json            # 公共命令和开发依赖
├── pnpm-workspace.yaml     # workspace 包范围
└── tsconfig.json           # 共享 TypeScript 配置
\`\`\`

宿主只负责入口、manifest、模块清单和启动；底座能力由 \`@liujitcn/kratos-app-core\` 提供，默认业务页面由 \`@liujitcn/kratos-app-system\` 提供。本地模块通过各自公开入口接入，不跨包相对引用源码。

## 开发

\`\`\`bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
\`\`\`

模块装配入口是 \`apps/app/src/module-manifest.ts\`。新增、删除或调整模块时同步维护该清单，并在模块自己的 README 中记录页面和接口职责。
`,
  )
  const dependencies = {
    '@liujitcn/kratos-app-core': '^0.0.1',
    '@liujitcn/kratos-app-system': '^0.0.1',
    '@dcloudio/uni-app': 'latest',
    '@dcloudio/uni-components': 'latest',
    '@dcloudio/uni-h5': 'latest',
    '@dcloudio/uni-mp-weixin': 'latest',
    '@dcloudio/uni-ui': 'latest',
    '@dcloudio/vite-plugin-uni': 'latest',
    pinia: '^2.0.27',
    sass: '^1.77.8',
    vite: '^5.2.8',
    vue: '^3.4.21',
    ...Object.fromEntries(modules.map((name) => [`@local/${name}`, 'workspace:*'])),
    ...Object.fromEntries(packages.map((name) => [name, 'latest'])),
  }
  write(
    target,
    'apps/app/package.json',
    json({
      name: '@liujitcn/kratos-app',
      description: `Private uni-app host for ${projectName}.`,
      version: '0.0.1',
      private: true,
      type: 'module',
      scripts: {
        'dev:h5': 'uni --mode development',
        'dev:mp-weixin': 'uni -p mp-weixin',
        'build:h5': 'uni build --mode production',
        'build:mp-weixin': 'uni build -p mp-weixin --mode production',
        tsc: 'vue-tsc --noEmit',
      },
      dependencies,
    }),
  )
  write(
    target,
    'apps/app/README.md',
    `# @liujitcn/kratos-app

\`apps/app\` 是 \`${projectName}\` 的私有 uni-app 宿主，负责组合模块并提供 H5、微信小程序的构建入口，不承载可复用业务实现。

## 目录结构

\`\`\`text
apps/app
├── src
│   ├── pages/bootstrap      # 固定启动页面
│   ├── App.vue              # 应用根组件和全局样式入口
│   ├── main.ts              # Vue 与 Kratos app 启动入口
│   ├── manifest.json        # uni-app 平台配置
│   ├── module-manifest.ts   # 唯一模块清单
│   └── pages.json           # 固定 bootstrap 路由
├── index.html               # H5 HTML 入口
├── package.json             # 宿主命令和运行依赖
└── vite.config.ts           # 模块页面装配与 uni-app 插件
\`\`\`

## 模块装配

\`src/module-manifest.ts\` 默认注册 core 和 system，并按声明顺序加载本地模块与已发布模块。业务页面、API 和状态应放在所属模块，宿主仅维护组合关系和平台配置。

## 命令

在 workspace 根目录执行：

\`\`\`bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
\`\`\`
`,
  )
  const imports = [
    "import { coreModule } from '@liujitcn/kratos-app-core/module'",
    "import systemModule from '@liujitcn/kratos-app-system'",
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
    'apps/app/src/module-manifest.ts',
    `${imports.join('\n')}\n\nexport const moduleManifest = [${members.join(', ')}]\n`,
  )
  write(
    target,
    'apps/app/src/pages.json',
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
    'apps/app/src/pages/bootstrap/index.vue',
    `<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { resolveStaticView } from '@liujitcn/kratos-app-core'
onLoad((options) => {
  const route = resolveStaticView('BOOTSTRAP_LOADING') ?? 'pages/status/index'
  const target = options?.route ? decodeURIComponent(options.route) : 'app/home'
  uni.reLaunch({ url: \`/\${route}?state=BOOTSTRAP_LOADING&bootstrap=1&route=\${encodeURIComponent(target)}\` })
})
</script>
<template><view /></template>
`,
  )
  write(
    target,
    'apps/app/src/main.ts',
    `import {
  bootstrapKratosApp,
  pinia,
  registerKratosAppModules,
} from '@liujitcn/kratos-app-core'
import { createSSRApp } from 'vue'
import App from './App.vue'
import { moduleManifest } from './module-manifest'

registerKratosAppModules(moduleManifest)

export function createApp() {
  return bootstrapKratosApp({ app: App, createSSRApp, pinia, modules: moduleManifest })
}
`,
  )
  write(
    target,
    'apps/app/src/App.vue',
    `<script setup lang="ts"></script>
<style lang="scss">@use '@liujitcn/kratos-app-core/styles/base.scss';</style>
`,
  )
  write(target, 'apps/app/src/uni.scss', "@forward '@liujitcn/kratos-app-core/uni.scss';\n")
  write(
    target,
    'apps/app/src/manifest.json',
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
    'apps/app/index.html',
    `<!doctype html>
<html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><title>${projectName}</title></head>
<body><div id="app"><!--app-html--></div><script type="module" src="/src/main.ts"></script></body></html>
`,
  )
  write(
    target,
    'apps/app/vite.config.ts',
    `import { createKratosUniPlugin, defineConfig, kratosApp } from '@liujitcn/kratos-app-core/vite'
import { moduleManifest } from './src/module-manifest'
export default defineConfig({
  resolve: { preserveSymlinks: true },
  plugins: [kratosApp({ modules: moduleManifest }), createKratosUniPlugin()],
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
        description: `Local Kratos app module for ${name}.`,
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
        dependencies: { '@liujitcn/kratos-app-core': '^0.0.1' },
      }),
    )
    write(
      target,
      `packages/modules/${name}/README.md`,
      `# @local/${name}

\`@local/${name}\` 是 \`${projectName}\` workspace 内的本地业务模块，通过 \`@liujitcn/kratos-app-core\` 的公开接口接入宿主。

## 目录结构

\`\`\`text
packages/modules/${name}
├── src
│   ├── views               # 模块页面；页面私有组件放在就近 components
│   ├── index.d.mts         # 模块入口类型
│   └── index.mjs           # 页面、视图键和图标注册入口
├── package.json            # 模块名称、依赖和 exports
└── README.md               # 模块职责与维护说明
\`\`\`

## 开发约束

- 页面放在 \`src/views\`，并在 \`src/index.mjs\` 的模块定义中登记页面和稳定视图键。
- 请求、认证、导航和状态能力只通过 \`@liujitcn/kratos-app-core\` 的公开 exports 使用。
- 页面替换使用稳定视图键，接口不能直接下发任意组件路径。
- 修改模块后从 workspace 根目录运行对应 H5 或微信小程序命令验证装配结果。
`,
    )
    write(
      target,
      `packages/modules/${name}/src/index.mjs`,
      `import { defineKratosAppModule } from '@liujitcn/kratos-app-core/module'
export default defineKratosAppModule({ name: '@local/${name}', pages: {}, views: {} })
`,
    )
    write(
      target,
      `packages/modules/${name}/src/index.d.mts`,
      `declare const module: import('@liujitcn/kratos-app-core/module').KratosAppModule
export default module
`,
    )
  })
  return target
}

function validateModuleName(name) {
  if (!/^[a-z][a-z0-9-]*$/.test(name)) throw new Error(`模块名无效：${name}`)
}

function validatePackageName(name) {
  if (!/^(?:@[a-z0-9-]+\/)?[a-z0-9-]+$/.test(name)) throw new Error(`包名无效：${name}`)
}

function write(root, file, content) {
  const target = resolve(root, file)
  mkdirSync(resolve(target, '..'), { recursive: true })
  writeFileSync(target, content)
}

function json(value) {
  return `${JSON.stringify(value, null, 2)}\n`
}
