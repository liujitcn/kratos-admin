import type { BaseTenantProject } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";

/** 租户项目业务数据查询键。 */
export interface TenantProjectKey {
  /** 租户 ID。 */
  tenant_id: number;
  /** 公共项目 ID。 */
  project_id: number;
}

/** 租户项目扩展数据，外层键由租户 ID 和项目 ID 组成。 */
export type TenantProjectExtraData = Record<string, Record<string, unknown>>;

/** 创建租户项目扩展数据的稳定键。 */
export function tenantProjectKey(tenantId: number, projectId: number): string {
  return `${tenantId}:${projectId}`;
}

/** 将外部业务字段合并到项目行，同时保护公共项目字段。 */
export function mergeTenantProjectExtraData(project: BaseTenantProject, extra?: Record<string, unknown>): BaseTenantProject & Record<string, unknown> {
  if (!extra) return { ...project } as BaseTenantProject & Record<string, unknown>;

  const protectedFields = new Set(["id", "tenant_id", "code", "name", "status", "sort", "remark", "created_at", "updated_at"]);
  const safeExtra = Object.fromEntries(Object.entries(extra).filter(([key]) => !protectedFields.has(key)));
  return { ...project, ...safeExtra };
}
