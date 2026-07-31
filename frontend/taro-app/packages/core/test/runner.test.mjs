import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
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
    assert.match(snapshot.homeWrapper, /@fixture\/two\/views\/pages\/override\/index/)
    assert.match(snapshot.homeWrapper, /KratosPageFrame/)
    assert.match(snapshot.homeWrapper, /navigationBarTitleText="second"/)
    assert.match(snapshot.detailWrapper, /@fixture\/one\/views\/pagesMember\/orders\/detail/)
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

test('runner 拒绝覆盖活动进程持有的页面事务', () => {
  const fixture = createRunnerFixture()
  try {
    write(
      fixture.transactionFile,
      `${JSON.stringify({
        inputDir: fixture.inputDir,
        appConfigFile: fixture.appConfigFile,
        originalAppConfig: fixture.originalAppConfig,
        generatedFiles: [],
        generatedStaticFiles: [],
        ownerPid: process.pid,
      })}\n`,
    )
    const result = runFixture(fixture)
    assert.notEqual(result.status, 0)
    assert.match(result.stderr, new RegExp(`页面装配已由进程 ${process.pid} 使用`))
    assert.equal(readFileSync(fixture.appConfigFile, 'utf8'), fixture.originalAppConfig)
    assert.ok(existsSync(fixture.transactionFile))
    assert.ok(!existsSync(fixture.snapshotFile))
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
    '{"pages":["pages/bootstrap/index"],"window":{}}\n',
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
}))
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

function runFixture(fixture) {
  return spawnSync(process.execPath, [runner, '--type', 'h5', '--mode', 'production'], {
    cwd: fixture.hostRoot,
    encoding: 'utf8',
    env: {
      ...process.env,
      KRATOS_RUNNER_SNAPSHOT: fixture.snapshotFile,
      PATH: `${fixture.binRoot}:${process.env.PATH ?? ''}`,
    },
  })
}

function write(file, content) {
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, content)
}
