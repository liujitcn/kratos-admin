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
        <span class="dynamic-translation-cell__source">{{ source || "--" }}</span>
        <el-tooltip v-if="editable" :content="t('common.action.edit')" placement="top">
          <el-button
            class="dynamic-translation-cell__edit"
            type="primary"
            text
            size="small"
            :icon="EditPen"
            :aria-label="t('common.action.edit')"
            @click.stop="emit('edit')"
          />
        </el-tooltip>
      </span>
    </template>

    <div class="dynamic-translation-cell__rows">
      <div v-for="item in translationRows" :key="item.locale" class="dynamic-translation-cell__row">
        <span class="dynamic-translation-cell__language">{{ t(`common.language.${item.locale}`) }}</span>
        <span class="dynamic-translation-cell__text">{{ item.text || t('system.common.value.none') }}</span>
        <el-tag v-if="showStatus" :type="statusTagType(item)" effect="light" size="small">
          {{ statusLabel(item) }}
        </el-tag>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EditPen } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { TranslationStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";

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
  /** 中文源文。 */
  source: string;
  /** 英语和日语翻译记录。 */
  translations?: DynamicTranslationCellRecord[];
  /** 从翻译记录中读取的文本字段。 */
  textField?: "text" | "name" | "label" | "title";
  /** 是否显示直接编辑按钮。 */
  editable?: boolean;
  /** 是否显示翻译状态标签。 */
  showStatus?: boolean;
}

const props = withDefaults(defineProps<DynamicTranslationCellProps>(), {
  translations: () => [],
  textField: "text",
  editable: false,
  showStatus: true
});

const emit = defineEmits<{ edit: [] }>();

const translationRows = computed(() =>
  (["en-US", "ja-JP"] as const).map(locale => {
    const record = props.translations.find(item => item.locale === locale);
    return {
      locale,
      text: record ? getTranslationText(record) : "",
      translation_status: record?.translation_status ?? TranslationStatus.TRANSLATION_STATUS_PENDING,
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
    }[item.translation_status] ?? "pending";
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

.dynamic-translation-cell__edit {
  flex: none;
  margin-left: 2px;
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
