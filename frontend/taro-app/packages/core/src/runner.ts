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
import { dirname, relative, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { spawn } from 'node:child_process'
import ts from 'typescript'
import type { KratosTaroBuildModule } from './build'

type PageEntry = {
  route: string
  source: string
  style?: Record<string, unknown>
}

type AppManifest = {
  pages: string[]
  subpackages?: Array<{ root: string; pages: string[] }>
  [key: string]: unknown
}

type BuildTransaction = {
  inputDir: string
  appConfigFile: string
  originalAppConfig: string
  generatedFiles: string[]
  generatedStaticFiles: string[]
  ownerPid?: number
  ready?: boolean
}

type PageAssembly = {
  transaction: BuildTransaction
  leaseFile: string
}

let sourceVersionCounter = 0
const PAGE_ASSEMBLY_WAIT_TIMEOUT_MS = 30_000
const PAGE_ASSEMBLY_POLL_INTERVAL_MS = 25

type CliOptions = {
  type: 'h5' | 'weapp'
  mode: string
  watch: boolean
  prepareOnly: boolean
  cleanupOnly: boolean
}

/** 运行 Taro 构建并在进程结束后恢复临时页面。 */
async function main(): Promise<void> {
  const options = parseArguments(process.argv.slice(2))
  const hostRoot = process.cwd()
  const inputDir = resolve(hostRoot, 'src')
  const appConfigFile = resolve(inputDir, 'app.config.ts')
  const baseConfigFile = resolve(inputDir, 'app.config.base.json')
  const transactionFile = resolve(hostRoot, '.kratos-taro-app-pages-state.json')
  recoverBuildTransaction(transactionFile, inputDir, appConfigFile)
  if (options.cleanupOnly) return

  const originalAppConfig = readFileSync(appConfigFile, 'utf8')
  const manifest = JSON.parse(readFileSync(baseConfigFile, 'utf8')) as AppManifest
  const modules = await loadBuildModules(hostRoot, resolve(inputDir, 'module-manifest.ts'))
  const pageMap = new Map<string, PageEntry>()
  modules.forEach((module) => {
    scanViews(resolve(module.root, 'src/views')).forEach((page) => {
      const config = module.pages[page.route] ?? {}
      const route = normalizeRoute(config.route ?? page.route)
      pageMap.set(route, {
        route,
        source: `${module.name}/views/${page.route}`,
        style: config.style,
      })
    })
  })

  const pageAssembly = preparePageAssembly({
    appConfigFile,
    baseManifest: manifest,
    inputDir,
    moduleStaticRoots: modules.map((module) => resolve(module.root, 'src/static')),
    originalAppConfig,
    pageMap: [...pageMap.values()],
    transactionFile,
  })

  let cleaned = false
  const cleanup = () => {
    if (cleaned) return
    cleaned = true
    releasePageAssembly(transactionFile, pageAssembly.leaseFile)
  }
  if (options.prepareOnly) {
    return
  }
  const handleSignal = (signal: NodeJS.Signals) => {
    child?.kill(signal)
  }
  let child: ReturnType<typeof spawn> | undefined
  process.once('SIGINT', () => handleSignal('SIGINT'))
  process.once('SIGTERM', () => handleSignal('SIGTERM'))

  try {
    const args = [
      'exec',
      'taro',
      'build',
      '--type',
      options.type,
      '--mode',
      options.mode,
      '--env-prefix',
      'VITE_APP_,TARO_APP_',
    ]
    if (options.watch) args.push('--watch')
    child = spawn(process.platform === 'win32' ? 'pnpm.cmd' : 'pnpm', args, {
      cwd: hostRoot,
      env: {
        ...process.env,
        NODE_ENV: options.mode === 'development' ? 'development' : 'production',
        KRATOS_TARO_OUTPUT_ROOT:
          process.env.KRATOS_TARO_OUTPUT_ROOT ||
          (options.type === 'h5' ? 'dist/h5' : 'dist/mp-weixin'),
      },
      stdio: 'inherit',
    })
    const exitCode = await new Promise<number>((complete, reject) => {
      child?.once('error', reject)
      child?.once('exit', (code, signal) => {
        if (signal) {
          complete(signal === 'SIGINT' ? 130 : 143)
          return
        }
        complete(code ?? 1)
      })
    })
    if (exitCode !== 0) process.exitCode = exitCode
  } finally {
    cleanup()
  }
}

function parseArguments(args: string[]): CliOptions {
  let type: CliOptions['type'] | undefined
  let mode = 'production'
  let watch = false
  let prepareOnly = false
  let cleanupOnly = false
  for (let index = 0; index < args.length; index += 1) {
    const argument = args[index]
    if (argument === '--type') {
      const value = args[++index]
      if (value !== 'h5' && value !== 'weapp') throw new Error(`不支持的 Taro 类型：${value}`)
      type = value
    } else if (argument === '--mode') {
      mode = args[++index] || mode
    } else if (argument === '--watch') {
      watch = true
    } else if (argument === '--prepare-only') {
      prepareOnly = true
    } else if (argument === '--cleanup-only') {
      cleanupOnly = true
    } else {
      throw new Error(`未知参数：${argument}`)
    }
  }
  if (!type) throw new Error('缺少 --type h5|weapp')
  return { type, mode, watch, prepareOnly, cleanupOnly }
}

async function loadBuildModules(
  hostRoot: string,
  manifestFile: string,
): Promise<KratosTaroBuildModule[]> {
  const source = readFileSync(manifestFile, 'utf8')
  const sourceFile = ts.createSourceFile(
    manifestFile,
    source,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TS,
  )
  const imports = new Map<string, string>()
  let moduleNames: string[] = []
  sourceFile.forEachChild((node) => {
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier)) {
      const importClause = node.importClause
      const bindings = node.importClause?.namedBindings
      const packageName = node.moduleSpecifier.text
      if (importClause?.name) imports.set(importClause.name.text, packageName)
      if (bindings && ts.isNamedImports(bindings)) {
        bindings.elements.forEach((element) => {
          imports.set(element.name.text, packageName)
        })
      }
      return
    }
    if (!ts.isVariableStatement(node)) return
    node.declarationList.declarations.forEach((declaration) => {
      if (
        ts.isIdentifier(declaration.name) &&
        declaration.name.text === 'moduleManifest' &&
        declaration.initializer &&
        ts.isArrayLiteralExpression(declaration.initializer)
      ) {
        moduleNames = declaration.initializer.elements
          .filter(ts.isIdentifier)
          .map((element) => element.text)
      }
    })
  })
  if (!moduleNames.length) throw new Error(`模块清单未导出 moduleManifest：${manifestFile}`)

  const hostRequire = createRequire(resolve(hostRoot, 'package.json'))
  return Promise.all(
    moduleNames.map(async (name) => {
      const packageName = imports.get(name)
      if (!packageName) throw new Error(`模块 ${name} 缺少静态 import`)
      const buildEntry = hostRequire.resolve(`${packageName}/build`)
      const loaded = (await import(pathToFileURL(buildEntry).href)) as {
        buildModule?: KratosTaroBuildModule
      }
      if (!loaded.buildModule) throw new Error(`${packageName}/build 未导出 buildModule`)
      return loaded.buildModule
    }),
  )
}

function recoverBuildTransaction(
  transactionFile: string,
  inputDir: string,
  appConfigFile: string,
): void {
  const lockFile = `${transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    const transaction = readBuildTransaction(transactionFile, inputDir, appConfigFile)
    if (!transaction || isPageAssemblyActive(transaction, transactionFile)) return
    restoreBuildTransaction(transaction, transactionFile)
  } finally {
    releasePageAssemblyLock(lockFile)
  }
}

function preparePageAssembly(options: {
  appConfigFile: string
  baseManifest: AppManifest
  inputDir: string
  moduleStaticRoots: string[]
  originalAppConfig: string
  pageMap: PageEntry[]
  transactionFile: string
}): PageAssembly {
  const lockFile = `${options.transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    let transaction = readBuildTransaction(
      options.transactionFile,
      options.inputDir,
      options.appConfigFile,
    )
    if (transaction && isPageAssemblyActive(transaction, options.transactionFile)) {
      return {
        transaction,
        leaseFile: createPageAssemblyLease(options.transactionFile),
      }
    }

    let originalAppConfig = options.originalAppConfig
    if (transaction) {
      originalAppConfig = transaction.originalAppConfig
      restoreBuildTransaction(transaction, options.transactionFile)
    }

    const generatedFiles = options.pageMap.flatMap((page) => {
      const pageFile = resolve(options.inputDir, `${page.route}.tsx`)
      const configFile = resolve(options.inputDir, `${page.route}.config.ts`)
      return [pageFile, configFile].filter(
        (file) => !existsSync(file) || isGeneratedPageFile(file) || isGeneratedPageConfig(file),
      )
    })
    const staticFiles = collectModuleStaticFiles(
      options.moduleStaticRoots,
      resolve(options.inputDir, 'static'),
    )
    transaction = {
      inputDir: options.inputDir,
      appConfigFile: options.appConfigFile,
      originalAppConfig,
      generatedFiles,
      generatedStaticFiles: staticFiles.map(({ target }) => target),
      ownerPid: process.pid,
      ready: false,
    }
    writeBuildTransaction(options.transactionFile, transaction, 'wx')
    const leaseFile = createPageAssemblyLease(options.transactionFile)
    try {
      options.pageMap.forEach((page) => {
        appendPage(options.baseManifest, page.route)
        const pageFile = resolve(options.inputDir, `${page.route}.tsx`)
        const configFile = resolve(options.inputDir, `${page.route}.config.ts`)
        if (!existsSync(pageFile)) {
          mkdirSync(dirname(pageFile), { recursive: true })
          writeFileSync(pageFile, createPageWrapper(page))
        }
        if (!existsSync(configFile)) writeFileSync(configFile, createPageConfig(page))
      })
      staticFiles.forEach(({ source, target }) => {
        mkdirSync(dirname(target), { recursive: true })
        copyFileSync(source, target)
      })
      writeFileSync(
        options.appConfigFile,
        `export default defineAppConfig(${JSON.stringify(options.baseManifest, null, 2)})\n`,
      )
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
  appConfigFile: string,
): BuildTransaction | undefined {
  if (!existsSync(transactionFile)) return
  const transaction = JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
  if (transaction.inputDir !== inputDir || transaction.appConfigFile !== appConfigFile) {
    throw new Error(`Taro 构建事务目录不匹配：${transactionFile}`)
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
  writeFileSync(transaction.appConfigFile, transaction.originalAppConfig)
  if (existsSync(transactionFile)) unlinkSync(transactionFile)
}

function releasePageAssembly(transactionFile: string, leaseFile: string): void {
  releasePageAssemblyLease(leaseFile)
  const lockFile = `${transactionFile}.lock`
  acquirePageAssemblyLock(lockFile)
  try {
    const transaction = readBuildTransactionFromFile(transactionFile)
    if (!transaction || isPageAssemblyActive(transaction, transactionFile)) return
    restoreBuildTransaction(transaction, transactionFile)
  } finally {
    releasePageAssemblyLock(lockFile)
  }
}

function readBuildTransactionFromFile(transactionFile: string): BuildTransaction | undefined {
  if (!existsSync(transactionFile)) return
  return JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
}

function createPageAssemblyLease(transactionFile: string): string {
  const leaseDirectory = `${transactionFile}.leases`
  mkdirSync(leaseDirectory, { recursive: true })
  const leaseFile = resolve(
    leaseDirectory,
    `${process.pid}-${Date.now()}-${++sourceVersionCounter}.lease`,
  )
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

function isPageAssemblyActive(transaction: BuildTransaction, transactionFile: string): boolean {
  if (listActivePageLeases(`${transactionFile}.leases`).length) return true
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
      if (Date.now() >= deadline) throw new Error(`等待 Taro 页面装配锁超时：${lockFile}`)
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

function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0)
    return true
  } catch (error) {
    return (error as NodeJS.ErrnoException).code === 'EPERM'
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
      } else if (entry.endsWith('.tsx')) {
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

function appendPage(manifest: AppManifest, route: string): void {
  const [first, ...rest] = route.split('/')
  if (!first.startsWith('pages') || first === 'pages') {
    if (!manifest.pages.includes(route)) manifest.pages.push(route)
    return
  }
  manifest.subpackages ??= []
  let subpackage = manifest.subpackages.find((item) => item.root === first)
  if (!subpackage) {
    subpackage = { root: first, pages: [] }
    manifest.subpackages.push(subpackage)
  }
  const subRoute = rest.join('/')
  if (!subpackage.pages.includes(subRoute)) subpackage.pages.push(subRoute)
}

/** 将动态页面配置安全编码为 TypeScript 单引号字符串。 */
function typescriptStringLiteral(value: string): string {
  return `'${value
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/\r/g, '\\r')
    .replace(/\n/g, '\\n')
    .replace(/\t/g, '\\t')}'`
}

/** 读取页面样式中的字符串配置，非法类型回退默认值。 */
function pageStyleString(style: Record<string, unknown> | undefined, key: string, fallback: string): string {
  const value = style?.[key]
  return typeof value === 'string' ? value : fallback
}

function createPageWrapper(page: PageEntry): string {
  return `import KratosPage from ${typescriptStringLiteral(`${page.source}.tsx`)}
import { KratosPageFrame } from '@liujitcn/kratos-taro-app-core/components/KratosPageFrame'
import { KratosTabBar } from '@liujitcn/kratos-taro-app-core/components/KratosTabBar'

/** 自动生成的模块页面包装器。 */
export default function KratosPageWrapper() {
  return (
    <>
      <KratosPageFrame
        navigationStyle=${typescriptStringLiteral(pageStyleString(page.style, 'navigationStyle', 'default'))}
        navigationBarTitleText=${typescriptStringLiteral(pageStyleString(page.style, 'navigationBarTitleText', ''))}
        navigationBarBackgroundColor=${typescriptStringLiteral(pageStyleString(page.style, 'navigationBarBackgroundColor', '#f8f8f8'))}
        navigationBarTextStyle=${typescriptStringLiteral(pageStyleString(page.style, 'navigationBarTextStyle', 'black'))}
      >
        <KratosPage />
      </KratosPageFrame>
      <KratosTabBar route=${typescriptStringLiteral(page.route)} />
    </>
  )
}
`
}

function createPageConfig(page: PageEntry): string {
  return `export default definePageConfig(${JSON.stringify(page.style ?? {}, null, 2)})\n`
}

function isGeneratedPageFile(file: string): boolean {
  return existsSync(file) && readFileSync(file, 'utf8').includes('自动生成的模块页面包装器')
}

function isGeneratedPageConfig(file: string): boolean {
  return existsSync(file) && readFileSync(file, 'utf8').includes('definePageConfig(')
}

function normalizeRoute(route: string): string {
  return route
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\.tsx$/, '')
}

function removeGeneratedStaticFiles(files: string[], staticRoot: string): void {
  for (const file of [...files].reverse()) {
    if (existsSync(file)) unlinkSync(file)
    removeEmptyParents(dirname(file), staticRoot)
  }
}

function removeEmptyParents(directory: string, boundary: string): void {
  let current = directory
  while (current !== boundary && current.startsWith(boundary) && existsSync(current)) {
    if (readdirSync(current).length) return
    rmdirSync(current)
    current = dirname(current)
  }
}

void main().catch((error: unknown) => {
  console.error(error)
  process.exitCode = 1
})
