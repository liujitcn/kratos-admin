import assert from 'node:assert/strict'
import test from 'node:test'

import { buildMenuTree, resolveRootMenuId } from '../src/navigation-tree.mjs'

const menus = [
  { id: 99901, parentId: 999, name: 'AppHome' },
  { id: 99909, parentId: 999, name: 'AppMy' },
  { id: 9990101, parentId: 99901, name: 'AppLogin' },
  { id: 999010101, parentId: 9990101, name: 'AppProtocol' },
  { id: 9990901, parentId: 99909, name: 'AppProfile' },
]

test('按上下级关系构建移动端菜单树', () => {
  const tree = buildMenuTree(menus, 999)

  assert.deepEqual(
    tree.map((menu) => menu.id),
    [99901, 99909],
  )
  assert.deepEqual(
    tree[0].children.map((menu) => menu.id),
    [9990101],
  )
  assert.deepEqual(
    tree[0].children[0].children.map((menu) => menu.id),
    [999010101],
  )
  assert.deepEqual(
    tree[1].children.map((menu) => menu.id),
    [9990901],
  )
})

test('嵌套页面沿父级关系归属二级 tab', () => {
  assert.equal(resolveRootMenuId(menus, 999010101, 999), 99901)
  assert.equal(resolveRootMenuId(menus, 9990901, 999), 99909)
  assert.equal(resolveRootMenuId(menus, 12345, 999), undefined)
})
