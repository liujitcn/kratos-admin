import assert from 'node:assert/strict'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import test from 'node:test'
import { scaffoldKratosApp } from '../src/index.mjs'

test('生成默认 system、本地模块和发布模块', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-app-cli-'))
  const target = resolve(root, 'demo')
  scaffoldKratosApp(target, { modules: ['orders'], packages: ['@acme/pay'] })
  const manifest = readFileSync(resolve(target, 'apps/uni-app/src/module-manifest.ts'), 'utf8')
  const main = readFileSync(resolve(target, 'apps/uni-app/src/main.ts'), 'utf8')
  const viteConfig = readFileSync(resolve(target, 'apps/uni-app/vite.config.ts'), 'utf8')
  const cliPackage = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../package.json'), 'utf8'),
  )
  const hostPackage = JSON.parse(readFileSync(resolve(target, 'apps/uni-app/package.json'), 'utf8'))
  const workspacePackage = JSON.parse(readFileSync(resolve(target, 'package.json'), 'utf8'))
  const modulePackage = JSON.parse(
    readFileSync(resolve(target, 'packages/modules/orders/package.json'), 'utf8'),
  )
  const productionEnv = readFileSync(resolve(target, '.env.production'), 'utf8')
  assert.match(manifest, /kratos-uni-app-system/)
  assert.match(manifest, /@local\/orders/)
  assert.match(manifest, /@acme\/pay/)
  assert.match(main, /import \{ createSSRApp \} from 'vue'/)
  assert.match(main, /registerKratosAppModules\(moduleManifest\)[\s\S]+export function createApp/)
  assert.match(main, /registerUserStoreExtension\(\{[\s\S]+onLogin: initializeAppNavigation/)
  assert.match(main, /bootstrapKratosApp\(\{ app: App, createSSRApp,/)
  assert.match(viteConfig, /server: \{[\s\S]+port: Number\(env\.VITE_APP_PORT \|\| 5004\)/)
  assert.equal(hostPackage.dependencies['@liujitcn/kratos-uni-app-core'], `^${cliPackage.version}`)
  assert.equal(
    hostPackage.dependencies['@liujitcn/kratos-uni-app-system'],
    `^${cliPackage.version}`,
  )
  assert.equal(
    modulePackage.dependencies['@liujitcn/kratos-uni-app-core'],
    `^${cliPackage.version}`,
  )
  assert.equal(hostPackage.dependencies['@dcloudio/vite-plugin-uni'], '3.0.0-5010520260709002')
  assert.equal(hostPackage.scripts['dev:h5'], 'pnpm run recover:pages && uni --mode development-h5')
  assert.equal(workspacePackage.packageManager, 'pnpm@10.13.1')
  assert.match(viteConfig, /VITE_APP_PORT/)
  assert.match(viteConfig, /kratosApp\(\{ modules: moduleManifest \}\)/)
  assert.ok(existsSync(resolve(target, 'packages/modules/orders/src/index.mjs')))
  assert.ok(existsSync(resolve(target, 'apps/uni-app/vite.config.ts')))
  assert.ok(existsSync(resolve(target, 'apps/uni-app/src/main.ts')))
  assert.ok(existsSync(resolve(target, 'apps/uni-app/src/manifest.json')))
  assert.ok(existsSync(resolve(target, '.env.development-h5')))
  assert.ok(existsSync(resolve(target, '.env.production-h5')))
  assert.doesNotMatch(productionEnv, /localhost/)
  const packageDirectories = ['', 'apps/uni-app', 'packages/modules/orders']
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
  const hostReadme = readFileSync(resolve(target, 'apps/uni-app/README.md'), 'utf8')
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
