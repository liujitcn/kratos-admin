import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmdirSync,
  rmSync,
  statSync,
  unlinkSync,
  writeFileSync,
} from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, isAbsolute, relative, resolve } from 'node:path'
import uniPlugin from '@dcloudio/vite-plugin-uni'
import { defineConfig, loadEnv } from 'vite'
import type { Plugin, UserConfig } from 'vite'
import type { KratosAppModule, KratosAppPageConfig } from './module'

export { defineConfig, loadEnv }
export type { ConfigEnv, UserConfig } from 'vite'

type PageEntry = { path: string; style?: Record<string, unknown> }
type SubPackage = { root: string; pages: PageEntry[] }
type PagesManifest = {
  pages: PageEntry[]
  subPackages?: SubPackage[]
  [key: string]: unknown
}
type ScannedPage = {
  route: string
  source: string
  style?: Record<string, unknown>
}
type BuildTransaction = {
  inputDir: string
  pagesFile: string
  originalPages: string
  generatedFiles: string[]
  generatedStatic: boolean
}

/** app Vite 插件参数。 */
export interface KratosAppViteOptions {
  /** 宿主模块清单，顺序同时决定覆盖优先级。 */
  modules: KratosAppModule[]
}

/** 创建 uni-app 官方 Vite 插件。 */
export function createKratosUniPlugin() {
  const createPlugin =
    typeof uniPlugin === 'function'
      ? uniPlugin
      : (uniPlugin as unknown as { default: typeof uniPlugin }).default
  return createPlugin()
}

/** 创建自动页面装配插件。 */
export function kratosApp(options: KratosAppViteOptions): Plugin {
  const state: {
    inputDir?: string
    pagesFile?: string
    originalPages?: string
    generatedFiles: string[]
    generatedStatic: boolean
    cleaned: boolean
    transactionFile?: string
    moduleRoots: Map<string, string>
  } = { generatedFiles: [], generatedStatic: false, cleaned: false, moduleRoots: new Map() }

  const cleanup = () => {
    if (state.cleaned) return
    state.cleaned = true
    process.removeListener('exit', cleanup)
    process.removeListener('SIGINT', handleSigint)
    process.removeListener('SIGTERM', handleSigterm)
    state.generatedFiles.reverse().forEach((file) => {
      if (existsSync(file)) unlinkSync(file)
      removeEmptyParents(dirname(file), state.inputDir ?? '')
    })
    if (state.generatedStatic && state.inputDir) {
      rmSync(resolve(state.inputDir, 'static'), { recursive: true, force: true })
    }
    if (state.pagesFile && state.originalPages !== undefined) {
      writeFileSync(state.pagesFile, state.originalPages)
    }
    if (state.transactionFile && existsSync(state.transactionFile)) {
      unlinkSync(state.transactionFile)
    }
  }
  const handleSigint = () => {
    cleanup()
    process.exit(130)
  }
  const handleSigterm = () => {
    cleanup()
    process.exit(143)
  }

  return {
    name: 'kratos-app-pages',
    enforce: 'pre',
    resolveId(source) {
      const [sourcePath, query = ''] = source.split('?', 2)
      let target: string | undefined
      if (
        sourcePath === '@liujitcn/kratos-app-core' ||
        sourcePath.startsWith('@liujitcn/kratos-app-core/')
      ) {
        const coreRoot = state.moduleRoots.get('@liujitcn/kratos-app-core')
        if (coreRoot) {
          target = resolveWorkspacePackageSource(
            coreRoot,
            sourcePath.slice('@liujitcn/kratos-app-core'.length).replace(/^\/+/, ''),
          )
        }
      } else if (
        sourcePath === '@liujitcn/kratos-app-system' ||
        sourcePath.startsWith('@liujitcn/kratos-app-system/')
      ) {
        const systemRoot = state.moduleRoots.get('@liujitcn/kratos-app-system')
        if (systemRoot) {
          target = resolveWorkspacePackageSource(
            systemRoot,
            sourcePath.slice('@liujitcn/kratos-app-system'.length).replace(/^\/+/, ''),
          )
        }
      } else if (sourcePath.startsWith('@kratos-app-system-source/')) {
        const systemRoot = state.moduleRoots.get('@liujitcn/kratos-app-system')
        if (systemRoot) {
          target = resolve(systemRoot, 'src', sourcePath.slice('@kratos-app-system-source/'.length))
        }
      } else if (sourcePath.startsWith('@kratos-app-core-source/')) {
        const coreRoot = state.moduleRoots.get('@liujitcn/kratos-app-core')
        if (coreRoot) {
          target = resolve(coreRoot, 'src', sourcePath.slice('@kratos-app-core-source/'.length))
        }
      }
      if (!target) return
      const resolved = resolveSourceFile(target)
      return `${resolved}${query ? `?${query}` : ''}`
    },
    transform(code, id) {
      const sourceFile = id.split('?', 1)[0]
      const isModuleSource = [...state.moduleRoots.values()].some((root) =>
        isWithinRoot(resolve(root, 'src'), sourceFile),
      )
      const isLinkedModuleSource =
        sourceFile.includes('/@liujitcn/kratos-app-core/') ||
        sourceFile.includes('/@liujitcn/kratos-app-system/')
      if (!isModuleSource && !isLinkedModuleSource) return
      const transformed = code
        .replace(/@system\//g, '@kratos-app-system-source/')
        .replace(/@\//g, '@kratos-app-core-source/')
      if (transformed === code) return
      return { code: transformed, map: null }
    },
    config(): UserConfig {
      const inputDir = process.env.UNI_INPUT_DIR || resolve(process.cwd(), 'src')
      const pagesFile = resolve(inputDir, 'pages.json')
      const transactionFile = resolve(inputDir, '../.kratos-app-pages-state.json')
      recoverBuildTransaction(transactionFile, inputDir, pagesFile)
      const hostRequire = createRequire(resolve(inputDir, '../package.json'))
      const originalPages = readFileSync(pagesFile, 'utf8')
      const manifest = JSON.parse(originalPages) as PagesManifest
      manifest.pages ??= []
      const pageMap = new Map<string, ScannedPage>()
      const moduleRoots = options.modules.map((module) => {
        const linkedRoot = resolve(inputDir, '../node_modules', ...module.name.split('/'))
        return {
          module,
          root: existsSync(resolve(linkedRoot, 'package.json'))
            ? linkedRoot
            : dirname(hostRequire.resolve(`${module.name}/package.json`)),
        }
      })
      state.moduleRoots = new Map(moduleRoots.map(({ module, root }) => [module.name, root]))

      moduleRoots.forEach(({ module, root }) => {
        scanViews(resolve(root, 'src/views')).forEach((page) => {
          const config = module.pages[page.route] ?? {}
          const route = normalizeRoute(config.route ?? page.route)
          pageMap.set(route, {
            route,
            source: `${module.name}/views/${page.route}.vue`,
            style: config.style,
          })
        })
      })

      state.inputDir = inputDir
      state.pagesFile = pagesFile
      state.originalPages = originalPages
      state.transactionFile = transactionFile
      state.generatedFiles = [...pageMap.values()]
        .map((page) => resolve(inputDir, `${page.route}.vue`))
        .filter((target) => !existsSync(target))
      state.generatedStatic = !existsSync(resolve(inputDir, 'static'))
      writeFileSync(
        transactionFile,
        `${JSON.stringify(
          {
            inputDir,
            pagesFile,
            originalPages,
            generatedFiles: state.generatedFiles,
            generatedStatic: state.generatedStatic,
          } satisfies BuildTransaction,
          null,
          2,
        )}\n`,
      )
      pageMap.forEach((page) => {
        const target = resolve(inputDir, `${page.route}.vue`)
        if (existsSync(target)) return
        mkdirSync(dirname(target), { recursive: true })
        writeFileSync(target, createPageWrapper(page))
        appendPage(manifest, page)
      })

      const staticTarget = resolve(inputDir, 'static')
      if (state.generatedStatic) {
        moduleRoots.forEach(({ root }) => {
          const source = resolve(root, 'src/static')
          if (existsSync(source)) cpSync(source, staticTarget, { recursive: true })
        })
      }
      writeFileSync(pagesFile, `${JSON.stringify(manifest, null, 2)}\n`)
      process.once('exit', cleanup)
      process.prependOnceListener('SIGINT', handleSigint)
      process.prependOnceListener('SIGTERM', handleSigterm)
      return {
        resolve: {
          alias: [
            {
              find: '@kratos-app-core-source',
              replacement: resolve(state.moduleRoots.get('@liujitcn/kratos-app-core') ?? '', 'src'),
            },
            {
              find: '@kratos-app-system-source',
              replacement: resolve(
                state.moduleRoots.get('@liujitcn/kratos-app-system') ?? '',
                'src',
              ),
            },
          ],
        },
        optimizeDeps: {
          exclude: options.modules.map((module) => module.name),
        },
      }
    },
    configureServer(server) {
      server.httpServer?.once('close', cleanup)
    },
    closeBundle: cleanup,
  }
}

function recoverBuildTransaction(
  transactionFile: string,
  inputDir: string,
  pagesFile: string,
): void {
  if (!existsSync(transactionFile)) return
  const transaction = JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
  if (transaction.inputDir !== inputDir || transaction.pagesFile !== pagesFile) {
    throw new Error(`app 构建事务目录不匹配：${transactionFile}`)
  }
  transaction.generatedFiles.reverse().forEach((file) => {
    if (existsSync(file)) unlinkSync(file)
    removeEmptyParents(dirname(file), inputDir)
  })
  if (transaction.generatedStatic) {
    rmSync(resolve(inputDir, 'static'), { recursive: true, force: true })
  }
  writeFileSync(pagesFile, transaction.originalPages)
  unlinkSync(transactionFile)
}

function scanViews(root: string): Array<{ route: string; source: string }> {
  if (!existsSync(root)) return []
  const files: string[] = []
  const visit = (directory: string) => {
    readdirSync(directory).forEach((entry) => {
      const path = resolve(directory, entry)
      if (statSync(path).isDirectory()) {
        if (entry !== 'components') visit(path)
      } else if (entry.endsWith('.vue')) {
        files.push(path)
      }
    })
  }
  visit(root)
  return files.map((source) => ({
    route: normalizeRoute(relative(root, source)),
    source,
  }))
}

function appendPage(manifest: PagesManifest, page: ScannedPage): void {
  const [first, ...rest] = page.route.split('/')
  if (!first.startsWith('pages') || first === 'pages') {
    manifest.pages.push({ path: page.route, style: page.style })
    return
  }
  manifest.subPackages ??= []
  let subPackage = manifest.subPackages.find((item) => item.root === first)
  if (!subPackage) {
    subPackage = { root: first, pages: [] }
    manifest.subPackages.push(subPackage)
  }
  subPackage.pages.push({ path: rest.join('/'), style: page.style })
}

function createPageWrapper(page: ScannedPage): string {
  const source = page.source.replace(/\\/g, '/')
  return `<script setup lang="ts">
import KratosPage from ${JSON.stringify(source)}
import { KratosTabBar } from '@liujitcn/kratos-app-core'

defineOptions({ inheritAttrs: false })
</script>

<template>
  <KratosPage />
  <KratosTabBar route=${JSON.stringify(page.route)} />
</template>
`
}

function normalizeRoute(route: string): string {
  return route
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\.vue$/, '')
}

function removeEmptyParents(directory: string, boundary: string): void {
  let current = directory
  while (current !== boundary && current.startsWith(boundary) && existsSync(current)) {
    if (readdirSync(current).length) return
    rmdirSync(current)
    current = dirname(current)
  }
}

function resolveSourceFile(target: string): string {
  const candidates = [
    target,
    `${target}.ts`,
    `${target}.js`,
    `${target}.vue`,
    resolve(target, 'index.ts'),
    resolve(target, 'index.js'),
  ]
  const resolved = candidates.find((candidate) => existsSync(candidate)) ?? target
  if (!isAbsolute(resolved)) throw new Error(`模块源码路径不是绝对路径：${resolved}`)
  return resolved
}

function resolveWorkspacePackageSource(packageRoot: string, subpath: string): string {
  if (!subpath) return resolve(packageRoot, 'src/index.ts')
  if (subpath === 'module') return resolve(packageRoot, 'src/module.ts')
  if (subpath === 'stores') return resolve(packageRoot, 'src/stores/index.ts')
  return resolve(packageRoot, 'src', subpath)
}

function isWithinRoot(root: string, target: string): boolean {
  const relativePath = relative(root, target)
  return relativePath === '' || (!relativePath.startsWith('..') && !isAbsolute(relativePath))
}
