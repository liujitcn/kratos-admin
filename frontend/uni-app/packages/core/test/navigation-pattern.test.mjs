import assert from 'node:assert/strict'
import test from 'node:test'
import { matchLogicalPath, parseLogicalQuery } from '../src/navigation-pattern.mjs'

test('匹配逻辑路由参数并解码', () => {
  assert.deepEqual(matchLogicalPath('app/order/:id', '/app/order/A%201/'), { id: 'A 1' })
  assert.equal(matchLogicalPath('app/order/:id', 'app/order'), undefined)
  assert.equal(matchLogicalPath('app/order/:id', 'app/profile/A'), undefined)
})

test('解析 query，后续可覆盖同名路径参数', () => {
  const params = matchLogicalPath('app/order/:id', 'app/order/1')
  const query = parseLogicalQuery('id=2&name=%E6%B5%8B%E8%AF%95')
  assert.deepEqual({ ...params, ...query }, { id: '2', name: '测试' })
})
