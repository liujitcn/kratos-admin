#!/usr/bin/env node

import { createRequire } from 'node:module'
import { existsSync, mkdirSync, readFileSync, symlinkSync } from 'node:fs'
import { delimiter, dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const sharedNodeModules = resolve(packageRoot, 'node_modules')
process.env.NODE_PATH = [sharedNodeModules, process.env.NODE_PATH].filter(Boolean).join(delimiter)
process.env.UNI_CLI_CONTEXT = packageRoot
process.env.UNI_INPUT_DIR ??= resolve(process.cwd(), 'src')
require('node:module').Module._initPaths()

const sharedPackage = JSON.parse(readFileSync(resolve(packageRoot, 'package.json'), 'utf8'))
const consumerNodeModules = resolve(process.cwd(), 'node_modules')
mkdirSync(consumerNodeModules, { recursive: true })
for (const packageName of Object.keys(sharedPackage.dependencies ?? {})) {
  const packageParts = packageName.split('/')
  const target = resolve(consumerNodeModules, ...packageParts)
  if (!existsSync(target)) {
    mkdirSync(resolve(consumerNodeModules, ...packageParts.slice(0, -1)), { recursive: true })
    symlinkSync(resolve(sharedNodeModules, ...packageParts), target)
  }
}

const cli = require.resolve('@dcloudio/vite-plugin-uni/bin/uni.js', {
  paths: [packageRoot],
})

require(cli)
