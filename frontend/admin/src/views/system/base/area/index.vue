<!-- 行政区域 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseAreaTable"
      :pagination="false"
      :indent="20"
      :lazy="true"
      :load="loadAreaChildren"
      :tree-props="{ children: 'children', hasChildren: 'has_children' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="640px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseAreaService } from "@/api/system/base_area";

import type { TreeBaseAreaRequest, BaseArea, BaseAreaForm } from "@/rpc/system/admin/v1/base_area";

import { normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseArea",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseAreaForm>({
  id: 0,
  parent_id: 0,
  name: ""
});

const rules = reactive<FormRules>({
  parent_id: [{ required: true, message: "父级区域不能为空", trigger: "change" }],
  name: [{ required: true, message: "区域名称不能为空", trigger: "blur" }, { max: 50, message: "区域名称不能超过 50 个字符", trigger: "blur" }]
});
const parentIdFormOptions = ref<ProFormOption[]>([]);

type GeneratedTreeOption = ProFormOption & {
  has_children?: boolean;
  isLeaf?: boolean;
};

/** 将树形接口返回的子节点转换为 Element Plus 懒加载节点。 */
function normalizeLazyTreeOptions(options: GeneratedTreeOption[] = []): ProFormOption[] {
  return options.map(option => ({
    ...option,
    isLeaf: !option.has_children
  }));
}

/** 懒加载parentIdForm树形选项的子节点。 */
async function loadParentIdFormTreeOptions(node: { level: number; value?: string | number; data?: { value?: string | number } }, resolve: (data: ProFormOption[]) => void) {
  const parentId = node.level === 0 ? 0 : Number(node.data?.value ?? node.value ?? 0);
  const response = await defBaseAreaService.OptionBaseArea({ "parent_id": parentId, lazy: true } as Parameters<typeof defBaseAreaService.OptionBaseArea>[0]);
  resolve(normalizeLazyTreeOptions((response.list ?? []) as GeneratedTreeOption[]));
}


void loadFormOptions();

/** 行政区域表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  { prop: "parent_id", label: "父级区域", component: "tree-select", options: parentIdFormOptions.value, props: { lazy: true, load: loadParentIdFormTreeOptions, placeholder: "请选择", filterable: true, style: { width: "100%" } } },
  { prop: "name", label: "区域名称", component: "input", props: { placeholder: "请输入区域名称" } }
]);

/** 行政区域表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: "区域名称", align: "left", search: { el: "input" } },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:area:update"],
        onClick: scope => handleOpenDialog((scope.row as BaseArea).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:area:delete"],
        onClick: scope => handleDelete(scope.row as BaseArea)
      }
    ]
  }
]);

/** 行政区域顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:area:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:area:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseArea[])
  }
];

/**
 * 请求行政区域列表，并适配 ProTable 固定列表字段。
 */
async function requestBaseAreaTable(params: TreeBaseAreaRequest) {
  const data = await defBaseAreaService.TreeBaseArea({ ...params, parent_id: params.parent_id ?? 0 });
  return { data: data.base_areas ?? [] };
}

/**
 * 懒加载行政区域表格的子节点。
 */
async function loadAreaChildren(row: BaseArea, _treeNode: unknown, resolve: (data: BaseArea[]) => void) {
  try {
    const data = await defBaseAreaService.TreeBaseArea({ parent_id: row.id, lazy: true });
    resolve(data.base_areas ?? []);
  } catch {
    ElMessage.error("加载行政区域子节点失败");
    resolve([]);
  }
}

/**
 * 刷新行政区域表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}
/** 加载表单选择项。 */
async function loadFormOptions() {
  const parentIdFormResponse = await defBaseAreaService.OptionBaseArea({ "parent_id": 0, lazy: true } as Parameters<typeof defBaseAreaService.OptionBaseArea>[0]);
  parentIdFormOptions.value = [{ label: "顶级节点", value: 0 }, ...normalizeLazyTreeOptions((parentIdFormResponse.list ?? []) as GeneratedTreeOption[]).filter(option => Number(option.value) !== 0)];
}

/**
 * 打开行政区域弹窗。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
  await loadFormOptions();
  dialog.title = id ? "修改行政区域" : "新增行政区域";
  dialog.visible = true;
  if (!id) return;

  const data = await defBaseAreaService.GetBaseArea({ id });
  Object.assign(formData, data);
}
/**
 * 关闭行政区域弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
/**
 * 重置行政区域表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.parent_id = 0;
  formData.name = "";
}
/**
 * 提交行政区域表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const payload = JSON.parse(JSON.stringify(formData)) as BaseAreaForm;
    const request = payload.id
      ? defBaseAreaService.UpdateBaseArea({ id: payload.id, base_area: payload })
      : defBaseAreaService.CreateBaseArea({ base_area: payload });
    request.then(() => {
      ElMessage.success(payload.id ? "修改行政区域成功" : "新增行政区域成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 删除行政区域，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseArea | BaseArea[]) {
  const rowList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseArea[])
    : selected && typeof selected === "object"
      ? [selected as BaseArea]
      : [];
  const ids = (
    rowList.length ? rowList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!ids) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = rowList.length === 1 ? "是否确定删除行政区域？" : "确认删除已选中的行政区域吗？";
  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseAreaService.DeleteBaseArea({ ids }).then(() => {
        ElMessage.success("删除行政区域成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除行政区域");
    }
  );
}
</script>
