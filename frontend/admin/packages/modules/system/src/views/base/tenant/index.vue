<!-- 租户管理 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseTenantTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey, { resource: t('system.base.tenant.resource') })"
      width="780px"
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
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/base_tenant";
import type {
  BaseTenant,
  BaseTenantForm,
  PageBaseTenantRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseTenant",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  titleKey: "common.action.create_resource",
  visible: false
});

const formData = reactive<BaseTenantForm>({
  /** 租户ID */
  id: 0,
  /** 租户编号 */
  code: "",
  /** 租户名称 */
  name: "",
  /** 联系人 */
  contact_name: "",
  /** 联系电话 */
  contact_phone: "",
  /** 状态 */
  status: Status.ENABLE,
  /** 备注 */
  remark: ""
});

/** 租户表单校验规则。 */
const rules = computed(() => ({
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.tenant.field.name") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.base.tenant.field.name"), max: 100 }),
      trigger: "blur"
    }
  ],
  contact_name: [
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.tenant.field.contact_name"), max: 50 }),
      trigger: "blur"
    }
  ],
  contact_phone: [
    {
      max: 20,
      message: t("common.validation.max_length", { field: t("system.base.tenant.field.contact_phone"), max: 20 }),
      trigger: "blur"
    },
    { pattern: /^1[3-9]\d{9}$/, message: t("system.base.tenant.message.phone_invalid"), trigger: "blur" }
  ],
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

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

/** 租户表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "code",
    label: t("system.base.tenant.field.code"),
    component: "input",
    props: { disabled: true },
    visible: model => Boolean(model.id)
  },
  {
    prop: "name",
    label: t("system.base.tenant.field.name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.tenant.field.name") }) }
  },
  {
    prop: "contact_name",
    label: t("system.base.tenant.field.contact_name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.tenant.field.contact_name") }) }
  },
  {
    prop: "contact_phone",
    label: t("system.base.tenant.field.contact_phone"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.tenant.field.contact_phone") }) }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { placeholder: t("common.placeholder.remark"), rows: 3 },
    colSpan: 24
  }
]);

/** 租户表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55, selectable: row => !isProtectedManagementTenant(row as BaseTenant) },
  { prop: "code", label: t("system.base.tenant.field.code"), minWidth: 140, search: { el: "input", order: 1 } },
  { prop: "name", label: t("system.base.tenant.field.name"), minWidth: 160, search: { el: "input", order: 2 } },
  { prop: "contact_name", label: t("system.base.tenant.field.contact_name"), minWidth: 120 },
  { prop: "contact_phone", label: t("system.base.tenant.field.contact_phone"), minWidth: 140 },
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
      disabled: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseTenant)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
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
        hidden: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:update"],
        params: scope => ({ tenantId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.tenantId as number | undefined) ?? (scope.row as BaseTenant).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:delete"],
        onClick: scope => handleDelete(scope.row as BaseTenant)
      }
    ]
  }
]);

/** 租户顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:tenant:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:tenant:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseTenant[])
  }
]);

/**
 * 请求租户列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseTenantTable(params: PageBaseTenantRequest) {
  const data = await defBaseTenantService.PageBaseTenant(buildPageRequest(params));
  return { data: { list: data.base_tenants ?? [], total: data.total } };
}

/**
 * 刷新租户表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 根据后端管理保护标记判断租户是否禁止通过租户管理操作。
 */
function isProtectedManagementTenant(row?: BaseTenant) {
  return Boolean(row?.is_protected);
}

/**
 * 打开租户弹窗，并按新增或编辑场景回填表单数据。
 */
async function handleOpenDialog(tenantId?: number) {
  resetForm();
  dialog.titleKey = tenantId ? "common.action.edit_resource" : "common.action.create_resource";
  dialog.visible = true;
  if (!tenantId) return;

  const data = await defBaseTenantService.GetBaseTenant({ id: tenantId });
  Object.assign(formData, data);
}

/**
 * 关闭租户弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置租户表单，避免新增时保留旧值。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.code = "";
  formData.name = "";
  formData.contact_name = "";
  formData.contact_phone = "";
  formData.status = Status.ENABLE;
  formData.remark = "";
}

/**
 * 提交租户表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseTenantForm;
    const request = submitData.id
      ? defBaseTenantService.UpdateBaseTenant({ base_tenant: submitData })
      : defBaseTenantService.CreateBaseTenant({ base_tenant: submitData });
    request.then(() => {
      ElMessage.success(
        t(submitData.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("system.base.tenant.resource")
        })
      );
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在租户状态切换前先完成确认与接口调用。
 */
async function handleBeforeSetStatus(row: BaseTenant) {
  if (isProtectedManagementTenant(row)) {
    ElMessage.warning(t("system.base.tenant.message.protected_status"));
    return false;
  }

  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: text,
        resource: t("system.base.tenant.resource"),
        field: t("system.base.tenant.field.name"),
        value: row.name || `ID:${row.id}`
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseTenantService.SetBaseTenantStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action: text }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除租户，兼容单项删除与多选删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseTenant | BaseTenant[]) {
  const tenantList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseTenant[])
    : selected && typeof selected === "object"
      ? [selected as BaseTenant]
      : [];
  if (tenantList.some(isProtectedManagementTenant)) {
    ElMessage.warning(t("system.base.tenant.message.protected_delete"));
    return;
  }

  const tenantIds = (
    tenantList.length
      ? tenantList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!tenantIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = tenantList.length
    ? tenantList.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("system.base.tenant.resource") })}\n${t("common.dialog.resource_field", { field: t("system.base.tenant.field.name"), value: tenantList[0].name || `ID:${tenantList[0].id}` })}`
      : t("common.dialog.delete_batch", { count: tenantList.length, unit: "", resource: t("system.base.tenant.resource") })
    : t("common.dialog.delete_selected", { resource: t("system.base.tenant.resource") });

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseTenantService.DeleteBaseTenant({ id: tenantIds }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.base.tenant.resource") }));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.base.tenant.resource") }));
    }
  );
}
</script>
