<template>
  <span>{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import { useTenantScope } from "@/tenant";

defineOptions({ name: "TenantText" });

/** 租户展示组件属性。 */
interface TenantTextProps {
  /** 租户ID。 */
  tenantId?: number | string;
  /** 已知的租户名称。 */
  tenantName?: string;
  /** 外部传入的租户选项。 */
  options?: SelectOptionResponse_Option[];
}

const props = defineProps<TenantTextProps>();
const { resolveTenantLabel } = useTenantScope();
const label = computed(() => {
  if (props.tenantName) return props.tenantName;
  return resolveTenantLabel({ tenant_id: props.tenantId }, "tenant_id", props.options);
});
</script>
