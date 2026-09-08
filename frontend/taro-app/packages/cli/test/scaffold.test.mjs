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
    const indexHtml = readFileSync(resolve(target, 'apps/taro-app/src/index.html'), 'utf8')
    const h5Env = readFileSync(resolve(target, '.env.development-h5'), 'utf8')

    assert.equal(rootPackage.packageManager, 'pnpm@10.13.1')
    assert.match(rootPackage.scripts['dev:h5'], /prepare:modules/)
    assert.equal(hostPackage.type, undefined)
    assert.equal(
      hostPackage.dependencies['@liujitcn/kratos-taro-app-core'],
      `^${cliPackage.version}`,
    )
    assert.equal(hostPackage.dependencies['@liujitcn/kratos-taro-app-ui'], `^${cliPackage.version}`)
    assert.equal(
      hostPackage.dependencies['@liujitcn/kratos-taro-app-system'],
      `^${cliPackage.version}`,
    )
    assert.equal(hostPackage.dependencies['@local/shop'], 'workspace:*')
    assert.equal(hostPackage.dependencies['@acme/customer-module'], 'latest')
    assert.equal(hostPackage.devDependencies['@pmmmwh/react-refresh-webpack-plugin'], '0.5.17')
    assert.equal(hostPackage.devDependencies['react-refresh'], '0.14.2')
    assert.equal(
      modulePackage.dependencies['@liujitcn/kratos-taro-app-core'],
      `^${cliPackage.version}`,
    )
    assert.equal(modulePackage.exports['./build'].import, './dist/build.mjs')
    assert.match(gitignore, /apps\/taro-app\/src\/pages\/\*/)
    assert.match(gitignore, /!apps\/taro-app\/src\/pages\/bootstrap\/\*\*/)
    assert.match(gitignore, /apps\/taro-app\/src\/pages\?\*\//)
    assert.match(manifest, /import \{ shopModule as localModule0 \} from '@local\/shop'/)
    assert.match(manifest, /import packageModule0 from '@acme\/customer-module'/)
    assert.match(config, /hostRequire\.resolve\(`\$\{name\}\/package\.json`\)/)
    assert.match(config, /sourceRoots\.forEach/)
    assert.match(config, /prebundle: \{ enable: false \}/)
    assert.match(config, /resolveHttpsOptions/)
    assert.match(config, /https: httpsOptions/)
    assert.match(config, /apiProxyOptions/)
    assert.match(config, /\.\.\/\.\.\/certs\/dev-key\.pem/)
    assert.match(config, /from: resolve\(__dirname, '\.\.\/src\/static'\)/)
    assert.match(config, /to: resolve\(__dirname, '\.\.', outputRoot, 'static'\)/)
    assert.match(config, /options: \{\}/)
    assert.match(config, /VITE_APP_BASE_API/)
    assert.doesNotMatch(
      config,
      /KRATOS_TARO_API_BASE|KRATOS_TARO_API_URL|KRATOS_TARO_PUBLIC_PATH|KRATOS_TARO_STATIC_URL/,
    )
    assert.ok(existsSync(resolve(target, '.env.development')))
    assert.ok(existsSync(resolve(target, '.env.development-h5')))
    assert.match(h5Env, /VITE_APP_HTTPS=false/)
    assert.match(h5Env, /VITE_APP_HTTPS_KEY=\.\.\/\.\.\/certs\/dev-key\.pem/)
    assert.ok(existsSync(resolve(target, '.env.production')))
    assert.ok(existsSync(resolve(target, '.env.production-h5')))
    assert.ok(existsSync(resolve(target, 'apps/taro-app/src/pages/bootstrap/index.tsx')))
    assert.ok(existsSync(resolve(target, 'apps/taro-app/src/static/favicon.ico')))
    assert.ok(existsSync(resolve(target, 'apps/taro-app/src/static/h5-root-font.js')))
    assert.match(
      indexHtml,
      /<link rel="icon" type="image\/x-icon" href="\.\/static\/favicon\.ico" \/>/,
    )
    assert.match(indexHtml, /<script src="\.\/static\/h5-root-font\.js" defer><\/script>/)
    assert.doesNotMatch(indexHtml, /htmlWebpackPlugin\.options\.script/)
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

// 验证实际发布包可独立创建项目，避免源码测试掩盖资源漏发。
test('打包后的 CLI 创建项目并完整复制静态资源', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-app-packed-'))
  const target = resolve(root, 'customer-app')
  try {
    const packed = spawnSync('pnpm', ['pack', '--json', '--pack-destination', root], {
      cwd: resolve(import.meta.dirname, '..'),
      encoding: 'utf8',
      timeout: 30000,
    })
    assert.equal(packed.status, 0, packed.error?.message ?? packed.stderr)
    const archive = JSON.parse(packed.stdout)
    const extracted = spawnSync('tar', ['-xzf', archive.filename, '-C', root], {
      encoding: 'utf8',
      timeout: 30000,
    })
    assert.equal(extracted.status, 0, extracted.error?.message ?? extracted.stderr)

    const created = spawnSync(
      process.execPath,
      [resolve(root, 'package/bin/kratos-taro-app.mjs'), 'create', target, '--module', 'app'],
      { cwd: root, encoding: 'utf8', timeout: 30000 },
    )
    assert.equal(created.status, 0, created.error?.message ?? created.stderr)
    assert.match(created.stdout, /已创建 Taro workspace/)
    assert.deepEqual(
      readFileSync(resolve(target, 'apps/taro-app/src/static/favicon.ico')),
      readFileSync(resolve(import.meta.dirname, '../assets/favicon.ico')),
    )
    assert.ok(existsSync(resolve(target, 'apps/taro-app/src/static/h5-root-font.js')))
    assert.ok(existsSync(resolve(target, 'packages/modules/app/src/index.ts')))
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})

test('脚手架校验项目、模块和包名', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-app-validation-'))
  try {
    assert.throws(() => scaffoldKratosTaroApp(resolve(root, 'BadName')), /kebab-case/)
    assert.throws(
      () => scaffoldKratosTaroApp(resolve(root, 'valid-name'), { modules: ['../system'] }),
      /模块名无效/,
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

test('CLI 独立生成 system 多模块、完整语言入口与项目构建配置', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-frontend-locales-'))
  const target = resolve(root, 'app')
  try {
    scaffoldKratosTaroApp(target, { modules: ['system', 'order'], kratosProject: true })
    for (const name of ['system', 'order']) {
      const moduleRoot = resolve(target, `packages/modules/${name}`)
      const entry = readFileSync(resolve(moduleRoot, 'src/index.ts'), 'utf8')
      assert.match(entry, /messages: LOCALE_MESSAGES/)
      const locales = readFileSync(resolve(moduleRoot, 'src/locales/generated.ts'), 'utf8')
      for (const locale of ['zh-CN', 'en-US', 'zh-TW', 'ja-JP']) {
        assert.match(locales, new RegExp(locale))
        assert.deepEqual(JSON.parse(readFileSync(resolve(moduleRoot, `src/locales/${locale}.json`))), {})
      }
      assert.ok(existsSync(resolve(moduleRoot, 'src/api/.gitkeep')))
      assert.ok(existsSync(resolve(moduleRoot, 'src/rpc/.gitkeep')))
      assert.ok(JSON.parse(readFileSync(resolve(moduleRoot, 'package.json'))).scripts.build)
    }
    const host = JSON.parse(readFileSync(resolve(target, 'apps/taro-app/package.json')))
    assert.match(host.scripts['build:h5'], /backend\/data\/taro-app/)
    assert.ok(existsSync(resolve(target, 'apps/taro-app/tsconfig.json')))
    assert.ok(existsSync(resolve(target, 'scripts/check-package-exports.mjs')))
    const checked = spawnSync(process.execPath, ['scripts/sync-locales.mjs'], { cwd: target, encoding: 'utf8' })
    assert.equal(checked.status, 0, checked.stderr)
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
