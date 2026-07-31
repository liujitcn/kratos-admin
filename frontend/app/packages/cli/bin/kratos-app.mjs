#!/usr/bin/env node
import { scaffoldKratosApp } from '../src/index.mjs'

const args = process.argv.slice(2)
if (args[0] !== 'create' || !args[1]) {
  console.error('用法: kratos-app create <目录> [--module <名称>] [--with <包名>]')
  process.exitCode = 1
} else {
  const modules = []
  const packages = []
  for (let index = 2; index < args.length; index += 1) {
    if (args[index] === '--module' && args[index + 1]) modules.push(args[++index])
    else if (args[index] === '--with' && args[index + 1]) packages.push(args[++index])
    else throw new Error(`未知参数：${args[index]}`)
  }
  scaffoldKratosApp(args[1], { modules, packages })
}
