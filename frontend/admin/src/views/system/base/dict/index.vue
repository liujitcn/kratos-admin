<!-- 字典 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDictTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="500px"
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
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseDictService } from "@/api/system/base_dict";
import type { BaseDict, BaseDictForm, PageBaseDictRequest } from "@/rpc/system/admin/v1/base_dict";
import router from "@/routers";
import { Status } from "@/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "BaseDict",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseDictForm>({
  /** 字典ID */
  id: 0,
  /** 字典编号 */
  code: "",
  /** 字典名称 */
  name: "",
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  name: [
    { required: true, message: "请输入字典名称", trigger: "blur" },
    { max: 50, message: "字典名称不能超过 50 个字符", trigger: "blur" }
  ],
  code: [
    { required: true, message: "请输入字典编码", trigger: "blur" },
    { max: 50, message: "字典编号不能超过 50 个字符", trigger: "blur" }
  ],
  status: [{ required: true, message: "状态不能为空", trigger: "change" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 字典表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "name", label: "字典名称", component: "input", props: { placeholder: "请输入字典名称" } },
  { prop: "code", label: "字典编码", component: "input", props: { placeholder: "请输入字典编码" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 字典表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "字典名称", minWidth: 140, search: { el: "input" } },
  { prop: "code", label: "字典编码", minWidth: 160, search: { el: "input" } },
  {
    prop: "status",
    label: "状态",
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:dict:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDict)
    }
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 240,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "字典数据",
        type: "primary",
        link: true,
        icon: List,
        hidden: () => !BUTTONS.value["base:dict:items"],
        onClick: scope => handleOpenBaseDictItem(scope.row as BaseDict)
      },
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dict:update"],
        params: scope => ({ dictId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.dictId as number | undefined) ?? (scope.row as BaseDict).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dict:delete"],
        onClick: scope => handleDelete(scope.row as BaseDict)
      }
    ]
  }
];

/** 字典顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dict:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dict:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDict[])
  }
];

/**
 * 请求字典列表，并由 ProTable 统一管理分页搜索。
 */
async function requestBaseDictTable(params: PageBaseDictRequest) {
  const data = await defBaseDictService.PageBaseDict(buildPageRequest(params));
  return { data: { list: data.base_dicts ?? [], total: data.total } };
}

/**
 * 刷新字典表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开字典编辑弹窗。
 */
function handleOpenDialog(dictId?: number) {
  resetForm();
  dialog.title = dictId ? "修改字典" : "新增字典";
  dialog.visible = true;
  if (!dictId) return;

  defBaseDictService.GetBaseDict({ id: dictId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭字典弹窗并恢复表单初始值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置字典表单，避免新增时带入上次编辑结果。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.code = "";
  formData.name = "";
  formData.status = Status.ENABLE;
}

/**
 * 提交字典表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDictForm;
    const request = submitData.id
      ? defBaseDictService.UpdateBaseDict({ base_dict: submitData })
      : defBaseDictService.CreateBaseDict({ base_dict: submitData });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改字典成功" : "新增字典成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDict) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const dictName = row.name || row.code || String(row.id);
  try {
    await ElMessageBox.confirm(`是否确定${text}字典？\n字典名称：${dictName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseDictService.SetBaseDictStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDict | BaseDict[]) {
  const dictList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDict[])
    : selected && typeof selected === "object"
      ? [selected as BaseDict]
      : [];
  const dictIds = (
    dictList.length ? dictList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = dictList.length
    ? dictList.length === 1
      ? `是否确定删除字典？\n字典名称：${dictList[0].name || dictList[0].code || `ID:${dictList[0].id}`}`
      : `确认删除已选中的 ${dictList.length} 个字典吗？`
    : "确认删除已选中的字典吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDict({ id: dictIds }).then(() => {
        ElMessage.success("删除字典成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除字典");
    }
  );
}

/**
 * 打开字典数据页面。
 */
function handleOpenBaseDictItem(row: BaseDict) {
  navigateTo(router, "/base/dict/item", { dictId: row.id, title: `【${row.name}】字典数据` });
}
</script>
