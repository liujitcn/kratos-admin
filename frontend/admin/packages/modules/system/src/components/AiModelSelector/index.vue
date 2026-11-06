<template>
  <el-popover
    ref="modelPopoverRef"
    placement="top-start"
    :width="324"
    trigger="click"
    :show-arrow="false"
    :persistent="false"
    popper-class="agent-model-popper"
  >
    <template #reference>
      <button
        class="agent-model-selector"
        :class="{ 'is-disabled': disabled || !modelOptions.length }"
        type="button"
        :disabled="disabled || !modelOptions.length"
        :aria-label="t('system.ai.model.title')"
      >
        <el-icon class="agent-model-selector__spark"><MagicStick /></el-icon>
        <span class="agent-model-selector__label">{{ selectedModelDisplayName || t("system.ai.model.model_placeholder") }}</span>
        <el-icon class="agent-model-selector__arrow"><ArrowDown /></el-icon>
      </button>
    </template>
    <div v-if="modelOptions.length" class="agent-model-panel">
      <div class="agent-model-panel__head">
        <span class="agent-model-panel__title">{{ t("system.ai.model.title") }}</span>
        <span class="agent-model-panel__meta">{{
          t("system.ai.model.provider_model_count", {
            providers: modelOptions.length,
            models: modelOptionCount
          })
        }}</span>
      </div>
      <div class="agent-model-panel__list">
        <template v-for="provider in modelOptions" :key="provider.providerId">
          <div class="agent-model-panel__group">{{ provider.label }}</div>
          <button
            v-for="model in provider.models"
            :key="model.name"
            class="agent-model-row"
            :disabled="disabled"
            :class="{ 'is-selected': isModelSelected(provider.providerId, model.name) }"
            type="button"
            @click="handleModelSelect(provider.providerId, model.name)"
          >
            <span class="agent-model-row__name">{{ model.displayName }}</span>
            <span v-if="model.displayName !== model.name" class="agent-model-row__raw">{{ model.name }}</span>
            <el-icon v-if="isModelSelected(provider.providerId, model.name)" class="agent-model-row__check">
              <Check />
            </el-icon>
          </button>
        </template>
      </div>
      <button v-if="aiProviderRoute" class="agent-model-panel__manage" type="button" @click="openModelManage">
        <el-icon><Setting /></el-icon>
        <span>{{ t("system.ai.model.manage") }}</span>
        <el-icon class="agent-model-panel__manage-arrow"><ArrowRight /></el-icon>
      </button>
    </div>
    <div v-else class="agent-model-panel agent-model-panel--empty">
      {{ t("system.ai.model.empty") }}
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import type { PopoverInstance } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { useAuthStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import { fallbackModelDisplayName } from "./modelDisplay";

const props = withDefaults(
  defineProps<{
    /** 已启用的供应商及模型。 */
    modelProviders: AiProviderModelOption[];
    /** 当前供应商编号。 */
    providerId: number;
    /** 当前模型名称。 */
    modelName: string;
    /** 选择器是否禁用。 */
    disabled?: boolean;
  }>(),
  { disabled: false }
);

const emit = defineEmits<{
  /** 更新当前供应商与模型。 */
  change: [providerId: number, modelName: string];
}>();
const modelPopoverRef = ref<PopoverInstance>();
const router = useRouter();
const authStore = useAuthStore();
const modelOptions = computed(() =>
  props.modelProviders.map(provider => ({
    providerId: provider.provider_id,
    label: provider.name,
    models: provider.models.map(model => ({
      name: model.model_name,
      displayName: model.display_name || fallbackModelDisplayName(model.model_name)
    }))
  }))
);
const modelOptionCount = computed(() => modelOptions.value.reduce((total, provider) => total + provider.models.length, 0));
const selectedModelDisplayName = computed(() => {
  const provider = modelOptions.value.find(item => item.providerId === props.providerId);
  return provider?.models.find(model => model.name === props.modelName)?.displayName ?? fallbackModelDisplayName(props.modelName);
});
const aiProviderRoute = computed(() => authStore.flatMenuListGet.find(item => item.name === "AiProvider" && item.path));

/** 判断指定供应商与模型是否为当前选择。 */
function isModelSelected(providerId: number, modelName: string) {
  return props.providerId === providerId && props.modelName === modelName;
}

/** 选中模型并关闭选择面板。 */
function handleModelSelect(providerId: number, modelName: string) {
  emit("change", providerId, modelName);
  modelPopoverRef.value?.hide();
}

/** 跳转到 AI 供应商配置页。 */
async function openModelManage() {
  if (!aiProviderRoute.value?.path) return;
  modelPopoverRef.value?.hide();
  await navigateTo(router, aiProviderRoute.value.path);
}
</script>

<style lang="scss" src="./styles.scss"></style>
