import type {
  BaseTenantProject,
  PageBaseTenantProjectRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";
import type { ColumnProps, TableActionProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { TenantProjectExtraData, TenantProjectKey } from "./tenant-project-manager-data";

export {
  arrangeTenantProjectColumns,
  mergeTenantProjectExtraData,
  tenantProjectKey,
  type TenantProjectExtraData,
  type TenantProjectKey
} from "./tenant-project-manager-data";

/** 租户项目单条业务上下文。 */
export interface TenantProjectContext<TData extends Record<string, unknown> = Record<string, unknown>> extends TenantProjectKey {
  /** 公共项目基础数据。 */
  project: BaseTenantProject;
  /** 外部业务模块返回的当前项目数据。 */
  data: TData;
}

/** 租户项目扩展字段批量加载上下文。 */
export interface TenantProjectExtraDataLoadContext {
  /** 当前页全部项目键。 */
  projects: TenantProjectKey[];
  /** 当前项目列表查询参数。 */
  query: PageBaseTenantProjectRequest;
}

/** 租户项目扩展数据加载器。 */
export type TenantProjectExtraDataLoader = (
  context: TenantProjectExtraDataLoadContext
) => Promise<TenantProjectExtraData | undefined>;

/** 租户项目扩展操作按钮。 */
export type TenantProjectAction = Omit<TableActionProps, "disabled" | "hidden" | "params" | "onClick"> & {
  /** 按项目上下文控制按钮显隐。 */
  hidden?: boolean | ((context: TenantProjectContext) => boolean);
  /** 按项目上下文控制按钮禁用。 */
  disabled?: boolean | ((context: TenantProjectContext) => boolean);
  /** 根据项目上下文生成按钮附加参数。 */
  params?: (context: TenantProjectContext) => Record<string, any>;
  /** 执行项目扩展操作。 */
  onClick: (context: TenantProjectContext, params?: Record<string, any>) => void | Promise<void>;
};

/** 租户项目管理的扩展列配置。 */
export type TenantProjectExtraColumn = ColumnProps & {
  /** 插入到指定基础字段后面，未配置时默认插入备注字段后面。 */
  after?: string;
};

/** 租户项目管理组件属性。 */
export interface TenantProjectManagerProps {
  /** 追加到基础列表的表格列，可通过 after 指定插入位置。 */
  extraColumns?: TenantProjectExtraColumn[];
  /** 追加到基础操作列的业务操作。 */
  extraActions?: TenantProjectAction[];
  /** 加载当前页外部业务字段。 */
  loadExtraData?: TenantProjectExtraDataLoader;
}
