import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'
import { loadConfigFromFile } from 'vite'

test('扫描模块页面并在结束时恢复宿主页表', async () => {
  const workspaceRoot = resolve(import.meta.dirname, '../../..')
  const inputDir = resolve(workspaceRoot, 'apps/app/src')
  const pagesFile = resolve(inputDir, 'pages.json')
  const transactionFile = resolve(inputDir, '../.kratos-app-pages-state.json')
  const original = existsSync(transactionFile)
    ? JSON.parse(readFileSync(transactionFile, 'utf8')).originalPages
    : readFileSync(pagesFile, 'utf8')
  process.env.UNI_INPUT_DIR = inputDir
  const loaded = await loadConfigFromFile(
    { command: 'build', mode: 'production-h5' },
    resolve(workspaceRoot, 'apps/app/vite.config.ts'),
  )
  const plugin = loaded.config.plugins
    .flat(Infinity)
    .find((item) => item?.name === 'kratos-app-pages')
  assert.ok(plugin)
  plugin.config()
  const canonicalCore = plugin.resolveId(
    '@liujitcn/kratos-app-core',
    resolve(workspaceRoot, 'packages/modules/system/src/index.ts'),
  )
  assert.match(
    canonicalCore,
    /apps\/app\/node_modules\/@liujitcn\/kratos-app-core\/src\/index\.ts$/,
  )
  const generated = JSON.parse(readFileSync(pagesFile, 'utf8'))
  const routes = [
    ...generated.pages.map((page) => page.path),
    ...generated.subPackages.flatMap((group) =>
      group.pages.map((page) => `${group.root}/${page.path}`),
    ),
  ]
  assert.ok(routes.includes('pages/index/index'))
  assert.ok(routes.includes('pages/my/my'))
  assert.ok(routes.includes('pagesMember/ai/index'))
  assert.ok(!routes.some((route) => route.includes('/components/')))
  assert.equal(generated.tabBar, undefined)
  assert.ok(existsSync(transactionFile))
  plugin.closeBundle()
  assert.equal(readFileSync(pagesFile, 'utf8'), original)
  assert.ok(!existsSync(transactionFile))
  delete process.env.UNI_INPUT_DIR
})
