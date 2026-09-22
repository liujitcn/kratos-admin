<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseTenantProjectTable"
    >
      <template v-for="slotName in forwardedSlotNames" #[slotName]="scope">
        <slot :name="slotName" v-bind="scope" :tenant-project="resolveTenantProjectSlotContext(scope)" />
      </template>
    </ProTable>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey, { resource: t('common.field.project') })"
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
import { computed, onMounted, reactive, ref, useSlots } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type {
  ColumnProps,
  HeaderActionProps,
  ProTableInstance,
  RenderScope,
  TableActionProps
} from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseTenantProjectService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project";
import type {
  BaseTenantProject,
  BaseTenantProjectForm,
  PageBaseTenantProjectRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { arrangeTenantProjectColumns } from "./tenant-project-manager-data";
import {
  mergeTenantProjectExtraData,
  tenantProjectKey,
  type TenantProjectAction,
  type TenantProjectContext,
  type TenantProjectExtraData,
  type TenantProjectManagerProps
} from "./tenant-project-manager";

defineOptions({
  name: "TenantProjectManager",
  inheritAttrs: false
});

const props = withDefaults(defineProps<TenantProjectManagerProps>(), {
  extraColumns: () => [],
  extraActions: () => []
});

const emit = defineEmits<{
  /** 外部业务字段加载失败。 */
  "extra-data-error": [error: unknown];
}>();

/** 项目表单状态，默认租户创建时必须选择目标租户。 */
type BaseTenantProjectFormState = Omit<BaseTenantProjectForm, "tenant_id"> & {
  /** 项目所属租户。 */
  tenant_id?: number;
};

const { BUTTONS } = useAuthButtons();
const slots = useSlots();
const forwardedSlotNames = computed(() => Object.keys(slots).filter(name => name !== "default"));
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const extraDataByProject = ref<TenantProjectExtraData>({});
let extraDataRequestRevision = 0;

const dialog = reactive({
  titleKey: "common.action.create_resource",
  visible: false
});
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const formData = reactive<BaseTenantProjectFormState>({
  /** 项目ID。 */
  id: 0,
  /** 项目所属租户。 */
  tenant_id: undefined,
  /** 项目名称。 */
  name: "",
  /** 项目编号。 */
  code: "",
  /** 显示顺序。 */
  sort: 1,
  /** 状态。 */
  status: Status.STATUS_ENABLE,
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
      message: t("common.validation.required_input", { field: t("system.base.tenant_project.field.name") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.base.tenant_project.field.name"), max: 100 }),
      trigger: "blur"
    }
  ],
  code: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.tenant_project.field.code") }),
      trigger: "blur"
    },
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.tenant_project.field.code"), max: 50 }),
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

const { isDefaultTenant, tenantColumns, tenantFormField, toRequestTenantId, loadTenantOptions } = useTenantScope();
onMounted(() => {
  if (isDefaultTenant.value) void loadTenantOptions();
});

/** 项目表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  tenantFormField({ label: t("common.field.tenant"), disabledOnEdit: true }),
  {
    prop: "name",
    label: t("system.base.tenant_project.field.name"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.tenant_project.field.name") }) }
  },
  {
    prop: "code",
    label: t("system.base.tenant_project.field.code"),
    component: "input",
    props: { placeholder: t("common.validation.required_input", { field: t("system.base.tenant_project.field.code") }) }
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

/** 将项目行转换成外部业务模块可用的单条上下文。 */
function toTenantProjectContext(row: BaseTenantProject): TenantProjectContext {
  const key = tenantProjectKey(row.tenant_id, row.id);
  return {
    tenant_id: row.tenant_id,
    project_id: row.id,
    project: row,
    data: extraDataByProject.value[key] ?? {}
  };
}

/** 将租户项目扩展操作适配成 ProTable 操作。 */
function toTableAction(action: TenantProjectAction): TableActionProps {
  const { disabled, hidden, params, onClick, ...rest } = action;
  const result: TableActionProps = {
    ...rest,
    onClick: (scope, resolvedParams) => onClick(toTenantProjectContext(scope.row as BaseTenantProject), resolvedParams)
  };
  if (disabled !== undefined) result.disabled = scope => resolveProjectBoolean(disabled, scope);
  if (hidden !== undefined) result.hidden = scope => resolveProjectBoolean(hidden, scope);
  if (params) result.params = scope => params(toTenantProjectContext(scope.row as BaseTenantProject));
  return result;
}

/** 解析项目扩展操作的布尔配置。 */
function resolveProjectBoolean(value: boolean | ((context: TenantProjectContext) => boolean), scope: RenderScope): boolean {
  return typeof value === "function" ? value(toTenantProjectContext(scope.row as BaseTenantProject)) : value;
}

/** 项目表格列配置。 */
const columns = computed<ColumnProps[]>(() => {
  const baseColumns: ColumnProps[] = [
    { type: "selection", width: 55 },
    ...tenantColumns({ label: t("common.field.tenant"), order: 1 }),
    { prop: "name", label: t("system.base.tenant_project.field.name"), minWidth: 180, search: { el: "input" } },
    { prop: "code", label: t("system.base.tenant_project.field.code"), minWidth: 140, search: { el: "input" } },
    { prop: "sort", label: t("common.field.sort"), minWidth: 90, align: "right" },
    {
      prop: "status",
      label: t("common.field.status"),
      minWidth: 100,
      search: { el: "select" },
      cellType: "status",
      statusProps: {
        activeValue: Status.STATUS_ENABLE,
        inactiveValue: Status.STATUS_DISABLE,
        activeText: t("common.status.enabled"),
        inactiveText: t("common.status.disabled"),
        disabled: () => !isDefaultTenant.value || !BUTTONS.value["base:tenant:project:status"],
        beforeChange: scope => handleBeforeSetStatus(scope.row as BaseTenantProject)
      }
    },
    { prop: "remark", label: t("common.field.remark"), minWidth: 160 },
    { prop: "created_at", align: "center", label: t("common.field.created_at"), minWidth: 180 },
    { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
    {
      prop: "operation",
      label: t("common.field.operation"),
      cellType: "actions",
      actions: [
        ...props.extraActions.map(toTableAction),
        {
          label: t("common.action.edit"),
          type: "primary",
          link: true,
          icon: EditPen,
          hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:tenant:project:update"],
          params: scope => ({ projectId: scope.row.id }),
          onClick: (scope, params) =>
            handleOpenDialog((params?.projectId as number | undefined) ?? (scope.row as BaseTenantProject).id)
        },
        {
          label: t("common.action.delete"),
          type: "danger",
          link: true,
          icon: Delete,
          hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:tenant:project:delete"],
          onClick: scope => handleDelete(scope.row as BaseTenantProject)
        }
      ]
    }
  ];
  return arrangeTenantProjectColumns(baseColumns, props.extraColumns);
});

/** 项目顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:tenant:project:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:tenant:project:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseTenantProject[])
  }
]);

/** 请求项目分页列表并加载当前页外部业务字段。 */
async function requestBaseTenantProjectTable(params: PageBaseTenantProjectRequest) {
  const revision = ++extraDataRequestRevision;
  const request = {
    ...buildPageRequest(params),
    tenant_id: toRequestTenantId(params.tenant_id)
  } as PageBaseTenantProjectRequest;
  const data = await defBaseTenantProjectService.PageBaseTenantProject(request);
  const projects = data.base_tenant_projects ?? [];
  if (!props.loadExtraData || !projects.length) {
    if (revision === extraDataRequestRevision) extraDataByProject.value = {};
    return { data: { list: projects, total: data.total } };
  }

  try {
    const extraData = (await props.loadExtraData({
      projects: projects.map(row => ({ tenant_id: row.tenant_id, project_id: row.id })),
      query: request
    })) ?? {};
    if (revision === extraDataRequestRevision) extraDataByProject.value = extraData;
    return {
      data: {
        list: projects.map(row => mergeTenantProjectExtraData(row, extraData[tenantProjectKey(row.tenant_id, row.id)])),
        total: data.total
      }
    };
  } catch (error) {
    if (revision === extraDataRequestRevision) extraDataByProject.value = {};
    emit("extra-data-error", error);
    return { data: { list: projects, total: data.total } };
  }
}

/** 刷新项目表格。 */
async function refreshTable() {
  await proTable.value?.getTableList();
}

/** 将表格插槽作用域补充为项目上下文。 */
function resolveTenantProjectSlotContext(scope: Record<string, any>) {
  const row = scope?.row as BaseTenantProject | undefined;
  return row?.id && row.tenant_id ? toTenantProjectContext(row) : undefined;
}

/** 打开项目编辑弹窗。 */
async function handleOpenDialog(id?: number) {
  await formDialogRef.value?.open({
    load: async () => {
      await loadTenantOptions();
      return id ? defBaseTenantProjectService.GetBaseTenantProject({ id }) : undefined;
    },
    commit: data => {
      resetForm();
      dialog.titleKey = id ? "common.action.edit_resource" : "common.action.create_resource";
      if (data) Object.assign(formData, data);
    }
  });
}

/** 关闭项目弹窗并清理表单。 */
function handleCloseDialog() {
  formDialogRef.value?.close();
  resetForm();
}

/** 重置项目表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.tenant_id = undefined;
  formData.name = "";
  formData.code = "";
  formData.sort = 1;
  formData.status = Status.STATUS_ENABLE;
  formData.remark = "";
}

/** 提交项目表单。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const submitData = JSON.parse(JSON.stringify(formData)) as Partial<BaseTenantProjectForm>;
    const tenantId = toRequestTenantId(formData.tenant_id);
    if (tenantId === undefined) delete submitData.tenant_id;
    else submitData.tenant_id = tenantId;
    const request = submitData.id
      ? defBaseTenantProjectService.UpdateBaseTenantProject({ base_tenant_project: submitData as BaseTenantProjectForm })
      : defBaseTenantProjectService.CreateBaseTenantProject({ base_tenant_project: submitData as BaseTenantProjectForm });
    request.then(() => {
      ElMessage.success(
        t(submitData.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("common.field.project")
        })
      );
      handleCloseDialog();
      void refreshTable();
    });
  });
}

/** 在项目状态切换前确认并提交。 */
async function handleBeforeSetStatus(row: BaseTenantProject) {
  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const text = t(nextStatus === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: text,
        resource: t("common.field.project"),
        field: t("system.base.tenant_project.field.name"),
        value: row.name || row.code || `ID:${row.id}`
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseTenantProjectService.SetBaseTenantProjectStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action: text }));
    return true;
  } catch {
    return false;
  }
}

/** 删除项目，兼容单条删除与批量删除。 */
function handleDelete(selected?: number | string | Array<number | string> | BaseTenantProject | BaseTenantProject[]) {
  const projectList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseTenantProject[])
    : selected && typeof selected === "object"
      ? [selected as BaseTenantProject]
      : [];
  const projectIds = (
    projectList.length ? projectList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!projectIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  const confirmMessage =
    projectList.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("common.field.project") })}\n${t("common.dialog.resource_field", { field: t("system.base.tenant_project.field.name"), value: projectList[0].name || projectList[0].code || `ID:${projectList[0].id}` })}`
      : projectList.length > 1
        ? t("common.dialog.delete_batch", { count: projectList.length, unit: "", resource: t("common.field.project") })
        : t("common.dialog.delete_selected", { resource: t("common.field.project") });
  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () =>
      defBaseTenantProjectService.DeleteBaseTenantProject({ id: projectIds }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("common.field.project") }));
        void refreshTable();
      }),
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("common.field.project") }))
  );
}

defineExpose({
  /** 刷新项目列表。 */
  refresh: refreshTable,
  /** 兼容表格实例的刷新方法名称。 */
  getTableList: refreshTable
});
</script>
