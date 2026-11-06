/** 流式放置只修改栅格坐标，浅拷贝卡片以支持响应式查询快照。 */
export function flowPlace<T extends { x?: number; y?: number; w: number; h: number }>(charts: T[], columns: number): T[] {
  if (
    !Number.isInteger(columns) ||
    columns < 1 ||
    charts.some(
      item =>
        !Number.isInteger(item.w) ||
        !Number.isInteger(item.h) ||
        (item.w ?? 6) < 1 ||
        (item.w ?? 6) > columns ||
        (item.h ?? 4) < 1
    )
  )
    throw new Error("卡片栅格尺寸无效");
  const placed = charts.map(item => ({ ...item }));
  const occupied = new Set<string>();
  const mark = (c: T) => {
    const w = c.w ?? 6;
    const h = c.h ?? 4;
    for (let dx = 0; dx < w; dx++) {
      for (let dy = 0; dy < h; dy++) occupied.add(`${(c.x ?? 0) + dx},${(c.y ?? 0) + dy}`);
    }
  };
  placed.forEach(c => {
    if (c.x !== undefined && c.y !== undefined) {
      mark(c);
    }
  });
  for (const c of placed) {
    if (c.x !== undefined && c.y !== undefined) continue;
    const w = c.w ?? 6;
    const h = c.h ?? 4;
    let found = false;
    for (let y = 0; !found; y++) {
      for (let x = 0; x + w <= columns; x++) {
        let free = true;
        for (let dx = 0; dx < w && free; dx++) {
          for (let dy = 0; dy < h && free; dy++) {
            if (occupied.has(`${x + dx},${y + dy}`)) free = false;
          }
        }
        if (free) {
          c.x = x;
          c.y = y;
          mark(c);
          found = true;
          break;
        }
      }
    }
  }
  return placed;
}
