<template>
  <DynamicI18nEditor
    v-if="enabled"
    :key="`${targetType}:${targetId}`"
    :model-value="values"
    :source="source"
    :readonly="!editable || !targetType || !targetId || saving"
    :maxlength="10000"
    @update:model-value="saveValues"
  >
    <template #trigger="{ open }">
      <el-link class="dynamic-i18n-cell" type="primary" underline="never" @click="open">
        <span>{{ displaySource || "--" }}</span>
      </el-link>
    </template>
  </DynamicI18nEditor>
  <span v-else>{{ source || "--" }}</span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { defBaseI18nService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_i18n";
import type { BaseI18n, I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import DynamicI18nEditor from "./DynamicI18nEditor.vue";
import { getEditableLanguageOptions, type DynamicI18nValue } from "./dynamicI18n";

/** 列表国际化字段入口，未启用非主语言时退回普通文本。 */
interface DynamicI18nCellProps {
  source: string;
  targetType?: I18nTargetType;
  targetId?: number;
  i18ns?: BaseI18n[];
  editable?: boolean;
}
const props = withDefaults(defineProps<DynamicI18nCellProps>(), { editable: true });
const { locale } = useLocaleStore();
const enabled = computed(() => getEditableLanguageOptions().length > 0);
const overrides = ref(new Map<string, string>());
const saving = ref(false);
let resourceVersion = 0;
const values = computed<DynamicI18nValue[]>(() =>
  getEditableLanguageOptions().map(({ value }) => {
    const record = props.i18ns?.find(item => item.locale === value);
    return { id: record?.id ?? 0, locale: value, text: overrides.value.get(value) ?? record?.name ?? "" };
  })
);
const displaySource = computed(() => values.value.find(item => item.locale === locale.value)?.text || props.source);

/** 仅在确认后保存发生变化的译文，打开窗口不会自动翻译或写库。 */
async function saveValues(next: DynamicI18nValue[]) {
  const targetType = props.targetType;
  const targetId = props.targetId;
  if (!props.editable || !targetType || !targetId || saving.value) return;
  const changed = next.filter(item => item.text !== values.value.find(value => value.locale === item.locale)?.text);
  if (!changed.length) return;
  const version = resourceVersion;
  saving.value = true;
  try {
    const results = await Promise.allSettled(
      changed.map(async item => {
        await defBaseI18nService.UpdateBaseI18n({
          id: item.id,
          target_type: targetType,
          target_id: targetId,
          locale: item.locale,
          name: item.text
        });
        return item;
      })
    );
    if (version !== resourceVersion) return;
    for (const result of results) {
      if (result.status === "fulfilled") overrides.value.set(result.value.locale, result.value.text);
    }
    if (results.some(result => result.status === "rejected")) ElMessage.error(t("common.message.system_error"));
    else ElMessage.success(t("common.message.operation_success"));
  } finally {
    if (version === resourceVersion) saving.value = false;
  }
}

watch(
  () => [props.targetType, props.targetId],
  () => {
    resourceVersion++;
    overrides.value.clear();
    saving.value = false;
  }
);
</script>

<style scoped>
.dynamic-i18n-cell {
  display: inline-flex;
  min-width: 0;
  max-width: 100%;
}
.dynamic-i18n-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
