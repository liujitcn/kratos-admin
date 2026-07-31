import assert from 'node:assert/strict'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import test from 'node:test'
import { scaffoldKratosApp } from '../src/index.mjs'

test('生成默认 system、本地模块和发布模块', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-app-cli-'))
  const target = resolve(root, 'demo')
  scaffoldKratosApp(target, { modules: ['orders'], packages: ['@acme/pay'] })
  const manifest = readFileSync(resolve(target, 'apps/app/src/module-manifest.ts'), 'utf8')
  const main = readFileSync(resolve(target, 'apps/app/src/main.ts'), 'utf8')
  assert.match(manifest, /kratos-app-system/)
  assert.match(manifest, /@local\/orders/)
  assert.match(manifest, /@acme\/pay/)
  assert.match(main, /import \{ createSSRApp \} from 'vue'/)
  assert.match(main, /registerKratosAppModules\(moduleManifest\)[\s\S]+export function createApp/)
  assert.match(main, /bootstrapKratosApp\(\{ app: App, createSSRApp,/)
  assert.ok(existsSync(resolve(target, 'packages/modules/orders/src/index.mjs')))
  assert.ok(existsSync(resolve(target, 'apps/app/vite.config.ts')))
  assert.ok(existsSync(resolve(target, 'apps/app/src/main.ts')))
  assert.ok(existsSync(resolve(target, 'apps/app/src/manifest.json')))
  const packageDirectories = ['', 'apps/app', 'packages/modules/orders']
  for (const directory of packageDirectories) {
    const packagePath = resolve(target, directory, 'package.json')
    const readmePath = resolve(target, directory, 'README.md')
    assert.ok(existsSync(packagePath))
    assert.ok(existsSync(readmePath))
    const packageMetadata = JSON.parse(readFileSync(packagePath, 'utf8'))
    assert.equal(typeof packageMetadata.description, 'string')
    assert.ok(packageMetadata.description.length > 0)
  }
  const workspaceReadme = readFileSync(resolve(target, 'README.md'), 'utf8')
  const hostReadme = readFileSync(resolve(target, 'apps/app/README.md'), 'utf8')
  const moduleReadme = readFileSync(resolve(target, 'packages/modules/orders/README.md'), 'utf8')
  assert.match(workspaceReadme, /# demo/)
  assert.match(hostReadme, /demo/)
  assert.match(moduleReadme, /@local\/orders/)
  assert.doesNotMatch(
    `${workspaceReadme}${hostReadme}${moduleReadme}`,
    /__PROJECT_NAME__|__MODULE_NAME__/,
  )
  assert.throws(() => scaffoldKratosApp(target), /已存在/)
  rmSync(root, { recursive: true, force: true })
})
