/**
 * @template {{id: number, parentId?: number}} T
 * @typedef {T & {children: MenuNode<T>[]}} MenuNode
 */

/**
 * 按父级编号将移动端扁平菜单构建为根目录下的树。
 * @template {{id: number, parentId?: number}} T
 * @param {ReadonlyArray<T>} menus
 * @param {number} rootId
 * @returns {MenuNode<T>[]}
 */
export function buildMenuTree(menus, rootId) {
  /** @type {Map<number, MenuNode<T>>} */
  const nodeMap = new Map(menus.map((menu) => [menu.id, { ...menu, children: [] }]))
  /** @type {MenuNode<T>[]} */
  const roots = []
  for (const node of nodeMap.values()) {
    if (node.parentId === rootId) {
      roots.push(node)
      continue
    }
    nodeMap.get(node.parentId)?.children.push(node)
  }
  return roots
}

/**
 * 沿父级关系查找页面所属的移动端二级菜单编号。
 * @param {ReadonlyArray<{id: number, parentId?: number}>} menus
 * @param {number} menuId
 * @param {number} rootId
 */
export function resolveRootMenuId(menus, menuId, rootId) {
  const menuMap = new Map(menus.map((menu) => [menu.id, menu]))
  const visited = new Set()
  let currentId = menuId
  while (!visited.has(currentId)) {
    visited.add(currentId)
    const current = menuMap.get(currentId)
    if (!current) return
    if (current.parentId === rootId) return current.id
    currentId = current.parentId
  }
}
