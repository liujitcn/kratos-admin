#!/usr/bin/env node
import { run } from '../src/index.mjs'

try {
  await run()
} catch (error) {
  const message = error instanceof Error ? error.message : String(error)
  process.stderr.write(`${message}\n`)
  process.exitCode = 1
}
