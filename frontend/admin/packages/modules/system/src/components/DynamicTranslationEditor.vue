<template>
  <div class="translation-editor">
    <div v-for="item in localValues" :key="item.locale" class="translation-editor__row">
      <div class="translation-editor__heading">
        <span>{{ t(`common.language.${item.locale}`) }}</span>
        <el-tag :type="statusTagType(item)" effect="light" size="small">{{ statusLabel(item) }}</el-tag>
      </div>
      <div class="translation-editor__control">
        <el-input
          :model-value="item.text"
          :maxlength="maxlength"
          :type="multiline ? 'textarea' : 'text'"
          :rows="multiline ? 6 : undefined"
          show-word-limit
          :placeholder="t('system.translation.placeholder.text', { language: t(`common.language.${item.locale}`) })"
          @update:model-value="value => updateText(item.locale, value)"
        />
        <el-tooltip
          v-if="draftEnabled && resourceId > 0"
          :content="t('system.translation.action.generateDraft', { language: t(`common.language.${item.locale}`) })"
        >
          <el-button
            :icon="MagicStick"
            :loading="loadingLocale === item.locale"
            :disabled="item.translation_status === TranslationStatus.TRANSLATION_STATUS_REVIEWED && !item.source_changed"
            @click="generateDraft(item.locale)"
          />
        </el-tooltip>
      </div>
    </div>
    <el-alert
      v-if="resourceId === 0 && draftEnabled"
      :title="t('system.translation.message.saveBeforeDraft')"
      type="info"
      :closable="false"
      show-icon
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { MagicStick } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTranslationService } from "@liujitcn/kratos-admin-system/api/system/base_translation";
import {
  BaseConfigTranslationField,
  TranslationResourceType,
  TranslationStatus
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import type { DynamicTranslationValue } from "./dynamicTranslation";

/** DynamicTranslationEditorProps 动态翻译编辑器属性。 */
interface DynamicTranslationEditorProps {
  modelValue: DynamicTranslationValue[];
  resourceId: number;
  resourceType: TranslationResourceType;
  field?: BaseConfigTranslationField;
  draftEnabled?: boolean;
  maxlength?: number;
  multiline?: boolean;
}

const props = withDefaults(defineProps<DynamicTranslationEditorProps>(), {
  draftEnabled: false,
  maxlength: 100,
  multiline: false
});

const emit = defineEmits<{ "update:modelValue": [value: DynamicTranslationValue[]] }>();
const localValues = ref<DynamicTranslationValue[]>([]);
const loadingLocale = ref("");

watch(
  () => props.modelValue,
  value => {
    localValues.value = value.map(item => ({ ...item }));
  },
  { immediate: true, deep: true }
);

/** updateText 更新指定语言的人工译文。 */
function updateText(locale: DynamicTranslationValue["locale"], text: string) {
  const values = localValues.value.map(item => (item.locale === locale ? { ...item, text } : item));
  localValues.value = values;
  emit("update:modelValue", values);
}

/** generateDraft 为指定目标语言生成机器翻译草稿。 */
async function generateDraft(locale: DynamicTranslationValue["locale"]) {
  loadingLocale.value = locale;
  try {
    const response = await defBaseTranslationService.GenerateTranslationDraft({
      resource_type: props.resourceType,
      resource_id: props.resourceId,
      target_locale: locale,
      field: props.field ?? BaseConfigTranslationField.BASE_CONFIG_TRANSLATION_FIELD_UNSPECIFIED
    });
    const values = localValues.value.map(item =>
      item.locale === locale
        ? {
            ...item,
            text: response.translation,
            translation_status: response.translation_status,
            source_changed: false,
            source_hash: response.source_hash,
            translation_provider: response.translation_provider,
            translated_at: response.translated_at
          }
        : item
    );
    localValues.value = values;
    emit("update:modelValue", values);
    ElMessage.success(t("system.translation.message.draftSuccess", { language: t(`common.language.${locale}`) }));
  } finally {
    loadingLocale.value = "";
  }
}

/** statusLabel 返回当前翻译审核状态文案。 */
function statusLabel(item: DynamicTranslationValue) {
  if (item.source_changed) return t("system.translation.status.sourceChanged");
  const statusKey =
    {
      [TranslationStatus.TRANSLATION_STATUS_PENDING]: "pending",
      [TranslationStatus.TRANSLATION_STATUS_MACHINE]: "machine",
      [TranslationStatus.TRANSLATION_STATUS_REVIEWED]: "reviewed"
    }[item.translation_status] ?? "pending";
  return t(`system.translation.status.${statusKey}`);
}

/** statusTagType 返回当前翻译审核状态对应的标签样式。 */
function statusTagType(item: DynamicTranslationValue) {
  if (item.source_changed) return "warning";
  if (item.translation_status === TranslationStatus.TRANSLATION_STATUS_REVIEWED) return "success";
  if (item.translation_status === TranslationStatus.TRANSLATION_STATUS_MACHINE) return "primary";
  return "info";
}
</script>

<style scoped>
.translation-editor {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}
.translation-editor__row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.translation-editor__heading,
.translation-editor__control {
  display: flex;
  gap: 8px;
  align-items: center;
}
.translation-editor__heading {
  justify-content: space-between;
  color: var(--el-text-color-regular);
}
.translation-editor__control :deep(.el-input) {
  flex: 1;
}
</style>
