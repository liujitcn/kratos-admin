import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmdirSync,
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
let sourceVersionCounter = 0
type BuildTransaction = {
  inputDir: string
  pagesFile: string
  originalPages: string
  generatedFiles: string[]
  generatedStaticFiles: string[]
  ownerPid?: number
  ready?: boolean
}

type PageAssembly = {
  transaction: BuildTransaction
  leaseFile: string
}

const PAGE_ASSEMBLY_WAIT_TIMEOUT_MS = 30_000
const PAGE_ASSEMBLY_POLL_INTERVAL_MS = 25

/** uni-app Vite 插件参数。 */
export interface KratosAppViteOptions {
  /** 宿主模块清单，顺序同时决定覆盖优先级。 */
  modules: KratosAppModule[]
}

/** 创建 uni-app 官方 Vite 插件。 */
export function createKratosUniPlugin() {
  recoverStalePageTransaction()
  const createPlugin =
    typeof uniPlugin === 'function'
      ? uniPlugin
      : (uniPlugin as unknown as { default: typeof uniPlugin }).default
  return createPlugin()
}

/** 创建自动页面装配插件。 */
export function kratosApp(options: KratosAppViteOptions): Plugin {
  const sourceVersion = `${process.pid}-${Date.now()}-${++sourceVersionCounter}`
  const state: {
    pagesFile?: string
    originalPages?: string
    cleaned: boolean
    persistent: boolean
    transactionFile?: string
    leaseFile?: string
    moduleRoots: Map<string, string>
  } = {
    cleaned: false,
    persistent: false,
    moduleRoots: new Map(),
  }

  const cleanup = () => {
    if (state.cleaned) return
    state.cleaned = true
    process.removeListener('exit', cleanup)
    process.removeListener('SIGINT', handleSigint)
    process.removeListener('SIGTERM', handleSigterm)
    if (state.transactionFile && state.leaseFile) {
      releasePageAssembly(state.transactionFile, state.leaseFile)
    } else if (state.pagesFile && state.originalPages !== undefined) {
      writeFileSync(state.pagesFile, state.originalPages)
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
    name: 'kratos-uni-app-pages',
    enforce: 'pre',
    configResolved(config) {
      state.persistent = config.command === 'serve' || Boolean(config.build.watch)
    },
    resolveId(source, importer) {
      const [sourcePath, query = ''] = source.split('?', 2)
      let target: string | undefined
      if (
        sourcePath === '@liujitcn/kratos-uni-app-core' ||
        sourcePath.startsWith('@liujitcn/kratos-uni-app-core/')
      ) {
        const coreRoot = state.moduleRoots.get('@liujitcn/kratos-uni-app-core')
        if (coreRoot) {
          target = resolveWorkspacePackageSource(
            coreRoot,
            sourcePath.slice('@liujitcn/kratos-uni-app-core'.length).replace(/^\/+/, ''),
          )
        }
      } else if (
        sourcePath === '@liujitcn/kratos-uni-app-system' ||
        sourcePath.startsWith('@liujitcn/kratos-uni-app-system/')
      ) {
        const systemRoot = state.moduleRoots.get('@liujitcn/kratos-uni-app-system')
        if (systemRoot) {
          target = resolveWorkspacePackageSource(
            systemRoot,
            sourcePath.slice('@liujitcn/kratos-uni-app-system'.length).replace(/^\/+/, ''),
          )
        }
      } else if (sourcePath.startsWith('@kratos-uni-app-system-source/')) {
        const systemRoot = state.moduleRoots.get('@liujitcn/kratos-uni-app-system')
        if (systemRoot) {
          target = resolve(
            systemRoot,
            'src',
            sourcePath.slice('@kratos-uni-app-system-source/'.length),
          )
        }
      } else if (sourcePath.startsWith('@kratos-uni-app-core-source/')) {
        const coreRoot = state.moduleRoots.get('@liujitcn/kratos-uni-app-core')
        if (coreRoot) {
          target = resolve(coreRoot, 'src', sourcePath.slice('@kratos-uni-app-core-source/'.length))
        }
      } else if (sourcePath.startsWith('.') && importer) {
        const importerPath = importer.split('?', 1)[0]
        const moduleRoot = [...state.moduleRoots.values()].find((root) =>
          isWithinRoot(resolve(root, 'src'), importerPath),
        )
        if (moduleRoot) target = resolve(dirname(importerPath), sourcePath)
      }
      if (!target) return
      const resolved = resolveSourceFile(target)
      const importQuery = query || dependencyVersionQuery(importer, sourceVersion)
      return `${resolved}${importQuery ? `?${importQuery}` : ''}`
    },
    transform(code, id) {
      const sourceFile = id.split('?', 1)[0]
      const isModuleSource = [...state.moduleRoots.values()].some((root) =>
        isWithinRoot(resolve(root, 'src'), sourceFile),
      )
      const isLinkedModuleSource =
        sourceFile.includes('/@liujitcn/kratos-uni-app-core/') ||
        sourceFile.includes('/@liujitcn/kratos-uni-app-system/')
      if (!isModuleSource && !isLinkedModuleSource) return
      const transformed = code
        .replace(/@system\//g, '@kratos-uni-app-system-source/')
        .replace(/@\//g, '@kratos-uni-app-core-source/')
      if (transformed === code) return
      return { code: transformed, map: null }
    },
    config(): UserConfig {
      const inputDir = process.env.UNI_INPUT_DIR || resolve(process.cwd(), 'src')
      const pagesFile = resolve(inputDir, 'pages.json')
      const transactionFile = resolve(inputDir, '../.kratos-uni-app-pages-state.json')
      const hostRequire = createRequire(resolve(inputDir, '../package.json'))
      const originalPages = readFileSync(pagesFile, 'utf8')
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

      state.pagesFile = pagesFile
      state.transactionFile = transactionFile
      const pageAssembly = preparePageAssembly({
        inputDir,
        pagesFile,
        transactionFile,
        originalPages,
        moduleRoots,
        pageMap: [...pageMap.values()],
        sourceVersion,
      })
      state.originalPages = pageAssembly.transaction.originalPages
      state.leaseFile = pageAssembly.leaseFile
      process.once('exit', cleanup)
      process.prependOnceListener('SIGINT', handleSigint)
      process.prependOnceListener('SIGTERM', handleSigterm)
      return {
        resolve: {
          alias: [
            {
              find: '@kratos-uni-app-core-source',
              replacement: resolve(
                state.moduleRoots.get('@liujitcn/kratos-uni-app-core') ?? '',
                'src',
              ),
            },
            {
              find: '@kratos-uni-app-system-source',
              replacement: resolve(
                state.moduleRoots.get('@liujitcn/kratos-uni-app-system') ?? '',
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
    closeBundle() {
      if (!state.persistent) cleanup()
    },
    closeWatcher: cleanup,
  }
}

function preparePageAssembly(options: {
  inputDir: string
  pagesFile: string
  transactionFile: string
  originalPages: string
  moduleRoots: Array<{ root: string }>
  pageMap: ScannedPage[]
  sourceVersion: string
}): PageAssembly {
  const lockFile = `${options.transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    let transaction = readBuildTransaction(
      options.transactionFile,
      options.inputDir,
      options.pagesFile,
    )
    if (transaction && isPageAssemblyActive(transaction)) {
      return {
        transaction,
        leaseFile: createPageAssemblyLease(options.transactionFile, options.sourceVersion),
      }
    }

    let originalPages = options.originalPages
    if (transaction) {
      originalPages = transaction.originalPages
      restoreBuildTransaction(transaction, options.transactionFile)
    }

    const manifest = JSON.parse(originalPages) as PagesManifest
    manifest.pages ??= []
    const generatedFiles = options.pageMap
      .map((page) => resolve(options.inputDir, `${page.route}.vue`))
      .filter((target) => !existsSync(target))
    const staticTarget = resolve(options.inputDir, 'static')
    const staticFiles = collectModuleStaticFiles(
      options.moduleRoots.map(({ root }) => resolve(root, 'src/static')),
      staticTarget,
    )
    const generatedStaticFiles = staticFiles.map(({ target }) => target)
    transaction = {
      inputDir: options.inputDir,
      pagesFile: options.pagesFile,
      originalPages,
      generatedFiles,
      generatedStaticFiles,
      ownerPid: process.pid,
      ready: false,
    }
    writeBuildTransaction(options.transactionFile, transaction, 'wx')
    const leaseFile = createPageAssemblyLease(options.transactionFile, options.sourceVersion)
    try {
      options.pageMap.forEach((page) => {
        const target = resolve(options.inputDir, `${page.route}.vue`)
        if (existsSync(target)) return
        mkdirSync(dirname(target), { recursive: true })
        writeFileSync(target, createPageWrapper(page))
        appendPage(manifest, page)
      })

      staticFiles.forEach(({ source, target }) => {
        mkdirSync(dirname(target), { recursive: true })
        copyFileSync(source, target)
      })
      writeFileSync(options.pagesFile, `${JSON.stringify(manifest, null, 2)}\n`)
      transaction.ready = true
      writeBuildTransaction(options.transactionFile, transaction)
      return { transaction, leaseFile }
    } catch (error) {
      releasePageAssemblyLease(leaseFile)
      restoreBuildTransaction(transaction, options.transactionFile)
      throw error
    }
  } finally {
    releasePageAssemblyLock(lockFile)
  }
}

function readBuildTransaction(
  transactionFile: string,
  inputDir: string,
  pagesFile: string,
): BuildTransaction | undefined {
  if (!existsSync(transactionFile)) return
  const transaction = JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
  if (transaction.inputDir !== inputDir || transaction.pagesFile !== pagesFile) {
    throw new Error(`uni-app 构建事务目录不匹配：${transactionFile}`)
  }
  return transaction
}

function writeBuildTransaction(
  transactionFile: string,
  transaction: BuildTransaction,
  flag?: 'wx',
): void {
  writeFileSync(
    transactionFile,
    `${JSON.stringify(transaction, null, 2)}\n`,
    flag ? { flag } : undefined,
  )
}

function restoreBuildTransaction(transaction: BuildTransaction, transactionFile: string): void {
  for (const file of [...transaction.generatedFiles].reverse()) {
    if (existsSync(file)) unlinkSync(file)
    removeEmptyParents(dirname(file), transaction.inputDir)
  }
  removeGeneratedStaticFiles(
    transaction.generatedStaticFiles,
    resolve(transaction.inputDir, 'static'),
  )
  writeFileSync(transaction.pagesFile, transaction.originalPages)
  if (existsSync(transactionFile)) unlinkSync(transactionFile)
}

function releasePageAssembly(transactionFile: string, leaseFile: string): void {
  releasePageAssemblyLease(leaseFile)
  const lockFile = `${transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    if (!existsSync(transactionFile)) return
    const transaction = JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
    const current = readBuildTransaction(
      transactionFile,
      transaction.inputDir,
      transaction.pagesFile,
    )
    if (!current || isPageAssemblyActive(current)) return
    restoreBuildTransaction(current, transactionFile)
  } finally {
    releasePageAssemblyLock(lockFile)
  }
}

function createPageAssemblyLease(transactionFile: string, sourceVersion: string): string {
  const leaseDirectory = `${transactionFile}.leases`
  mkdirSync(leaseDirectory, { recursive: true })
  const leaseFile = resolve(leaseDirectory, `${process.pid}-${sourceVersion}.lease`)
  writeFileSync(leaseFile, `${JSON.stringify({ ownerPid: process.pid })}\n`, { flag: 'wx' })
  return leaseFile
}

function releasePageAssemblyLease(leaseFile: string): void {
  if (existsSync(leaseFile)) unlinkSync(leaseFile)
  const leaseDirectory = dirname(leaseFile)
  if (existsSync(leaseDirectory) && !readdirSync(leaseDirectory).length) {
    try {
      rmdirSync(leaseDirectory)
    } catch (error) {
      const code = (error as NodeJS.ErrnoException).code
      if (code !== 'ENOTEMPTY' && code !== 'EEXIST') throw error
    }
  }
}

function isPageAssemblyActive(transaction: BuildTransaction): boolean {
  const leaseDirectory = resolve(
    dirname(transaction.pagesFile),
    '../.kratos-uni-app-pages-state.json.leases',
  )
  if (listActivePageLeases(leaseDirectory).length) return true
  return (
    transaction.ownerPid !== undefined &&
    transaction.ownerPid !== process.pid &&
    isProcessRunning(transaction.ownerPid)
  )
}

function listActivePageLeases(leaseDirectory: string): number[] {
  if (!existsSync(leaseDirectory)) return []
  const activePids: number[] = []
  for (const entry of readdirSync(leaseDirectory)) {
    const leaseFile = resolve(leaseDirectory, entry)
    try {
      const lease = JSON.parse(readFileSync(leaseFile, 'utf8')) as { ownerPid?: number }
      if (lease.ownerPid && isProcessRunning(lease.ownerPid)) {
        activePids.push(lease.ownerPid)
      } else {
        unlinkSync(leaseFile)
      }
    } catch {
      unlinkSync(leaseFile)
    }
  }
  return activePids
}

function acquirePageAssemblyLock(lockFile: string): void {
  const deadline = Date.now() + PAGE_ASSEMBLY_WAIT_TIMEOUT_MS
  while (true) {
    try {
      writeFileSync(lockFile, `${JSON.stringify({ ownerPid: process.pid })}\n`, { flag: 'wx' })
      return
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'EEXIST') throw error
      const ownerPid = readPageAssemblyLockOwner(lockFile)
      if (ownerPid !== undefined && !isProcessRunning(ownerPid)) {
        unlinkSync(lockFile)
        continue
      }
      if (Date.now() >= deadline) {
        throw new Error(`等待 uni-app 页面装配锁超时：${lockFile}`)
      }
      waitForPageAssembly(PAGE_ASSEMBLY_POLL_INTERVAL_MS)
    }
  }
}

function releasePageAssemblyLock(lockFile: string): void {
  if (!existsSync(lockFile)) return
  if (readPageAssemblyLockOwner(lockFile) === process.pid) unlinkSync(lockFile)
}

function readPageAssemblyLockOwner(lockFile: string): number | undefined {
  try {
    return (JSON.parse(readFileSync(lockFile, 'utf8')) as { ownerPid?: number }).ownerPid
  } catch {
    return
  }
}

function waitForPageAssembly(milliseconds: number): void {
  const buffer = new SharedArrayBuffer(4)
  Atomics.wait(new Int32Array(buffer), 0, 0, milliseconds)
}

// recoverStalePageTransaction 在官方 uni 插件创建前恢复页面事务，避免官方插件先读取到空 pages.json。
export function recoverStalePageTransaction(): void {
  const inputDir = process.env.UNI_INPUT_DIR || resolve(process.cwd(), 'src')
  const transactionFile = resolve(inputDir, '../.kratos-uni-app-pages-state.json')
  const pagesFile = resolve(inputDir, 'pages.json')
  const lockFile = `${transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    recoverBuildTransaction(transactionFile, inputDir, pagesFile)
  } finally {
    releasePageAssemblyLock(lockFile)
  }
}

function recoverBuildTransaction(
  transactionFile: string,
  inputDir: string,
  pagesFile: string,
): void {
  const transaction = readBuildTransaction(transactionFile, inputDir, pagesFile)
  if (!transaction || isPageAssemblyActive(transaction)) return
  restoreBuildTransaction(transaction, transactionFile)
}

// isProcessRunning 判断页面事务的所属进程是否仍然存活。
function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0)
    return true
  } catch (error) {
    return (error as NodeJS.ErrnoException).code === 'EPERM'
  }
}

function collectModuleStaticFiles(
  sourceRoots: string[],
  targetRoot: string,
): Array<{ source: string; target: string }> {
  const fileMap = new Map<string, { source: string; target: string }>()
  sourceRoots.forEach((sourceRoot) => {
    scanFiles(sourceRoot).forEach((source) => {
      const target = resolve(targetRoot, relative(sourceRoot, source))
      if (!existsSync(target)) fileMap.set(target, { source, target })
    })
  })
  return [...fileMap.values()]
}

function scanFiles(root: string): string[] {
  if (!existsSync(root)) return []
  const files: string[] = []
  const visit = (directory: string) => {
    readdirSync(directory).forEach((entry) => {
      const path = resolve(directory, entry)
      if (statSync(path).isDirectory()) visit(path)
      else files.push(path)
    })
  }
  visit(root)
  return files
}

function removeGeneratedStaticFiles(files: string[], staticRoot: string): void {
  for (const file of [...files].reverse()) {
    if (existsSync(file)) unlinkSync(file)
    removeEmptyParents(dirname(file), staticRoot)
  }
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
    if (manifest.pages.some((item) => normalizeRoute(item.path) === page.route)) return
    manifest.pages.push({ path: page.route, style: page.style })
    return
  }
  manifest.subPackages ??= []
  let subPackage = manifest.subPackages.find((item) => item.root === first)
  if (!subPackage) {
    subPackage = { root: first, pages: [] }
    manifest.subPackages.push(subPackage)
  }
  if (subPackage.pages.some((item) => normalizeRoute(`${first}/${item.path}`) === page.route))
    return
  subPackage.pages.push({ path: rest.join('/'), style: page.style })
}

function createPageWrapper(page: ScannedPage): string {
  const source = page.source.replace(/\\/g, '/')
  return `<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { setAppPageTitle } from '@liujitcn/kratos-uni-app-core'
import KratosPage from ${JSON.stringify(source)}
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.vue'

defineOptions({ inheritAttrs: false })
onShow(() => setAppPageTitle(${JSON.stringify(page.route)}))
</script>

<template>
  <KratosPage v-bind="$attrs" />
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
  const resolved =
    candidates.find((candidate) => existsSync(candidate) && statSync(candidate).isFile()) ?? target
  if (!isAbsolute(resolved)) throw new Error(`模块源码路径不是绝对路径：${resolved}`)
  return resolved
}

// dependencyVersionQuery 继承依赖版本并隔离开发服务缓存，避免源码重复实例或跨服务过期。
function dependencyVersionQuery(importer: string | undefined, sourceVersion: string): string {
  const query = importer?.split('?', 2)[1] ?? ''
  const version = new URLSearchParams(query).get('v')
  const params = new URLSearchParams()
  if (version) params.set('v', version)
  params.set('kratos', sourceVersion)
  return params.toString()
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
