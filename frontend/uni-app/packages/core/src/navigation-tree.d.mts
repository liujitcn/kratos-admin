export interface MenuNode<T extends { id: number; parentId?: number }> extends T {
  children: MenuNode<T>[]
}

export function buildMenuTree<T extends { id: number; parentId?: number }>(
  menus: readonly T[],
  rootId: number,
): MenuNode<T>[]

export function resolveRootMenuId(
  menus: readonly { id: number; parentId?: number }[],
  menuId: number,
  rootId: number,
): number | undefined
