import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { dirname, relative, resolve } from 'node:path'

const appRoot = resolve(import.meta.dirname, '..')
const workspaceRoot = resolve(appRoot, '../..')
const modules = [
  resolvePackage('@liujitcn/kratos-uni-app-core'),
  resolvePackage('@liujitcn/kratos-uni-app-system'),
  ...scanLocalModules(),
].filter(Boolean)

const pageStyles = {
  'pages/webview/webview': { navigationBarTitleText: '' },
}

const pages = [
  { route: 'pages/bootstrap/index', packageName: null },
  ...modules.flatMap((module) =>
    scanViews(module.sourceRoot).map((route) => ({ route, packageName: module.packageName })),
  ),
]

const uniquePages = new Map(pages.map((page) => [page.route, page]))
for (const page of uniquePages.values()) {
  if (page.packageName === null) continue
  const target = resolve(appRoot, `${page.route}.uvue`)
  mkdirSync(dirname(target), { recursive: true })
  writeFileSync(target, createPageWrapper(page.route, page.packageName), 'utf8')
}

const pagesManifest = {
  easycom: {
    autoscan: true,
    custom: {
      '^uni-(.*)': '@dcloudio/uni-ui/lib/uni-$1/uni-$1.uvue',
    },
  },
  pages: [...uniquePages.values()]
    .filter(({ route }) => !route.startsWith('pagesMember/'))
    .map(({ route }) => ({ path: route, style: pageStyle(route) })),
  globalStyle: {
    navigationBarTextStyle: 'black',
    navigationBarTitleText: '',
    navigationBarBackgroundColor: '#F8F8F8',
    backgroundColor: '#F8F8F8',
  },
  subPackages: [
    {
      root: 'pagesMember',
      pages: [...uniquePages.values()]
        .filter(({ route }) => route.startsWith('pagesMember/'))
        .map(({ route }) => ({
          path: route.slice('pagesMember/'.length),
          style: pageStyle(route),
        })),
    },
  ],
}
writeFileSync(resolve(appRoot, 'pages.json'), `${JSON.stringify(pagesManifest, null, 2)}\n`, 'utf8')

const staticRoot = resolve(appRoot, 'static')
rmSync(staticRoot, { recursive: true, force: true })
for (const module of modules) copyStatic(module.staticRoot, staticRoot)

function createPageWrapper(route, packageName) {
  const source = `${packageName}/views/${route}.uvue`
  return `<script setup lang="ts">
import { onShow } from '@dcloudio/uni-app'
import { setAppPageTitle } from '@liujitcn/kratos-uni-app-core'
import KratosPage from '${source}'
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.uvue'

defineOptions({ inheritAttrs: false })
onShow(() => setAppPageTitle('${route}'))
</script>

<template>
  <KratosPage v-bind="$attrs" />
  <KratosTabBar route="${route}" />
</template>
`
}

function pageStyle(route) {
  return {
    navigationStyle: 'custom',
    navigationBarTitleText: '',
    ...(pageStyles[route] ?? {}),
  }
}

function scanViews(root) {
  if (!existsSync(root)) return []
  const files = []
  const visit = (directory) => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const path = resolve(directory, entry.name)
      if (entry.isDirectory()) {
        if (entry.name !== 'components') visit(path)
        continue
      }
      if (entry.name.endsWith('.uvue')) files.push(path)
    }
  }
  visit(root)
  return files.map((file) =>
    relative(root, file)
      .replaceAll('\\', '/')
      .replace(/\.uvue$/, ''),
  )
}

function copyStatic(source, target) {
  if (!existsSync(source)) return
  for (const entry of readdirSync(source, { withFileTypes: true })) {
    const from = resolve(source, entry.name)
    const to = resolve(target, entry.name)
    if (existsSync(to)) continue
    mkdirSync(dirname(to), { recursive: true })
    cpSync(from, to, { recursive: true })
  }
}

function resolvePackage(packageName) {
  const packageRoot = resolve(appRoot, 'node_modules', ...packageName.split('/'))
  if (!existsSync(resolve(packageRoot, 'package.json'))) return undefined
  return {
    packageName,
    sourceRoot: resolve(packageRoot, 'src/views'),
    staticRoot: resolve(packageRoot, 'src/static'),
  }
}

function scanLocalModules() {
  const modulesRoot = resolve(workspaceRoot, 'packages/modules')
  if (!existsSync(modulesRoot)) return []
  return readdirSync(modulesRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => {
      const packageRoot = resolve(modulesRoot, entry.name)
      const packageFile = resolve(packageRoot, 'package.json')
      if (!existsSync(packageFile)) return undefined
      const packageName = JSON.parse(readFileSync(packageFile, 'utf8')).name
      if (typeof packageName !== 'string') return undefined
      return {
        packageName,
        sourceRoot: resolve(packageRoot, 'src/views'),
        staticRoot: resolve(packageRoot, 'src/static'),
      }
    })
    .filter(Boolean)
}
