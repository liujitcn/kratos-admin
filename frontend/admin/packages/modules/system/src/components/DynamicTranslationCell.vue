<template>
  <el-popover
    placement="top-start"
    trigger="hover"
    :show-after="120"
    :width="360"
    :enterable="true"
    :teleported="true"
    popper-class="dynamic-translation-cell__popover"
  >
    <template #reference>
      <span class="dynamic-translation-cell">
        <span class="dynamic-translation-cell__source">{{ displaySource || "--" }}</span>
      </span>
    </template>

    <div class="dynamic-translation-cell__rows">
      <div v-for="item in translationRows" :key="item.locale" class="dynamic-translation-cell__row">
        <span class="dynamic-translation-cell__language">{{ getLanguageLabel(item.locale) }}</span>
        <span class="dynamic-translation-cell__text">{{ item.text || t('system.common.value.none') }}</span>
        <el-tag v-if="showStatus && !item.isSource" :type="statusTagType(item)" effect="light" size="small">
          {{ statusLabel(item) }}
        </el-tag>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { useEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";
import { TranslationStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import { getLanguageLabel } from "./dynamicTranslation";

/** DynamicTranslationCellRecord 描述列表翻译预览所需的最小记录结构。 */
export interface DynamicTranslationCellRecord {
  /** 语言区域。 */
  locale: string;
  /** 通用翻译文本，供系统配置使用。 */
  text?: string;
  /** 字典名称翻译。 */
  name?: string;
  /** 字典项标签翻译。 */
  label?: string;
  /** 菜单标题翻译。 */
  title?: string;
  /** 翻译审核状态。 */
  translation_status?: TranslationStatus;
  /** 中文源文是否已变化。 */
  source_changed?: boolean;
}

/** DynamicTranslationCellProps 动态翻译列表单元格属性。 */
interface DynamicTranslationCellProps {
  /** 资源源文。 */
  source: string;
  /** 非主语言翻译记录。 */
  translations?: DynamicTranslationCellRecord[];
  /** 源文所属语言，仅用于语言配置尚未加载时的回退。 */
  sourceLocale?: string;
  /** 从翻译记录中读取的文本字段。 */
  textField?: "text" | "name" | "label" | "title";
  /** 是否显示翻译状态标签。 */
  showStatus?: boolean;
}

const props = withDefaults(defineProps<DynamicTranslationCellProps>(), {
  translations: () => [],
  sourceLocale: "zh-CN",
  textField: "text",
  showStatus: true
});

const { locale } = useLocaleStore();
const { languages } = useEnabledBaseLanguages();
const primaryLocale = computed(() => languages.value.find(item => item.is_primary)?.language_code ?? props.sourceLocale ?? "zh-CN");

const displaySource = computed(() => {
  if (locale.value === primaryLocale.value) return props.source;
  const record = props.translations.find(item => item.locale === locale.value);
  if (record?.translation_status !== TranslationStatus.TRANSLATION_STATUS_REVIEWED) return props.source;
  return getTranslationText(record) || props.source;
});

const translationRows = computed(() =>
  languages.value
    .filter(item => item.language_code !== locale.value)
    .map(item => {
      const itemLocale = item.language_code;
      const isSource = itemLocale === primaryLocale.value;
      const record = props.translations.find(item => item.locale === itemLocale);
      const text = isSource ? props.source : record ? getTranslationText(record) : "";
      return {
        locale: itemLocale,
        text,
        isSource,
        translation_status: record?.translation_status,
        source_changed: record?.source_changed ?? false
      };
    })
);

/** getTranslationText 读取当前资源对应的翻译字段。 */
function getTranslationText(record: DynamicTranslationCellRecord) {
  return record[props.textField] ?? "";
}

/** statusLabel 返回当前翻译审核状态文案。 */
function statusLabel(item: (typeof translationRows.value)[number]) {
  if (item.source_changed) return t("system.translation.status.sourceChanged");
  const statusKey =
    {
      [TranslationStatus.TRANSLATION_STATUS_PENDING]: "pending",
      [TranslationStatus.TRANSLATION_STATUS_MACHINE]: "machine",
      [TranslationStatus.TRANSLATION_STATUS_REVIEWED]: "reviewed"
    }[item.translation_status ?? TranslationStatus.TRANSLATION_STATUS_PENDING] ?? "pending";
  return t(`system.translation.status.${statusKey}`);
}

/** statusTagType 返回当前翻译审核状态对应的标签样式。 */
function statusTagType(item: (typeof translationRows.value)[number]) {
  if (item.source_changed) return "warning";
  if (item.translation_status === TranslationStatus.TRANSLATION_STATUS_REVIEWED) return "success";
  if (item.translation_status === TranslationStatus.TRANSLATION_STATUS_MACHINE) return "primary";
  return "info";
}
</script>

<style scoped>
.dynamic-translation-cell {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 100%;
}

.dynamic-translation-cell__source {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

</style>

<style>
.dynamic-translation-cell__popover .dynamic-translation-cell__rows {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dynamic-translation-cell__popover {
  max-width: calc(100vw - 24px);
  box-sizing: border-box;
}

.dynamic-translation-cell__popover .dynamic-translation-cell__row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.dynamic-translation-cell__popover .dynamic-translation-cell__language {
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.dynamic-translation-cell__popover .dynamic-translation-cell__text {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}
</style>
