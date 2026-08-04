<!-- 代码生成表配置 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      class="code-gen-table"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :restore-selected-row-keys="progressSelectedTableIds"
      :request-api="requestCodeGenTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialogTitle"
      width="min(920px, calc(100vw - 32px))"
      top="4vh"
      :model="formData"
      :fields="formFields"
      :rules="tableRules"
      :confirm-loading="saving"
      label-width="116px"
      :gutter="16"
      :col-span="12"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #tableI18nConfig>
        <CodeGenLocaleEditor
          :model-value="formData.i18n_config"
          :source-comment="formData.comment"
          :show-left-tree-comment="formData.page_type === 'left_tree'"
          :source-left-tree-comment="formData.left_tree_config?.comment"
          @update:model-value="value => (formData.i18n_config = value)"
        />
      </template>
    </FormDialog>

    <CodeGenProgressDialog
      v-model="progressDialogVisible"
      :task-id="progressTaskId"
      @update:model-value="handleProgressDialogVisibleChange"
      @completed="handleProgressCompleted"
      @unavailable="handleProgressUnavailable"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import {
  CirclePlus,
  Clock,
  Connection,
  Delete,
  Document,
  EditPen,
  Promotion,
  RefreshRight,
  SetUp,
  View
} from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseMenuService } from "@liujitcn/kratos-admin-system/api/system/base_menu";
import { defBaseDictService } from "@liujitcn/kratos-admin-system/api/system/base_dict";
import { defCodeGenService } from "@liujitcn/kratos-admin-system/api/system/code_gen";
import { defCodeGenColumnService } from "@liujitcn/kratos-admin-system/api/system/code_gen_column";
import { defCodeGenTableService } from "@liujitcn/kratos-admin-system/api/system/code_gen_table";
import type { CodeGenDatabaseColumn } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_column";
import type {
  CodeGenDatabaseTable,
  CodeGenTable,
  CodeGenTableForm,
  PageCodeGenTableRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_table";
import type { BaseMenu } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_menu";
import type { OptionBaseDictResponse_BaseDictItem } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dict";
import { BaseMenuType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/common";
import { CodeGenTableStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_table";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import CodeGenProgressDialog from "../components/CodeGenProgressDialog.vue";
import CodeGenLocaleEditor from "../components/CodeGenLocaleEditor.vue";
import {
  codeGenPageTypeOptions,
  codeGenTableRules,
  createDefaultCodeGenLeftTreeConfig,
  createDefaultCodeGenTableForm,
  isCodeGenTreePageType
} from "../config";

defineOptions({
  name: "CodeGenTable",
  inheritAttrs: false
});

/** 代码生成表配置删除入参。 */
type CodeGenDeleteTarget = number | string | Array<number | string> | CodeGenTable | CodeGenTable[];

/** 代码生成单项或批量目标。 */
type CodeGenGenerateTarget = CodeGenTable | CodeGenTable[];

/** 代码生成单项或批量还原目标。 */
type CodeGenRestoreTarget = CodeGenTable | CodeGenTable[];

/** 代码生成表配置弹窗状态，未选择父级菜单时保持空白。 */
type CodeGenTableFormState = Omit<CodeGenTableForm, "parent_menu_id"> & {
  /** 父级菜单ID。 */
  parent_menu_id?: number;
};

const codeGenTaskStorageKey = "code-gen-progress-task-id";
const codeGenProgressDialogVisibleStorageKey = "code-gen-progress-dialog-visible";
const codeGenProgressSelectedTableIdsStorageKey = "code-gen-progress-selected-table-ids";
const codeGenStatusDisabled = CodeGenTableStatus.CODE_GEN_TABLE_STATUS_DISABLED;

const { BUTTONS } = useAuthButtons();
const router = useRouter();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const saving = ref(false);
const databaseTables = ref<CodeGenDatabaseTable[]>([]);
const businessModuleItems = ref<OptionBaseDictResponse_BaseDictItem[]>([]);
const databaseColumns = ref<CodeGenDatabaseColumn[]>([]);
const leftTreeDatabaseColumns = ref<CodeGenDatabaseColumn[]>([]);
const parentMenuOptions = ref<ProFormOption[]>([]);
const progressTaskId = ref(typeof window === "undefined" ? "" : (window.sessionStorage.getItem(codeGenTaskStorageKey) ?? ""));
const progressDialogVisible = ref(
  !!progressTaskId.value &&
    typeof window !== "undefined" &&
    window.sessionStorage.getItem(codeGenProgressDialogVisibleStorageKey) === "true"
);
const progressTaskAvailable = ref(!!progressTaskId.value);
const generating = ref(!!progressTaskId.value);
const progressSelectedTableIds = ref<Array<string | number>>(readProgressSelectedTableIds());

const dialog = reactive({ editing: false, visible: false });

const formData = reactive<CodeGenTableFormState>({ ...createDefaultCodeGenTableForm(), parent_menu_id: undefined });
const dialogTitle = computed(() => t(dialog.editing ? "system.code.gen.table.title.edit" : "system.code.gen.table.title.create"));
const tableRules = computed(() => codeGenTableRules());
const pageTypeOptions = computed(() => codeGenPageTypeOptions());

/** 当前业务表选择项。 */
const databaseTableOptions = computed<ProFormOption[]>(() =>
  databaseTables.value.map(item => ({
    label: item.comment ? `${item.name}（${item.comment}）` : item.name,
    value: item.name,
    disabled: item.disabled && item.name !== formData.name
  }))
);

/** 左树来源表选择项。 */
const leftTreeTableOptions = computed<ProFormOption[]>(() =>
  databaseTables.value.map(item => ({
    label: item.comment ? `${item.name}（${item.comment}）` : item.name,
    value: item.name
  }))
);

/** 业务模块选择项，仅允许新增或编辑时选择启用项；已保存的停用项保留为只读选项。 */
const businessModuleOptions = computed<ProFormOption[]>(() => {
  const options: ProFormOption[] = businessModuleItems.value.map(item => ({ label: item.label, value: item.value }));
  if (formData.business_module && !options.some(item => item.value === formData.business_module)) {
    options.push({
      label: t("system.code.gen.table.value.disabled_module", { module: formData.business_module }),
      value: formData.business_module,
      disabled: true
    });
  }
  return options;
});

/** 当前业务表字段选择项。 */
const databaseColumnOptions = computed<ProFormOption[]>(() => createDatabaseColumnOptions(databaseColumns.value));

/** 左树来源表字段选择项。 */
const leftTreeColumnOptions = computed<ProFormOption[]>(() => createDatabaseColumnOptions(leftTreeDatabaseColumns.value));

/** 代码生成表配置表单字段。 */
const formFields = computed<ProFormField[]>(() => [
  // 标签提示与当前生成器的实际读写逻辑保持一致，方便配置时判断影响范围。
  {
    prop: "name",
    label: t("system.code.gen.table.field.name"),
    component: "select",
    options: databaseTableOptions.value,
    colSpan: 24,
    labelTooltip: t("system.code.gen.table.tooltip.name"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.database_table"),
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleTableNameChange
    }
  },
  {
    prop: "comment",
    label: t("system.code.gen.table.field.comment"),
    component: "input",
    colSpan: 24,
    labelTooltip: t("system.code.gen.table.tooltip.comment"),
    props: { placeholder: t("system.code.gen.table.placeholder.comment") }
  },
  {
    prop: "i18n_config",
    label: t("system.code.gen.i18n.field.translations"),
    component: "slot",
    slotName: "tableI18nConfig",
    colSpan: 24,
    labelTooltip: t("system.code.gen.i18n.tooltip.table")
  },
  {
    prop: "business_module",
    label: t("system.code.gen.table.field.business_module"),
    component: "select",
    options: businessModuleOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.business_module"),
    props: { placeholder: t("system.code.gen.table.placeholder.business_module"), filterable: true, style: { width: "100%" } }
  },
  {
    prop: "parent_menu_id",
    label: t("system.code.gen.table.field.parent_menu"),
    component: "tree-select",
    options: parentMenuOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.parent_menu"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.parent_menu"),
      clearable: true,
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  {
    prop: "page_type",
    label: t("system.code.gen.table.field.page_type"),
    component: "segmented",
    options: pageTypeOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.page_type"),
    props: { onChange: handlePageTypeChange }
  },
  {
    prop: "parent_column",
    label: t("system.code.gen.table.field.parent_column"),
    component: "select",
    options: databaseColumnOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.parent_column"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.parent_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => isCodeGenTreePageType(model.page_type)
  },
  {
    prop: "tree_label_column",
    label: t("system.code.gen.table.field.tree_label_column"),
    component: "select",
    options: databaseColumnOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.tree_label_column"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.tree_label_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => isCodeGenTreePageType(model.page_type)
  },
  {
    prop: "left_tree_config.table_name",
    label: t("system.code.gen.table.field.left_tree_table"),
    component: "select",
    options: leftTreeTableOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_table"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.left_tree_table"),
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleLeftTreeTableNameChange
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.comment",
    label: t("system.code.gen.table.field.left_tree_comment"),
    component: "input",
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_comment"),
    props: { placeholder: t("system.code.gen.table.placeholder.left_tree_comment") },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.filter_column",
    label: t("system.code.gen.table.field.filter_column"),
    labelTooltip: t("system.code.gen.table.tooltip.filter_column"),
    component: "select",
    options: databaseColumnOptions.value,
    props: {
      placeholder: t("system.code.gen.table.placeholder.filter_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.parent_column",
    label: t("system.code.gen.table.field.left_tree_parent_column"),
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_parent_column"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.left_tree_parent_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.label_column",
    label: t("system.code.gen.table.field.left_tree_label_column"),
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_label_column"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.left_tree_label_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.value_column",
    label: t("system.code.gen.table.field.left_tree_value_column"),
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_value_column"),
    props: {
      placeholder: t("system.code.gen.table.placeholder.left_tree_value_column"),
      clearable: true,
      filterable: true,
      style: { width: "100%" }
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.lazy",
    label: t("system.code.gen.table.field.left_tree_lazy"),
    component: "switch",
    labelTooltip: t("system.code.gen.table.tooltip.left_tree_lazy"),
    props: { activeText: t("system.code.gen.value.lazy"), inactiveText: t("system.code.gen.value.eager") },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "gen_backend",
    label: t("system.code.gen.table.field.gen_backend"),
    component: "switch",
    labelTooltip: t("system.code.gen.table.tooltip.gen_backend"),
    // 三个生成开关始终从新行开始并排展示。
    rowBreakBefore: true,
    colSpan: 8,
    props: { activeText: t("system.code.gen.value.generate"), inactiveText: t("system.code.gen.value.skip") }
  },
  {
    prop: "gen_frontend",
    label: t("system.code.gen.table.field.gen_frontend"),
    component: "switch",
    labelTooltip: t("system.code.gen.table.tooltip.gen_frontend"),
    colSpan: 8,
    props: { activeText: t("system.code.gen.value.generate"), inactiveText: t("system.code.gen.value.skip") }
  },
  {
    prop: "gen_sql",
    label: t("system.code.gen.table.field.gen_sql"),
    component: "switch",
    colSpan: 8,
    labelTooltip: t("system.code.gen.table.tooltip.gen_sql"),
    props: { activeText: t("system.code.gen.value.generate"), inactiveText: t("system.code.gen.value.skip") }
  },
  {
    prop: "status",
    label: t("system.code.gen.table.field.status"),
    component: "dict",
    colSpan: 24,
    props: { code: "code_gen_table_status", codeType: "number", type: "radio" },
    labelTooltip: t("system.code.gen.table.tooltip.status")
  },
  {
    prop: "remark",
    label: t("system.code.gen.table.field.remark"),
    component: "textarea",
    colSpan: 24,
    labelTooltip: t("system.code.gen.table.tooltip.remark"),
    props: { placeholder: t("system.code.gen.table.placeholder.remark"), rows: 3 }
  }
]);

/** 代码生成表配置列表列。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: t("system.code.gen.table.field.name"), minWidth: 160, search: { el: "input" } },
  { prop: "comment", label: t("system.code.gen.table.field.comment"), minWidth: 160, showOverflowTooltip: true },
  {
    prop: "business_module",
    label: t("system.code.gen.table.field.business_module"),
    minWidth: 140,
    dictCode: "business_module",
    dictValueType: "string",
    search: { el: "select" }
  },
  {
    prop: "page_type",
    label: t("system.code.gen.table.field.page_type"),
    minWidth: 120,
    enum: pageTypeOptions.value,
    search: { el: "select" }
  },
  {
    prop: "status",
    label: t("system.code.gen.table.field.status"),
    width: 100,
    dictCode: "code_gen_table_status",
    search: { el: "select" }
  },
  { prop: "remark", label: t("system.code.gen.table.field.remark"), minWidth: 180, showOverflowTooltip: true },
  { prop: "created_at", label: t("system.code.gen.table.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 660,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("system.code.gen.table.action.columns"),
        type: "success",
        link: true,
        icon: SetUp,
        hidden: () => !BUTTONS.value["tool:code-gen-table:column"],
        onClick: scope => handleOpenColumnConfig((scope.row as CodeGenTable).id)
      },
      {
        label: t("system.code.gen.table.action.proto"),
        type: "warning",
        link: true,
        icon: Connection,
        hidden: () => !BUTTONS.value["tool:code-gen-table:proto"],
        onClick: scope => handleOpenProtoConfig((scope.row as CodeGenTable).id)
      },
      {
        label: t("system.code.gen.table.action.page_preview"),
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["tool:code-gen-table:preview"],
        onClick: scope => handleOpenPreview((scope.row as CodeGenTable).id)
      },
      {
        label: t("system.code.gen.table.action.code_preview"),
        type: "primary",
        link: true,
        icon: Document,
        hidden: () => !BUTTONS.value["tool:code-gen-table:code-preview"],
        onClick: scope => handleOpenCodePreview((scope.row as CodeGenTable).id)
      },
      {
        label: t("system.code.gen.action.generate"),
        type: "success",
        link: true,
        icon: Promotion,
        disabled: () => generating.value,
        hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
        onClick: scope => handleGenerate(scope.row as CodeGenTable)
      },
      {
        label: t("system.code.gen.action.restore"),
        type: "warning",
        link: true,
        icon: RefreshRight,
        disabled: scope => !(scope.row as CodeGenTable).restore_available,
        hidden: () => !BUTTONS.value["tool:code-gen-table:restore"],
        onClick: scope => handleRestore(scope.row as CodeGenTable)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["tool:code-gen-table:update"],
        onClick: scope => handleOpenDialog((scope.row as CodeGenTable).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
        onClick: scope => handleDelete(scope.row as CodeGenTable)
      }
    ]
  }
]);

/** 打开已经保存的代码生成页面预览。 */
async function handleOpenPreview(tableId: number) {
  await router.push(`/code/gen/preview/${tableId}`);
}

/** 打开字段配置页面。 */
async function handleOpenColumnConfig(tableId: number) {
  await router.push(`/code/gen/column/${tableId}`);
}

/** 打开Proto接口配置页面。 */
async function handleOpenProtoConfig(tableId: number) {
  await router.push(`/code/gen/proto/${tableId}`);
}

/** 代码生成表配置列表顶部操作。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["tool:code-gen-table:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("system.code.gen.action.batch_generate"),
    type: "primary",
    icon: Promotion,
    hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
    disabled: scope => generating.value || !scope.selectedList.length,
    onClick: scope => handleGenerate(scope.selectedList as CodeGenTable[])
  },
  {
    label: t("system.code.gen.action.batch_restore"),
    type: "warning",
    icon: RefreshRight,
    hidden: () => !BUTTONS.value["tool:code-gen-table:restore"],
    disabled: scope => scope.selectedList.every(item => !(item as CodeGenTable).restore_available),
    onClick: scope => handleRestore(scope.selectedList as CodeGenTable[])
  },
  {
    label: t("system.code.gen.action.recent_task"),
    icon: Clock,
    hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
    disabled: () => !progressTaskAvailable.value,
    onClick: handleOpenProgress
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as CodeGenTable[])
  }
]);

/** 打开已经保存的代码生成文件预览。 */
async function handleOpenCodePreview(tableId: number) {
  await router.push(`/code/gen/code/preview/${tableId}`);
}

/** 创建单项或批量代码生成任务。 */
async function handleGenerate(selected: CodeGenGenerateTarget) {
  const tables = Array.isArray(selected) ? selected : [selected];
  if (!tables.length) {
    ElMessage.warning(t("system.code.gen.table.message.select_generate"));
    return;
  }
  const disabledTable = tables.find(table => table.status === codeGenStatusDisabled);
  if (disabledTable) {
    ElMessage.warning(t("system.code.gen.table.message.disabled", { name: disabledTable.name }));
    return;
  }
  const message =
    tables.length === 1
      ? t("system.code.gen.table.dialog.generate_one", { name: tables[0].name })
      : t("system.code.gen.table.dialog.generate_batch", { count: tables.length });
  try {
    await ElMessageBox.confirm(message, t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
  } catch {
    return;
  }
  generating.value = true;
  try {
    const data = await defCodeGenService.StartCodeGenTask({
      table_ids: tables.map(table => table.id)
    });
    progressTaskId.value = data.task_id;
    progressTaskAvailable.value = true;
    progressSelectedTableIds.value = tables.map(table => table.id);
    window.sessionStorage.setItem(codeGenTaskStorageKey, data.task_id);
    window.sessionStorage.setItem(codeGenProgressSelectedTableIdsStorageKey, JSON.stringify(progressSelectedTableIds.value));
    handleProgressDialogVisibleChange(true);
  } catch (error) {
    generating.value = false;
    throw error;
  }
}

/** 还原单项或批量代码生成结果。 */
async function handleRestore(selected: CodeGenRestoreTarget) {
  const tables = Array.isArray(selected) ? selected : [selected];
  const restorableTables = tables.filter(table => table.restore_available);
  if (!restorableTables.length) {
    ElMessage.warning(t("system.code.gen.table.message.select_restore"));
    return;
  }
  const message =
    restorableTables.length === 1
      ? t("system.code.gen.table.dialog.restore_one", { name: restorableTables[0].name })
      : t("system.code.gen.table.dialog.restore_batch", { count: restorableTables.length });
  try {
    await ElMessageBox.confirm(message, t("common.title.warning"), {
      confirmButtonText: t("system.code.gen.action.confirm_restore"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
  } catch {
    return;
  }
  await defCodeGenService.RestoreCodeGen({ table_ids: restorableTables.map(table => table.id) });
  ElMessage.success(t("system.code.gen.table.message.restore_success"));
  progressSelectedTableIds.value = [];
  proTable.value?.clearSelection();
  refreshTable();
}

/** 打开最近一次代码生成任务。 */
function handleOpenProgress() {
  if (progressTaskId.value) handleProgressDialogVisibleChange(true);
}

/** 同步进度弹窗可见状态，确保热更新后仅恢复任务运行期间主动打开的弹窗。 */
function handleProgressDialogVisibleChange(visible: boolean) {
  progressDialogVisible.value = visible;
  if (visible) {
    window.sessionStorage.setItem(codeGenProgressDialogVisibleStorageKey, "true");
    return;
  }
  window.sessionStorage.removeItem(codeGenProgressDialogVisibleStorageKey);
}

/** 生成任务结束后刷新列表。 */
function handleProgressCompleted() {
  generating.value = false;
  window.sessionStorage.removeItem(codeGenProgressDialogVisibleStorageKey);
  progressSelectedTableIds.value = [];
  proTable.value?.clearSelection();
  removeProgressSelectedTableIds();
  refreshTable();
}

/** 清理不可恢复的最近任务。 */
function handleProgressUnavailable() {
  generating.value = false;
  progressTaskId.value = "";
  progressTaskAvailable.value = false;
  progressSelectedTableIds.value = [];
  proTable.value?.clearSelection();
  handleProgressDialogVisibleChange(false);
  removeProgressSelectedTableIds();
  window.sessionStorage.removeItem(codeGenTaskStorageKey);
}

/** 读取上次页面重建前的批量生成选择项。 */
function readProgressSelectedTableIds(): Array<string | number> {
  if (typeof window === "undefined") return [];
  try {
    const selectedTableIds = JSON.parse(window.sessionStorage.getItem(codeGenProgressSelectedTableIdsStorageKey) ?? "[]");
    return Array.isArray(selectedTableIds) ? selectedTableIds.filter(id => typeof id === "string" || typeof id === "number") : [];
  } catch {
    return [];
  }
}

/** 清理跨热更新恢复选择所需的会话记录，保留当前页面的选择状态。 */
function removeProgressSelectedTableIds() {
  window.sessionStorage.removeItem(codeGenProgressSelectedTableIdsStorageKey);
}

/** 请求代码生成表配置列表。 */
async function requestCodeGenTable(params: PageCodeGenTableRequest) {
  const data = await defCodeGenTableService.PageCodeGenTable(buildPageRequest(params));
  return { data: { ...data, list: data.code_gen_tables ?? [] } };
}

/** 刷新代码生成表配置列表。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 打开新增或编辑弹窗，并加载当前表单所需选项。 */
async function handleOpenDialog(tableId?: number) {
  resetForm();
  const [tableData, menuData, dictionaryData] = await Promise.all([
    defCodeGenTableService.ListCodeGenDatabaseTable({}),
    defBaseMenuService.TreeBaseMenu({}),
    defBaseDictService.OptionBaseDict({})
  ]);
  databaseTables.value = tableData.tables ?? [];
  parentMenuOptions.value = convertMenuOptions(menuData.base_menus ?? []);
  businessModuleItems.value = dictionaryData.base_dicts?.find(item => item.code === "business_module")?.items ?? [];
  if (tableId) {
    const detail = await defCodeGenTableService.GetCodeGenTable({ id: tableId });
    Object.assign(formData, detail);
    formData.parent_menu_id = detail.parent_menu_id || undefined;
    formData.left_tree_config ??= createDefaultCodeGenLeftTreeConfig();
    if (!formData.comment) {
      formData.comment = databaseTables.value.find(item => item.name === formData.name)?.comment ?? "";
    }
    if (!formData.left_tree_config.comment) {
      formData.left_tree_config.comment =
        databaseTables.value.find(item => item.name === formData.left_tree_config?.table_name)?.comment ?? "";
    }
    await Promise.all([loadDatabaseColumns(databaseColumns, formData.name), loadLeftTreeDatabaseColumns()]);
    dialog.editing = true;
  }
  dialog.visible = true;
}

/** 关闭弹窗并清理表单状态。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/** 重置弹窗表单和字段选项。 */
function resetForm() {
  Object.assign(formData, { ...createDefaultCodeGenTableForm(), parent_menu_id: undefined });
  dialog.editing = false;
  databaseColumns.value = [];
  leftTreeDatabaseColumns.value = [];
  void nextTick(() => {
    formDialogRef.value?.resetFields();
    formDialogRef.value?.clearValidate();
  });
}

/** 选择业务表后同步数据库注释、默认命名、字段选项和树字段默认值。 */
async function handleTableNameChange(tableName: string) {
  const table = databaseTables.value.find(item => item.name === tableName);
  formData.comment = table?.comment ?? "";
  formData.i18n_config = new Map();
  await loadDatabaseColumns(databaseColumns, tableName);
  resetUnavailableTableColumns();
  formData.parent_column = resolveDefaultColumn(databaseColumns.value, "parent_id");
  formData.tree_label_column = resolveDefaultColumn(databaseColumns.value, "name");
}

/** 页面类型变化时清理不再生效的页面字段。 */
function handlePageTypeChange(pageType: string) {
  if (!isCodeGenTreePageType(pageType)) {
    formData.parent_column = "";
    formData.tree_label_column = "";
  }
  if (pageType !== "left_tree") {
    resetLeftTreeConfig();
  }
}

/** 左树来源表变化时覆盖描述、加载字段选项并设置约定默认字段。 */
async function handleLeftTreeTableNameChange(tableName: string) {
  const config = ensureLeftTreeConfig();
  const table = databaseTables.value.find(item => item.name === tableName);
  config.comment = table?.comment ?? "";
  await loadLeftTreeDatabaseColumns();
  resetUnavailableLeftTreeColumns();
  config.parent_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "parent_id");
  config.label_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "name");
  config.value_column = resolveDefaultColumn(leftTreeDatabaseColumns.value, "id");
}

/** 提交代码生成表配置。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;
  if (!formData.parent_menu_id) return;

  const payload: CodeGenTableForm = { ...formData, parent_menu_id: formData.parent_menu_id };
  saving.value = true;
  try {
    if (formData.id) {
      await defCodeGenTableService.UpdateCodeGenTable({ id: formData.id, code_gen_table: payload });
      ElMessage.success(t("system.code.gen.table.message.update_success"));
    } else {
      await defCodeGenTableService.CreateCodeGenTable({ code_gen_table: payload });
      ElMessage.success(t("system.code.gen.table.message.create_success"));
    }
    handleCloseDialog();
    refreshTable();
  } finally {
    saving.value = false;
  }
}

/** 删除单项或批量代码生成表配置。 */
async function handleDelete(selected?: CodeGenDeleteTarget) {
  const tableList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as CodeGenTable[])
    : selected && typeof selected === "object"
      ? [selected as CodeGenTable]
      : [];
  const tableIds = (
    tableList.length ? tableList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!tableIds) {
    ElMessage.warning(t("system.code.gen.table.message.select_delete"));
    return;
  }
  const confirmMessage =
    tableList.length === 1
      ? t("system.code.gen.table.dialog.delete_one", { name: tableList[0].name })
      : t("system.code.gen.table.dialog.delete_batch", { count: tableList.length });
  try {
    await ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
  } catch {
    ElMessage.info(t("system.code.gen.table.message.delete_canceled"));
    return;
  }
  await defCodeGenTableService.DeleteCodeGenTable({ ids: tableIds });
  ElMessage.success(t("system.code.gen.table.message.delete_success"));
  refreshTable();
}

/** 查询数据库表字段选项。 */
async function loadDatabaseColumns(target: { value: CodeGenDatabaseColumn[] }, tableName: string) {
  if (!tableName) {
    target.value = [];
    return;
  }
  const data = await defCodeGenColumnService.ListCodeGenDatabaseColumn({ table_name: tableName });
  target.value = data.columns ?? [];
}

/** 查询左树来源表字段选项。 */
async function loadLeftTreeDatabaseColumns() {
  const config = ensureLeftTreeConfig();
  if (formData.page_type !== "left_tree") {
    leftTreeDatabaseColumns.value = [];
    return;
  }
  await loadDatabaseColumns(leftTreeDatabaseColumns, config.table_name);
}

/** 转换数据库字段为 ProForm 选择项。 */
function createDatabaseColumnOptions(columns: CodeGenDatabaseColumn[]): ProFormOption[] {
  return columns.map(item => ({
    label: item.comment
      ? `${item.name}（${item.comment} / ${item.column_type || item.db_type}）`
      : `${item.name}（${item.column_type || item.db_type}）`,
    value: item.name
  }));
}

/** 从字段列表中解析存在的约定默认字段。 */
function resolveDefaultColumn(columns: CodeGenDatabaseColumn[], columnName: string) {
  return columns.some(item => item.name === columnName) ? columnName : "";
}

/** 转换菜单树为 ProForm 树形选择项。 */
function convertMenuOptions(options: BaseMenu[]): ProFormOption[] {
  return options
    .filter(item => item.type === BaseMenuType.BASE_MENU_TYPE_FOLDER)
    .map(item => ({
      label: item.meta?.title || item.name || item.path,
      value: item.id,
      disabled: item.id < 100 || item.id > 99999,
      children: convertMenuOptions(item.children ?? [])
    }));
}

/** 清理当前业务表已不存在的字段配置。 */
function resetUnavailableTableColumns() {
  const columnNames = new Set(databaseColumns.value.map(item => item.name));
  if (formData.parent_column && !columnNames.has(formData.parent_column)) formData.parent_column = "";
  if (formData.tree_label_column && !columnNames.has(formData.tree_label_column)) formData.tree_label_column = "";
  const config = ensureLeftTreeConfig();
  if (config.filter_column && !columnNames.has(config.filter_column)) {
    config.filter_column = "";
  }
}

/** 清理左树来源表已不存在的字段配置。 */
function resetUnavailableLeftTreeColumns() {
  const columnNames = new Set(leftTreeDatabaseColumns.value.map(item => item.name));
  const config = ensureLeftTreeConfig();
  if (config.parent_column && !columnNames.has(config.parent_column)) {
    config.parent_column = "";
  }
  if (config.label_column && !columnNames.has(config.label_column)) {
    config.label_column = "";
  }
  if (config.value_column && !columnNames.has(config.value_column)) {
    config.value_column = "";
  }
}

/** 清空左树右表专属配置。 */
function resetLeftTreeConfig() {
  formData.left_tree_config = createDefaultCodeGenLeftTreeConfig();
  leftTreeDatabaseColumns.value = [];
}

/** 确保左树配置对象存在并返回当前配置。 */
function ensureLeftTreeConfig() {
  formData.left_tree_config ??= createDefaultCodeGenLeftTreeConfig();
  return formData.left_tree_config;
}
</script>

<style scoped lang="scss">
/* 固定操作列表头与普通表头使用同一主题背景，并保持行内操作单行展示。 */
:deep(.code-gen-table) {
  --el-table-header-bg-color: var(--el-fill-color-light);

  td.el-table-fixed-column--right .cell {
    white-space: nowrap;
  }
}
</style>
