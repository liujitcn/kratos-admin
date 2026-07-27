#!/usr/bin/env node

import { createRequire } from 'node:module'
import { existsSync, lstatSync, mkdirSync, readFileSync, symlinkSync, unlinkSync } from 'node:fs'
import { delimiter, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const sharedNodeModules = resolve(packageRoot, 'node_modules')
const consumerNodeModules = resolve(process.cwd(), 'node_modules')
process.env.NODE_PATH = [sharedNodeModules, consumerNodeModules, process.env.NODE_PATH]
  .filter(Boolean)
  .join(delimiter)
process.env.UNI_CLI_CONTEXT = packageRoot
process.env.UNI_INPUT_DIR ??= resolve(process.cwd(), 'src')
require('node:module').Module._initPaths()

const sharedPackage = JSON.parse(readFileSync(resolve(packageRoot, 'package.json'), 'utf8'))
mkdirSync(consumerNodeModules, { recursive: true })

/**
 * 解析依赖的真实包目录，兼容 npm 嵌套依赖和 pnpm 虚拟存储布局。
 */
const resolveDependencyRoot = (packageName) => {
  let resolvedPath
  try {
    resolvedPath = require.resolve(`${packageName}/package.json`, { paths: [packageRoot] })
  } catch {
    resolvedPath = require.resolve(packageName, { paths: [packageRoot] })
  }

  let currentPath = dirname(resolvedPath)
  while (currentPath !== dirname(currentPath)) {
    const packageJSONPath = resolve(currentPath, 'package.json')
    if (existsSync(packageJSONPath)) {
      try {
        const packageMetadata = JSON.parse(readFileSync(packageJSONPath, 'utf8'))
        if (packageMetadata.name === packageName) {
          return currentPath
        }
      } catch {
        // 继续向上查找包根目录。
      }
    }
    currentPath = dirname(currentPath)
  }

  throw new Error(`无法解析依赖包目录：${packageName}`)
}

for (const packageName of Object.keys(sharedPackage.dependencies ?? {})) {
  const packageParts = packageName.split('/')
  const target = resolve(consumerNodeModules, ...packageParts)
  if (!existsSync(target)) {
    try {
      if (lstatSync(target).isSymbolicLink()) {
        unlinkSync(target)
      }
    } catch (error) {
      if (error.code !== 'ENOENT') {
        throw error
      }
    }
    mkdirSync(resolve(consumerNodeModules, ...packageParts.slice(0, -1)), { recursive: true })
    symlinkSync(resolveDependencyRoot(packageName), target, 'junction')
  }
}

const cli = require.resolve('@dcloudio/vite-plugin-uni/bin/uni.js', {
  paths: [packageRoot],
})

require(cli)
