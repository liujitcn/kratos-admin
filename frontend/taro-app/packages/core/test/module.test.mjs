import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

test('模块注册使用后注册优先级并隔离列表变更', async () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-module-'))
  const output = resolve(root, 'module.mjs')
  try {
    await build({
      entryPoints: [resolve(import.meta.dirname, '../src/module.ts')],
      bundle: true,
      platform: 'node',
      format: 'esm',
      outfile: output,
    })
    const runtime = await import(`${pathToFileURL(output).href}?test=${Date.now()}`)
    const first = runtime.defineKratosTaroModule({
      name: 'first',
      pages: {},
      views: { HOME: 'pages/first/index' },
      icons: { HOME: 'static/first.png' },
    })
    const second = runtime.defineKratosTaroModule({
      name: 'second',
      pages: {},
      views: { HOME: 'pages/second/index' },
      icons: { HOME: 'static/second.png' },
    })
    runtime.registerKratosTaroModules([first, second])

    assert.equal(runtime.resolveStaticView('HOME'), 'pages/second/index')
    assert.equal(runtime.resolveModuleIcon('https://cdn.example/icon.png'), 'https://cdn.example/icon.png')
    process.env.VITE_APP_BASE_PATH = '/app/'
    assert.equal(runtime.resolveModuleIcon('HOME'), '/app/static/second.png')
    assert.equal(runtime.resolveBundledAsset('static/images/logo.png'), '/app/static/images/logo.png')
    assert.equal(runtime.resolveBundledAsset('https://cdn.example.com/logo.png'), 'https://cdn.example.com/logo.png')
    delete process.env.VITE_APP_BASE_PATH

    const registered = runtime.getRegisteredKratosTaroModules()
    registered.length = 0
    assert.equal(runtime.getRegisteredKratosTaroModules().length, 2)
  } finally {
    delete process.env.VITE_APP_BASE_PATH
    rmSync(root, { recursive: true, force: true })
  }
})
