<template>
  <span>{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-core/rpc/common/v1/common";

defineOptions({ name: "TenantProjectText" });

/** 租户项目展示组件属性。 */
interface TenantProjectTextProps {
  /** 包含租户和项目字段的列表行。 */
  row: Record<string, any>;
  /** 项目名称字段。 */
  projectField?: string;
  /** 租户名称字段。 */
  tenantField?: string;
  /** 租户选项，用于只有租户ID时解析名称。 */
  tenantOptions?: SelectOptionResponse_Option[];
}

const props = withDefaults(defineProps<TenantProjectTextProps>(), {
  projectField: "project_name",
  tenantField: "tenant_name",
  tenantOptions: () => []
});
const { isDefaultTenant, resolveTenantLabel } = useTenantScope();

const label = computed(() => {
  const projectName = props.row[props.projectField] ?? props.row.project_name ?? props.row.name ?? props.row.project_id ?? "";
  if (!isDefaultTenant.value) return String(projectName);
  const tenantName = props.row[props.tenantField] ?? resolveTenantLabel(props.row, "tenant_id", props.tenantOptions);
  return tenantName ? `${tenantName} / ${projectName}` : String(projectName);
});
</script>
