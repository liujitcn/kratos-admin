import assert from 'node:assert/strict'
import { execFileSync, spawnSync } from 'node:child_process'
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import test from 'node:test'

// 使用实际 tarball 执行 CLI，避免源码测试遗漏发布包中的资源缺失。
test('打包后的 CLI 可生成业务模块并复制 favicon', () => {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-uni-app-packed-'))
  const cliRoot = resolve(import.meta.dirname, '..')
  try {
    execFileSync('pnpm', ['pack', '--pack-destination', root], { cwd: cliRoot })
    const archives = readdirSync(root).filter((name) => name.endsWith('.tgz'))
    assert.equal(archives.length, 1)
    execFileSync('tar', ['-xzf', resolve(root, archives[0]), '-C', root])

    const target = resolve(root, 'demo')
    const created = spawnSync(
      process.execPath,
      [resolve(root, 'package/bin/kratos-uni-app.mjs'), 'create', target, '--module', 'app'],
      { cwd: root, encoding: 'utf8' },
    )
    assert.equal(created.status, 0, created.stderr)
    assert.deepEqual(
      readFileSync(resolve(target, 'apps/uni-app/favicon.ico')),
      readFileSync(resolve(cliRoot, 'assets/favicon.ico')),
    )
    assert.match(
      readFileSync(resolve(target, 'apps/uni-app/index.html'), 'utf8'),
      /<link rel="icon" href="\.\/favicon\.ico">/,
    )
    assert.ok(existsSync(resolve(target, 'apps/uni-app/package.json')))
    assert.ok(existsSync(resolve(target, 'packages/modules/app/src/index.mjs')))
  } finally {
    rmSync(root, { recursive: true, force: true })
  }
})
