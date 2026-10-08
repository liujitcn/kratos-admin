import { computed, h, ref } from "vue";
import type { ComputedRef, Ref } from "vue";
import type { ColumnProps, SearchRenderScope } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { t } from "@liujitcn/kratos-admin-core";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-core/rpc/common/v1/common";
import type { TreeBaseTenantProjectResponse_Option } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";
import TenantProjectSelect, { type TenantProjectSelection as TenantProjectNodeSelection } from "./TenantProjectSelect.vue";
import TenantProjectText from "./TenantProjectText.vue";
import { getTenantProjectDisplayMap, loadTenantProjectOptions, type TenantProjectDisplayInfo, type TenantProjectScopeSelection } from "./tenant-project-catalog";
import { parseTenantProjectValue } from "./tenant-project-manager-data";

export type { TenantProjectDisplayInfo, TenantProjectScopeRow, TenantProjectScopeSelection } from "./tenant-project-catalog";
export { getTenantProjectDisplayMap, resolveTenantProjectLabel } from "./tenant-project-catalog";
export { parseTenantProjectValue } from "./tenant-project-manager-data";

/** 租户项目范围参数。 */
export type TenantProjectScopeParams = {
  /** 租户项目树值。 */
  tenant_project_tree_value?: string;
  /** 租户 ID。 */
  tenant_id?: number;
  /** 项目 ID。 */
  project_id?: number;
};

/** 租户二级资源范围配置。 */
export interface TenantProjectScopeOptions {
  /** 二级资源名称国际化键，默认使用 common.field.project。 */
  resourceLabelKey?: string;
  /** 全部二级资源名称国际化键。 */
  allResourceLabelKey?: string;
  /** 搜索参数键。 */
  searchKey?: string;
  /** 搜索项排序。 */
  order?: number;
  /** 表格列最小宽度。 */
  minWidth?: number;
  /** 表格列绑定的项目 ID 字段。 */
  projectField?: string;
  /** 表格列绑定的租户 ID 字段。 */
  tenantField?: string;
}

/** 租户项目范围状态。 */
export interface TenantProjectScope {
  /** 当前账号是否为默认租户。 */
  isDefaultTenant: ComputedRef<boolean>;
  /** 当前范围列标题。 */
  scopeLabel: ComputedRef<string>;
  /** 当前范围选择器占位文案。 */
  allLabel: ComputedRef<string>;
  /** 当前账号可见的树选项。 */
  treeOptions: Ref<TreeBaseTenantProjectResponse_Option[]>;
  /** 当前账号可见的普通项目选项。 */
  projectOptions: Ref<SelectOptionResponse_Option[]>;
  /** 当前目录名称映射。 */
  displayMap: ComputedRef<Map<string, TenantProjectDisplayInfo>>;
  /** 加载当前账号可见范围。 */
  loadOptions: (force?: boolean) => Promise<TreeBaseTenantProjectResponse_Option[] | SelectOptionResponse_Option[]>;
  /** 当前范围的全部项目。 */
  projectSelections: (tenantId?: number) => TenantProjectScopeSelection[];
  /** 归一化搜索参数。 */
  normalizeParams: <T extends TenantProjectScopeParams>(params: T) => T & TenantProjectScopeParams;
  /** 创建与搜索条件配套的表格列。 */
  column: ComputedRef<ColumnProps>;
}

/** 创建租户项目范围状态。 */
export function useTenantProjectScope(options: TenantProjectScopeOptions = {}): TenantProjectScope {
  const { isDefaultTenant } = useTenantScope();
  const treeOptions = ref<TreeBaseTenantProjectResponse_Option[]>([]);
  const projectOptions = ref<SelectOptionResponse_Option[]>([]);
  const resourceLabelKey = options.resourceLabelKey ?? "common.field.project";
  const allResourceLabelKey = options.allResourceLabelKey ?? "system.base.tenant_project.all";
  const scopeLabel = computed(() => isDefaultTenant.value ? `${t("common.field.tenant")} / ${t(resourceLabelKey)}` : t(resourceLabelKey));
  const allLabel = computed(() => t(allResourceLabelKey));
  const displayMap = computed(() => getTenantProjectDisplayMap());

  async function loadOptions(force = false) {
    const catalog = await loadTenantProjectOptions(isDefaultTenant.value, force);
    treeOptions.value = catalog.treeOptions;
    projectOptions.value = catalog.projectOptions;
    return isDefaultTenant.value ? treeOptions.value : projectOptions.value;
  }

  function projectSelections(tenantId = 0) {
    if (!isDefaultTenant.value) return projectOptions.value.map(option => ({ tenant_id: 0, project_id: Number(option.value) })).filter(item => item.project_id > 0);
    const selections: TenantProjectScopeSelection[] = [];
    const collect = (options: TreeBaseTenantProjectResponse_Option[]) => {
      for (const option of options) {
        if (option.type === "project" && option.project_id > 0 && (!tenantId || option.tenant_id === tenantId)) selections.push({ tenant_id: option.tenant_id, project_id: option.project_id });
        if (option.children?.length) collect(option.children);
      }
    };
    collect(treeOptions.value);
    return selections;
  }

  function normalizeParams<T extends TenantProjectScopeParams>(params: T) {
    const result = { ...params } as T & TenantProjectScopeParams;
    const selection = parseTenantProjectValue(params.tenant_project_tree_value);
    result.tenant_id = isDefaultTenant.value ? selection.tenant_id ?? 0 : 0;
    result.project_id = selection.project_id ?? Number(params.project_id ?? 0);
    delete result.tenant_project_tree_value;
    return result;
  }

  const column = computed<ColumnProps>(() => ({
    prop: options.projectField ?? "project_id",
    label: scopeLabel.value,
    minWidth: options.minWidth ?? (isDefaultTenant.value ? 220 : 160),
    showOverflowTooltip: true,
    render: ({ row }) => h(TenantProjectText, {
      row,
      projectId: row[options.projectField ?? "project_id"],
      tenantId: row[options.tenantField ?? "tenant_id"],
      resourceLabelKey
    }),
    search: {
      key: options.searchKey ?? "tenant_project_tree_value",
      order: options.order ?? 1,
      render: (scope: SearchRenderScope) => renderTenantProjectSearch(scope, loadOptions, treeOptions, allLabel.value)
    },
    enum: async () => ({ data: await loadOptions() }),
    isFilterEnum: false
  }));

  return { isDefaultTenant, scopeLabel, allLabel, treeOptions, projectOptions, displayMap, loadOptions, projectSelections, normalizeParams, column };
}

function renderTenantProjectSearch(
  scope: SearchRenderScope,
  loadOptions: (force?: boolean) => Promise<TreeBaseTenantProjectResponse_Option[] | SelectOptionResponse_Option[]>,
  options: Ref<TreeBaseTenantProjectResponse_Option[]>,
  placeholder: string
) {
  const value = scope.searchParam.tenant_project_tree_value;
  return h(TenantProjectSelect, {
    modelValue: value ? { value: String(value) } : undefined,
    options: options.value,
    clearable: scope.clearable,
    filterable: true,
    checkStrictly: true,
    renderAfterExpand: false,
    placeholder: scope.placeholder || placeholder,
    style: { width: "100%" },
    onVisibleChange: (visible: boolean) => { if (visible) void loadOptions(); },
    "onUpdate:modelValue": (selection: TenantProjectNodeSelection | undefined) => { scope.searchParam.tenant_project_tree_value = selection?.value; }
  });
}

/** 创建项目范围列。 */
export function createTenantProjectColumn(scope: TenantProjectScope) {
  return scope.column;
}
