export type MenuNode<T> = T & { children: Array<MenuNode<T>> }

export function buildMenuTree<T extends { id: number; parentId?: number }>(
  menus: ReadonlyArray<T>,
  rootId: number,
): Array<MenuNode<T>>

export function resolveRootMenuId(
  menus: ReadonlyArray<{ id: number; parentId?: number }>,
  menuId: number,
  rootId: number,
): number | undefined
