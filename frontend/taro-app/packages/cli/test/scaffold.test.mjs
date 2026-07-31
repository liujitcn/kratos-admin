import assert from 'node:assert/strict'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { scaffoldKratosTaroApp } from '../src/index.mjs'

test('生成可扩展的 Taro workspace、本地模块和发布模块清单', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-app-cli-'))
  const target = resolve(root, 'customer-app')
  try {
    scaffoldKratosTaroApp(target, {
      modules: ['shop', 'shop'],
      packages: ['@acme/customer-module'],
    })
    const cliPackage = readJson(resolve(import.meta.dirname, '../package.json'))
    const rootPackage = readJson(resolve(target, 'package.json'))
    const hostPackage = readJson(resolve(target, 'apps/taro-app/package.json'))
    const modulePackage = readJson(resolve(target, 'packages/modules/shop/package.json'))
    const gitignore = readFileSync(resolve(target, '.gitignore'), 'utf8')
    const manifest = readFileSync(resolve(target, 'apps/taro-app/src/module-manifest.ts'), 'utf8')
    const config = readFileSync(resolve(target, 'apps/taro-app/config/index.ts'), 'utf8')

    assert.equal(rootPackage.packageManager, 'pnpm@10.13.1')
    assert.match(rootPackage.scripts['dev:h5'], /prepare:modules/)
    assert.equal(hostPackage.type, undefined)
    assert.equal(hostPackage.dependencies['@liujitcn/kratos-taro-app-core'], `^${cliPackage.version}`)
    assert.equal(hostPackage.dependencies['@liujitcn/kratos-taro-app-ui'], `^${cliPackage.version}`)
    assert.equal(hostPackage.dependencies['@liujitcn/kratos-taro-app-system'], `^${cliPackage.version}`)
    assert.equal(hostPackage.dependencies['@local/shop'], 'workspace:*')
    assert.equal(hostPackage.dependencies['@acme/customer-module'], 'latest')
    assert.equal(hostPackage.devDependencies['@pmmmwh/react-refresh-webpack-plugin'], '0.5.17')
    assert.equal(hostPackage.devDependencies['react-refresh'], '0.14.2')
    assert.equal(modulePackage.dependencies['@liujitcn/kratos-taro-app-core'], `^${cliPackage.version}`)
    assert.equal(modulePackage.exports['./build'].import, './dist/build.mjs')
    assert.match(gitignore, /apps\/taro-app\/src\/pages\/\*/)
    assert.match(gitignore, /!apps\/taro-app\/src\/pages\/bootstrap\/\*\*/)
    assert.match(gitignore, /apps\/taro-app\/src\/pages\?\*\//)
    assert.match(manifest, /import \{ shopModule \} from '@local\/shop'/)
    assert.match(manifest, /import packageModule0 from '@acme\/customer-module'/)
    assert.match(config, /hostRequire\.resolve\(`\$\{name\}\/package\.json`\)/)
    assert.match(config, /sourceRoots\.forEach/)
    assert.match(config, /prebundle: \{ enable: false \}/)
    assert.match(config, /from: resolve\(__dirname, '\.\.\/src\/static'\)/)
    assert.match(config, /to: resolve\(__dirname, '\.\.', outputRoot, 'static'\)/)
    assert.match(config, /options: \{\}/)
    assert.ok(existsSync(resolve(target, 'apps/taro-app/src/pages/bootstrap/index.tsx')))
    assert.ok(existsSync(resolve(target, 'apps/taro-app/scripts/run-taro.mjs')))
    assert.ok(existsSync(resolve(target, 'packages/modules/shop/src/build.ts')))
    assert.ok(existsSync(resolve(target, 'packages/modules/shop/src/pages.ts')))

    const documentation = [
      readFileSync(resolve(target, 'README.md'), 'utf8'),
      readFileSync(resolve(target, 'apps/taro-app/README.md'), 'utf8'),
      readFileSync(resolve(target, 'packages/modules/shop/README.md'), 'utf8'),
    ].join('\n')
    assert.match(documentation, /customer-app/)
    assert.doesNotMatch(documentation, /__PROJECT_NAME__|__MODULE_NAME__/)
    assert.throws(() => scaffoldKratosTaroApp(target), /已存在/)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('CLI 解析重复选项并拒绝未知参数', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-app-bin-'))
  const target = resolve(root, 'shop-app')
  const bin = resolve(import.meta.dirname, '../bin/kratos-taro-app.mjs')
  try {
    const created = spawnSync(
      process.execPath,
      [bin, 'create', target, '--module', 'shop,order', '--with', '@acme/pay'],
      { encoding: 'utf8' },
    )
    assert.equal(created.status, 0, created.stderr)
    assert.match(created.stdout, /已创建 Taro workspace/)
    assert.ok(existsSync(resolve(target, 'packages/modules/shop/src/index.ts')))
    assert.ok(existsSync(resolve(target, 'packages/modules/order/src/index.ts')))

    const invalid = spawnSync(process.execPath, [bin, 'create', resolve(root, 'bad'), '--wat'], {
      encoding: 'utf8',
    })
    assert.equal(invalid.status, 1)
    assert.match(invalid.stderr, /未知参数/)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('脚手架校验项目、模块和包名', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-app-validation-'))
  try {
    assert.throws(() => scaffoldKratosTaroApp(resolve(root, 'BadName')), /kebab-case/)
    assert.throws(
      () => scaffoldKratosTaroApp(resolve(root, 'valid-name'), { modules: ['system'] }),
      /保留名称/,
    )
    assert.throws(
      () => scaffoldKratosTaroApp(resolve(root, 'valid-name'), { packages: ['Bad Package'] }),
      /包名无效/,
    )
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

function readJson(file) {
  return JSON.parse(readFileSync(file, 'utf8'))
}
