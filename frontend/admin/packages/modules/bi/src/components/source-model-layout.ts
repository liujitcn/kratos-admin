import type { BiSourceModel } from "../rpc/bi/admin/v1/bi_source";

/** 将事实表置于中央，关联子树分布两侧并为连线保留间距。 */
export function formatSourceModelLayout(model: BiSourceModel): void {
  const tables = new Map(model.tables.map(table => [table.key, table]));
  const children = new Map<string, string[]>();
  const targets = new Set<string>();
  for (const relation of model.relations) {
    if (!tables.has(relation.left_table) || !tables.has(relation.right_table)) continue;
    const branch = children.get(relation.left_table) ?? [];
    branch.push(relation.right_table);
    children.set(relation.left_table, branch);
    targets.add(relation.right_table);
  }

  if (tables.has(model.base_table)) {
    const branches = children.get(model.base_table) ?? [];
    for (const table of model.tables) {
      if (table.key !== model.base_table && !targets.has(table.key)) branches.push(table.key);
    }
    children.set(model.base_table, branches);
  }

  const visited = new Set<string>();
  let row = 0;

  /** 为子树分配独立行，避免同层节点和不同分支相互遮挡。 */
  function placeTable(key: string, depth: number, direction: number): number | undefined {
    const table = tables.get(key);
    if (!table || visited.has(key)) return;
    visited.add(key);
    const childRows = (children.get(key) ?? [])
      .map(child => placeTable(child, depth + 1, direction))
      .filter((value): value is number => value !== undefined);
    const center = childRows.length ? (childRows[0] + childRows[childRows.length - 1]) / 2 : row++;
    table.x = direction * depth * 520;
    table.y = 40 + center * 280;
    return center;
  }

  const base = tables.get(model.base_table);
  if (base) {
    visited.add(base.key);
    const branches = children.get(base.key) ?? [];
    const middle = Math.ceil(branches.length / 2);
    for (const key of branches.slice(0, middle)) placeTable(key, 1, 1);
    const rightRows = row;
    const rightTables = model.tables.filter(table => visited.has(table.key) && table !== base);
    row = 0;
    for (const key of branches.slice(middle)) placeTable(key, 1, -1);
    const leftRows = row;
    row = Math.max(1, leftRows, rightRows);
    for (const table of rightTables) table.y += (row - rightRows) * 140;
    for (const table of model.tables.filter(table => visited.has(table.key) && table.x < 0)) table.y += (row - leftRows) * 140;
    base.x = 0;
    base.y = 40 + (row - 1) * 140;
  }
  for (const table of model.tables.filter(table => !targets.has(table.key))) placeTable(table.key, 0, 1);
  for (const table of model.tables) placeTable(table.key, 0, 1);
  const minX = Math.min(0, ...model.tables.map(table => table.x));
  for (const table of model.tables) table.x += 40 - minX;
}
