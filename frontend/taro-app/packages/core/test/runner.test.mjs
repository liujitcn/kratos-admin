import assert from 'node:assert/strict'
import { spawn, spawnSync } from 'node:child_process'
import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve } from 'node:path'
import test from 'node:test'

const workspaceRoot = resolve(import.meta.dirname, '../../..')
const runner = resolve(workspaceRoot, 'packages/core/dist/runner.mjs')

test('runner 扫描页面、应用后注册覆盖并在构建后恢复宿主', () => {
  const fixture = createRunnerFixture()
  try {
    const staleFile = resolve(fixture.inputDir, 'pages/stale/index.tsx')
    write(staleFile, 'stale')
    write(
      fixture.transactionFile,
      `${JSON.stringify({
        inputDir: fixture.inputDir,
        appConfigFile: fixture.appConfigFile,
        originalAppConfig: fixture.originalAppConfig,
        generatedFiles: [staleFile],
        generatedStaticFiles: [],
        ownerPid: 2147483647,
      })}\n`,
    )
    write(fixture.appConfigFile, 'mutated')

    const result = runFixture(fixture)
    assert.equal(result.status, 0, result.stderr)
    const snapshot = JSON.parse(readFileSync(fixture.snapshotFile, 'utf8'))
    const generatedConfig = snapshot.appConfig
    assert.match(generatedConfig, /pages\/index\/index/)
    assert.match(generatedConfig, /"root": "pagesMember"/)
    assert.match(generatedConfig, /"orders\/detail"/)
    assert.match(generatedConfig, /"custom": true/)
    assert.match(snapshot.homeWrapper, /@fixture\/two\/views\/pages\/override\/index\.tsx/)
    assert.match(snapshot.homeWrapper, /KratosPageFrame/)
    assert.match(snapshot.homeWrapper, /navigationBarTitleText='second'/)
    assert.doesNotMatch(snapshot.homeWrapper, /navigationBarTitleText="/)
    assert.match(snapshot.detailWrapper, /@fixture\/one\/views\/pagesMember\/orders\/detail\.tsx/)
    assert.match(snapshot.homeConfig, /"title": "second"/)
    assert.equal(snapshot.sharedStatic, 'second')
    assert.equal(snapshot.hostStatic, 'host')

    assert.equal(readFileSync(fixture.appConfigFile, 'utf8'), fixture.originalAppConfig)
    assert.ok(!existsSync(resolve(fixture.inputDir, 'pages/index/index.tsx')))
    assert.ok(!existsSync(resolve(fixture.inputDir, 'pagesMember/orders/detail.tsx')))
    assert.ok(!existsSync(resolve(fixture.inputDir, 'static/shared.txt')))
    assert.ok(existsSync(resolve(fixture.inputDir, 'static/host.txt')))
    assert.ok(!existsSync(staleFile))
    assert.ok(!existsSync(fixture.transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('runner 复用活动进程持有的页面事务', () => {
  const fixture = createRunnerFixture()
  try {
    const generatedFiles = [
      resolve(fixture.inputDir, 'pages/index/index.tsx'),
      resolve(fixture.inputDir, 'pages/index/index.config.ts'),
      resolve(fixture.inputDir, 'pagesMember/orders/detail.tsx'),
      resolve(fixture.inputDir, 'pagesMember/orders/detail.config.ts'),
    ]
    generatedFiles.forEach((file) => write(file, 'generated'))
    const generatedStaticFile = resolve(fixture.inputDir, 'static/shared.txt')
    write(generatedStaticFile, 'shared')
    write(
      fixture.transactionFile,
      `${JSON.stringify({
        inputDir: fixture.inputDir,
        appConfigFile: fixture.appConfigFile,
        originalAppConfig: fixture.originalAppConfig,
        generatedFiles,
        generatedStaticFiles: [generatedStaticFile],
        ownerPid: process.pid,
      })}\n`,
    )
    write(fixture.appConfigFile, 'mutated')
    const result = runFixture(fixture)
    assert.equal(result.status, 0, result.stderr)
    assert.equal(readFileSync(fixture.appConfigFile, 'utf8'), 'mutated')
    assert.ok(existsSync(fixture.transactionFile))
    assert.ok(existsSync(generatedFiles[0]))
    assert.ok(existsSync(generatedStaticFile))
  } finally {
    fixture.dispose()
  }
})

test('runner 准备类型检查后由清理模式恢复页面事务', () => {
  const fixture = createRunnerFixture()
  try {
    const prepareResult = runFixture(fixture, { prepareOnly: true })
    assert.equal(prepareResult.status, 0, prepareResult.stderr)
    assert.ok(existsSync(fixture.transactionFile))
    assert.ok(existsSync(resolve(fixture.inputDir, 'pages/index/index.tsx')))

    const cleanupResult = runFixture(fixture, { cleanupOnly: true })
    assert.equal(cleanupResult.status, 0, cleanupResult.stderr)
    assert.equal(readFileSync(fixture.appConfigFile, 'utf8'), fixture.originalAppConfig)
    assert.ok(!existsSync(resolve(fixture.inputDir, 'pages/index/index.tsx')))
    assert.ok(!existsSync(fixture.transactionFile))
  } finally {
    fixture.dispose()
  }
})

test('并行 runner 共享页面装配并在最后一个进程退出后恢复', async () => {
  const fixture = createRunnerFixture()
  const firstSnapshot = resolve(fixture.root, 'first-snapshot.json')
  const secondSnapshot = resolve(fixture.root, 'second-snapshot.json')
  const first = runFixtureAsync(fixture, {
    delay: '5000',
    snapshotFile: firstSnapshot,
  })
  let second
  try {
    await waitForFile(fixture.transactionFile)
    second = runFixtureAsync(fixture, { snapshotFile: secondSnapshot })
    const secondResult = await waitForExit(second)
    assert.equal(secondResult.status, 0, secondResult.stderr)
    assert.ok(existsSync(fixture.transactionFile))
    assert.notEqual(readFileSync(fixture.appConfigFile, 'utf8'), fixture.originalAppConfig)

    first.kill('SIGTERM')
    const firstResult = await waitForExit(first)
    assert.equal(firstResult.status, 143, firstResult.stderr)
    assert.equal(readFileSync(fixture.appConfigFile, 'utf8'), fixture.originalAppConfig)
    assert.ok(!existsSync(fixture.transactionFile))
    assert.ok(!existsSync(resolve(fixture.inputDir, 'pages/index/index.tsx')))
    assert.ok(!existsSync(resolve(fixture.inputDir, 'static/shared.txt')))
  } finally {
    second?.kill('SIGTERM')
    first.kill('SIGTERM')
    await Promise.all([waitForExit(first), second ? waitForExit(second) : undefined])
    fixture.dispose()
  }
})

test('runner 为 H5 和微信小程序设置独立默认产物目录', () => {
  const fixture = createRunnerFixture()
  const h5Snapshot = resolve(fixture.root, 'h5-snapshot.json')
  const weappSnapshot = resolve(fixture.root, 'weapp-snapshot.json')
  try {
    assert.equal(runFixture(fixture, { snapshotFile: h5Snapshot }).status, 0)
    assert.equal(runFixture(fixture, { snapshotFile: weappSnapshot, type: 'weapp' }).status, 0)
    assert.equal(JSON.parse(readFileSync(h5Snapshot, 'utf8')).outputRoot, 'dist/h5')
    assert.equal(JSON.parse(readFileSync(weappSnapshot, 'utf8')).outputRoot, 'dist/mp-weixin')
  } finally {
    fixture.dispose()
  }
})

function createRunnerFixture() {
  const root = realpathSync(mkdtempSync(resolve(tmpdir(), 'kratos-taro-runner-')))
  const hostRoot = resolve(root, 'app')
  const inputDir = resolve(hostRoot, 'src')
  const appConfigFile = resolve(inputDir, 'app.config.ts')
  const transactionFile = resolve(hostRoot, '.kratos-taro-app-pages-state.json')
  const snapshotFile = resolve(root, 'snapshot.json')
  const binRoot = resolve(root, 'bin')
  const originalAppConfig = "import config from './app.config.base.json'\nexport default defineAppConfig(config)\n"
  write(resolve(hostRoot, 'package.json'), '{"type":"module"}\n')
  write(appConfigFile, originalAppConfig)
  write(
    resolve(inputDir, 'app.config.base.json'),
    '{"pages":["pages/bootstrap/index"],"window":{},"tabBar":{"custom":true,"list":[{"pagePath":"pages/index/index","text":""},{"pagePath":"pages/my/my","text":""}]}}\n',
  )
  write(
    resolve(inputDir, 'module-manifest.ts'),
    "import { firstModule } from '@fixture/one'\nimport secondModule from '@fixture/two'\nexport const moduleManifest = [firstModule, secondModule]\n",
  )
  write(resolve(inputDir, 'static/host.txt'), 'host')

  const firstRoot = createFixtureModule(hostRoot, '@fixture/one', 'one', {
    'pages/home/index': { route: 'pages/index/index', style: { title: 'first' } },
    'pagesMember/orders/detail': { style: { title: 'detail' } },
  })
  write(resolve(firstRoot, 'src/views/pages/home/index.tsx'), 'export default function Page() {}\n')
  write(
    resolve(firstRoot, 'src/views/pagesMember/orders/detail.tsx'),
    'export default function Page() {}\n',
  )
  write(
    resolve(firstRoot, 'src/views/pagesMember/orders/components/Editor.tsx'),
    'export default function Editor() {}\n',
  )
  write(resolve(firstRoot, 'src/static/shared.txt'), 'first')
  write(resolve(firstRoot, 'src/static/host.txt'), 'module-host')

  const secondRoot = createFixtureModule(hostRoot, '@fixture/two', 'two', {
    'pages/override/index': {
      route: 'pages/index/index',
      style: { title: 'second', navigationBarTitleText: 'second' },
    },
  })
  write(
    resolve(secondRoot, 'src/views/pages/override/index.tsx'),
    'export default function Page() {}\n',
  )
  write(resolve(secondRoot, 'src/static/shared.txt'), 'second')

  write(
    resolve(binRoot, 'pnpm'),
    `#!/usr/bin/env node
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const root = process.cwd()
const read = (file) => readFileSync(resolve(root, file), 'utf8')
writeFileSync(process.env.KRATOS_RUNNER_SNAPSHOT, JSON.stringify({
  appConfig: read('src/app.config.ts'),
  homeWrapper: read('src/pages/index/index.tsx'),
  homeConfig: read('src/pages/index/index.config.ts'),
  detailWrapper: read('src/pagesMember/orders/detail.tsx'),
  sharedStatic: read('src/static/shared.txt'),
  hostStatic: read('src/static/host.txt'),
  outputRoot: process.env.KRATOS_TARO_OUTPUT_ROOT,
}))
const delay = Number(process.env.KRATOS_RUNNER_DELAY_MS || 0)
if (delay) await new Promise((resolve) => setTimeout(resolve, delay))
`,
  )
  chmodSync(resolve(binRoot, 'pnpm'), 0o755)

  return {
    appConfigFile,
    binRoot,
    dispose: () => rmSync(root, { recursive: true, force: true }),
    hostRoot,
    inputDir,
    originalAppConfig,
    root,
    snapshotFile,
    transactionFile,
  }
}

function createFixtureModule(hostRoot, packageName, directory, pages) {
  const moduleRoot = resolve(hostRoot, 'node_modules/@fixture', directory)
  write(
    resolve(moduleRoot, 'package.json'),
    `${JSON.stringify({
      name: packageName,
      type: 'module',
      exports: { './build': './dist/build.mjs' },
    })}\n`,
  )
  write(
    resolve(moduleRoot, 'dist/build.mjs'),
    `export const buildModule = ${JSON.stringify({ name: packageName, root: moduleRoot, pages })}\n`,
  )
  return moduleRoot
}

function runFixture(fixture, options = {}) {
  const args = [runner, '--type', options.type ?? 'h5', '--mode', 'production']
  if (options.prepareOnly) args.push('--prepare-only')
  if (options.cleanupOnly) args.push('--cleanup-only')
  return spawnSync(process.execPath, args, {
    cwd: fixture.hostRoot,
    encoding: 'utf8',
    env: {
      ...process.env,
      KRATOS_RUNNER_SNAPSHOT: options.snapshotFile ?? fixture.snapshotFile,
      PATH: `${fixture.binRoot}:${process.env.PATH ?? ''}`,
    },
  })
}

function runFixtureAsync(fixture, options = {}) {
  const child = spawn(
    process.execPath,
    [runner, '--type', options.type ?? 'h5', '--mode', 'production'],
    {
      cwd: fixture.hostRoot,
      env: {
        ...process.env,
        KRATOS_RUNNER_DELAY_MS: options.delay ?? '0',
        KRATOS_RUNNER_SNAPSHOT: options.snapshotFile ?? fixture.snapshotFile,
        PATH: `${fixture.binRoot}:${process.env.PATH ?? ''}`,
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    },
  )
  return child
}

function waitForExit(child) {
  if (!child || child.exitCode !== null) {
    return Promise.resolve({ status: child?.exitCode ?? 0, stderr: '' })
  }
  let stderr = ''
  child.stderr?.on('data', (chunk) => {
    stderr += chunk
  })
  return new Promise((complete) => {
    child.once('exit', (code, signal) => {
      complete({ status: signal ? (signal === 'SIGINT' ? 130 : 143) : (code ?? 1), stderr })
    })
  })
}

async function waitForFile(file) {
  const deadline = Date.now() + 2000
  while (!existsSync(file)) {
    if (Date.now() >= deadline) throw new Error(`等待文件超时：${file}`)
    await new Promise((complete) => setTimeout(complete, 10))
  }
}

function write(file, content) {
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, content)
}
