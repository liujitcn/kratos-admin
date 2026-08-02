<template>
  <div class="code-gen-locale-editor">
    <el-alert
      v-if="missingLocales.length"
      :title="t('system.codegen.i18n.message.missing', { locales: missingLocales.join(', ') })"
      type="warning"
      :closable="false"
      show-icon
    />
    <div v-for="locale in editableLocales" :key="locale.value" class="code-gen-locale-editor__group">
      <div class="code-gen-locale-editor__locale">
        <span>{{ locale.label }}</span>
        <el-tag v-if="hasComment(locale.value)" size="small" type="success" effect="plain">
          {{ t("system.codegen.i18n.status.complete") }}
        </el-tag>
        <el-tag v-else size="small" type="warning" effect="plain">
          {{ t("system.codegen.i18n.status.missing") }}
        </el-tag>
      </div>
      <el-input
        :model-value="localeValue(locale.value).comment"
        :placeholder="t('system.codegen.i18n.placeholder.comment', { source: sourceComment || '-' })"
        maxlength="255"
        @update:model-value="value => updateLocale(locale.value, 'comment', value)"
      />
      <el-input
        v-if="showLeftTreeComment"
        :model-value="localeValue(locale.value).left_tree_comment"
        :placeholder="t('system.codegen.i18n.placeholder.leftTreeComment', { source: sourceLeftTreeComment || '-' })"
        maxlength="255"
        @update:model-value="value => updateLocale(locale.value, 'left_tree_comment', value)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { CodeGenLocaleConfig } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";

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

const editableLocales = computed(() => [
  { value: "en-US", label: t("common.language.en-US") },
  { value: "ja-JP", label: t("common.language.ja-JP") }
]);

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
</style>
