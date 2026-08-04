<template>
  <el-link class="dynamic-translation-cell" type="primary" :underline="false" @click="openDialog">
    <span class="dynamic-translation-cell__source">{{ displaySource || "--" }}</span>
  </el-link>

  <ProDialog
    v-model="dialogVisible"
    :title="displaySource"
    width="760px"
    append-to-body
    :destroy-on-close="true"
    @closed="handleDialogClosed"
  >
    <el-table v-loading="loading" :data="translationRows" border>
      <el-table-column :label="t('common.field.language')" width="150">
        <template #default="{ row }">
          {{ getLanguageLabel(row.locale) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('system.base.translation.field.translations')" min-width="360">
        <template #default="{ row }">
          <el-input
            v-if="row.editing"
            v-model="row.text"
            :maxlength="2000"
            show-word-limit
            :placeholder="t('system.base.translation.placeholder.text', { language: getLanguageLabel(row.locale) })"
          />
          <span v-else class="dynamic-translation-dialog__text">
            {{ row.text || t("common.value.none") }}
          </span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.field.operation')" width="150" fixed="right">
        <template #default="{ row }">
          <el-button v-if="!row.editing" link type="primary" :icon="EditPen" @click="startEdit(row)">
            {{ t("common.action.edit") }}
          </el-button>
          <template v-else>
            <el-button link type="primary" :icon="Check" :loading="row.saving" @click="saveTranslation(row)">
              {{ t("common.action.save") }}
            </el-button>
            <el-button link :icon="Close" :disabled="row.saving" @click="cancelEdit(row)">
              {{ t("common.action.cancel") }}
            </el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <template #footer>
      <el-button @click="dialogVisible = false">{{ t("common.action.cancel") }}</el-button>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { Check, Close, EditPen } from "@element-plus/icons-vue";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { loadEnabledBaseLanguages, useEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";
import { defBaseTranslationService } from "@liujitcn/kratos-admin-system/api/system/base_translation";
import type { BaseTranslation } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import { TranslationTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import { getLanguageLabel } from "./dynamicTranslation";

/** DynamicTranslationCellProps 动态翻译列表单元格属性。 */
interface DynamicTranslationCellProps {
  /** 资源源文。 */
  source: string;
  /** 统一翻译表目标类型。 */
  targetType?: TranslationTargetType;
  /** 目标资源编号。 */
  targetId?: number;
  /** 资源列表返回的非主语言翻译。 */
  translations?: BaseTranslation[];
}

const props = defineProps<DynamicTranslationCellProps>();

const { languages } = useEnabledBaseLanguages();
const { locale } = useLocaleStore();
const dialogVisible = ref(false);
const loading = ref(false);
const translationRows = ref<DynamicTranslationRow[]>([]);
const translationOverrides = ref(new Map<string, string>());

const displaySource = computed(() => {
  const override = translationOverrides.value.get(locale.value);
  if (override) return override;
  const translation = props.translations?.find(item => item.locale === locale.value && item.name);
  return translation?.name || props.source;
});

watch(
  () => [props.targetType, props.targetId],
  () => {
    translationRows.value = [];
    translationOverrides.value.clear();
    dialogVisible.value = false;
  }
);

/** DynamicTranslationRow 描述弹窗内可编辑的单语言翻译行。 */
interface DynamicTranslationRow {
  /** 翻译记录主键，新增翻译时为空。 */
  id: number;
  /** 语言区域。 */
  locale: string;
  /** 翻译文本。 */
  text: string;
  /** 是否处于编辑状态。 */
  editing: boolean;
  /** 是否正在保存。 */
  saving: boolean;
  /** 编辑前的文本，用于取消编辑。 */
  originalText: string;
}

/** openDialog 打开翻译编辑弹窗并加载当前资源译文。 */
async function openDialog() {
  dialogVisible.value = true;
  await loadTranslations();
}

/** loadTranslations 查询当前资源的翻译并补齐启用语言行。 */
async function loadTranslations() {
  const targetType = props.targetType;
  const targetId = props.targetId;
  if (!targetType || !targetId) return;
  loading.value = true;
  try {
    await loadEnabledBaseLanguages();
    const rows = await queryTranslationRows();
    translationRows.value = rows;
    const missingRows = rows.filter(row => !row.text);
    if (missingRows.length === 0) return;
    const sourceLocale = languages.value.find(item => item.is_primary)?.language_code || "";
    if (!sourceLocale || !props.source) return;
    const results = await Promise.allSettled(
      missingRows.map(async row => {
        const draft = await defBaseTranslationService.DraftBaseTranslation({
          source: props.source,
          source_locale: sourceLocale,
          target_locale: row.locale
        });
        await defBaseTranslationService.UpdateBaseTranslation({
          id: row.id,
          target_type: targetType,
          target_id: targetId,
          locale: row.locale,
          name: draft.translation
        });
        return { row, text: draft.translation };
      })
    );
    for (const result of results) {
      if (result.status !== "fulfilled") continue;
      result.value.row.text = result.value.text;
      result.value.row.originalText = result.value.text;
      translationOverrides.value.set(result.value.row.locale, result.value.text);
    }
  } catch {
    ElMessage.error(t("common.message.system_error"));
  } finally {
    loading.value = false;
  }
}

/** queryTranslationRows 将列表返回的翻译记录补齐为可编辑语言行。 */
async function queryTranslationRows(): Promise<DynamicTranslationRow[]> {
  const records = new Map((props.translations ?? []).map(item => [item.locale, item]));
  return languages.value
    .filter(item => !item.is_primary)
    .map(item => {
      const record = records.get(item.language_code);
      const text = translationOverrides.value.get(item.language_code) ?? record?.name ?? "";
      return {
        id: record?.id ?? 0,
        locale: item.language_code,
        text,
        editing: false,
        saving: false,
        originalText: text
      };
    });
}

/** startEdit 开启指定语言的编辑状态。 */
function startEdit(row: DynamicTranslationRow) {
  row.originalText = row.text;
  row.editing = true;
}

/** cancelEdit 恢复编辑前的内容。 */
function cancelEdit(row: DynamicTranslationRow) {
  row.text = row.originalText;
  row.editing = false;
}

/** saveTranslation 使用统一更新接口保存指定语言译文。 */
async function saveTranslation(row: DynamicTranslationRow) {
  const targetType = props.targetType;
  const targetId = props.targetId;
  if (row.saving || !targetType || !targetId) return;
  row.saving = true;
  try {
    await defBaseTranslationService.UpdateBaseTranslation({
      id: row.id,
      target_type: targetType,
      target_id: targetId,
      locale: row.locale,
      name: row.text
    });
    ElMessage.success(t("common.message.operation_success"));
    row.editing = false;
    row.originalText = row.text;
    translationOverrides.value.set(row.locale, row.text);
  } catch {
    ElMessage.error(t("common.message.system_error"));
  } finally {
    row.saving = false;
  }
}

/** handleDialogClosed 清理弹窗状态，避免复用旧资源译文。 */
function handleDialogClosed() {
  translationRows.value = [];
  loading.value = false;
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
.dynamic-translation-dialog__text {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
