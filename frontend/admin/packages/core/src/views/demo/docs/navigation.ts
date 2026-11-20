import type { AdminComponentDoc } from "../../../demo.js";

/** 组件文档 API 分项。 */
export type ComponentDocsApi = "props" | "events" | "slots" | "methods";

/** 返回时恢复的文档位置与筛选条件。 */
export interface ComponentDocsLocation {
  id: string;
  query: string;
  api: ComponentDocsApi;
  apiPage: number;
}

/** 组件文档浏览状态，记录当前组件及返回历史。 */
export interface ComponentDocsState extends ComponentDocsLocation {
  history: ComponentDocsLocation[];
}

/** 创建以指定组件为起点的文档位置。 */
export function createComponentDocsState(id: string): ComponentDocsState {
  return { id, query: "", api: "props", apiPage: 1, history: [] };
}

/** 保存来源位置后切换组件，当前组件不重复入栈。 */
export function openComponentDoc(state: ComponentDocsState, id: string): void {
  if (state.id === id) return;
  const { history, ...location } = state;
  state.history = [...history, location];
  state.id = id;
  state.api = "props";
  state.apiPage = 1;
}

/** 逐级返回来源文档，没有来源时保持当前组件。 */
export function returnComponentDoc(state: ComponentDocsState): void {
  const previous = state.history.pop();
  if (!previous) return;
  const { apiPage, ...location } = previous;
  // API 分项变化会同步重置页码，恢复时最后回填来源页码。
  Object.assign(state, location, { apiPage });
}

/** 按中文名称、组件标识及职责搜索已启用模块的文档。 */
export function filterComponentDocs(entries: AdminComponentDoc[], query: string): AdminComponentDoc[] {
  const keyword = query.trim().toLowerCase();
  return entries.filter(entry => `${entry.id} ${entry.label} ${entry.description}`.toLowerCase().includes(keyword));
}

/** 对 API 列表分页，结果减少时回到有效页码。 */
export function getComponentDocsPage<T>(entries: T[], requestedPage: number, size: number) {
  const pages = Math.max(1, Math.ceil(entries.length / size));
  const page = Math.max(1, Math.min(requestedPage, pages));
  return { page, pages, items: entries.slice((page - 1) * size, page * size) };
}
