<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :indent="20"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDeptTable"
      :pagination="false"
      :default-expand-all="false"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey, { resource: t('system.base.dept.resource') })"
      width="600px"
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
import { defBaseDeptService } from "@liujitcn/kratos-admin-system/api/system/base_dept";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/base_tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import type { BaseDept, BaseDeptForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dept";
import type { SelectOptionResponse_Option, TreeOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseDept",
  inheritAttrs: false
});

/** 部门表单状态，新增时租户必须由默认租户管理员显式选择。 */
type BaseDeptFormState = Omit<BaseDeptForm, "tenant_id"> & {
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

const deptOptions = ref<TreeOptionResponse_Option[]>([]);
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
const currentTenantId = ref<number | undefined>();
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

const formData = reactive<BaseDeptFormState>({
  /** 部门ID */
  id: 0,
  /** 租户ID */
  tenant_id: undefined,
  /** 父节点ID */
  parent_id: 0,
  /** 部门名称 */
  name: "",
  /** 排序 */
  sort: 1,
  /** 菜单状态 */
  status: Status.ENABLE,
  /** 备注 */
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
  parent_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.dept.field.parent") }),
      trigger: "change"
    }
  ],
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.dept.field.name") }),
      trigger: "blur"
    },
    {
      max: 255,
      message: t("common.validation.max_length", { field: t("system.base.dept.field.name"), max: 255 }),
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
  ]
}));

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 部门表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "tenant_id",
    label: t("common.field.tenant"),
    component: "select",
    props: {
      placeholder: t("common.validation.required_select", { field: t("common.field.tenant") }),
      filterable: true,
      disabled: Boolean(formData.id),
      onChange: handleFormTenantChange
    },
    visible: () => isDefaultTenant.value,
    options: tenantOptions.value
  },
  {
    prop: "parent_id",
    label: t("system.base.dept.field.parent"),
    component: "tree-select",
    options: deptOptions.value,
    props: {
      placeholder: t("common.validation.required_select", { field: t("system.base.dept.field.parent") }),
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  {
    prop: "name",
    label: t("system.base.dept.field.name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.dept.field.name") }) }
  },
  {
    prop: "sort",
    label: t("common.field.sort"),
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { placeholder: t("common.placeholder.remark") }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

/** 部门树表格列配置。 */
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
  { prop: "name", label: t("system.base.dept.field.name"), minWidth: 140, align: "left", search: { el: "input" } },
  { prop: "remark", label: t("common.field.remark"), minWidth: 160, search: { el: "input" } },
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
      disabled: () => !BUTTONS.value["base:dept:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDept)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 220,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.create"),
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: () => !BUTTONS.value["base:dept:create"],
        params: scope => ({ parent_id: scope.row.id, tenantId: scope.row.tenant_id }),
        onClick: (_, params) =>
          handleOpenDialog((params?.parent_id as number | undefined) ?? 0, undefined, params?.tenantId as number | undefined)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dept:update"],
        params: scope => ({
          parent_id: scope.row.parent_id,
          deptId: scope.row.id
        }),
        onClick: (_, params) => handleOpenDialog(params?.parent_id as number | undefined, params?.deptId as number | undefined)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dept:delete"],
        onClick: scope => handleDelete(scope.row as BaseDept)
      }
    ]
  }
]);

/** 部门顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dept:create"],
    onClick: () => handleOpenDialog(0)
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dept:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDept[])
  }
]);

/**
 * 按搜索条件递归过滤部门树，命中父节点或子节点时保留当前节点。
 */
function filterDeptTree(deptList: BaseDept[], keywordMap: { name: string; remark: string; status: string }) {
  const nameKeyword = keywordMap.name.trim().toLowerCase();
  const remarkKeyword = keywordMap.remark.trim().toLowerCase();
  const statusKeyword = keywordMap.status.trim();

  return deptList.reduce<BaseDept[]>((result, item) => {
    const children = filterDeptTree(item.children ?? [], keywordMap);
    const name = item.name?.toLowerCase() ?? "";
    const remark = item.remark?.toLowerCase() ?? "";
    const status = String(item.status ?? "");
    const matched =
      (!nameKeyword || name.includes(nameKeyword)) &&
      (!remarkKeyword || remark.includes(remarkKeyword)) &&
      (!statusKeyword || status === statusKeyword);

    if (!matched && !children.length) return result;

    result.push({
      ...item,
      children
    });
    return result;
  }, []);
}

/**
 * 请求部门树数据，并按搜索条件过滤树形结构。
 */
async function requestBaseDeptTable(params: Record<string, string>) {
  currentTenantId.value = params.tenant_id ? Number(params.tenant_id) : undefined;
  const data = await defBaseDeptService.TreeBaseDept({ tenant_id: currentTenantId.value });
  const keywordMap = {
    name: params.name ?? "",
    remark: params.remark ?? "",
    status: String(params.status ?? "")
  };
  return { data: filterDeptTree(data.base_depts ?? [], keywordMap) };
}

/**
 * 刷新部门树表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载部门下拉树数据，供弹窗选择上级部门。
 */
async function loadDeptOptions() {
  // 默认租户未选择目标租户时仅保留顶级部门，避免混入其他租户的部门树。
  if (isDefaultTenant.value && !formData.tenant_id) {
    deptOptions.value = [{ value: 0, label: t("system.base.dept.value.root"), disabled: false, has_children: true, children: [] }];
    return;
  }
  const optionBaseDeptResponse = await defBaseDeptService.OptionBaseDept({
    tenant_id: isDefaultTenant.value ? formData.tenant_id : undefined
  });
  deptOptions.value = [
    {
      value: 0,
      label: t("system.base.dept.value.root"),
      disabled: false,
      has_children: true,
      children: optionBaseDeptResponse.list
    }
  ];
}

/**
 * 加载租户下拉选项。
 */
async function loadTenantOptions() {
  if (!isDefaultTenant.value || tenantOptions.value.length) return;
  const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  tenantOptions.value = response.list ?? [];
}

/**
 * 切换表单租户时重新加载对应租户的上级部门树。
 */
async function handleFormTenantChange() {
  formData.parent_id = 0;
  await loadDeptOptions();
}

/**
 * 打开部门弹窗。
 */
async function handleOpenDialog(parent_id?: number, deptId?: number, tenantId?: number) {
  resetForm();
  await loadTenantOptions();
  if (deptId) {
    dialog.titleKey = "common.action.edit_resource";
    dialog.visible = true;
    defBaseDeptService.GetBaseDept({ id: deptId }).then(async data => {
      Object.assign(formData, data);
      await loadDeptOptions();
    });
    return;
  }

  // 从部门行新增子部门时，继承父部门租户并加载同租户上级部门树。
  formData.tenant_id = tenantId;
  await loadDeptOptions();
  dialog.titleKey = "common.action.create_resource";
  dialog.visible = true;
  formData.parent_id = parent_id ?? 0;
}

/**
 * 关闭部门弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置部门表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.tenant_id = undefined;
  formData.parent_id = 0;
  formData.name = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
  formData.remark = "";
  deptOptions.value = [];
}

/**
 * 提交部门表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDeptForm;
    const request = submitData.id
      ? defBaseDeptService.UpdateBaseDept({ base_dept: submitData })
      : defBaseDeptService.CreateBaseDept({ base_dept: submitData });
    request.then(() => {
      ElMessage.success(
        t(submitData.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("system.base.dept.resource")
        })
      );
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在部门状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDept) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const deptName = row.name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: text,
        resource: t("system.base.dept.resource"),
        field: t("system.base.dept.field.name"),
        value: deptName
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseDeptService.SetBaseDeptStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action: text }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除部门，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDept | BaseDept[]) {
  const deptList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDept[])
    : selected && typeof selected === "object"
      ? [selected as BaseDept]
      : [];
  const deptIds = (
    deptList.length ? deptList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!deptIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = deptList.length
    ? deptList.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("system.base.dept.resource") })}\n${t("common.dialog.resource_field", { field: t("system.base.dept.field.name"), value: deptList[0].name || `ID:${deptList[0].id}` })}`
      : t("common.dialog.delete_batch", { count: deptList.length, unit: "", resource: t("system.base.dept.resource") })
    : t("common.dialog.delete_selected", { resource: t("system.base.dept.resource") });

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseDeptService.DeleteBaseDept({ id: deptIds }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.base.dept.resource") }));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.base.dept.resource") }));
    }
  );
}
</script>
