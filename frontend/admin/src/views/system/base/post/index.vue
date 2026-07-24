<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBasePostTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="560px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="90px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBasePostService } from "@/api/system/base_post";
import type { BasePost, BasePostForm, PageBasePostRequest } from "@/rpc/system/admin/v1/base_post";
import { defBaseTenantService } from "@/api/system/base_tenant";
import type { SelectOptionResponse_Option } from "@/rpc/common/v1/common";
import { Status } from "@/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@/utils/tenant";
import { useUserStore } from "@/stores/modules/user";

defineOptions({
  name: "BasePost",
  inheritAttrs: false
});

/** 岗位表单状态，新增时租户由默认租户管理员选择。 */
type BasePostFormState = Omit<BasePostForm, "tenant_id"> & {
  /** 租户ID。 */
  tenant_id?: number;
};

const { BUTTONS } = useAuthButtons();
const userStore = useUserStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
});
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];
const formData = reactive<BasePostFormState>({
  /** 岗位ID。 */
  id: 0,
  /** 租户ID。 */
  tenant_id: undefined,
  /** 岗位名称。 */
  name: "",
  /** 岗位编号。 */
  code: "",
  /** 显示顺序。 */
  sort: 1,
  /** 状态。 */
  status: Status.ENABLE,
  /** 备注。 */
  remark: ""
});
const rules = reactive({
  tenant_id: [{ required: true, message: "请选择所属租户", trigger: "change" }],
  name: [
    { required: true, message: "请输入岗位名称", trigger: "blur" },
    { max: 30, message: "岗位名称不能超过 30 个字符", trigger: "blur" }
  ],
  code: [
    { required: true, message: "请输入岗位编号", trigger: "blur" },
    { max: 20, message: "岗位编号不能超过 20 个字符", trigger: "blur" }
  ],
  sort: [{ required: true, type: "number", min: 1, message: "排序必须大于 0", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "change" }],
  remark: [{ max: 500, message: "备注不能超过 500 个字符", trigger: "blur" }]
});

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 岗位表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "tenant_id",
    label: "所属租户",
    component: "select",
    props: { placeholder: "请选择所属租户", filterable: true, disabled: Boolean(formData.id) },
    visible: () => isDefaultTenant.value,
    options: tenantOptions.value
  },
  { prop: "name", label: "岗位名称", component: "input", props: { placeholder: "请输入岗位名称" } },
  { prop: "code", label: "岗位编号", component: "input", props: { placeholder: "请输入岗位编号" } },
  {
    prop: "sort",
    label: "排序",
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注" } }
]);

/** 岗位表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...(isDefaultTenant.value
    ? ([
        {
          prop: "tenant_id",
          label: "租户",
          minWidth: 140,
          showOverflowTooltip: true,
          search: { el: "select", key: "tenant_id", props: { filterable: true }, order: 1 },
          enum: requestTenantOptions
        }
      ] satisfies ColumnProps[])
    : []),
  { prop: "name", label: "岗位名称", minWidth: 140, search: { el: "input" } },
  { prop: "code", label: "岗位编号", minWidth: 140, search: { el: "input" } },
  { prop: "sort", label: "排序", minWidth: 90, align: "right" },
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
      disabled: () => !BUTTONS.value["base:post:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BasePost)
    }
  },
  { prop: "remark", label: "备注", minWidth: 160 },
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
        hidden: () => !BUTTONS.value["base:post:update"],
        params: scope => ({ postId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.postId as number | undefined) ?? (scope.row as BasePost).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:post:delete"],
        onClick: scope => handleDelete(scope.row as BasePost)
      }
    ]
  }
]);

/** 岗位顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:post:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:post:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BasePost[])
  }
];

/** 请求岗位分页列表。 */
async function requestBasePostTable(params: PageBasePostRequest) {
  const data = await defBasePostService.PageBasePost({
    ...buildPageRequest(params),
    tenant_id: isDefaultTenant.value ? params.tenant_id : undefined
  });
  return { data: { list: data.base_posts ?? [], total: data.total } };
}

/** 刷新岗位表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 加载租户选项。 */
async function loadTenantOptions() {
  if (!isDefaultTenant.value || tenantOptions.value.length) return;
  const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  tenantOptions.value = response.list ?? [];
}

/** 打开岗位编辑弹窗。 */
async function handleOpenDialog(id?: number) {
  resetForm();
  await loadTenantOptions();
  dialog.title = id ? "修改岗位" : "新增岗位";
  dialog.visible = true;
  if (id) Object.assign(formData, await defBasePostService.GetBasePost({ id }));
}

/** 关闭岗位弹窗并清理表单。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/** 重置岗位表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.tenant_id = undefined;
  formData.name = "";
  formData.code = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/** 提交岗位表单。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const submitData = JSON.parse(JSON.stringify(formData)) as BasePostForm;
    const request = submitData.id
      ? defBasePostService.UpdateBasePost({ base_post: submitData })
      : defBasePostService.CreateBasePost({ base_post: submitData });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改岗位成功" : "新增岗位成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/** 在岗位状态切换前确认并提交。 */
async function handleBeforeSetStatus(row: BasePost) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  try {
    await ElMessageBox.confirm(`是否确定${text}岗位？\n岗位名称：${row.name || row.code || `ID:${row.id}`}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBasePostService.SetBasePostStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/** 删除岗位，兼容单条删除与批量删除。 */
function handleDelete(selected?: number | string | Array<number | string> | BasePost | BasePost[]) {
  const postList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BasePost[])
    : selected && typeof selected === "object"
      ? [selected as BasePost]
      : [];
  const postIds = (postList.length ? postList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)).join(",");
  if (!postIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  const confirmMessage =
    postList.length === 1
      ? `是否确定删除岗位？\n岗位名称：${postList[0].name || postList[0].code || `ID:${postList[0].id}`}`
      : postList.length > 1
        ? `确认删除已选中的 ${postList.length} 个岗位吗？`
        : "确认删除已选中的岗位吗？";
  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () =>
      defBasePostService.DeleteBasePost({ id: postIds }).then(() => {
        ElMessage.success("删除岗位成功");
        refreshTable();
      }),
    () => ElMessage.info("已取消删除岗位")
  );
}
</script>
