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
      :title="t(dialog.titleKey, { resource: t('system.base.post.resource') })"
      width="560px"
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
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBasePostService } from "@liujitcn/kratos-admin-system/api/system/base_post";
import type { BasePost, BasePostForm, PageBasePostRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_post";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/base_tenant";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@liujitcn/kratos-admin-core/tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { t } from "@liujitcn/kratos-admin-core";

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
  titleKey: "common.action.create_resource",
  visible: false
});
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);
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
const rules = computed(() => ({
  tenant_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.tenant") }),
      trigger: "change"
    }
  ],
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.post.field.name") }),
      trigger: "blur"
    },
    {
      max: 30,
      message: t("common.validation.max_length", { field: t("system.base.post.field.name"), max: 30 }),
      trigger: "blur"
    }
  ],
  code: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.post.field.code") }),
      trigger: "blur"
    },
    {
      max: 20,
      message: t("common.validation.max_length", { field: t("system.base.post.field.code"), max: 20 }),
      trigger: "blur"
    }
  ],
  sort: [{ required: true, type: "number", min: 1, message: t("common.validation.sort_positive"), trigger: "blur" }],
  status: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.status") }),
      trigger: "change"
    }
  ],
  remark: [
    {
      max: 500,
      message: t("common.validation.max_length", { field: t("common.field.remark"), max: 500 }),
      trigger: "blur"
    }
  ]
}));

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 岗位表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "tenant_id",
    label: t("common.field.tenant"),
    component: "select",
    props: {
      placeholder: t("common.validation.required_select", { field: t("common.field.tenant") }),
      filterable: true,
      disabled: Boolean(formData.id)
    },
    visible: () => isDefaultTenant.value,
    options: tenantOptions.value
  },
  {
    prop: "name",
    label: t("system.base.post.field.name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.post.field.name") }) }
  },
  {
    prop: "code",
    label: t("system.base.post.field.code"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.post.field.code") }) }
  },
  {
    prop: "sort",
    label: t("common.field.sort"),
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { placeholder: t("common.placeholder.remark") }
  }
]);

/** 岗位表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...(isDefaultTenant.value
    ? ([
        {
          prop: "tenant_id",
          label: t("common.field.tenant"),
          minWidth: 140,
          showOverflowTooltip: true,
          search: { el: "select", key: "tenant_id", props: { filterable: true }, order: 1 },
          enum: requestTenantOptions
        }
      ] satisfies ColumnProps[])
    : []),
  { prop: "name", label: t("system.base.post.field.name"), minWidth: 140, search: { el: "input" } },
  { prop: "code", label: t("system.base.post.field.code"), minWidth: 140, search: { el: "input" } },
  { prop: "sort", label: t("common.field.sort"), minWidth: 90, align: "right" },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:post:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BasePost)
    }
  },
  { prop: "remark", label: t("common.field.remark"), minWidth: 160 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:post:update"],
        params: scope => ({ postId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.postId as number | undefined) ?? (scope.row as BasePost).id)
      },
      {
        label: t("common.action.delete"),
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
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:post:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:post:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BasePost[])
  }
]);

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
  dialog.titleKey = id ? "common.action.edit_resource" : "common.action.create_resource";
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
      ElMessage.success(
        t(submitData.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("system.base.post.resource")
        })
      );
      handleCloseDialog();
      refreshTable();
    });
  });
}

/** 在岗位状态切换前确认并提交。 */
async function handleBeforeSetStatus(row: BasePost) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: text,
        resource: t("system.base.post.resource"),
        field: t("system.base.post.field.name"),
        value: row.name || row.code || `ID:${row.id}`
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBasePostService.SetBasePostStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action: text }));
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
  const postIds = (
    postList.length ? postList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!postIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  const confirmMessage =
    postList.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("system.base.post.resource") })}\n${t("common.dialog.resource_field", { field: t("system.base.post.field.name"), value: postList[0].name || postList[0].code || `ID:${postList[0].id}` })}`
      : postList.length > 1
        ? t("common.dialog.delete_batch", { count: postList.length, unit: "", resource: t("system.base.post.resource") })
        : t("common.dialog.delete_selected", { resource: t("system.base.post.resource") });
  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () =>
      defBasePostService.DeleteBasePost({ id: postIds }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.base.post.resource") }));
        refreshTable();
      }),
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.base.post.resource") }))
  );
}
</script>
