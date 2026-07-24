<!-- 代码生成完整页面效果预览 -->
<template>
  <div v-loading="loading" class="app-container code-gen-page-preview-page">
    <template v-if="snapshot">
      <section v-if="pageType === 'left_tree'" class="main-box code-gen-left-tree-preview">
        <TreeFilter
          id="value"
          label="label"
          :title="leftTreeTitle"
          :data="leftTreeLazy ? undefined : leftTreeOptions"
          :request-api="leftTreeLazy ? requestPreviewLeftTree : undefined"
          :lazy="leftTreeLazy"
          :parent-key="leftTreeParentColumn"
          :show-all="false"
          @change="handleLeftTreeChange"
        />
        <div class="code-gen-left-tree-preview__table table-box">
          <ProTable
            :key="previewTableKey"
            ref="proTable"
            :row-key="primaryColumn"
            :columns="tableColumns"
            :header-actions="headerActions"
            :request-api="requestPreviewTable"
            scrollbar-always-on
            class="code-gen-page-preview-table"
          />
        </div>
      </section>

      <div v-else class="code-gen-page-preview__table table-box">
        <ProTable
          :key="previewTableKey"
          ref="proTable"
          :row-key="primaryColumn"
          :columns="tableColumns"
          :header-actions="headerActions"
          :request-api="requestPreviewTable"
          :pagination="pageType !== 'tree' && pageType !== 'tree_lazy'"
          :indent="20"
          :lazy="pageType === 'tree_lazy'"
          :load="pageType === 'tree_lazy' ? loadPreviewTreeChildren : undefined"
          :tree-props="
            pageType === 'tree' || pageType === 'tree_lazy'
              ? { children: 'children', hasChildren: pageType === 'tree_lazy' ? 'has_children' : 'hasChildren' }
              : undefined
          "
          scrollbar-always-on
          class="code-gen-page-preview-table"
        />
      </div>

      <FormDialog
        v-model="dialog.visible"
        ref="formDialogRef"
        :title="dialog.title"
        width="min(920px, calc(100vw - 32px))"
        top="4vh"
        :model="previewFormModel"
        :fields="formFields"
        label-width="116px"
        :gutter="20"
        :col-span="12"
        @confirm="handleSubmit"
        @close="handleCloseDialog"
      >
        <template #passwordStrength>
          <PasswordStrength :password="String(previewFormModel[passwordFieldName])" />
        </template>
        <template #codeGenPreviewSlot="{ field }">
          <el-input v-model="previewFormModel[field.prop]" placeholder="自定义插槽内容">
            <template #append>自定义</template>
          </el-input>
        </template>
      </FormDialog>
    </template>
    <el-empty v-else-if="!loading" description="暂无页面配置" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import ProTable from "@/components/ProTable/index.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import PasswordStrength from "@/components/PasswordStrength/index.vue";
import type { ProFormComponentType, ProFormField, ProFormOption } from "@/components/ProForm/interface";
import TreeFilter from "@/components/TreeFilter/index.vue";
import { useTabsStore } from "@/stores/modules/tabs";
import { defBaseDictService } from "@/api/system/base_dict";
import { defCodeGenColumnService } from "@/api/system/code_gen_column";
import { defCodeGenProtoService } from "@/api/system/code_gen_proto";
import { defCodeGenTableService } from "@/api/system/code_gen_table";
import type { OptionBaseDictResponse_BaseDict } from "@/rpc/system/admin/v1/base_dict";
import type { CodeGenColumn, CodeGenColumnOptionConfig } from "@/rpc/system/admin/v1/code_gen_column";
import type { CodeGenProtoCheck } from "@/rpc/system/admin/v1/code_gen_proto";
import type { CodeGenDatabaseTable } from "@/rpc/system/admin/v1/code_gen_table";
import { codeGenFormComponentOptions } from "../config";
import { resolveCodeGenPreviewCapabilities } from "./capabilities";
import {
  buildCodeGenPreviewTree,
  createCodeGenLeftTreeOptions,
  createCodeGenPreviewOptionMap,
  createCodeGenPreviewRows,
  filterCodeGenPreviewRows,
  flattenCodeGenPreviewOptions,
  isNumericColumn,
  resolveCodeGenPreviewOptions,
  resolveCodeGenPrimaryColumn,
  type CodeGenPagePreviewSnapshot,
  type CodeGenPreviewRow
} from "./data";

defineOptions({
  name: "CodeGenPreview",
  inheritAttrs: false
});

const route = useRoute();
const tabsStore = useTabsStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const loading = ref(false);
const snapshot = ref<CodeGenPagePreviewSnapshot | null>(null);
const protoChecks = ref<CodeGenProtoCheck[]>([]);
const dictionaries = ref<OptionBaseDictResponse_BaseDict[]>([]);
const databaseTables = ref<CodeGenDatabaseTable[]>([]);
const mockRows = ref<CodeGenPreviewRow[]>([]);
const previewFormModel = reactive<Record<string, any>>({});
const editingRowKey = ref<string | number>();
const selectedLeftTreeValues = ref<Array<string | number | boolean>>([]);
const passwordFieldName = computed(() => snapshot.value?.columns.find(column => column.form_config?.component === "password")?.name ?? "pwd");
const supportedFormComponents = new Set(codeGenFormComponentOptions.map(item => String(item.value)));

const dialog = reactive({
  title: "",
  visible: false
});

/** 当前代码生成表配置 ID。 */
const tableId = computed(() => {
  const value = route.params.tableId;
  const id = Number(Array.isArray(value) ? value[0] : value);
  return Number.isFinite(id) && id > 0 ? id : 0;
});

/** 当前页面类型。 */
const pageType = computed(() => snapshot.value?.table.page_type || "normal");

/** 页面预览表格重建键。 */
const previewTableKey = computed(() => `${tableId.value}:${pageType.value}`);

/** 当前真实主键字段。 */
const primaryColumn = computed(() => resolveCodeGenPrimaryColumn(snapshot.value?.columns ?? []));

/** 当前字段配置对应的全部模拟选项。 */
const optionMap = computed(() => createCodeGenPreviewOptionMap(snapshot.value?.columns ?? [], dictionaries.value));

/** 左树右表页面的模拟节点。 */
const leftTreeOptions = computed(() => createCodeGenLeftTreeOptions(snapshot.value?.table.left_tree_config));

/** 左树是否启用懒加载。 */
const leftTreeLazy = computed(() => pageType.value === "left_tree" && Boolean(snapshot.value?.table.left_tree_config?.lazy));

/** 左树懒加载请求使用的父节点字段。 */
const leftTreeParentColumn = computed(() => snapshot.value?.table.left_tree_config?.parent_column || "parent_id");

/** 旧配置缺少左树描述时，从数据库表元数据中读取描述。 */
const leftTreeTableComment = computed(() => {
  const tableName = snapshot.value?.table.left_tree_config?.table_name;
  return databaseTables.value.find(item => item.name === tableName)?.comment || "";
});

/** 左树标题优先使用可编辑描述，旧配置使用数据表描述补齐。 */
const leftTreeTitle = computed(
  () =>
    snapshot.value?.table.left_tree_config?.comment ||
    leftTreeTableComment.value ||
    snapshot.value?.table.left_tree_config?.table_name ||
    "分类列表"
);

/** 当前实体已经存在或已经选择生成的 Proto 维护能力。 */
const protoCapabilities = computed(() =>
  resolveCodeGenPreviewCapabilities(toPascalCase(snapshot.value?.table.name ?? ""), protoChecks.value)
);

/** 根据字段配置生成最终页面的查询项和列表列。 */
const tableColumns = computed<ColumnProps[]>(() => {
  const columns = snapshot.value?.columns ?? [];
  const treeParentColumn = pageType.value === "tree" || pageType.value === "tree_lazy" ? snapshot.value?.table.parent_column : "";
  const treeLabelColumn = pageType.value === "tree" || pageType.value === "tree_lazy" ? snapshot.value?.table.tree_label_column : "";
  const configuredColumns = columns
    .filter(
      column =>
        column.name !== "deleted_at" &&
        (column.list_config?.enabled ||
          column.query_config?.enabled ||
          column.name === treeLabelColumn ||
          column.name === treeParentColumn)
    )
    .sort((left, right) => left.sort - right.sort)
    .sort((left, right) => {
      if (left.name === treeLabelColumn) return -1;
      if (right.name === treeLabelColumn) return 1;
      return 0;
    })
    .map(column => {
      const isTreeLabel = column.name === treeLabelColumn;
      const isTreeParent = column.name === treeParentColumn && treeParentColumn !== treeLabelColumn;
      const listConfig = column.list_config ?? { enabled: false, component: "input", option: undefined };
      const previewColumn = isTreeLabel && !listConfig.enabled
        ? { ...column, list_config: { ...listConfig, enabled: true } }
        : column;
      const result = createPreviewTableColumn(previewColumn);
      if (isTreeLabel) result.align = "left";
      if (isTreeParent) {
        result.isShow = false;
        result.isSetting = false;
      }
      return result;
    });
  const result: ColumnProps[] = [...configuredColumns];
  if (protoCapabilities.value.delete) result.unshift({ type: "selection", width: 55, fixed: "left" });
  const actions: NonNullable<ColumnProps["actions"]> = [];
  if (protoCapabilities.value.update) {
    actions.push({
      label: "编辑",
      type: "primary",
      link: true,
      icon: EditPen,
      onClick: scope => handleOpenDialog(scope.row as CodeGenPreviewRow)
    });
  }
  if (protoCapabilities.value.delete) {
    actions.push({
      label: "删除",
      type: "danger",
      link: true,
      icon: Delete,
      onClick: scope => handleDelete([scope.row as CodeGenPreviewRow])
    });
  }
  if (actions.length) {
    result.push({
      prop: "operation",
      label: "操作",
      width: actions.length === 1 ? 100 : 180,
      fixed: "right",
      cellType: "actions",
      actions
    });
  }
  return result;
});

/** 页面预览表格顶部操作。 */
const headerActions = computed<HeaderActionProps[]>(() => {
  const actions: HeaderActionProps[] = [];
  if (protoCapabilities.value.create && formFields.value.length) {
    actions.push({
      label: "新增",
      type: "success",
      icon: CirclePlus,
      onClick: () => handleOpenDialog()
    });
  }
  if (protoCapabilities.value.delete) {
    actions.push({
      label: "删除",
      type: "danger",
      icon: Delete,
      disabled: scope => !scope.selectedList.length,
      onClick: scope => handleDelete(scope.selectedList as CodeGenPreviewRow[])
    });
  }
  return actions;
});

/** 根据真实表单字段配置生成新增、编辑弹窗。 */
const formFields = computed<ProFormField[]>(() => {
  return (snapshot.value?.columns ?? [])
    .filter(
      column =>
        !column.is_primary && !column.is_auto_increment && column.name !== "deleted_at" && column.form_config?.enabled
    )
    .sort((left, right) => left.sort - right.sort)
    .flatMap(column => {
      const label = column.comment || column.name;
      const isTreeParent =
        (pageType.value === "tree" || pageType.value === "tree_lazy") && column.name === snapshot.value?.table.parent_column;
      const component = isTreeParent ? "tree-select" : resolvePreviewFormComponent(column.form_config?.component);
      const isMultipleTreeSelect = component === "tree-select" && Boolean(column.form_config?.multiple);
      const options = isTreeParent
        ? treeParentOptions.value
        : resolveCodeGenPreviewOptions(optionMap.value, column.name, "form");
      const field: ProFormField = {
        prop: column.name,
        label,
        component,
        props: {
          ...createPreviewFormProps(component, label, column.form_config?.option, isMultipleTreeSelect),
          ...(isTreeParent ? { disabled: Boolean(previewFormModel[primaryColumn.value]) } : {})
        },
        options,
        checkboxLabel: component === "checkbox" ? `启用${label}` : undefined,
        slotName: component === "slot" ? "codeGenPreviewSlot" : undefined,
        visible: component === "password" ? model => !model[primaryColumn.value] : undefined,
        rules: column.form_config?.required ? [{ required: true, message: `${label}不能为空` }] : undefined,
        colSpan: resolvePreviewColSpan(component)
      };
      if (component !== "password") return [field];
      return [
        field,
        {
          prop: "passwordStrength",
          label: "强度提示",
          component: "slot",
          slotName: "passwordStrength",
          visible: model => !model[primaryColumn.value]
        }
      ];
    });
});

/** 树形表格新增弹窗中的父节点选项。 */
const treeParentOptions = computed<ProFormOption[]>(() => {
  if (!snapshot.value?.table.parent_column || (pageType.value !== "tree" && pageType.value !== "tree_lazy")) return [];
  const treeRows = buildCodeGenPreviewTree(mockRows.value, primaryColumn.value, snapshot.value.table.parent_column);
  return [{ label: "顶级节点", value: 0 }, ...mapPreviewRowsToOptions(treeRows)];
});

// 路由生成对象变化时重新载入对应预览。
watch(tableId, () => {
  void loadPreview();
});

/** 请求左树预览节点，懒加载时只返回当前父节点的直接子节点。 */
async function requestPreviewLeftTree(params?: Record<string, any>) {
  const parentValue = params?.[leftTreeParentColumn.value] ?? 0;
  return { data: findPreviewTreeChildren(leftTreeOptions.value, parentValue) };
}

/** 查找模拟树中指定父节点的直接子节点。 */
function findPreviewTreeChildren(options: ProFormOption[], parentValue: unknown): ProFormOption[] {
  if (String(parentValue) === "0") {
    return options.map(({ children: _children, ...option }) => option);
  }
  for (const option of options) {
    if (String(option.value) === String(parentValue)) {
      return (option.children ?? []).map(({ children: _children, ...child }) => child);
    }
    const nested = findPreviewTreeChildren(option.children ?? [], parentValue);
    if (nested.length) return nested;
  }
  return [];
}

/** 请求前端模拟列表，并复用最终页面的查询与分页交互。 */
async function requestPreviewTable(params: Record<string, any>) {
  const columns = snapshot.value?.columns ?? [];
  let rows = filterCodeGenPreviewRows(mockRows.value, columns, params);
  if (
    pageType.value === "left_tree" &&
    snapshot.value?.table.left_tree_config?.filter_column &&
    selectedLeftTreeValues.value.length
  ) {
    const filterColumn = snapshot.value.table.left_tree_config.filter_column;
    rows = rows.filter(row => selectedLeftTreeValues.value.some(value => String(value) === String(row[filterColumn])));
  }
  if (pageType.value === "tree_lazy" && snapshot.value?.table.parent_column) {
    const parentColumn = snapshot.value.table.parent_column;
    const parentValue = params[parentColumn] ?? 0;
    const allRows = mockRows.value;
    rows = rows
      .filter(row => String(row[parentColumn] ?? 0) === String(parentValue))
      .map(row => ({
        ...row,
        has_children: allRows.some(child => String(child[parentColumn] ?? 0) === String(row[primaryColumn.value]))
      }));
    return { data: rows };
  }
  if (pageType.value === "tree" && snapshot.value?.table.parent_column) {
    return { data: buildCodeGenPreviewTree(rows, primaryColumn.value, snapshot.value.table.parent_column) };
  }
  const pageNum = Number(params.pageNum ?? 1);
  const pageSize = Number(params.pageSize ?? 10);
  const start = (pageNum - 1) * pageSize;
  return { data: { list: rows.slice(start, start + pageSize), total: rows.length } };
}

/** 请求懒加载树表格的子节点。 */
async function loadPreviewTreeChildren(
  row: CodeGenPreviewRow,
  _treeNode: unknown,
  resolve: (data: CodeGenPreviewRow[]) => void
) {
  const parentColumn = snapshot.value?.table.parent_column;
  if (!parentColumn) {
    resolve([]);
    return;
  }
  const response = await requestPreviewTable({ [parentColumn]: row[primaryColumn.value], lazy: true });
  resolve(Array.isArray(response.data) ? response.data : []);
}

/** 刷新预览表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 加载当前表、字段和 Proto 配置并创建页面预览。 */
async function loadPreview() {
  loading.value = true;
  try {
    snapshot.value = null;
    protoChecks.value = [];
    dictionaries.value = [];
    databaseTables.value = [];
    if (!tableId.value) return;
    const [table, columnResponse, protoResponse, dictionaryResponse, databaseTableResponse] = await Promise.all([
      defCodeGenTableService.GetCodeGenTable({ id: tableId.value }),
      defCodeGenColumnService.ListCodeGenColumn({ table_id: tableId.value }),
      defCodeGenProtoService.ListCodeGenProto({ table_id: tableId.value }),
      defBaseDictService.OptionBaseDict({}),
      defCodeGenTableService.ListCodeGenDatabaseTable({})
    ]);
    snapshot.value = { table, columns: columnResponse.code_gen_columns ?? [] };
    protoChecks.value = protoResponse.code_gen_protos ?? [];
    dictionaries.value = dictionaryResponse.base_dicts ?? [];
    databaseTables.value = databaseTableResponse.tables ?? [];
    createMockRows();
    syncWorkspaceTitle();
  } finally {
    loading.value = false;
  }
}

/** 根据 TreeFilter 当前节点筛选该节点及其全部子节点。 */
function handleLeftTreeChange(value: string | number | boolean | undefined) {
  const selectedNode = flattenCodeGenPreviewOptions(leftTreeOptions.value).find(option => String(option.value) === String(value));
  selectedLeftTreeValues.value = selectedNode
    ? flattenCodeGenPreviewOptions([selectedNode]).map(option => option.value)
    : [];
  proTable.value?.search();
}

/** 打开新增或编辑模拟记录弹窗。 */
function handleOpenDialog(row?: CodeGenPreviewRow) {
  resetPreviewForm(row);
  editingRowKey.value = row?.[primaryColumn.value];
  dialog.title = row ? `编辑${snapshot.value?.table.comment || "数据"}` : `新增${snapshot.value?.table.comment || "数据"}`;
  dialog.visible = true;
}

/** 关闭模拟表单并清理编辑上下文。 */
function handleCloseDialog() {
  dialog.visible = false;
  editingRowKey.value = undefined;
  resetPreviewForm();
}

/** 重置预览表单并按组件类型写入结构正确的初始值。 */
function resetPreviewForm(row?: CodeGenPreviewRow) {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.keys(previewFormModel).forEach(key => delete previewFormModel[key]);
  formFields.value.forEach(field => {
    previewFormModel[field.prop] = row ? resolvePreviewFormValue(field, row[field.prop]) : createPreviewFormValue(field);
  });
}

/** 提交模拟新增或编辑，并刷新当前列表布局。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(async valid => {
    if (!valid || !snapshot.value) return;
    if (editingRowKey.value !== undefined) {
      const rowIndex = mockRows.value.findIndex(row => String(row[primaryColumn.value]) === String(editingRowKey.value));
      if (rowIndex >= 0) mockRows.value[rowIndex] = { ...mockRows.value[rowIndex], ...clonePreviewValue(previewFormModel) };
    } else {
      const template = createCodeGenPreviewRows(snapshot.value, optionMap.value, leftTreeOptions.value)[0] ?? {};
      const nextPrimaryValue = createNextPrimaryValue();
      const newRow = { ...template, ...clonePreviewValue(previewFormModel), [primaryColumn.value]: nextPrimaryValue };
      if (
        pageType.value === "left_tree" &&
        snapshot.value.table.left_tree_config?.filter_column &&
        selectedLeftTreeValues.value.length
      ) {
        newRow[snapshot.value.table.left_tree_config.filter_column] = selectedLeftTreeValues.value[0];
      }
      mockRows.value.unshift(newRow);
    }
    const successMessage = editingRowKey.value !== undefined ? "修改成功" : "新增成功";
    handleCloseDialog();
    await nextTick();
    refreshTable();
    ElMessage.success(successMessage);
  });
}

/** 删除一条或多条模拟记录。 */
async function handleDelete(rows: CodeGenPreviewRow[]) {
  if (!rows.length) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  try {
    await ElMessageBox.confirm(`确认删除选中的 ${rows.length} 条数据吗？`, "删除确认", {
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      type: "warning"
    });
  } catch {
    return;
  }
  const keys = new Set(rows.map(row => String(row[primaryColumn.value])));
  mockRows.value = mockRows.value.filter(row => !keys.has(String(row[primaryColumn.value])));
  await nextTick();
  refreshTable();
  ElMessage.success("删除成功");
}

/** 创建当前页面类型使用的完整模拟数据。 */
function createMockRows() {
  mockRows.value = snapshot.value ? createCodeGenPreviewRows(snapshot.value, optionMap.value, leftTreeOptions.value) : [];
  selectedLeftTreeValues.value = [];
}

/** 同步预览页签和浏览器标题。 */
function syncWorkspaceTitle() {
  const title = snapshot.value?.table.comment || snapshot.value?.table.name || "数据列表";
  tabsStore.setTabsTitle(title);
  document.title = `${title} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

/** 将数据库表名转换为生成器使用的实体名。 */
function toPascalCase(value: string) {
  return value
    .split("_")
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

/** 创建最终 ProTable 单列配置。 */
function createPreviewTableColumn(column: CodeGenColumn): ColumnProps {
  const label = column.comment || column.name;
  const listOptions = resolveCodeGenPreviewOptions(optionMap.value, column.name, "list");
  const queryOptions = resolveCodeGenPreviewOptions(optionMap.value, column.name, "query");
  const result: ColumnProps = {
    prop: column.name,
    label,
    minWidth: resolvePreviewColumnWidth(column),
    isShow: Boolean(column.list_config?.enabled),
    isSetting: Boolean(column.list_config?.enabled),
    align: resolvePreviewColumnAlign(column)
  };
  if (column.query_config?.enabled) {
    result.search = {
      el: resolvePreviewSearchComponent(column.query_config.component),
      props: createPreviewSearchProps(column)
    };
    if (queryOptions.length) result.enum = queryOptions;
  }
  applyPreviewListComponent(result, column, listOptions);
  return result;
}

/** 根据字段类型和选项数据源解析预览表格列对齐方式。 */
function resolvePreviewColumnAlign(column: CodeGenColumn): NonNullable<ColumnProps["align"]> {
  const listComponent = column.list_config?.component;
  if (listComponent === "money") return "right";
  if (listComponent === "image") return "center";

  const optionSources = [column.query_config?.option?.source_type, column.list_config?.option?.source_type].filter(Boolean);
  if (optionSources.includes("table")) return "left";
  if (optionSources.includes("dict") || optionSources.includes("static")) return "center";
  if (isNumericColumn(column)) return "right";
  return "left";
}

/** 将列表展示组件映射为 ProTable 列能力。 */
function applyPreviewListComponent(result: ColumnProps, column: CodeGenColumn, options: ProFormOption[]) {
  const component = column.list_config?.component;
  if (component === "image") {
    result.cellType = "image";
    result.width = 120;
    result.imageProps = {
      width: 52,
      height: 52,
      src: scope => {
        const value = scope.row[column.name];
        return Array.isArray(value) ? String(value[0] ?? "") : String(value ?? "");
      }
    };
    return;
  }
  if (component === "money") {
    result.cellType = "money";
    result.align = "right";
    return;
  }
  if (component === "switch") {
    result.width = 110;
    result.render = scope => renderPreviewOptionValue(scope, column.name, options);
    return;
  }
  if (options.length) {
    result.render = scope => renderPreviewOptionValue(scope, column.name, options);
  }
}

/** 渲染列表选择值，树形子选项同样可以正确匹配。 */
function renderPreviewOptionValue(scope: RenderScope, columnName: string, options: ProFormOption[]) {
  const flatOptions = flattenCodeGenPreviewOptions(options);
  const value = scope.row[columnName];
  const matched = flatOptions.find(option => String(option.value) === String(value));
  return matched?.label || String(value ?? "--");
}

/** 还原编辑态字段值，保证多选树形选择始终接收数组。 */
function resolvePreviewFormValue(field: ProFormField, value: unknown) {
  const props = field.props && typeof field.props !== "function" ? field.props : {};
  if (field.component !== "tree-select" || !props.multiple || Array.isArray(value)) return clonePreviewValue(value);
  if (typeof value !== "string") return [];
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

/** 创建新增记录不与现有主键冲突的模拟编号。 */
function createNextPrimaryValue() {
  const values = mockRows.value.map(row => Number(row[primaryColumn.value])).filter(Number.isFinite);
  if (values.length === mockRows.value.length) return Math.max(0, ...values) + 1;
  return `record-${Date.now()}`;
}

/** 将树形模拟记录转换成父节点选择项。 */
function mapPreviewRowsToOptions(rows: CodeGenPreviewRow[], parentPath = ""): ProFormOption[] {
  const labelColumn = snapshot.value?.table.tree_label_column || primaryColumn.value;
  return rows.map(row => {
    const label = String(row[labelColumn] ?? row[primaryColumn.value]);
    const fullPath = [parentPath, label].filter(Boolean).join("/");
    return {
      label: fullPath,
      value: row[primaryColumn.value],
      children: row.children?.length ? mapPreviewRowsToOptions(row.children, fullPath) : undefined
    };
  });
}

/** 将配置中的组件字符串收敛为 ProForm 支持类型，字典预览使用模拟下拉避免接口请求。 */
function resolvePreviewFormComponent(component?: string): ProFormComponentType {
  if (component === "dict") return "select";
  return component && supportedFormComponents.has(component) ? (component as ProFormComponentType) : "input";
}

/** 创建不同 ProForm 组件在最终新增弹窗中的参数。 */
function createPreviewFormProps(
  component: ProFormComponentType,
  label: string,
  option?: CodeGenColumnOptionConfig,
  isMultipleTreeSelect = false
) {
  const fullWidthStyle = { width: "100%" };
  switch (component) {
    case "input":
      return { placeholder: `请输入${label}`, clearable: true, style: fullWidthStyle };
    case "password":
      return { placeholder: `请输入${label}`, clearable: true, showPassword: true, style: fullWidthStyle };
    case "textarea":
      return { placeholder: `请输入${label}`, rows: 4 };
    case "input-number":
      return { min: 0, controlsPosition: "right", style: fullWidthStyle };
    case "segmented":
      return { block: true };
    case "select":
      return { placeholder: `请选择${label}`, clearable: true, filterable: true, checkStrictly: true, style: fullWidthStyle };
    case "tree-select":
      return {
        placeholder: `请选择${label}`,
        clearable: true,
        filterable: true,
        checkStrictly: true,
        ...(isMultipleTreeSelect ? { multiple: true, showCheckbox: true, nodeKey: "value" } : {}),
        style: fullWidthStyle
      };
    case "date-picker":
      return { type: "datetime", placeholder: `请选择${label}`, style: fullWidthStyle };
    case "transfer":
      return { titles: ["可选项", "已选项"] };
    case "image-upload":
    case "images-upload":
    case "file-upload":
    case "files-upload":
      return { disabled: true };
    case "dynamic-list":
      return { inputProps: { placeholder: `请输入${label}` } };
    case "kv-list":
      return { keyInputProps: { placeholder: "键" }, valueInputProps: { placeholder: "值" } };
    default:
      return option?.source_value ? { placeholder: option.source_value } : {};
  }
}

/** 将查询组件映射为 SearchForm 支持类型。 */
function resolvePreviewSearchComponent(component?: string) {
  if (["input", "input-number", "select", "tree-select", "date-picker"].includes(component || "")) return component as any;
  return "input";
}

/** 创建查询组件参数，区间查询使用日期范围。 */
function createPreviewSearchProps(column: CodeGenColumn) {
  const props: Record<string, any> = { clearable: true, style: { width: "100%" } };
  if (column.query_config?.component === "date-picker") {
    props.type = column.query_config.operator === "between" ? "datetimerange" : "datetime";
    props.rangeSeparator = "至";
    props.startPlaceholder = "开始时间";
    props.endPlaceholder = "结束时间";
  }
  if (column.query_config?.component === "tree-select") {
    props.checkStrictly = true;
    props.renderAfterExpand = false;
  }
  return props;
}

/** 创建不同组件的空白新增表单初始值。 */
function createPreviewFormValue(field: ProFormField) {
  switch (field.component) {
    case "input-number":
    case "segmented":
    case "select":
    case "radio-group":
    case "date-picker":
      return undefined;
    case "tree-select":
      return field.props && typeof field.props !== "function" && field.props.multiple ? [] : undefined;
    case "switch":
    case "checkbox":
      return false;
    case "checkbox-group":
    case "transfer":
    case "images-upload":
    case "files-upload":
    case "dynamic-list":
    case "kv-list":
      return [];
    default:
      return "";
  }
}

/** 宽内容组件占满整行，其余组件在桌面端双列展示。 */
function resolvePreviewColSpan(component: ProFormComponentType) {
  return new Set([
    "textarea",
    "checkbox-group",
    "transfer",
    "image-upload",
    "images-upload",
    "file-upload",
    "files-upload",
    "rich-text",
    "dynamic-list",
    "kv-list",
    "slot"
  ]).has(component)
    ? 24
    : 12;
}

/** 根据列表组件和字段名称分配稳定列宽。 */
function resolvePreviewColumnWidth(column: CodeGenColumn) {
  if (["created_at", "updated_at"].includes(column.name) || column.list_config?.component === "date") return 180;
  if (column.list_config?.component === "image") return 120;
  if (column.list_config?.component === "switch") return 110;
  return 150;
}

/** 深拷贝模拟表单和行数据，避免编辑时直接污染列表。 */
function clonePreviewValue<T>(value: T): T {
  if (value === undefined || value === null) return value;
  return JSON.parse(JSON.stringify(value)) as T;
}

onMounted(() => {
  void loadPreview();
});
</script>

<style scoped lang="scss">
.code-gen-page-preview-page {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.code-gen-page-preview__table,
.code-gen-left-tree-preview {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.code-gen-left-tree-preview__table {
  min-width: 0;
  min-height: 0;
}

:deep(.code-gen-page-preview-table) {
  --el-table-header-bg-color: var(--el-fill-color-light);
}

@media (width <= 768px) {
  :deep(.el-dialog .el-col-12) {
    max-width: 100%;
    flex: 0 0 100%;
  }
}
</style>
