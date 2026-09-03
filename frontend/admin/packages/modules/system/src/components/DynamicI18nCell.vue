<template>
  <el-link class="dynamic-i18n-cell" type="primary" underline="never" @click="openDialog">
    <span class="dynamic-i18n-cell__source">{{ displaySource || "--" }}</span>
  </el-link>

  <ProDialog
    v-model="dialogVisible"
    :title="displaySource"
    width="760px"
    append-to-body
    :show-footer="false"
    :destroy-on-close="true"
    @closed="handleDialogClosed"
  >
    <el-table v-loading="loading" :data="i18nRows" border>
      <el-table-column :label="t('common.field.language')" width="150">
        <template #default="{ row }">
          {{ getLanguageLabel(row.locale) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('system.base.i18n.field.i18ns')" min-width="360">
        <template #default="{ row }">
          <el-input
            v-if="row.editing"
            v-model="row.text"
            :placeholder="t('system.base.i18n.placeholder.text', { language: getLanguageLabel(row.locale) })"
          />
          <span v-else class="dynamic-i18n-dialog__text">
            {{ row.text || t("common.value.none") }}
          </span>
        </template>
      </el-table-column>
      <el-table-column v-if="editable" :label="t('common.field.operation')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button v-if="editable && !row.editing" link type="primary" :icon="EditPen" @click="startEdit(row)">
            {{ t("common.action.edit") }}
          </el-button>
          <template v-else>
            <el-button link type="primary" :icon="Check" :loading="row.saving" @click="saveI18n(row)">
              {{ t("common.action.save") }}
            </el-button>
            <el-button link :icon="Close" :disabled="row.saving" @click="cancelEdit(row)">
              {{ t("common.action.cancel") }}
            </el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>
  </ProDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { Check, Close, EditPen } from "@element-plus/icons-vue";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import { loadEnabledBaseLanguages, useEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import { defBaseI18nService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_i18n";
import type { BaseI18n } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import { I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import { getLanguageLabel } from "./dynamicI18n";

/** DynamicI18nCellProps 动态翻译列表单元格属性。 */
interface DynamicI18nCellProps {
  /** 资源源文。 */
  source: string;
  /** 统一翻译表目标类型。 */
  targetType?: I18nTargetType;
  /** 目标资源编号。 */
  targetId?: number;
  /** 资源列表返回的非主语言翻译。 */
  i18ns?: BaseI18n[];
  /** 是否允许编辑翻译。 */
  editable?: boolean;
}

const props = withDefaults(defineProps<DynamicI18nCellProps>(), {
  editable: true
});

const editable = computed(() => props.editable);

const { languages } = useEnabledBaseLanguages();
const { locale } = useLocaleStore();
const dialogVisible = ref(false);
const loading = ref(false);
const i18nRows = ref<DynamicI18nRow[]>([]);
const i18nOverrides = ref(new Map<string, string>());

const displaySource = computed(() => {
  const override = i18nOverrides.value.get(locale.value);
  if (override) return override;
  const i18n = props.i18ns?.find(item => item.locale === locale.value && item.name);
  return i18n?.name || props.source;
});

watch(
  () => [props.targetType, props.targetId],
  () => {
    i18nRows.value = [];
    i18nOverrides.value.clear();
    dialogVisible.value = false;
  }
);

/** DynamicI18nRow 描述弹窗内可编辑的单语言翻译行。 */
interface DynamicI18nRow {
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
  await loadI18ns();
}

/** loadI18ns 查询当前资源的翻译并补齐启用语言行。 */
async function loadI18ns() {
  const targetType = props.targetType;
  const targetId = props.targetId;
  if (!targetType || !targetId) return;
  loading.value = true;
  try {
    await loadEnabledBaseLanguages();
    const rows = await queryI18nRows();
    i18nRows.value = rows;
    if (!props.editable) return;
    const missingRows = rows.filter(row => !row.text);
    if (missingRows.length === 0) return;
    if (!props.source) return;
    const draft = await defBaseI18nService.DraftBaseI18n({ source: props.source });
    const i18ns = new Map(draft.i18ns.map(item => [item.locale, item.i18n]));
    const results = await Promise.allSettled(
      missingRows.map(async row => {
        const i18n = i18ns.get(row.locale);
        if (!i18n) throw new Error("i18n not found");
        await defBaseI18nService.UpdateBaseI18n({
          id: row.id,
          target_type: targetType,
          target_id: targetId,
          locale: row.locale,
          name: i18n
        });
        return { row, text: i18n };
      })
    );
    for (const result of results) {
      if (result.status !== "fulfilled") continue;
      result.value.row.text = result.value.text;
      result.value.row.originalText = result.value.text;
      i18nOverrides.value.set(result.value.row.locale, result.value.text);
    }
  } catch {
    ElMessage.error(t("common.message.system_error"));
  } finally {
    loading.value = false;
  }
}

/** queryI18nRows 将列表返回的翻译记录补齐为可编辑语言行。 */
async function queryI18nRows(): Promise<DynamicI18nRow[]> {
  const records = new Map((props.i18ns ?? []).map(item => [item.locale, item]));
  return languages.value
    .filter(item => !item.is_primary)
    .map(item => {
      const record = records.get(item.language_code);
      const text = i18nOverrides.value.get(item.language_code) ?? record?.name ?? "";
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
function startEdit(row: DynamicI18nRow) {
  if (!props.editable) return;
  row.originalText = row.text;
  row.editing = true;
}

/** cancelEdit 恢复编辑前的内容。 */
function cancelEdit(row: DynamicI18nRow) {
  row.text = row.originalText;
  row.editing = false;
}

/** saveI18n 使用统一更新接口保存指定语言译文。 */
async function saveI18n(row: DynamicI18nRow) {
  const targetType = props.targetType;
  const targetId = props.targetId;
  if (!props.editable || row.saving || !targetType || !targetId) return;
  if (row.id === 0 && !row.text) {
    row.editing = false;
    return;
  }
  row.saving = true;
  try {
    await defBaseI18nService.UpdateBaseI18n({
      id: row.id,
      target_type: targetType,
      target_id: targetId,
      locale: row.locale,
      name: row.text
    });
    ElMessage.success(t("common.message.operation_success"));
    row.editing = false;
    row.originalText = row.text;
    i18nOverrides.value.set(row.locale, row.text);
  } catch {
    ElMessage.error(t("common.message.system_error"));
  } finally {
    row.saving = false;
  }
}

/** handleDialogClosed 清理弹窗状态，避免复用旧资源译文。 */
function handleDialogClosed() {
  i18nRows.value = [];
  loading.value = false;
}
</script>

<style scoped>
.dynamic-i18n-cell {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 100%;
}
.dynamic-i18n-cell__source {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dynamic-i18n-dialog__text {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
