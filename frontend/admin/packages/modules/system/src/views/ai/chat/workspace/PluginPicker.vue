<template>
  <el-select
    :model-value="modelValue.map(pluginSelectionKey)"
    multiple
    clearable
    collapse-tags
    collapse-tags-tooltip
    :disabled="disabled"
    :placeholder="t('system.ai.workspace.plugins')"
    :aria-label="t('system.ai.workspace.plugins')"
    @update:model-value="keys => emit('update:modelValue', keys.map(selectionFromKey))"
  >
    <el-option v-for="option in options" :key="option.key" :value="option.key" :label="option.label" :disabled="!!option.reason">
      <span>{{ option.label }}</span
      ><small v-if="option.reason" class="plugin-reason">{{ option.reason }}</small>
    </el-option>
  </el-select>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { AiPlugin, AiPluginConnection, AiPluginSelection } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";
import { pluginSelectionKey, selectionFromKey } from "./state";
const props = defineProps<{
  modelValue: AiPluginSelection[];
  plugins: AiPlugin[];
  connections: AiPluginConnection[];
  disabled?: boolean;
}>();
const emit = defineEmits<{ "update:modelValue": [value: AiPluginSelection[]] }>();
const options = computed(() => {
  const result = props.plugins.flatMap(plugin => {
    const reason =
      plugin.unavailable_reason ||
      (!plugin.enabled || !plugin.installation?.enabled
        ? t("system.ai.workspace.disabled")
        : !plugin.assembled && plugin.source !== "MCP"
          ? t("system.ai.workspace.not_assembled")
          : "");
    if (plugin.source !== "MCP") return [{ key: `${plugin.id}:`, label: plugin.name, reason }];
    const accounts = props.connections.filter(item => item.plugin_id === plugin.id);
    return accounts.map(account => ({
      key: `${plugin.id}:${account.id}`,
      label: `${plugin.name} · ${account.label}`,
      reason:
        reason || (account.state !== "connected" ? account.last_error || t("system.ai.workspace.connection_unavailable") : "")
    }));
  });
  for (const value of props.modelValue) {
    const key = pluginSelectionKey(value);
    if (!result.some(item => item.key === key))
      result.push({ key, label: key, reason: t("system.ai.workspace.unavailable_selection") });
  }
  return result;
});
</script>
<style scoped>
.plugin-reason {
  display: block;
  color: var(--el-color-warning);
  font-size: 12px;
}
:deep(.el-select-dropdown__item) {
  height: auto;
  min-height: 34px;
  line-height: 22px;
  padding-top: 6px;
  padding-bottom: 6px;
}
</style>
