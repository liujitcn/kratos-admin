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
      :title="t(dialog.titleKey)"
      width="640px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseAreaService } from "@liujitcn/kratos-admin-system/api/system/base_area";

import type { TreeBaseAreaRequest, BaseArea, BaseAreaForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_area";

import { normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseArea",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  titleKey: "system.base.area.action.create",
  visible: false
});

const formData = reactive<BaseAreaForm>({
  id: 0,
  parent_id: 0,
  name: ""
});

const rules = computed<FormRules>(() => ({
  parent_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.area.field.parent") }),
      trigger: "change"
    }
  ],
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.area.field.name") }),
      trigger: "blur"
    },
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.area.field.name"), max: 50 }),
      trigger: "blur"
    }
  ]
}));
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
async function loadParentIdFormTreeOptions(
  node: { level: number; value?: string | number; data?: { value?: string | number } },
  resolve: (data: ProFormOption[]) => void
) {
  const parentId = node.level === 0 ? 0 : Number(node.data?.value ?? node.value ?? 0);
  const response = await defBaseAreaService.OptionBaseArea({ parent_id: parentId, lazy: true } as Parameters<
    typeof defBaseAreaService.OptionBaseArea
  >[0]);
  resolve(normalizeLazyTreeOptions((response.list ?? []) as GeneratedTreeOption[]));
}

void loadFormOptions();

/** 行政区域表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "parent_id",
    label: t("system.base.area.field.parent"),
    component: "tree-select",
    options: parentIdFormOptions.value,
    props: {
      lazy: true,
      load: loadParentIdFormTreeOptions,
      placeholder: t("common.placeholder.select"),
      filterable: true,
      style: { width: "100%" }
    }
  },
  {
    prop: "name",
    label: t("system.base.area.field.name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.area.field.name") }) }
  }
]);

/** 行政区域表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: t("system.base.area.field.name"), align: "left", search: { el: "input" } },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:area:update"],
        onClick: scope => handleOpenDialog((scope.row as BaseArea).id)
      },
      {
        label: t("common.action.delete"),
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
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:area:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:area:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseArea[])
  }
]);

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
    ElMessage.error(t("common.message.load_children_failed", { resource: t("system.base.area.resource") }));
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
  const parentIdFormResponse = await defBaseAreaService.OptionBaseArea({ parent_id: 0, lazy: true } as Parameters<
    typeof defBaseAreaService.OptionBaseArea
  >[0]);
  parentIdFormOptions.value = [
    { label: t("system.base.area.value.root"), value: 0 },
    ...normalizeLazyTreeOptions((parentIdFormResponse.list ?? []) as GeneratedTreeOption[]).filter(
      option => Number(option.value) !== 0
    )
  ];
}

/**
 * 打开行政区域弹窗。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
  await loadFormOptions();
  dialog.titleKey = id ? "system.base.area.action.edit" : "system.base.area.action.create";
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
      ElMessage.success(
        t(payload.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("system.base.area.resource")
        })
      );
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
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = t(rowList.length === 1 ? "system.base.area.message.delete_single" : "system.base.area.message.delete_batch");
  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseAreaService.DeleteBaseArea({ ids }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.base.area.resource") }));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.base.area.resource") }));
    }
  );
}
</script>
