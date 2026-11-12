import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const hostRoot = resolve(import.meta.dirname, '../../../apps/uni-app')

test('宿主使用 uni-app x 根目录结构', () => {
  const manifest = JSON.parse(readFileSync(resolve(hostRoot, 'manifest.json'), 'utf8'))
  const pages = JSON.parse(readFileSync(resolve(hostRoot, 'pages.json'), 'utf8'))

  assert.ok(manifest['uni-app-x'])
  assert.equal(
    pages.pages.some((page) => page.path === 'pages/index/index'),
    true,
  )
  assert.equal(pages.subPackages[0].root, 'pagesMember')
  const app = readFileSync(resolve(hostRoot, 'App.uvue'), 'utf8')
  const main = readFileSync(resolve(hostRoot, 'main.ts'), 'utf8')
  assert.equal(app.includes('script setup'), true)
  assert.doesNotMatch(app, /<style[\s\S]*base\.scss/)
  assert.match(main, /@liujitcn\/kratos-uni-app-core\/styles\/base\.scss/)
})

test('x 页面生成器使用 uvue 页面和静态页面配置', () => {
  const generator = readFileSync(resolve(hostRoot, 'scripts/generate-pages.mjs'), 'utf8')
  assert.match(generator, /\.uvue/)
  assert.match(generator, /pagesMember/)
  assert.doesNotMatch(generator, /app-plus/)
})

test('core Vite 入口只封装官方 uni-app x 插件', () => {
  const source = readFileSync(resolve(import.meta.dirname, '../src/vite.ts'), 'utf8')
  assert.match(source, /createKratosUniPlugin/)
  assert.doesNotMatch(source, /kratosApp|recoverStalePageTransaction/)
})
