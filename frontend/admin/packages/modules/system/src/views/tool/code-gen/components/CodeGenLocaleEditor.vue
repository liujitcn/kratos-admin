<template>
  <div class="code-gen-locale-editor">
    <el-alert
      v-if="missingLocales.length"
      :title="t('system.code.gen.i18n.message.missing', { locales: missingLocales.join(', ') })"
      type="warning"
      :closable="false"
      show-icon
    />
    <div v-for="locale in editableLocales" :key="locale.value" class="code-gen-locale-editor__group">
      <div class="code-gen-locale-editor__locale">
        <span>{{ locale.label }}</span>
        <el-tag v-if="hasComment(locale.value)" size="small" type="success" effect="plain">
          {{ t("system.code.gen.i18n.status.complete") }}
        </el-tag>
        <el-tag v-else size="small" type="warning" effect="plain">
          {{ t("system.code.gen.i18n.status.missing") }}
        </el-tag>
      </div>
      <div class="code-gen-locale-editor__control">
        <el-input
          :model-value="localeValue(locale.value).comment"
          :placeholder="t('system.code.gen.i18n.placeholder.comment', { source: sourceComment || '-' })"
          :disabled="isTranslating(locale.value, 'comment')"
          maxlength="255"
          @update:model-value="value => updateLocale(locale.value, 'comment', value)"
        />
        <el-button
          class="code-gen-locale-editor__translate"
          link
          type="primary"
          size="small"
          :icon="Promotion"
          :loading="isTranslating(locale.value, 'comment')"
          :disabled="!canTranslate(locale.value, 'comment', sourceComment)"
          @click="handleTranslate(locale.value, 'comment', sourceComment, locale.label)"
        >
          {{ t("system.base.translation.action.translate") }}
        </el-button>
      </div>
      <div v-if="showLeftTreeComment" class="code-gen-locale-editor__control">
        <el-input
          :model-value="localeValue(locale.value).left_tree_comment"
          :placeholder="t('system.code.gen.i18n.placeholder.left_tree_comment', { source: sourceLeftTreeComment || '-' })"
          :disabled="isTranslating(locale.value, 'left_tree_comment')"
          maxlength="255"
          @update:model-value="value => updateLocale(locale.value, 'left_tree_comment', value)"
        />
        <el-button
          class="code-gen-locale-editor__translate"
          link
          type="primary"
          size="small"
          :icon="Promotion"
          :loading="isTranslating(locale.value, 'left_tree_comment')"
          :disabled="!canTranslate(locale.value, 'left_tree_comment', sourceLeftTreeComment)"
          @click="handleTranslate(locale.value, 'left_tree_comment', sourceLeftTreeComment, locale.label)"
        >
          {{ t("system.base.translation.action.translate") }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Promotion } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseI18nService } from "@liujitcn/kratos-admin-system/api/system/base_i18n";
import { loadEnabledBaseLanguages, useEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";
import type { CodeGenLocaleConfig } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

/** 代码生成多语言编辑器属性。 */
interface CodeGenLocaleEditorProps {
  /** 按语言区域索引的生成配置。 */
  modelValue: Map<string, CodeGenLocaleConfig>;
  /** 中文业务或字段描述。 */
  sourceComment?: string;
  /** 是否编辑左树标题。 */
  showLeftTreeComment?: boolean;
  /** 中文左树标题。 */
  sourceLeftTreeComment?: string;
}

const props = withDefaults(defineProps<CodeGenLocaleEditorProps>(), {
  sourceComment: "",
  showLeftTreeComment: false,
  sourceLeftTreeComment: ""
});

const emit = defineEmits<{
  "update:modelValue": [value: Map<string, CodeGenLocaleConfig>];
}>();

const { languages } = useEnabledBaseLanguages();
const translatingField = ref("");
void loadEnabledBaseLanguages();

const editableLocales = computed(() =>
  languages.value
    .filter(item => !item.is_primary)
    .map(item => ({ value: item.language_code, label: item.native_name || item.language_name || item.language_code }))
);

const missingLocales = computed(() =>
  editableLocales.value.filter(locale => !hasComment(locale.value)).map(locale => locale.label)
);

/** 返回指定语言配置并补齐空字段。 */
function localeValue(locale: string): CodeGenLocaleConfig {
  return props.modelValue?.get(locale) ?? { comment: "", left_tree_comment: "" };
}

/** 判断指定语言是否已填写业务或字段描述。 */
function hasComment(locale: string) {
  const value = localeValue(locale);
  return Boolean(value.comment && (!props.showLeftTreeComment || value.left_tree_comment));
}

/** 更新单个语言字段并保持 Map 引用语义。 */
function updateLocale(locale: string, field: keyof CodeGenLocaleConfig, value: string) {
  const next = new Map(props.modelValue ?? []);
  next.set(locale, { ...localeValue(locale), [field]: value });
  emit("update:modelValue", next);
}

/** 判断指定语言字段是否正在翻译。 */
function isTranslating(locale: string, field: keyof CodeGenLocaleConfig) {
  return translatingField.value === `${locale}:${field}`;
}

/** 判断指定语言字段当前是否允许自动翻译。 */
function canTranslate(locale: string, field: keyof CodeGenLocaleConfig, source: string) {
  return Boolean(source) && !localeValue(locale)[field] && !translatingField.value;
}

/** 翻译指定语言的单个代码生成描述。 */
async function handleTranslate(locale: string, field: keyof CodeGenLocaleConfig, source: string, language: string) {
  if (!canTranslate(locale, field, source)) return;
  translatingField.value = `${locale}:${field}`;
  try {
    const response = await defBaseI18nService.DraftBaseTranslation({ source, locale });
    const translation = response.translations.find(item => item.locale === locale)?.translation;
    if (!translation) throw new Error("translation not found");
    updateLocale(locale, field, translation);
    ElMessage.success(t("system.base.translation.message.translate_success", { language }));
  } catch {
    ElMessage.error(t("system.base.translation.message.translate_failed", { language }));
  } finally {
    translatingField.value = "";
  }
}
</script>

<style scoped lang="scss">
.code-gen-locale-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.code-gen-locale-editor__group {
  display: grid;
  gap: 8px;
}

.code-gen-locale-editor__locale {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 24px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.code-gen-locale-editor__control {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}

.code-gen-locale-editor__control :deep(.el-input) {
  flex: 1;
}

.code-gen-locale-editor__control :deep(.el-input__suffix) {
  padding-right: 78px;
}

.code-gen-locale-editor__translate {
  position: absolute;
  z-index: 1;
  top: 50%;
  right: 8px;
  transform: translateY(-50%);
}
</style>
