<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'current-tenant'"
      row-key="grant_key"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestProjectGrantTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey, { resource: t('system.base.tenant_project_grant.title') })"
      width="640px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      :confirm-loading="loading"
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
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseDeptService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_dept";
import { defBasePostService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_post";
import { defBaseRoleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_role";
import { defBaseTenantProjectService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project";
import { defBaseTenantProjectGrantService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project_grant";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import {
  BaseTenantProjectGrantSubjectType,
  type BaseTenantProjectGrant,
  type PageBaseTenantProjectGrantRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project_grant";

defineOptions({
  name: "BaseTenantProjectGrant",
  inheritAttrs: false
});

type ProjectGrantFormState = {
  tenant_id?: number;
  subject_type: BaseTenantProjectGrantSubjectType;
  subject_id?: number;
  project_id: number[];
  all: boolean;
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const { isDefaultTenant, tenantOptions, tenantColumns, tenantFormField, toRequestTenantId, loadTenantOptions } = useTenantScope();
const projectOptions = ref<SelectOptionResponse_Option[]>([]);
const subjectOptions = ref<ProFormOption[]>([]);
const loading = ref(false);
const dialog = reactive({
  visible: false,
  editing: false,
  titleKey: "common.action.create_resource"
});
const DEFAULT_SUBJECT_TYPE = BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_POST;
const formData = reactive<ProjectGrantFormState>({
  tenant_id: undefined,
  subject_type: DEFAULT_SUBJECT_TYPE,
  subject_id: undefined,
  project_id: [],
  all: false
});
const subjectTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.tenant_project_grant.subject_type.post"), value: BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_POST },
  { label: t("system.base.tenant_project_grant.subject_type.role"), value: BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_ROLE },
  { label: t("system.base.tenant_project_grant.subject_type.dept"), value: BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_DEPT },
  { label: t("system.base.tenant_project_grant.subject_type.user"), value: BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_USER }
]);
const subjectTypeLabel = computed(() => new Map(subjectTypeOptions.value.map(item => [item.value, item.label])));
const rules = computed(() => ({
  tenant_id: [{ required: true, message: t("common.validation.required_select", { field: t("common.field.tenant") }), trigger: "change" }],
  subject_type: [{ required: true, type: "number", min: 1, message: t("common.validation.required_select", { field: t("system.base.tenant_project_grant.field.subject_type") }), trigger: "change" }],
  subject_id: [{ required: true, type: "number", min: 1, message: t("common.validation.required_select", { field: t("system.base.tenant_project_grant.field.subject") }), trigger: "change" }]
}));
const formFields = computed<ProFormField[]>(() => [
  tenantFormField({
    label: t("common.field.tenant"),
    props: { disabled: dialog.editing || loading.value, onChange: handleTenantChange }
  }),
  {
    prop: "subject_type",
    label: t("system.base.tenant_project_grant.field.subject_type"),
    component: "select",
    options: subjectTypeOptions.value,
    props: { filterable: false, disabled: dialog.editing || loading.value, onChange: handleSubjectTypeChange }
  },
  {
    prop: "subject_id",
    label: t("system.base.tenant_project_grant.field.subject"),
    component: formData.subject_type === BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_DEPT ? "tree-select" : "select",
    options: subjectOptions.value,
    props:
      formData.subject_type === BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_DEPT
        ? { filterable: true, checkStrictly: true, renderAfterExpand: false, style: { width: "100%" }, disabled: dialog.editing || !formData.tenant_id || loading.value }
        : { filterable: true, clearable: true, disabled: dialog.editing || !formData.tenant_id || formData.subject_type === 0 || loading.value }
  },
  {
    prop: "all",
    label: t("system.base.tenant_project_grant.field.scope"),
    labelTooltip: t("system.base.tenant_project_grant.tooltip.scope"),
    component: "switch",
    checkboxLabel: t("system.base.tenant_project_grant.all_projects"),
    props: { disabled: !formData.tenant_id || !formData.subject_id || loading.value }
  },
  {
    prop: "project_id",
    label: t("system.base.tenant_project_grant.field.projects"),
    component: "select",
    options: projectOptions.value,
    visible: () => !formData.all,
    props: { multiple: true, filterable: true, clearable: true, disabled: !formData.tenant_id || loading.value }
  }
]);
const columns = computed<ColumnProps[]>(() => [
  ...tenantColumns({ label: t("common.field.tenant"), minWidth: 150, order: 1 }),
  {
    prop: "subject_type",
    label: t("system.base.tenant_project_grant.field.subject_type"),
    minWidth: 110,
    search: { el: "select", enum: subjectTypeOptions.value },
    render: scope => subjectTypeLabel.value.get((scope.row as BaseTenantProjectGrant).subject_type) ?? String((scope.row as BaseTenantProjectGrant).subject_type)
  },
  { prop: "subject_name", label: t("system.base.tenant_project_grant.field.subject"), minWidth: 160, showOverflowTooltip: true },
  {
    prop: "project_names",
    label: t("system.base.tenant_project_grant.field.projects"),
    minWidth: 220,
    showOverflowTooltip: true,
    render: scope => {
      const row = scope.row as BaseTenantProjectGrant;
      return row.project_id.length === 1 && row.project_id[0] === 0
        ? t("system.base.tenant_project_grant.all_projects")
        : row.project_names.join(", ") || row.project_id.join(", ") || t("common.value.none");
    }
  },
  {
    prop: "operation",
    label: t("common.field.operation"),
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:tenant:project:grant:update"],
        onClick: scope => handleOpenDialog(scope.row as BaseTenantProjectGrant)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:tenant:project:grant:delete"],
        onClick: scope => handleDelete(scope.row as BaseTenantProjectGrant)
      }
    ]
  }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:tenant:project:grant:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:tenant:project:grant:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseTenantProjectGrant[])
  }
]);

/** 请求项目授权分页列表。 */
async function requestProjectGrantTable(params: Record<string, unknown>) {
  const page = buildPageRequest(params) as unknown as PageBaseTenantProjectGrantRequest;
  const tenantId = toRequestTenantId(params.tenant_id);
  const data = await defBaseTenantProjectGrantService.PageBaseTenantProjectGrant({ ...page, tenant_id: tenantId });
  return { data: { list: data.grants ?? [], total: data.total } };
}

/** 打开项目授权新增或编辑弹窗。 */
async function handleOpenDialog(row?: BaseTenantProjectGrant) {
  resetForm();
  dialog.editing = Boolean(row);
  dialog.titleKey = row ? "common.action.edit_resource" : "common.action.create_resource";
  loading.value = true;
  try {
    await loadTenantOptions(true);
    if (!isDefaultTenant.value) formData.tenant_id = Number(tenantOptions.value[0]?.value) || undefined;
    if (row) {
      formData.tenant_id = row.tenant_id;
      formData.subject_type = row.subject_type;
      formData.subject_id = row.subject_id;
      formData.all = row.project_id.length === 1 && row.project_id[0] === 0;
      formData.project_id = formData.all ? [] : [...row.project_id];
    }
    await Promise.all([loadProjectOptions(), loadSubjectOptions()]);
    dialog.visible = true;
  } finally {
    loading.value = false;
  }
}

/** 加载目标租户项目选项。 */
async function loadProjectOptions() {
  projectOptions.value = formData.tenant_id ? (await defBaseTenantProjectService.OptionBaseTenantProject({ tenant_id: formData.tenant_id })).list ?? [] : [];
}

/** 根据授权类型加载岗位、角色、部门或用户主体选项。 */
async function loadSubjectOptions() {
  subjectOptions.value = [];
  if (!formData.tenant_id || formData.subject_type === BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_UNSPECIFIED) return;
  switch (formData.subject_type) {
    case BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_POST:
      subjectOptions.value = (await defBasePostService.OptionBasePost({ tenant_id: formData.tenant_id })).list ?? [];
      return;
    case BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_ROLE:
      subjectOptions.value = (await defBaseRoleService.OptionBaseRole({ tenant_id: formData.tenant_id })).list ?? [];
      return;
    case BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_DEPT:
      subjectOptions.value = ((await defBaseDeptService.OptionBaseDept({ tenant_id: formData.tenant_id })).list ?? []) as unknown as ProFormOption[];
      return;
    case BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_USER:
      subjectOptions.value = (await defBaseUserService.OptionBaseUser({ keyword: "", tenant_id: formData.tenant_id })).list ?? [];
  }
}

/** 切换目标租户时重置主体和项目选择。 */
async function handleTenantChange(value: unknown) {
  if (loading.value) return;
  formData.tenant_id = Number(value) || undefined;
  formData.subject_type = DEFAULT_SUBJECT_TYPE;
  formData.subject_id = undefined;
  formData.project_id = [];
  formData.all = false;
  await Promise.all([loadProjectOptions(), loadSubjectOptions()]);
}

/** 切换授权类型时重新加载对应主体数据。 */
async function handleSubjectTypeChange(value: unknown) {
  if (loading.value) return;
  formData.subject_type = Number(value) as BaseTenantProjectGrantSubjectType;
  formData.subject_id = undefined;
  await loadSubjectOptions();
}

/** 提交项目授权新增或编辑。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid || !formData.tenant_id || !formData.subject_id || formData.subject_type === 0) return;
    const grant: BaseTenantProjectGrant = {
      tenant_id: formData.tenant_id,
      subject_type: formData.subject_type,
      subject_id: formData.subject_id,
      project_id: formData.all ? [0] : [...formData.project_id],
      tenant_name: "",
      subject_name: "",
      subject_code: "",
      project_names: [],
      grant_key: ""
    };
    loading.value = true;
    const request = dialog.editing
      ? defBaseTenantProjectGrantService.UpdateBaseTenantProjectGrant({ base_tenant_project_grant: grant })
      : defBaseTenantProjectGrantService.CreateBaseTenantProjectGrant({ base_tenant_project_grant: grant });
    request.then(() => {
      ElMessage.success(t(dialog.editing ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.tenant_project_grant.title") }));
      handleCloseDialog();
      proTable.value?.getTableList();
    }).finally(() => { loading.value = false; });
  });
}

/** 关闭并重置项目授权弹窗。 */
function handleCloseDialog() {
  dialog.visible = false;
  dialog.editing = false;
  resetForm();
}

/** 重置项目授权表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.tenant_id = undefined;
  formData.subject_type = DEFAULT_SUBJECT_TYPE;
  formData.subject_id = undefined;
  formData.project_id = [];
  formData.all = false;
  projectOptions.value = [];
  subjectOptions.value = [];
}

/** 删除项目授权，支持单条和批量记录。 */
function handleDelete(selected: BaseTenantProjectGrant | BaseTenantProjectGrant[]) {
  const rows = Array.isArray(selected) ? selected : [selected];
  if (!rows.length) return;
  const label = rows.length === 1 ? rows[0].subject_name || rows[0].subject_code || rows[0].grant_key : t("common.dialog.delete_batch", { count: rows.length, unit: "", resource: t("system.base.tenant_project_grant.title") });
  const message = rows.length === 1
    ? `${t("common.dialog.delete_single", { resource: t("system.base.tenant_project_grant.title") })}\n${t("common.dialog.resource_field", { field: t("system.base.tenant_project_grant.field.subject"), value: label })}`
    : label;
  ElMessageBox.confirm(message, t("common.title.warning"), { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }).then(async () => {
    await Promise.all(rows.map(row => defBaseTenantProjectGrantService.DeleteBaseTenantProjectGrant({ tenant_id: row.tenant_id, subject_type: row.subject_type, subject_id: row.subject_id })));
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.tenant_project_grant.title") }));
    proTable.value?.getTableList();
  });
}
</script>
