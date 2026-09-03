import assert from 'node:assert/strict'
import test from 'node:test'
import { buildMenuTree, resolveRootMenuId } from '../src/navigation-tree.mjs'

const menus = [
  { id: 99010000, parentId: 99000000, name: 'AppHome' },
  { id: 99090000, parentId: 99000000, name: 'AppMy' },
  { id: 99010100, parentId: 99010000, name: 'AppLogin' },
  { id: 99010101, parentId: 99010100, name: 'AppProtocol' },
  { id: 99090100, parentId: 99090000, name: 'AppProfile' },
]

test('按上下级关系构建移动端菜单树', () => {
  const tree = buildMenuTree(menus, 99000000)
  assert.deepEqual(tree.map((menu) => menu.id), [99010000, 99090000])
  assert.deepEqual(tree[0].children.map((menu) => menu.id), [99010100])
  assert.deepEqual(tree[0].children[0].children.map((menu) => menu.id), [99010101])
  assert.deepEqual(tree[1].children.map((menu) => menu.id), [99090100])
})

test('嵌套页面沿父级关系归属根 tab，并防止循环关系', () => {
  assert.equal(resolveRootMenuId(menus, 99010101, 99000000), 99010000)
  assert.equal(resolveRootMenuId(menus, 99090100, 99000000), 99090000)
  assert.equal(resolveRootMenuId(menus, 12345, 99000000), undefined)
  assert.equal(
    resolveRootMenuId(
      [
        { id: 1, parentId: 2 },
        { id: 2, parentId: 1 },
      ],
      1,
      99000000,
    ),
    undefined,
  )
})
