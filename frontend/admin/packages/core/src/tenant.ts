import { computed, ref, type ComputedRef, type Ref } from "vue";
import type { ProFormField } from "@/components/ProForm/interface";
import type { ColumnProps, EnumProps, RenderScope, SearchType } from "@/components/ProTable/interface";
import { defBaseTenantService } from "@/api/system/admin/v1/base_tenant";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import { useUserStore } from "@/stores/runtime";
import { DEFAULT_TENANT_CODE, requestTenantOptions as requestTenantOptionData } from "./utils/tenant";

export * from "./utils/tenant";

/** 查询租户编码筛选选项，值为租户编码而不是租户 ID。 */
export async function requestTenantCodeOptions() {
  const options: EnumProps[] = [];
  const pageSize = 100;
  for (let pageNum = 1; ; pageNum += 1) {
    const response = await defBaseTenantService.PageBaseTenant({ code: "", name: "", page_num: pageNum, page_size: pageSize });
    const tenants = response.base_tenants ?? [];
    options.push(...tenants.map(tenant => ({ value: tenant.code, label: tenant.name })));
    if (options.length >= response.total || tenants.length === 0) break;
  }
  return { data: options };
}

/** 租户上下文状态。 */
export interface TenantScope {
  /** 当前账号是否为默认租户。 */
  isDefaultTenant: ComputedRef<boolean>;
  /** 当前账号可用的租户选项。 */
  tenantOptions: Ref<SelectOptionResponse_Option[]>;
  /** 租户选项名称映射。 */
  tenantNameMap: ComputedRef<Map<string, string>>;
  /** 加载默认租户可选择的租户。 */
  loadTenantOptions: (includeCurrentTenant?: boolean) => Promise<void>;
  /** 将页面租户筛选值转换成后端请求值。 */
  toRequestTenantId: (value: unknown) => number | undefined;
  /** 创建默认租户可见的表单字段。 */
  tenantFormField: (options: TenantFormFieldOptions) => ProFormField;
  /** 创建默认租户可见的表格租户列。 */
  tenantColumns: (options: TenantColumnOptions) => ColumnProps[];
  /** 将行数据转换成租户名称。 */
  resolveTenantLabel: (row: Record<string, any>, field?: string, options?: SelectOptionResponse_Option[]) => string;
}

/** 租户表单字段配置。 */
export interface TenantFormFieldOptions {
  /** 字段绑定名称。 */
  prop?: string;
  /** 字段标签。 */
  label: string;
  /** 表单控件附加属性。 */
  props?: Record<string, any>;
  /** 是否允许编辑态修改租户。 */
  disabledOnEdit?: boolean;
}

/** 租户表格列配置。 */
export interface TenantColumnOptions {
  /** 列绑定字段。 */
  prop?: string;
  /** 列标题。 */
  label: string;
  /** 查询字段排序。 */
  order?: number;
  /** 查询参数字段，默认使用列字段。 */
  searchKey?: string;
  /** 列最小宽度。 */
  minWidth?: number | string;
  /** 是否在表格中显示，隐藏时仍可保留查询项。 */
  isShow?: boolean;
  /** 是否允许在列设置中切换。 */
  isSetting?: boolean;
  /** 是否带租户查询项。 */
  searchable?: boolean;
  /** 自定义租户选项来源。 */
  enum?: ColumnProps["enum"];
  /** 查询控件类型。 */
  searchEl?: SearchType;
  /** 自定义单元格展示。 */
  render?: (scope: RenderScope) => ReturnType<NonNullable<ColumnProps["render"]>>;
}

/** 创建租户范围能力，统一默认租户和普通租户的前端行为。 */
export function useTenantScope(): TenantScope {
  const userStore = useUserStore();
  const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);
  const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
  const tenantNameMap = computed(() => new Map(tenantOptions.value.map(item => [String(item.value), item.label] as const)));

  async function loadTenantOptions(includeCurrentTenant = false) {
    if (!includeCurrentTenant && !isDefaultTenant.value) return;
    if (tenantOptions.value.length) return;
    const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
    tenantOptions.value = response.list ?? [];
  }

  function toRequestTenantId(value: unknown) {
    if (!isDefaultTenant.value || value === undefined || value === null || value === "") return undefined;
    const tenantId = Number(value);
    return Number.isFinite(tenantId) && tenantId > 0 ? tenantId : undefined;
  }

  function tenantFormField(options: TenantFormFieldOptions): ProFormField {
    const prop = options.prop ?? "tenant_id";
    return {
      prop,
      label: options.label,
      component: "tenant-select",
      props: (model: Record<string, any>) => ({
        filterable: true,
        ...(options.disabledOnEdit ? { disabled: Boolean(model.id) } : {}),
        ...options.props
      }),
      visible: () => isDefaultTenant.value,
    };
  }

  function tenantColumns(options: TenantColumnOptions): ColumnProps[] {
    if (!isDefaultTenant.value) return [];
    const prop = options.prop ?? "tenant_id";
    return [
      {
        prop,
        label: options.label,
        minWidth: options.minWidth ?? 140,
        isShow: options.isShow,
        isSetting: options.isSetting,
        align: "left",
        showOverflowTooltip: true,
        search:
          options.searchable === false
            ? undefined
            : { el: options.searchEl ?? "tenant-select", key: options.searchKey ?? prop, props: { filterable: true }, order: options.order },
        enum: options.enum ?? requestTenantOptionData,
        render: options.render ?? (scope => resolveTenantLabel(scope.row, prop))
      }
    ];
  }

  function resolveTenantLabel(row: Record<string, any>, field = "tenant_id", options = tenantOptions.value) {
    const directLabel = row.tenant_name ?? row[`${field}_name`];
    if (directLabel) return String(directLabel);
    const value = row[field];
    if (value === undefined || value === null || value === "") return "";
    const nameMap = new Map(options.map(item => [String(item.value), item.label] as const));
    return nameMap.get(String(value)) ?? String(value);
  }

  return {
    isDefaultTenant,
    tenantOptions,
    tenantNameMap,
    loadTenantOptions,
    toRequestTenantId,
    tenantFormField,
    tenantColumns,
    resolveTenantLabel
  };
}
