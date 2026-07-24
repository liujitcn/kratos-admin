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
      :title="dialog.title"
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
import { reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseTenantService } from "@/api/system/base_tenant";
import type { BaseTenant, BaseTenantForm, PageBaseTenantRequest } from "@/rpc/system/admin/v1/base_tenant";
import { Status } from "@/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseTenant",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
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
const rules = reactive({
  name: [
    { required: true, message: "请输入租户名称", trigger: "blur" },
    { max: 100, message: "租户名称不能超过 100 个字符", trigger: "blur" }
  ],
  contact_name: [{ max: 50, message: "联系人不能超过 50 个字符", trigger: "blur" }],
  contact_phone: [
    { max: 20, message: "联系电话不能超过 20 个字符", trigger: "blur" },
    { pattern: /^1[3-9]\d{9}$/, message: "请输入正确的联系电话", trigger: "blur" }
  ],
  status: [{ required: true, message: "请选择状态", trigger: "change" }],
  remark: [{ max: 500, message: "备注不能超过 500 个字符", trigger: "blur" }]
});

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 租户表单字段配置。 */
const formFields: ProFormField[] = [
  {
    prop: "code",
    label: "租户编号",
    component: "input",
    props: { disabled: true },
    visible: model => Boolean(model.id)
  },
  { prop: "name", label: "租户名称", component: "input", props: { placeholder: "请输入租户名称" } },
  { prop: "contact_name", label: "联系人", component: "input", props: { placeholder: "请输入联系人" } },
  { prop: "contact_phone", label: "联系电话", component: "input", props: { placeholder: "请输入联系电话" } },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions },
  { prop: "remark", label: "备注", component: "textarea", props: { placeholder: "请输入备注", rows: 3 }, colSpan: 24 }
];

/** 租户表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55, selectable: row => !isProtectedManagementTenant(row as BaseTenant) },
  { prop: "code", label: "租户编号", minWidth: 140, search: { el: "input", order: 1 } },
  { prop: "name", label: "租户名称", minWidth: 160, search: { el: "input", order: 2 } },
  { prop: "contact_name", label: "联系人", minWidth: 120 },
  { prop: "contact_phone", label: "联系电话", minWidth: 140 },
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
      disabled: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseTenant)
    }
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
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
        hidden: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:update"],
        params: scope => ({ tenantId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.tenantId as number | undefined) ?? (scope.row as BaseTenant).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope => isProtectedManagementTenant(scope.row as BaseTenant) || !BUTTONS.value["base:tenant:delete"],
        onClick: scope => handleDelete(scope.row as BaseTenant)
      }
    ]
  }
];

/** 租户顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:tenant:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:tenant:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseTenant[])
  }
];

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
  dialog.title = tenantId ? "修改租户" : "新增租户";
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
      ElMessage.success(submitData.id ? "修改租户成功" : "新增租户成功");
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
    ElMessage.warning("默认租户不能修改状态");
    return false;
  }

  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  try {
    await ElMessageBox.confirm(`是否确定${text}租户？\n租户名称：${row.name || `ID:${row.id}`}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseTenantService.SetBaseTenantStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
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
    ElMessage.warning("默认租户不能删除");
    return;
  }

  const tenantIds = (
    tenantList.length
      ? tenantList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!tenantIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = tenantList.length
    ? tenantList.length === 1
      ? `是否确定删除租户？\n租户名称：${tenantList[0].name || `ID:${tenantList[0].id}`}`
      : `确认删除已选中的 ${tenantList.length} 项租户吗？`
    : "确认删除已选中的租户吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseTenantService.DeleteBaseTenant({ id: tenantIds }).then(() => {
        ElMessage.success("删除租户成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除租户");
    }
  );
}
</script>
