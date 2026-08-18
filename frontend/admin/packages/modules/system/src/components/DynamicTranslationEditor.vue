<template>
  <div class="translation-editor">
    <div class="translation-editor__toolbar">
      <el-button
        type="primary"
        size="small"
        :icon="Promotion"
        :loading="translating"
        :disabled="!canTranslate"
        @click="handleBatchTranslate"
      >
        {{ t("system.base.translation.action.batch_translate") }}
      </el-button>
    </div>
    <div v-for="item in localValues" :key="item.locale" class="translation-editor__row">
      <div class="translation-editor__heading">
        <span>{{ getLanguageLabel(item.locale) }}</span>
      </div>
      <div class="translation-editor__control" :class="{ 'translation-editor__control--multiline': multiline }">
        <el-input
          :model-value="item.text"
          :maxlength="maxlength"
          :type="multiline ? 'textarea' : 'text'"
          :rows="multiline ? 6 : undefined"
          :disabled="translating || translatingLocale === item.locale"
          show-word-limit
          :placeholder="t('system.base.translation.placeholder.text', { language: getLanguageLabel(item.locale) })"
          @update:model-value="value => updateText(item.locale, value)"
        />
        <el-button
          class="translation-editor__translate"
          link
          type="primary"
          size="small"
          :icon="Promotion"
          :loading="translatingLocale === item.locale"
          :disabled="!props.source || Boolean(item.text) || translating || Boolean(translatingLocale)"
          @click="handleTranslate(item.locale)"
        >
          {{ t("system.base.translation.action.translate") }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { Promotion } from "@element-plus/icons-vue";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { defBaseI18nService } from "@liujitcn/kratos-admin-system/api/system/base_i18n";
import { getLanguageLabel, type DynamicTranslationValue } from "./dynamicTranslation";

/** DynamicTranslationEditorProps 动态翻译编辑器属性。 */
interface DynamicTranslationEditorProps {
  modelValue: DynamicTranslationValue[];
  /** 当前语言源文本。 */
  source?: string;
  maxlength?: number;
  multiline?: boolean;
}

const props = withDefaults(defineProps<DynamicTranslationEditorProps>(), {
  source: "",
  maxlength: 100,
  multiline: false
});

const { locale: currentLocale } = useLocaleStore();
const emit = defineEmits<{ "update:modelValue": [value: DynamicTranslationValue[]] }>();
const localValues = ref<DynamicTranslationValue[]>([]);
const translating = ref(false);
const translatingLocale = ref("");
const canTranslate = computed(() => Boolean(props.source) && localValues.value.some(item => !item.text && item.locale !== currentLocale.value));

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

/** handleBatchTranslate 批量翻译当前仍为空的语言输入框。 */
async function handleBatchTranslate() {
  if (!canTranslate.value || translating.value) return;
  translating.value = true;
  try {
    const pending = localValues.value.filter(item => !item.text && item.locale !== currentLocale.value);
    const response = await defBaseI18nService.DraftBaseTranslation({ source: props.source });
    const translations = new Map(response.translations.map(item => [item.locale, item.translation]));
    const failed = pending.filter(item => !translations.has(item.locale)).length;
    if (translations.size) {
      const values = localValues.value.map(item => {
        const translation = translations.get(item.locale);
        return translation === undefined ? item : { ...item, text: translation };
      });
      localValues.value = values;
      emit("update:modelValue", values);
    }
    if (failed) {
      ElMessage.warning(
        t("system.base.translation.message.batch_translate_partial", {
          success: translations.size,
          failed
        })
      );
    } else {
      ElMessage.success(t("system.base.translation.message.batch_translate_success", { count: translations.size }));
    }
  } catch {
    ElMessage.error(t("system.base.translation.message.batch_translate_failed"));
  } finally {
    translating.value = false;
  }
}

/** handleTranslate 翻译指定语言的单个输入框。 */
async function handleTranslate(locale: string) {
  const item = localValues.value.find(value => value.locale === locale);
  if (!item || item.text || item.locale === currentLocale.value || !props.source || translating.value || translatingLocale.value) return;
  translatingLocale.value = locale;
  try {
    const response = await defBaseI18nService.DraftBaseTranslation({ source: props.source, locale });
    const translation = response.translations.find(item => item.locale === locale)?.translation;
    if (!translation) throw new Error("translation not found");
    const values = localValues.value.map(value => (value.locale === locale ? { ...value, text: translation } : value));
    localValues.value = values;
    emit("update:modelValue", values);
    ElMessage.success(t("system.base.translation.message.translate_success", { language: getLanguageLabel(locale) }));
  } catch {
    ElMessage.error(t("system.base.translation.message.translate_failed", { language: getLanguageLabel(locale) }));
  } finally {
    translatingLocale.value = "";
  }
}

</script>

<style scoped>
.translation-editor {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}
.translation-editor__toolbar {
  display: flex;
  justify-content: flex-end;
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
  color: var(--el-text-color-regular);
}
.translation-editor__control {
  position: relative;
  width: 100%;
}
.translation-editor__control :deep(.el-input) {
  flex: 1;
}
.translation-editor__control--multiline :deep(.el-textarea__inner) {
  padding-right: 78px;
}
.translation-editor__control:not(.translation-editor__control--multiline) :deep(.el-input__suffix) {
  padding-right: 78px;
}
.translation-editor__translate {
  position: absolute;
  z-index: 1;
  top: 50%;
  right: 8px;
  transform: translateY(-50%);
}
.translation-editor__control--multiline .translation-editor__translate {
  top: 8px;
  transform: none;
}
</style>
