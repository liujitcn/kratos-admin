import type { RouteItem } from "../rpc/system/admin/v1/auth";

/** 获取菜单节点的首个可跳转子路径。 */
function getFirstRoutePath(item: RouteItem): string {
  if (item.path) return item.path;
  for (const child of item.children ?? []) {
    const path = getFirstRoutePath(child);
    if (path) return path;
  }
  return "";
}

/** 获取菜单节点在 Element Plus 菜单树中的稳定索引。 */
export function getRouteMenuKey(item: RouteItem): string {
  if (item.path) return item.path;
  const title = item.meta?.title ?? "";
  return `folder:${title}:${getFirstRoutePath(item)}`;
}
