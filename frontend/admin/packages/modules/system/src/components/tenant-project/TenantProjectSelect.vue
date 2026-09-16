<template>
  <el-tree-select
    v-model="selectedValue"
    :data="treeOptions"
    :props="treeProps"
    node-key="value"
    check-strictly
    filterable
    clearable
    render-after-expand
    v-bind="$attrs"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { defBaseTenantProjectService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project";
import type { TreeBaseTenantProjectResponse_Option } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";

defineOptions({ name: "TenantProjectSelect", inheritAttrs: false });

/** 租户项目选择值。 */
export interface TenantProjectSelection {
  /** 节点唯一值。 */
  value?: string;
  /** 目标租户ID。 */
  tenant_id?: number;
  /** 目标项目ID。 */
  project_id?: number;
  /** 节点类型。 */
  type?: "tenant" | "project";
}

/** 租户项目选择组件属性。 */
interface TenantProjectSelectProps {
  /** 当前选中的租户项目。 */
  modelValue?: TenantProjectSelection;
  /** 外部传入的租户项目树。 */
  options?: TreeBaseTenantProjectResponse_Option[];
}

const props = withDefaults(defineProps<TenantProjectSelectProps>(), {
  modelValue: undefined,
  options: undefined
});
const emit = defineEmits<{
  /** 更新租户项目选择。 */
  (event: "update:modelValue", value: TenantProjectSelection | undefined): void;
  /** 租户项目选择变化。 */
  (event: "change", value: TenantProjectSelection | undefined): void;
}>();

const loadedOptions = ref<TreeBaseTenantProjectResponse_Option[]>([]);
const treeOptions = computed(() => props.options ?? loadedOptions.value);
const treeProps = { label: "label", value: "value", children: "children" };

const selectedValue = computed<string | undefined>({
  get: () => props.modelValue?.value,
  set: value => {
    const selection = parseSelection(value);
    emit("update:modelValue", selection);
    emit("change", selection);
  }
});

onMounted(async () => {
  if (props.options) return;
  const response = await defBaseTenantProjectService.TreeBaseTenantProject({ keyword: "" });
  loadedOptions.value = response.list ?? [];
});

function parseSelection(value?: string): TenantProjectSelection | undefined {
  if (!value) return undefined;
  const [type, tenantId, projectId] = value.split(":");
  if (type === "tenant") {
    return { value, type, tenant_id: Number(tenantId) };
  }
  if (type === "project") {
    return { value, type, tenant_id: Number(tenantId), project_id: Number(projectId) };
  }
  return { value };
}
</script>
