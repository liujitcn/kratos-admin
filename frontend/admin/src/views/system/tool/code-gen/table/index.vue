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
      :title="dialog.title"
      width="min(920px, calc(100vw - 32px))"
      top="4vh"
      :model="formData"
      :fields="formFields"
      :rules="codeGenTableRules"
      :confirm-loading="saving"
      label-width="116px"
      :gutter="16"
      :col-span="12"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />

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
import { CirclePlus, Clock, Connection, Delete, Document, EditPen, Promotion, RefreshRight, SetUp, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseMenuService } from "@/api/system/base_menu";
import { defBaseDictService } from "@/api/system/base_dict";
import { defCodeGenService } from "@/api/system/code_gen";
import { defCodeGenColumnService } from "@/api/system/code_gen_column";
import { defCodeGenTableService } from "@/api/system/code_gen_table";
import type { CodeGenDatabaseColumn } from "@/rpc/system/admin/v1/code_gen_column";
import type {
  CodeGenDatabaseTable,
  CodeGenTable,
  CodeGenTableForm,
  PageCodeGenTableRequest
} from "@/rpc/system/admin/v1/code_gen_table";
import type { BaseMenu } from "@/rpc/system/admin/v1/base_menu";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";
import { BaseMenuType, CodeGenTableStatus } from "@/rpc/system/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import CodeGenProgressDialog from "../components/CodeGenProgressDialog.vue";
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
const codeGenStatusDisabled = CodeGenTableStatus.DISABLED_CGTS;

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

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<CodeGenTableFormState>({ ...createDefaultCodeGenTableForm(), parent_menu_id: undefined });

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
    options.push({ label: `${formData.business_module}（已停用）`, value: formData.business_module, disabled: true });
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
    label: "业务表名",
    component: "select",
    options: databaseTableOptions.value,
    colSpan: 24,
    labelTooltip: "选择代码生成的源数据库表。生成器据此读取字段元数据、定位数据模型和生成目标；同一张表不能重复配置。",
    props: {
      placeholder: "请选择数据库表",
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleTableNameChange
    }
  },
  {
    prop: "comment",
    label: "业务表描述",
    component: "input",
    colSpan: 24,
    labelTooltip: "业务的中文描述。生成器优先用它写入 Proto、后端和前端文案，并作为生成菜单标题；修改后在下次生成生效。",
    props: { placeholder: "选择业务表后自动带出，可修改" }
  },
  {
    prop: "business_module",
    label: "业务模块",
    component: "select",
    options: businessModuleOptions.value,
    labelTooltip: "选择业务模块后，Proto、后端服务、前端 API 与页面路径均由模块和表名自动推导。",
    props: { placeholder: "请选择业务模块", filterable: true, style: { width: "100%" } }
  },
  {
    prop: "parent_menu_id",
    label: "父级菜单",
    component: "tree-select",
    options: parentMenuOptions.value,
    labelTooltip:
      "选择一级模块目录或二级业务目录作为挂载点。仅当同时生成前端、开启“生成SQL”且页面接口完整时，生成流程才会同步页面菜单和按钮权限。",
    props: {
      placeholder: "请选择生成页面挂载菜单",
      clearable: true,
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  {
    prop: "page_type",
    label: "页面类型",
    component: "segmented",
    options: codeGenPageTypeOptions,
    labelTooltip: "普通表格生成分页 CRUD；树形表格生成完整层级列表；树形懒加载按节点请求子列表；左树右表生成左侧树筛选与右侧列表。",
    props: { onChange: handlePageTypeChange }
  },
  {
    prop: "parent_column",
    label: "父节点字段",
    component: "select",
    options: databaseColumnOptions.value,
    labelTooltip: "树形表格中指向父记录的字段，例如 parent_id。它决定生成的树查询和选项接口如何组织父子层级。",
    props: { placeholder: "请选择父节点字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => isCodeGenTreePageType(model.page_type)
  },
  {
    prop: "tree_label_column",
    label: "树显示字段",
    component: "select",
    options: databaseColumnOptions.value,
    labelTooltip: "树节点显示的文字字段，例如 name。它会写入生成的树查询和选项接口响应，并显示在前端树节点上。",
    props: { placeholder: "请选择树显示字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => isCodeGenTreePageType(model.page_type)
  },
  {
    prop: "left_tree_config.table_name",
    label: "左树数据表",
    component: "select",
    options: leftTreeTableOptions.value,
    labelTooltip: "左侧树的数据来源。它决定要调用哪个实体的树选项接口，并限定左树父、显示和值字段的可选范围。",
    props: {
      placeholder: "请选择左树数据表",
      clearable: true,
      filterable: true,
      style: { width: "100%" },
      onChange: handleLeftTreeTableNameChange
    },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.comment",
    label: "左树描述",
    component: "input",
    labelTooltip: "左侧树的中文说明。页面预览会优先显示该标题；它不改变生成接口、路由或文件路径。",
    props: { placeholder: "选择左树数据表后自动带出，可修改" },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.filter_column",
    label: "筛选字段",
    labelTooltip: "当前业务表中关联左树节点值的字段。点击左树节点后，生成页面会把节点值作为该字段的查询条件传给右侧列表。",
    component: "select",
    options: databaseColumnOptions.value,
    props: { placeholder: "请选择当前表筛选字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.parent_column",
    label: "左树父字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: "左树数据表中指向父节点的字段，例如 parent_id。它决定左侧选项接口如何返回层级 children。",
    props: { placeholder: "请选择左树父字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.label_column",
    label: "左树显示字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: "左树节点显示的文字字段，例如 name。它会映射为左侧 TreeFilter 组件的节点标签。",
    props: { placeholder: "请选择左树显示字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.value_column",
    label: "左树值字段",
    component: "select",
    options: leftTreeColumnOptions.value,
    labelTooltip: "左树节点的唯一值字段，通常为主键 id。它作为点击节点后传给右侧列表筛选字段的值。",
    props: { placeholder: "请选择左树值字段", clearable: true, filterable: true, style: { width: "100%" } },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "left_tree_config.lazy",
    label: "左树懒加载",
    component: "switch",
    labelTooltip: "开启后左树只请求当前展开节点的子节点，适合层级较深或数据量较大的树。",
    props: { activeText: "懒加载", inactiveText: "全部加载" },
    visible: model => model.page_type === "left_tree"
  },
  {
    prop: "gen_backend",
    label: "生成后端",
    component: "switch",
    labelTooltip: "开启后生成 Proto、后端 Biz/Service 及注册代码；关闭后本次任务不会写入这些后端文件。",
    // 三个生成开关始终从新行开始并排展示。
    rowBreakBefore: true,
    colSpan: 8,
    props: { activeText: "生成", inactiveText: "跳过" }
  },
  {
    prop: "gen_frontend",
    label: "生成前端",
    component: "switch",
    labelTooltip: "开启后生成前端 API 和 Vue 页面；同时它也是同步页面菜单与按钮权限的前置条件。",
    colSpan: 8,
    props: { activeText: "生成", inactiveText: "跳过" }
  },
  {
    prop: "gen_sql",
    label: "生成SQL",
    component: "switch",
    colSpan: 8,
    labelTooltip: "开启后在满足前端生成与页面接口完整的条件下，同时同步菜单和按钮权限到数据库，并更新 sql/default-data.sql 脚本。",
    props: { activeText: "生成", inactiveText: "跳过" }
  },
  {
    prop: "status",
    label: "状态",
    component: "dict",
    colSpan: 24,
    props: { code: "code_gen_table_status", codeType: "number", type: "radio" },
    labelTooltip: "草稿和已生成配置均可再次生成；停用后只能查看，生成任务会拒绝写入任何文件。成功生成后状态自动更新为“已生成”。"
  },
  {
    prop: "remark",
    label: "备注",
    component: "textarea",
    colSpan: 24,
    labelTooltip: "仅保存给维护人员的配置备注，不参与生成文件、接口、路由或权限的命名。",
    props: { placeholder: "请输入备注", rows: 3 }
  }
]);

/** 代码生成表配置列表列。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "业务表名", minWidth: 160, search: { el: "input" } },
  { prop: "comment", label: "业务表描述", minWidth: 160, showOverflowTooltip: true },
  {
    prop: "business_module",
    label: "业务模块",
    minWidth: 140,
    dictCode: "business_module",
    dictValueType: "string",
    search: { el: "select" }
  },
  { prop: "page_type", label: "页面类型", minWidth: 120, enum: codeGenPageTypeOptions, search: { el: "select" } },
  { prop: "status", label: "状态", width: 100, dictCode: "code_gen_table_status", search: { el: "select" } },
  { prop: "remark", label: "备注", minWidth: 180, showOverflowTooltip: true },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 660,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "字段配置",
        type: "success",
        link: true,
        icon: SetUp,
        hidden: () => !BUTTONS.value["tool:code-gen-table:column"],
        onClick: scope => handleOpenColumnConfig((scope.row as CodeGenTable).id)
      },
      {
        label: "Proto配置",
        type: "warning",
        link: true,
        icon: Connection,
        hidden: () => !BUTTONS.value["tool:code-gen-table:proto"],
        onClick: scope => handleOpenProtoConfig((scope.row as CodeGenTable).id)
      },
      {
        label: "页面预览",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["tool:code-gen-table:preview"],
        onClick: scope => handleOpenPreview((scope.row as CodeGenTable).id)
      },
      {
        label: "代码预览",
        type: "primary",
        link: true,
        icon: Document,
        hidden: () => !BUTTONS.value["tool:code-gen-table:code-preview"],
        onClick: scope => handleOpenCodePreview((scope.row as CodeGenTable).id)
      },
      {
        label: "生成",
        type: "success",
        link: true,
        icon: Promotion,
        disabled: () => generating.value,
        hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
        onClick: scope => handleGenerate(scope.row as CodeGenTable)
      },
      {
        label: "还原",
        type: "warning",
        link: true,
        icon: RefreshRight,
        disabled: scope => !(scope.row as CodeGenTable).restore_available,
        hidden: () => !BUTTONS.value["tool:code-gen-table:restore"],
        onClick: scope => handleRestore(scope.row as CodeGenTable)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["tool:code-gen-table:update"],
        onClick: scope => handleOpenDialog((scope.row as CodeGenTable).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
        onClick: scope => handleDelete(scope.row as CodeGenTable)
      }
    ]
  }
];

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
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["tool:code-gen-table:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "批量生成",
    type: "primary",
    icon: Promotion,
    hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
    disabled: scope => generating.value || !scope.selectedList.length,
    onClick: scope => handleGenerate(scope.selectedList as CodeGenTable[])
  },
  {
    label: "批量还原",
    type: "warning",
    icon: RefreshRight,
    hidden: () => !BUTTONS.value["tool:code-gen-table:restore"],
    disabled: scope => scope.selectedList.every(item => !(item as CodeGenTable).restore_available),
    onClick: scope => handleRestore(scope.selectedList as CodeGenTable[])
  },
  {
    label: "最近任务",
    icon: Clock,
    hidden: () => !BUTTONS.value["tool:code-gen-table:generate"],
    disabled: () => !progressTaskAvailable.value,
    onClick: handleOpenProgress
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["tool:code-gen-table:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as CodeGenTable[])
  }
];

/** 打开已经保存的代码生成文件预览。 */
async function handleOpenCodePreview(tableId: number) {
  await router.push(`/code/gen/code/preview/${tableId}`);
}

/** 创建单项或批量代码生成任务。 */
async function handleGenerate(selected: CodeGenGenerateTarget) {
  const tables = Array.isArray(selected) ? selected : [selected];
  if (!tables.length) {
    ElMessage.warning("请勾选生成项");
    return;
  }
  const disabledTable = tables.find(table => table.status === codeGenStatusDisabled);
  if (disabledTable) {
    ElMessage.warning(`代码生成表配置 ${disabledTable.name} 已停用`);
    return;
  }
  const message = tables.length === 1 ? `确认生成业务表：${tables[0].name}？` : `确认按勾选顺序生成 ${tables.length} 个业务表？`;
  try {
    await ElMessageBox.confirm(message, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
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
    ElMessage.warning("请选择有可用还原快照的代码生成项");
    return;
  }
  const message =
    restorableTables.length === 1
      ? `确认还原业务表：${restorableTables[0].name}？生成文件、页面菜单和按钮权限将恢复到生成前状态。`
      : `确认批量还原 ${restorableTables.length} 个业务表？批量生成的共享文件必须整批还原。`;
  try {
    await ElMessageBox.confirm(message, "警告", {
      confirmButtonText: "确认还原",
      cancelButtonText: "取消",
      type: "warning"
    });
  } catch {
    return;
  }
  await defCodeGenService.RestoreCodeGen({ table_ids: restorableTables.map(table => table.id) });
  ElMessage.success("还原代码生成结果成功");
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
    dialog.title = "编辑代码生成表配置";
  } else {
    dialog.title = "新增代码生成表配置";
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
      ElMessage.success("编辑代码生成表配置成功");
    } else {
      await defCodeGenTableService.CreateCodeGenTable({ code_gen_table: payload });
      ElMessage.success("新增代码生成表配置成功");
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
    ElMessage.warning("请勾选删除项");
    return;
  }
  const confirmMessage =
    tableList.length === 1 ? `确认删除业务表：${tableList[0].name}？` : `确认删除已选中的 ${tableList.length} 条配置？`;
  try {
    await ElMessageBox.confirm(confirmMessage, "警告", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    });
  } catch {
    ElMessage.info("已取消删除代码生成表配置");
    return;
  }
  await defCodeGenTableService.DeleteCodeGenTable({ ids: tableIds });
  ElMessage.success("删除代码生成表配置成功");
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
    .filter(item => item.type === BaseMenuType.FOLDER)
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
