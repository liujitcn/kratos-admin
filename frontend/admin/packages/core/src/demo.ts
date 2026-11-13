import type { AdminModule, AdminViewLoader } from "./modules/index.js";
import { getRegisteredAdminModules } from "./modules/index.js";

/** 模块贡献组件文档的统一扩展名。 */
export const ADMIN_COMPONENT_DOCS = "kratos-admin.component-docs";

/** 从组件源码提取的 API 条目。 */
export interface AdminComponentApiEntry {
  name: string;
  description: string;
  type: string;
  default?: string;
  required?: boolean;
}

/** 组件声明的属性、事件、插槽和实例成员。 */
export interface AdminComponentApi {
  props: AdminComponentApiEntry[];
  events: AdminComponentApiEntry[];
  slots: AdminComponentApiEntry[];
  methods: AdminComponentApiEntry[];
}

/** 业务模块提供的单个组件说明及本地示例。 */
export interface AdminComponentDoc {
  id: string;
  label: string;
  group: string;
  owner: "Core" | "System" | "BI";
  /** 相对于所属包 src/components 的组件文件路径。 */
  file: string;
  description: string;
  usage: string;
  code: string;
  api: AdminComponentApi;
  /** 示例仅使用本地数据，接收 componentId 属性。 */
  example?: AdminViewLoader;
  related?: string[];
}

/** 聚合当前启用模块的组件文档，保持模块声明顺序。 */
export function getAdminComponentDocs(modules: AdminModule[] = getRegisteredAdminModules()): AdminComponentDoc[] {
  return modules.flatMap(module => (module.extensions?.[ADMIN_COMPONENT_DOCS] as AdminComponentDoc[] | undefined) ?? []);
}
