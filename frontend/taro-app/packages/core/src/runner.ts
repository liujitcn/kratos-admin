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
  ownerPid: number
}

type CliOptions = {
  type: 'h5' | 'weapp'
  mode: string
  watch: boolean
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

  const generatedFiles = [...pageMap.values()].flatMap((page) => {
    const pageFile = resolve(inputDir, `${page.route}.tsx`)
    const configFile = resolve(inputDir, `${page.route}.config.ts`)
    return [pageFile, configFile].filter((file) => !existsSync(file))
  })
  const staticRoot = resolve(inputDir, 'static')
  const staticFiles = collectModuleStaticFiles(
    modules.map((module) => resolve(module.root, 'src/static')),
    staticRoot,
  )
  const generatedStaticFiles = staticFiles.map(({ target }) => target)
  const transaction: BuildTransaction = {
    inputDir,
    appConfigFile,
    originalAppConfig,
    generatedFiles,
    generatedStaticFiles,
    ownerPid: process.pid,
  }
  writeFileSync(transactionFile, `${JSON.stringify(transaction, null, 2)}\n`, { flag: 'wx' })

  let cleaned = false
  const cleanup = () => {
    if (cleaned) return
    cleaned = true
    cleanupTransaction(transaction)
    if (existsSync(transactionFile)) unlinkSync(transactionFile)
  }
  const handleSignal = (signal: NodeJS.Signals) => {
    child?.kill(signal)
  }
  let child: ReturnType<typeof spawn> | undefined
  process.once('SIGINT', () => handleSignal('SIGINT'))
  process.once('SIGTERM', () => handleSignal('SIGTERM'))

  try {
    pageMap.forEach((page) => {
      appendPage(manifest, page.route)
      const pageFile = resolve(inputDir, `${page.route}.tsx`)
      const configFile = resolve(inputDir, `${page.route}.config.ts`)
      if (!existsSync(pageFile)) {
        mkdirSync(dirname(pageFile), { recursive: true })
        writeFileSync(pageFile, createPageWrapper(page))
      }
      if (!existsSync(configFile)) {
        writeFileSync(configFile, createPageConfig(page))
      }
    })
    staticFiles.forEach(({ source, target }) => {
      mkdirSync(dirname(target), { recursive: true })
      copyFileSync(source, target)
    })
    writeFileSync(
      appConfigFile,
      `export default defineAppConfig(${JSON.stringify(manifest, null, 2)})\n`,
    )

    const args = ['exec', 'taro', 'build', '--type', options.type, '--mode', options.mode]
    if (options.watch) args.push('--watch')
    child = spawn(process.platform === 'win32' ? 'pnpm.cmd' : 'pnpm', args, {
      cwd: hostRoot,
      env: {
        ...process.env,
        NODE_ENV: options.mode === 'development' ? 'development' : 'production',
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
    } else {
      throw new Error(`未知参数：${argument}`)
    }
  }
  if (!type) throw new Error('缺少 --type h5|weapp')
  return { type, mode, watch }
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
  if (!existsSync(transactionFile)) return
  const transaction = JSON.parse(readFileSync(transactionFile, 'utf8')) as BuildTransaction
  if (transaction.inputDir !== inputDir || transaction.appConfigFile !== appConfigFile) {
    throw new Error(`Taro 构建事务目录不匹配：${transactionFile}`)
  }
  if (transaction.ownerPid !== process.pid && isProcessRunning(transaction.ownerPid)) {
    throw new Error(
      `页面装配已由进程 ${transaction.ownerPid} 使用，请先停止另一个 H5 或小程序开发进程`,
    )
  }
  cleanupTransaction(transaction)
  unlinkSync(transactionFile)
}

function cleanupTransaction(transaction: BuildTransaction): void {
  for (const file of [...transaction.generatedFiles].reverse()) {
    if (existsSync(file)) unlinkSync(file)
    removeEmptyParents(dirname(file), transaction.inputDir)
  }
  removeGeneratedStaticFiles(transaction.generatedStaticFiles, resolve(transaction.inputDir, 'static'))
  writeFileSync(transaction.appConfigFile, transaction.originalAppConfig)
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

function createPageWrapper(page: PageEntry): string {
  return `import KratosPage from ${JSON.stringify(page.source)}
import { KratosPageFrame } from '@liujitcn/kratos-taro-app-core/components/KratosPageFrame'
import { KratosTabBar } from '@liujitcn/kratos-taro-app-core/components/KratosTabBar'

/** 自动生成的模块页面包装器。 */
export default function KratosPageWrapper() {
  return (
    <>
      <KratosPageFrame
        navigationStyle=${JSON.stringify(page.style?.navigationStyle ?? 'default')}
        navigationBarTitleText=${JSON.stringify(page.style?.navigationBarTitleText ?? '')}
        navigationBarBackgroundColor=${JSON.stringify(page.style?.navigationBarBackgroundColor ?? '#f8f8f8')}
        navigationBarTextStyle=${JSON.stringify(page.style?.navigationBarTextStyle ?? 'black')}
      >
        <KratosPage />
      </KratosPageFrame>
      <KratosTabBar route=${JSON.stringify(page.route)} />
    </>
  )
}
`
}

function createPageConfig(page: PageEntry): string {
  return `export default definePageConfig(${JSON.stringify(page.style ?? {}, null, 2)})\n`
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
