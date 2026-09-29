<template>
  <el-tree-select
    v-model="selectedValue"
    :data="treeOptions"
    :props="treeProps"
    node-key="value"
    check-strictly
    filterable
    clearable
    v-bind="treeSelectAttrs"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useAttrs } from "vue";
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

const attrs = useAttrs();
const loadedOptions = ref<TreeBaseTenantProjectResponse_Option[]>([]);
const treeOptions = computed(() => props.options ?? loadedOptions.value);
const treeProps = { label: "label", value: "value", children: "children" };

/** 消费方传入的 renderAfterExpand=false 会清空 el-tree-select 的选中标签缓存，导致已选值显示成节点原始值，这里统一忽略并保持标签缓存可用。 */
const treeSelectAttrs = computed(() => {
  const { renderAfterExpand: _ignored, ...rest } = attrs;
  return { ...rest, renderAfterExpand: true };
});

const selectedValue = computed<string | undefined>({
  get: () => resolveSelectedKey(props.modelValue),
  set: value => {
    const selection = parseSelection(value);
    emit("update:modelValue", selection);
    emit("change", selection);
  }
});

/** 解析选中的树节点值；容错误包了一层对象或非字符串的值，避免选中后显示 [object Object]。 */
function resolveSelectedKey(modelValue?: TenantProjectSelection): string | undefined {
  const raw = modelValue?.value as unknown;
  if (typeof raw === "string") return raw || undefined;
  if (raw && typeof raw === "object") {
    const nested = (raw as { value?: unknown }).value;
    if (typeof nested === "string") return nested || undefined;
  }
  return undefined;
}

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
