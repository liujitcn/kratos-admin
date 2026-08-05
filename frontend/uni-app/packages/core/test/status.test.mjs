import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const statusPage = readFileSync(
  resolve(import.meta.dirname, '../src/views/pages/status/index.vue'),
  'utf8',
)

test('启动状态页不阻塞 uni-app 页面生命周期等待导航请求', () => {
  assert.doesNotMatch(statusPage, /onLoad\(async\s*\(/)
  assert.match(statusPage, /void initializeAppNavigation\(\)/)
})
