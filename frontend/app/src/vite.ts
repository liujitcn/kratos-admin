import { fileURLToPath } from 'node:url'
import { dirname, isAbsolute, relative, resolve } from 'node:path'
import { existsSync, mkdirSync, readFileSync, rmSync, unlinkSync, writeFileSync } from 'node:fs'
import type { Plugin } from 'vite'
import type { KratosAppModule } from './modules'

type JsonObject = Record<string, unknown>

type PageEntry = JsonObject & {
  path: string
  style?: JsonObject
}

type SubPackage = JsonObject & {
  root: string
  pages: PageEntry[]
}

type PagesManifest = JsonObject & {
  pages: PageEntry[]
  subPackages?: SubPackage[]
  subpackages?: SubPackage[]
}

type PageRecord = {
  route: string
  page: PageEntry
  section: 'pages' | 'subPackages'
  root?: string
}

/**
 * app 页面注册插件参数。
 */
export interface KratosAppViteOptions {
  /** 公共 app 源码包根目录。 */
  packageRoot?: string
  /** 已注册的业务模块，保留给宿主构建配置声明模块边界。 */
  modules?: KratosAppModule[]
}

/**
 * 创建 app 页面合并插件。
 *
 * uni-app 要求 pages.json 中的页面在构建时存在于 UNI_INPUT_DIR。
 * 插件只在构建进程中补齐默认页面入口，进程结束后恢复宿主文件。
 */
export function kratosApp(options: KratosAppViteOptions = {}): Plugin {
  const packageRoot = options.packageRoot ?? resolve(dirname(fileURLToPath(import.meta.url)), '..')
  const packageSourceRoot = resolve(packageRoot, 'src')
  const state: {
    pagesFile?: string
    originalPagesJson?: string
    generatedFiles: string[]
    cleaned: boolean
  } = {
    generatedFiles: [],
    cleaned: false,
  }

  const cleanup = () => {
    if (state.cleaned) {
      return
    }
    state.cleaned = true

    state.generatedFiles.forEach((file) => {
      if (existsSync(file)) {
        unlinkSync(file)
      }
    })

    if (state.pagesFile && state.originalPagesJson !== undefined) {
      writeFileSync(state.pagesFile, state.originalPagesJson)
    }
  }

  return {
    name: 'kratos-app-pages',
    enforce: 'pre',
    resolveId(source, importer) {
      if (source === '@kratos-app') {
        return resolve(packageSourceRoot, 'index.ts')
      }
      if (source.startsWith('@kratos-app/')) {
        const [sourcePath, query = ''] = source.split('?', 2)
        const target = resolve(packageSourceRoot, sourcePath.slice('@kratos-app/'.length))
        if (!isWithinRoot(packageSourceRoot, target)) {
          return
        }
        const resolvedTarget = resolvePackageSourceFile(target)
        return `${resolvedTarget}${query ? `?${query}` : ''}`
      }
      if (!importer || !source.startsWith('@/')) {
        return
      }

      const importerPath = stripViteQuery(importer)
      const importerRelativePath = relative(packageSourceRoot, importerPath)
      if (importerRelativePath.startsWith('..') || isAbsolute(importerRelativePath)) {
        return
      }

      const [sourcePath, query = ''] = source.split('?', 2)
      const target = resolve(packageSourceRoot, sourcePath.slice(2))
      if (!isWithinRoot(packageSourceRoot, target)) {
        return
      }

      return `${target}${query ? `?${query}` : ''}`
    },
    transform(code, id) {
      const sourceFile = stripViteQuery(id)
      if (
        !isWithinRoot(packageSourceRoot, sourceFile) ||
        !/\.(?:vue|[cm]?[jt]sx?)$/.test(sourceFile)
      ) {
        return
      }

      const transformedCode = code.replace(/@\/(?=[\w-])/g, '@kratos-app/')
      if (transformedCode === code) {
        return
      }
      return {
        code: transformedCode,
        map: null,
      }
    },
    config() {
      const inputDir = process.env.UNI_INPUT_DIR || resolve(process.cwd(), 'src')
      const pagesFile = resolve(inputDir, 'pages.json')
      const packagePagesFile = resolve(packageRoot, 'src/pages.json')

      if (!existsSync(pagesFile) || !existsSync(packagePagesFile)) {
        return
      }

      const originalPagesJson = readFileSync(pagesFile, 'utf8')
      const hostManifest = parsePagesManifest(originalPagesJson, pagesFile)
      const packageManifest = parsePagesManifest(
        readFileSync(packagePagesFile, 'utf8'),
        packagePagesFile,
      )
      const hostPages = collectPageRecords(hostManifest)
      const packagePages = collectPageRecords(packageManifest)
      const hostPageMap = new Map(hostPages.map((record) => [record.route, record]))
      const generatedFiles: string[] = []
      let manifestChanged = false

      packagePages.forEach((defaultRecord) => {
        const hostRecord = hostPageMap.get(defaultRecord.route)
        if (hostRecord) {
          const mergedStyle = {
            ...(defaultRecord.page.style ?? {}),
            ...(hostRecord.page.style ?? {}),
          }
          if (JSON.stringify(mergedStyle) !== JSON.stringify(hostRecord.page.style ?? {})) {
            hostRecord.page.style = mergedStyle
            manifestChanged = true
          }
          return
        }

        const pageFile = resolve(inputDir, `${defaultRecord.route}.vue`)
        mkdirSync(dirname(pageFile), { recursive: true })
        writeFileSync(pageFile, createPageWrapper(packageRoot, defaultRecord.route))
        generatedFiles.push(pageFile)
        manifestChanged = true

        if (defaultRecord.section === 'pages') {
          hostManifest.pages.push({ ...defaultRecord.page })
          return
        }

        const subPackages = getSubPackages(hostManifest)
        let subPackage = subPackages.find((item) => item.root === defaultRecord.root)
        if (!subPackage) {
          subPackage = {
            root: defaultRecord.root ?? '',
            pages: [],
          }
          subPackages.push(subPackage)
        }
        subPackage.pages.push({ ...defaultRecord.page })
      })

      const nextPagesJson = `${JSON.stringify(hostManifest, null, 2)}\n`
      if (manifestChanged) {
        state.pagesFile = pagesFile
        state.originalPagesJson = originalPagesJson
        state.generatedFiles = generatedFiles
        writeFileSync(pagesFile, nextPagesJson)
      } else {
        generatedFiles.forEach((file) => rmSync(file, { force: true }))
      }

      if (manifestChanged) {
        process.once('exit', cleanup)
        process.once('SIGINT', () => {
          cleanup()
          process.exit(130)
        })
        process.once('SIGTERM', () => {
          cleanup()
          process.exit(143)
        })
      }
    },
    configureServer(server) {
      server.httpServer?.once('close', cleanup)
    },
    closeBundle: cleanup,
  }
}

/**
 * 解析 pages.json，统一把格式错误转换成带文件位置的构建错误。
 */
function parsePagesManifest(content: string, file: string): PagesManifest {
  try {
    const manifest = JSON.parse(content) as PagesManifest
    manifest.pages ??= []
    return manifest
  } catch (error) {
    throw new Error(`无法解析 ${file}: ${String(error)}`)
  }
}

/**
 * 收集主包与分包页面，使用完整路由作为覆盖键。
 */
function collectPageRecords(manifest: PagesManifest): PageRecord[] {
  const records: PageRecord[] = manifest.pages.map((page) => ({
    route: normalizeRoute(page.path),
    page,
    section: 'pages' as const,
  }))

  getSubPackages(manifest).forEach((subPackage) => {
    subPackage.pages.forEach((page) => {
      records.push({
        route: normalizeRoute(`${subPackage.root}/${page.path}`),
        page,
        section: 'subPackages',
        root: subPackage.root,
      })
    })
  })

  return records
}

/**
 * 获取宿主分包数组，并保留 uni-app 两种字段命名的兼容性。
 */
function getSubPackages(manifest: PagesManifest): SubPackage[] {
  if (manifest.subPackages) {
    return manifest.subPackages
  }
  if (manifest.subpackages) {
    return manifest.subpackages
  }
  manifest.subPackages = []
  return manifest.subPackages
}

/**
 * 规范化页面路径，统一处理前导斜杠和文件后缀。
 */
function normalizeRoute(route: string): string {
  return route
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\.vue$/, '')
}

/**
 * 生成宿主页面的薄入口，不复制公共页面实现。
 */
function createPageWrapper(packageRoot: string, route: string): string {
  const defaultPage = normalizePath(resolve(packageRoot, 'src', `${normalizeRoute(route)}.vue`))
  return `<script setup lang="ts">
import DefaultPage from ${JSON.stringify(defaultPage)}
</script>

<template>
  <DefaultPage />
</template>
`
}

/**
 * 统一绝对路径分隔符，生成的入口可被各平台 Vite 解析。
 */
function normalizePath(path: string): string {
  return path.replace(/\\/g, '/')
}

/**
 * 判断解析后的路径仍在公共 app 包目录内，避免别名穿越包边界。
 */
function isWithinRoot(root: string, target: string): boolean {
  const relativePath = relative(root, target)
  return relativePath === '' || (!relativePath.startsWith('..') && !isAbsolute(relativePath))
}

/**
 * 移除 Vite 虚拟模块查询参数，用于判断真实导入文件位置。
 */
function stripViteQuery(path: string): string {
  return path.split('?', 1)[0]
}

/**
 * 补齐包内源码导入的常见扩展名与目录入口。
 */
function resolvePackageSourceFile(target: string): string {
  const candidates = [
    target,
    `${target}.ts`,
    `${target}.js`,
    `${target}.vue`,
    resolve(target, 'index.ts'),
    resolve(target, 'index.js'),
  ]
  return candidates.find((candidate) => existsSync(candidate)) ?? target
}
