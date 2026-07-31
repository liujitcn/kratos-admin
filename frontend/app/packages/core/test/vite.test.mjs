import assert from 'node:assert/strict'
import { EventEmitter } from 'node:events'
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
  const fixture = await createViteFixture()
  const {
    generatedStaticFile,
    hostRoot,
    hostStaticFile,
    original,
    pagesFile,
    plugin,
    transactionFile,
    workspaceRoot,
  } = fixture

  try {
    const canonicalCore = plugin.resolveId(
      '@liujitcn/kratos-app-core',
      resolve(workspaceRoot, 'packages/modules/system/src/index.ts'),
    )
    assert.match(
      canonicalCore,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/index\.ts\?kratos=[^?&]+$/,
    )
    const coreStores = plugin.resolveId('./stores', canonicalCore)
    assert.match(
      coreStores,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/stores\/index\.ts\?kratos=[^?&]+$/,
    )
    const versionedCore = plugin.resolveId(
      '@liujitcn/kratos-app-core/utils/http',
      `${resolve(
        hostRoot,
        'node_modules/@liujitcn/kratos-app-system/src/api/base/ai_session.ts',
      )}?v=dependency-version`,
    )
    assert.match(
      versionedCore,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/utils\/http\.ts\?v=dependency-version&kratos=[^?&]+$/,
    )
    const tabBarComponent = plugin.resolveId(
      '@liujitcn/kratos-app-core/components/KratosTabBar.vue',
      resolve(hostRoot, 'src/pages/index/index.vue'),
    )
    assert.match(
      tabBarComponent,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/components\/KratosTabBar\.vue\?kratos=[^?&]+$/,
    )
    const tabBarNavigation = plugin.resolveId('../navigation', tabBarComponent)
    assert.match(
      tabBarNavigation,
      /node_modules\/@liujitcn\/kratos-app-core\/src\/navigation\.ts\?kratos=[^?&]+$/,
    )
    assert.equal(
      new URL(`file://${tabBarNavigation}`).searchParams.get('kratos'),
      new URL(`file://${tabBarComponent}`).searchParams.get('kratos'),
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
    assert.match(
      readFileSync(resolve(hostRoot, 'src/pages/index/index.vue'), 'utf8'),
      /import KratosTabBar from '@liujitcn\/kratos-app-core\/components\/KratosTabBar\.vue'/,
    )
    assert.ok(existsSync(transactionFile))
    assert.ok(existsSync(hostStaticFile))
    assert.ok(existsSync(generatedStaticFile))
    plugin.closeBundle()
    assert.equal(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(existsSync(hostStaticFile))
    assert.ok(!existsSync(generatedStaticFile))
    assert.ok(!existsSync(transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('watch 构建在关闭监听器前保留生成页面', async () => {
  const fixture = await createViteFixture()
  const { generatedStaticFile, original, pagesFile, plugin, transactionFile } = fixture

  try {
    plugin.configResolved?.({ build: { watch: {} } })
    plugin.closeBundle()
    assert.notEqual(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(existsSync(generatedStaticFile))
    assert.ok(existsSync(transactionFile))

    plugin.closeWatcher()
    assert.equal(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(!existsSync(generatedStaticFile))
    assert.ok(!existsSync(transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('H5 serve 在 HTTP 服务关闭前保留生成页面', async () => {
  const fixture = await createViteFixture()
  const { generatedStaticFile, original, pagesFile, plugin, transactionFile } = fixture
  const httpServer = new EventEmitter()

  try {
    plugin.configResolved?.({ command: 'serve', build: { watch: null } })
    plugin.configureServer({ httpServer })
    plugin.closeBundle()
    assert.notEqual(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(existsSync(generatedStaticFile))
    assert.ok(existsSync(transactionFile))

    httpServer.emit('close')
    assert.equal(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(!existsSync(generatedStaticFile))
    assert.ok(!existsSync(transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('活动页面事务阻止第二个开发进程覆盖生成页面', async () => {
  const fixture = await createViteFixture({ configure: false })
  const { configure, inputDir, original, pagesFile, transactionFile } = fixture
  writeFileSync(
    transactionFile,
    `${JSON.stringify({
      inputDir,
      pagesFile,
      originalPages: original,
      generatedFiles: [],
      generatedStaticFiles: [],
      ownerPid: process.ppid,
    })}\n`,
  )

  try {
    assert.throws(configure, new RegExp(`页面装配已由进程 ${process.ppid} 使用`))
    assert.equal(readFileSync(pagesFile, 'utf8'), original)
    assert.ok(existsSync(transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('不同开发服务使用独立模块版本避免复用旧源码缓存', async () => {
  const firstFixture = await createViteFixture()
  const secondFixture = await createViteFixture()

  try {
    const importer = `${resolve(
      firstFixture.hostRoot,
      'node_modules/@liujitcn/kratos-app-system/src/api/base/ai_session.ts',
    )}?v=dependency-version`
    const firstResolved = new URL(
      `file://${firstFixture.plugin.resolveId('@liujitcn/kratos-app-core/navigation', importer)}`,
    )
    const secondResolved = new URL(
      `file://${secondFixture.plugin.resolveId('@liujitcn/kratos-app-core/navigation', importer)}`,
    )
    assert.equal(firstResolved.searchParams.get('v'), 'dependency-version')
    assert.equal(secondResolved.searchParams.get('v'), 'dependency-version')
    assert.ok(firstResolved.searchParams.get('kratos'))
    assert.notEqual(
      firstResolved.searchParams.get('kratos'),
      secondResolved.searchParams.get('kratos'),
    )
  } finally {
    firstFixture.dispose()
    secondFixture.dispose()
  }
})

async function createViteFixture(options = {}) {
  const workspaceRoot = resolve(import.meta.dirname, '../../..')
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-app-vite-'))
  const hostRoot = resolve(root, 'app')
  const inputDir = resolve(hostRoot, 'src')
  const pagesFile = resolve(inputDir, 'pages.json')
  const transactionFile = resolve(hostRoot, '.kratos-app-pages-state.json')
  const hostStaticFile = resolve(inputDir, 'static/host.txt')
  const generatedStaticFile = resolve(inputDir, 'static/tabs/home_default.png')
  const original = '{"pages":[]}\n'
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

  process.env.UNI_INPUT_DIR = inputDir
  const loaded = await loadConfigFromFile(
    { command: 'build', mode: 'production-h5' },
    resolve(workspaceRoot, 'apps/app/vite.config.ts'),
  )
  const plugin = loaded.config.plugins
    .flat(Infinity)
    .find((item) => item?.name === 'kratos-app-pages')
  assert.ok(plugin)
  delete process.env.UNI_INPUT_DIR
  const configure = () => {
    process.env.UNI_INPUT_DIR = inputDir
    try {
      return plugin.config()
    } finally {
      delete process.env.UNI_INPUT_DIR
    }
  }
  if (options.configure !== false) configure()

  return {
    configure,
    generatedStaticFile,
    hostRoot,
    hostStaticFile,
    inputDir,
    original,
    pagesFile,
    plugin,
    transactionFile,
    workspaceRoot,
    dispose() {
      plugin.closeWatcher?.()
      plugin.closeBundle()
      delete process.env.UNI_INPUT_DIR
      rmSync(root, { recursive: true, force: true })
    },
  }
}
