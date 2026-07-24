<!-- 字典数据 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDictItemTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="820px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #tagTypeField>
        <div class="dict-item-tag-type">
          <el-tag v-if="formData.tag_type" :type="formatTagType(formData.tag_type)" class="mr-2">
            {{ formData.label || "标签预览" }}
          </el-tag>
          <el-radio-group v-model="formData.tag_type">
            <el-radio value="success" border size="small">success</el-radio>
            <el-radio value="warning" border size="small">warning</el-radio>
            <el-radio value="info" border size="small">info</el-radio>
            <el-radio value="primary" border size="small">primary</el-radio>
            <el-radio value="danger" border size="small">danger</el-radio>
            <el-radio value="" border size="small">清空</el-radio>
          </el-radio-group>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseDictService } from "@/api/system/base_dict";
import type { BaseDictItem, BaseDictItemForm, PageBaseDictItemRequest } from "@/rpc/system/admin/v1/base_dict";
import { Status } from "@/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseDictItem",
  inheritAttrs: false
});

const route = useRoute();
const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dictId = ref(Number(route.query.dictId ?? 0));

const dialog = reactive({
  title: "",
  visible: false
});

const formData = reactive<BaseDictItemForm>({
  /** 字典项ID */
  id: 0,
  /** 字典ID */
  dict_id: dictId.value,
  /** 字典值 */
  value: "",
  /** 字典项标签 */
  label: "",
  /** 标签类型 */
  tag_type: "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  value: [{ required: true, max: 50, message: "请输入字典值（不超过50个字符）", trigger: "blur" }],
  label: [{ required: true, max: 100, message: "请输入字典标签（不超过100个字符）", trigger: "blur" }],
  tag_type: [{ max: 50, message: "标签类型不能超过 50 个字符", trigger: "blur" }],
  sort: [{ required: true, message: "请输入排序", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "change" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 字典项表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "label", label: "字典标签", component: "input", props: { placeholder: "请输入字典标签" } },
  { prop: "value", label: "字典值", component: "input", props: { placeholder: "请输入字典值" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "tag_type", label: "标签类型", component: "slot", slotName: "tagTypeField", colSpan: 24 }
];

/**
 * 规范化标签类型，兼容 Element Plus Tag 的可选值。
 */
function formatTagType(tag_type: string) {
  if (["success", "info", "warning", "primary", "danger"].includes(tag_type)) {
    return tag_type as "success" | "info" | "warning" | "primary" | "danger";
  }
  return undefined;
}

/**
 * 渲染标签类型列，统一复用字典项标签的展示方式。
 */
function renderTagTypeCell(scope: RenderScope<BaseDictItem>) {
  if (!scope.row.tag_type) return "无";
  return h(
    ElTag,
    {
      type: formatTagType(scope.row.tag_type),
      effect: "plain"
    },
    () => scope.row.label
  );
}

/** 字典项表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "label", label: "字典标签", minWidth: 140, search: { el: "input" } },
  { prop: "value", label: "字典值", minWidth: 140 },
  { prop: "sort", label: "排序", minWidth: 90, align: "right" },
  {
    prop: "tag_type",
    label: "标签类型",
    minWidth: 120,
    render: scope => renderTagTypeCell(scope as unknown as RenderScope<BaseDictItem>)
  },
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
      disabled: () => !BUTTONS.value["base:dict-item:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDictItem)
    }
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dict-item:update"],
        params: scope => ({ dictItemId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.dictItemId as number | undefined) ?? (scope.row as BaseDictItem).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dict-item:delete"],
        onClick: scope => handleDelete(scope.row as BaseDictItem)
      }
    ]
  }
];

/** 字典项顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dict-item:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dict-item:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDictItem[])
  }
];

watch(
  () => route.query.dictId,
  value => {
    dictId.value = Number(value ?? 0);
    formData.dict_id = dictId.value;
    proTable.value?.search();
  }
);

/**
 * 请求字典项分页列表，并补充当前路由上的字典 ID。
 */
async function requestBaseDictItemTable(params: PageBaseDictItemRequest) {
  const data = await defBaseDictService.PageBaseDictItem({ ...buildPageRequest(params), dict_id: dictId.value });
  return { data: { list: data.base_dict_items ?? [], total: data.total } };
}

/**
 * 刷新字典项表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开字典项编辑弹窗。
 */
function handleOpenDialog(dictItemId?: number) {
  resetForm();
  dialog.title = dictItemId ? "修改字典数据" : "新增字典数据";
  dialog.visible = true;
  if (!dictItemId) return;

  defBaseDictService.GetBaseDictItem({ id: dictItemId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭字典项弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置字典项表单，避免弹窗之间相互污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.dict_id = dictId.value;
  formData.value = "";
  formData.label = "";
  formData.tag_type = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
}

/**
 * 提交字典项表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.dict_id = dictId.value;
    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDictItemForm;
    const request = submitData.id
      ? defBaseDictService.UpdateBaseDictItem({ base_dict_item: submitData })
      : defBaseDictService.CreateBaseDictItem({ base_dict_item: submitData });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改字典数据成功" : "新增字典数据成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典项状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDictItem) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const itemName = row.label || row.value || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}字典数据？\n字典标签：${itemName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseDictService.SetBaseDictItemStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典项，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDictItem | BaseDictItem[]) {
  const dictItemList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDictItem[])
    : selected && typeof selected === "object"
      ? [selected as BaseDictItem]
      : [];
  const dictItemIds = (
    dictItemList.length
      ? dictItemList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictItemIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const singleItemName = dictItemList[0]?.label || dictItemList[0]?.value || `ID:${dictItemList[0]?.id ?? ""}`;
  const confirmMessage = dictItemList.length
    ? dictItemList.length === 1
      ? `是否确定删除字典数据？\n字典标签：${singleItemName}`
      : `确认删除已选中的 ${dictItemList.length} 项字典数据吗？`
    : "确认删除已选中的字典数据吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDictItem({ id: dictItemIds }).then(() => {
        ElMessage.success("删除字典数据成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除字典数据");
    }
  );
}
</script>

<style scoped>
.dict-item-tag-type {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
