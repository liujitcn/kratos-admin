<template>
  <DynamicI18nEditor
    :model-value="values"
    :source="sourceComment"
    :maxlength="255"
    :disabled="disabled"
    @update:model-value="updateValues"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/DynamicI18nEditor.vue";
import { getEditableLanguageOptions, type DynamicI18nValue } from "@liujitcn/kratos-admin-system/components/dynamicI18n";
import type { CodeGenLocaleConfig } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";

/** 代码生成描述使用统一国际化入口，按字段保存回语言配置。 */
interface CodeGenLocaleEditorProps {
  modelValue: Map<string, CodeGenLocaleConfig>;
  sourceComment?: string;
  field?: keyof CodeGenLocaleConfig;
  disabled?: boolean;
}
const props = withDefaults(defineProps<CodeGenLocaleEditorProps>(), { sourceComment: "", field: "comment", disabled: false });
const emit = defineEmits<{ "update:modelValue": [value: Map<string, CodeGenLocaleConfig>] }>();
const values = computed<DynamicI18nValue[]>(() =>
  getEditableLanguageOptions().map(({ value }) => ({
    locale: value,
    id: 0,
    text: props.modelValue.get(value)?.[props.field] ?? ""
  }))
);

/** 更新当前描述译文，保留其他字段和语言配置。 */
function updateValues(values: DynamicI18nValue[]) {
  const next = new Map(props.modelValue);
  for (const item of values)
    next.set(item.locale, { comment: "", left_tree_comment: "", ...next.get(item.locale), [props.field]: item.text });
  emit("update:modelValue", next);
}
</script>
