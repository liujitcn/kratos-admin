<template>
  <el-select
    v-if="isDefaultTenant"
    :model-value="modelValue"
    v-bind="$attrs"
    :loading="isLoading"
    @update:model-value="value => emit('update:modelValue', value)"
    @change="value => emit('change', value)"
  >
    <el-option v-for="option in options" :key="String(option.value)" :label="option.label" :value="option.value" :disabled="option.disabled" />
  </el-select>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import { useTenantScope } from "@/tenant";

defineOptions({ name: "TenantSelect", inheritAttrs: false });

/** 租户选择组件属性。 */
interface TenantSelectProps {
  /** 当前选择的租户。 */
  modelValue?: number | string;
  /** 外部传入的租户选项。 */
  options?: SelectOptionResponse_Option[];
  /** 是否显示加载状态。 */
  loading?: boolean;
}

const props = withDefaults(defineProps<TenantSelectProps>(), {
  modelValue: undefined,
  options: undefined,
  loading: false
});
const emit = defineEmits<{
  /** 更新当前租户。 */
  (event: "update:modelValue", value: number | string | undefined): void;
  /** 租户发生变化。 */
  (event: "change", value: number | string | undefined): void;
}>();

const { isDefaultTenant, tenantOptions, loadTenantOptions } = useTenantScope();
const internalLoading = ref(false);
const options = computed(() => props.options ?? tenantOptions.value);
const isLoading = computed(() => props.loading || internalLoading.value);

onMounted(async () => {
  if (!isDefaultTenant.value || props.options) return;
  internalLoading.value = true;
  try {
    await loadTenantOptions();
  } finally {
    internalLoading.value = false;
  }
});
</script>
