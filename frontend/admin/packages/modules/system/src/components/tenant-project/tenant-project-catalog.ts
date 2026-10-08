import { reactive } from "vue";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-core/rpc/common/v1/common";
import { defBaseTenantProjectService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project";
import type { TreeBaseTenantProjectResponse_Option } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";

/** 租户项目范围。 */
export type TenantProjectScopeSelection = {
  /** 租户 ID。 */
  tenant_id: number;
  /** 项目 ID。 */
  project_id: number;
};

/** 租户项目名称信息。 */
export type TenantProjectDisplayInfo = {
  /** 租户名称。 */
  tenantName: string;
  /** 项目名称。 */
  projectName: string;
};

/** 租户项目表格行范围字段。 */
export type TenantProjectScopeRow = {
  /** 租户 ID。 */
  tenant_id?: number;
  /** 项目 ID。 */
  project_id?: number;
  /** 租户名称。 */
  tenant_name?: string;
  /** 项目名称。 */
  project_name?: string;
};

/** 当前账号租户项目目录。 */
export type TenantProjectCatalog = {
  /** 租户项目树。 */
  treeOptions: TreeBaseTenantProjectResponse_Option[];
  /** 普通租户项目选项。 */
  projectOptions: SelectOptionResponse_Option[];
};

type SharedCatalog = TenantProjectCatalog & {
  mode?: boolean;
  loaded: boolean;
  request?: Promise<TenantProjectCatalog>;
  displayMap: Map<string, TenantProjectDisplayInfo>;
  projectMap: Map<string, TenantProjectDisplayInfo>;
};

// 共享目录使用响应式对象，目录异步加载完成后已渲染的名称单元格能自动刷新。
const sharedCatalog = reactive<SharedCatalog>({
  loaded: false,
  treeOptions: [],
  projectOptions: [],
  displayMap: new Map(),
  projectMap: new Map()
});

/** 创建租户项目键。 */
function scopeKey(tenantId: number, projectId: number) {
  return `${tenantId}:${projectId}`;
}

/** 清空共享项目目录缓存。 */
function resetCatalog(mode: boolean) {
  sharedCatalog.mode = mode;
  sharedCatalog.loaded = false;
  sharedCatalog.treeOptions = [];
  sharedCatalog.projectOptions = [];
  sharedCatalog.displayMap.clear();
  sharedCatalog.projectMap.clear();
}

/** 将项目目录写入共享名称缓存。 */
function indexProject(tenantId: number, projectId: number, projectName: string, tenantName: string) {
  const info = { tenantName, projectName };
  sharedCatalog.displayMap.set(scopeKey(tenantId, projectId), info);
  sharedCatalog.projectMap.set(String(projectId), info);
}

function indexTree(options: TreeBaseTenantProjectResponse_Option[], tenantName = "") {
  for (const option of options) {
    if (option.type === "project") {
      indexProject(option.tenant_id, option.project_id, option.label, tenantName);
      continue;
    }
    indexTree(option.children ?? [], option.label);
  }
}

/** 读取当前共享租户项目名称映射。 */
export function getTenantProjectDisplayMap() {
  return sharedCatalog.displayMap;
}

/** 读取项目目录中的名称信息。 */
export function resolveTenantProjectDisplay(tenantId?: number, projectId?: number) {
  if (!projectId) return undefined;
  return sharedCatalog.displayMap.get(scopeKey(tenantId ?? 0, projectId)) ?? sharedCatalog.projectMap.get(String(projectId));
}

/** 根据租户项目 ID 生成展示文本。 */
export function resolveTenantProjectLabel(row: TenantProjectScopeRow, isDefaultTenant: boolean) {
  const displayInfo = resolveTenantProjectDisplay(row.tenant_id, row.project_id);
  const projectName = row.project_name ?? displayInfo?.projectName ?? row.project_id ?? "-";
  if (!isDefaultTenant) return String(projectName);
  const tenantName = row.tenant_name ?? displayInfo?.tenantName;
  return tenantName ? `${tenantName} / ${projectName}` : String(projectName);
}

/** 加载并缓存当前账号可见的租户项目目录。 */
export async function loadTenantProjectOptions(isDefaultTenant: boolean, force = false): Promise<TenantProjectCatalog> {
  if (sharedCatalog.mode !== isDefaultTenant) resetCatalog(isDefaultTenant);
  if (sharedCatalog.loaded && !force) return sharedCatalog;
  if (sharedCatalog.request && !force) return sharedCatalog.request;
  sharedCatalog.request = (async () => {
    if (isDefaultTenant) {
      const response = await defBaseTenantProjectService.TreeBaseTenantProject({ keyword: "" });
      sharedCatalog.treeOptions = response.list ?? [];
      indexTree(sharedCatalog.treeOptions);
    } else {
      const response = await defBaseTenantProjectService.OptionBaseTenantProject({ tenant_id: 0 });
      sharedCatalog.projectOptions = response.list ?? [];
      for (const option of sharedCatalog.projectOptions) indexProject(0, Number(option.value), option.label, "");
      sharedCatalog.treeOptions = sharedCatalog.projectOptions.map(option => ({
        value: `project:0:${Number(option.value)}`,
        label: option.label,
        type: "project",
        tenant_id: 0,
        project_id: Number(option.value),
        disabled: option.disabled,
        children: []
      }));
    }
    sharedCatalog.loaded = true;
    return sharedCatalog;
  })().finally(() => {
    sharedCatalog.request = undefined;
  });
  return sharedCatalog.request;
}
