<template>
  <div class="i18n-editor">
    <div class="i18n-editor__toolbar">
      <el-button
        type="primary"
        size="small"
        :icon="Promotion"
        :loading="translating"
        :disabled="!canTranslate"
        @click="handleBatchTranslate"
      >
        {{ t("system.base.i18n.action.batch_translate") }}
      </el-button>
    </div>
    <div v-for="item in localValues" :key="item.locale" class="i18n-editor__row">
      <div class="i18n-editor__heading">
        <span>{{ getLanguageLabel(item.locale) }}</span>
      </div>
      <div class="i18n-editor__control" :class="{ 'i18n-editor__control--multiline': multiline }">
        <el-input
          :model-value="item.text"
          :maxlength="maxlength"
          :type="multiline ? 'textarea' : 'text'"
          :rows="multiline ? 6 : undefined"
          :disabled="translating || translatingLocale === item.locale"
          show-word-limit
          :placeholder="t('system.base.i18n.placeholder.text', { language: getLanguageLabel(item.locale) })"
          @update:model-value="value => updateText(item.locale, value)"
        />
        <el-button
          class="i18n-editor__translate"
          link
          type="primary"
          size="small"
          :icon="Promotion"
          :loading="translatingLocale === item.locale"
          :disabled="!props.source || Boolean(item.text) || translating || Boolean(translatingLocale)"
          @click="handleTranslate(item.locale)"
        >
          {{ t("system.base.i18n.action.translate") }}
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
import { getLanguageLabel, type DynamicI18nValue } from "./dynamicI18n";

/** DynamicI18nEditorProps 动态翻译编辑器属性。 */
interface DynamicI18nEditorProps {
  modelValue: DynamicI18nValue[];
  /** 当前语言源文本。 */
  source?: string;
  maxlength?: number;
  multiline?: boolean;
}

const props = withDefaults(defineProps<DynamicI18nEditorProps>(), {
  source: "",
  maxlength: 100,
  multiline: false
});

const { locale: currentLocale } = useLocaleStore();
const emit = defineEmits<{ "update:modelValue": [value: DynamicI18nValue[]] }>();
const localValues = ref<DynamicI18nValue[]>([]);
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
function updateText(locale: DynamicI18nValue["locale"], text: string) {
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
    const response = await defBaseI18nService.DraftBaseI18n({ source: props.source });
    const i18ns = new Map(response.i18ns.map(item => [item.locale, item.i18n]));
    const failed = pending.filter(item => !i18ns.has(item.locale)).length;
    if (i18ns.size) {
      const values = localValues.value.map(item => {
        const i18n = i18ns.get(item.locale);
        return i18n === undefined ? item : { ...item, text: i18n };
      });
      localValues.value = values;
      emit("update:modelValue", values);
    }
    if (failed) {
      ElMessage.warning(
        t("system.base.i18n.message.batch_translate_partial", {
          success: i18ns.size,
          failed
        })
      );
    } else {
      ElMessage.success(t("system.base.i18n.message.batch_translate_success", { count: i18ns.size }));
    }
  } catch {
    ElMessage.error(t("system.base.i18n.message.batch_translate_failed"));
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
    const response = await defBaseI18nService.DraftBaseI18n({ source: props.source, locale });
    const i18n = response.i18ns.find(item => item.locale === locale)?.i18n;
    if (!i18n) throw new Error("i18n not found");
    const values = localValues.value.map(value => (value.locale === locale ? { ...value, text: i18n } : value));
    localValues.value = values;
    emit("update:modelValue", values);
    ElMessage.success(t("system.base.i18n.message.translate_success", { language: getLanguageLabel(locale) }));
  } catch {
    ElMessage.error(t("system.base.i18n.message.translate_failed", { language: getLanguageLabel(locale) }));
  } finally {
    translatingLocale.value = "";
  }
}

</script>

<style scoped>
.i18n-editor {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}
.i18n-editor__toolbar {
  display: flex;
  justify-content: flex-end;
}
.i18n-editor__row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.i18n-editor__heading,
.i18n-editor__control {
  display: flex;
  gap: 8px;
  align-items: center;
}
.i18n-editor__heading {
  color: var(--el-text-color-regular);
}
.i18n-editor__control {
  position: relative;
  width: 100%;
}
.i18n-editor__control :deep(.el-input) {
  flex: 1;
}
.i18n-editor__control--multiline :deep(.el-textarea__inner) {
  padding-right: 78px;
}
.i18n-editor__control:not(.i18n-editor__control--multiline) :deep(.el-input__suffix) {
  padding-right: 78px;
}
.i18n-editor__translate {
  position: absolute;
  z-index: 1;
  top: 50%;
  right: 8px;
  transform: translateY(-50%);
}
.i18n-editor__control--multiline .i18n-editor__translate {
  top: 8px;
  transform: none;
}
</style>
