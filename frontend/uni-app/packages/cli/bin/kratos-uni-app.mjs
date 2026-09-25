#!/usr/bin/env node
import { scaffoldKratosApp } from '../src/index.mjs'
import { cliMessage } from '../src/messages.mjs'

const args = process.argv.slice(2)
if (args[0] !== 'create' || !args[1]) {
  console.error(cliMessage('usage'))
  process.exitCode = 1
} else {
  const modules = []
  const packages = []
  let kratosProject = false
  for (let index = 2; index < args.length; index += 1) {
    if (args[index] === '--module' && args[index + 1]) modules.push(args[++index])
    else if (args[index] === '--with' && args[index + 1]) packages.push(args[++index])
    else if (args[index] === '--kratos-project') kratosProject = true
    else throw new Error(cliMessage('unknown_argument', { argument: args[index] }))
  }
  scaffoldKratosApp(args[1], { modules, packages, kratosProject })
}
