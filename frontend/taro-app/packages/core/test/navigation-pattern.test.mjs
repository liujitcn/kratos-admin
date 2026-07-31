import assert from 'node:assert/strict'
import test from 'node:test'
import { matchLogicalPath, parseLogicalQuery } from '../src/navigation-pattern.mjs'

test('匹配逻辑路由参数并解码', () => {
  assert.deepEqual(matchLogicalPath('app/order/:id', '/app/order/A%201/'), { id: 'A 1' })
  assert.equal(matchLogicalPath('app/order/:id', 'app/order'), undefined)
  assert.equal(matchLogicalPath('app/order/:id', 'app/profile/A'), undefined)
})

test('解析 query 并保留空值', () => {
  assert.deepEqual(parseLogicalQuery('id=2&name=%E6%B5%8B%E8%AF%95&empty='), {
    id: '2',
    name: '测试',
    empty: '',
  })
})
