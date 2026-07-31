import assert from 'node:assert/strict'
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve } from 'node:path'
import test from 'node:test'
import { loadConfigFromFile } from 'vite'

test('扫描模块页面、合并静态资源并在结束时恢复宿主文件', async () => {
  const workspaceRoot = resolve(import.meta.dirname, '../../..')
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-app-vite-'))
  const hostRoot = resolve(root, 'app')
  const inputDir = resolve(hostRoot, 'src')
  const pagesFile = resolve(inputDir, 'pages.json')
  const transactionFile = resolve(hostRoot, '.kratos-app-pages-state.json')
  const hostStaticFile = resolve(inputDir, 'static/host.txt')
  const generatedStaticFile = resolve(inputDir, 'static/tabs/home_default.png')
  const original = '{"pages":[]}\n'
  let plugin
  mkdirSync(dirname(hostStaticFile), { recursive: true })
  mkdirSync(resolve(hostRoot, 'node_modules/@liujitcn'), { recursive: true })
  writeFileSync(resolve(hostRoot, 'package.json'), '{"type":"module"}\n')
  writeFileSync(pagesFile, original)
  writeFileSync(resolve(inputDir, 'manifest.json'), '{}\n')
  writeFileSync(hostStaticFile, 'host')
  symlinkSync(
    resolve(workspaceRoot, 'packages/core'),
    resolve(hostRoot, 'node_modules/@liujitcn/kratos-app-core'),
    'dir',
  )
  symlinkSync(
    resolve(workspaceRoot, 'packages/modules/system'),
    resolve(hostRoot, 'node_modules/@liujitcn/kratos-app-system'),
    'dir',
  )

  try {
    process.env.UNI_INPUT_DIR = inputDir
    const loaded = await loadConfigFromFile(
      { command: 'build', mode: 'production-h5' },
      resolve(workspaceRoot, 'apps/app/vite.config.ts'),
    )
    plugin = loaded.config.plugins.flat(Infinity).find((item) => item?.name === 'kratos-app-pages')
    assert.ok(plugin)
    plugin.config()
    const canonicalCore = plugin.resolveId(
      '@liujitcn/kratos-app-core',
      resolve(workspaceRoot, 'packages/modules/system/src/index.ts'),
    )
    assert.match(canonicalCore, /node_modules\/@liujitcn\/kratos-app-core\/src\/index\.ts$/)
    const versionedCore = plugin.resolveId(
      '@liujitcn/kratos-app-core/utils/http',
      `${resolve(
        hostRoot,
        'node_modules/@liujitcn/kratos-app-system/src/api/base/ai_session.ts',
      )}?v=dependency-version`,
    )
    assert.match(
      versionedCore,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/utils\/http\.ts\?v=dependency-version$/,
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
    assert.ok(existsSync(hostStaticFile))
    assert.ok(existsSync(generatedStaticFile))
    plugin.closeBundle()
    assert.equal(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(existsSync(hostStaticFile))
    assert.ok(!existsSync(generatedStaticFile))
    assert.ok(!existsSync(transactionFile))
  } finally {
    plugin?.closeBundle()
    delete process.env.UNI_INPUT_DIR
    rmSync(root, { recursive: true, force: true })
  }
})
