<!-- 定时任务 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestBaseJobTable" />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.job.action.edit' : 'system.base.job.action.create')"
      width="1000px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #i18ns>
        <DynamicI18nEditor
          v-model="i18nValues"
          :source="formData.name"
          :maxlength="50"
        />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, type VNode } from "vue";
import { ElButton, ElMessage, ElMessageBox, ElTag } from "element-plus";
import { CirclePlus, Delete, EditPen, Promotion, Tickets, VideoPause, VideoPlay } from "@element-plus/icons-vue";
import type {
  ColumnProps,
  HeaderActionProps,
  ProTableInstance,
  RenderScope
} from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseJobService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_job";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import type {
  BaseJob,
  BaseJobArgs,
  BaseJobForm,
  PageBaseJobRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_job";
import router, { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import DynamicI18nCell from "@liujitcn/kratos-admin-system/components/DynamicI18nCell.vue";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/DynamicI18nEditor.vue";
import {
  normalizeDynamicI18ns,
  serializeDynamicI18ns,
  type DynamicI18nRecord,
  type DynamicI18nValue
} from "@liujitcn/kratos-admin-system/components/dynamicI18n";

defineOptions({
  name: "BaseJob",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const i18nValues = ref<DynamicI18nValue[]>(normalizeDynamicI18ns(undefined));

const dialog = reactive({
  editing: false,
  visible: false
});

const formData = reactive<BaseJobForm>({
  /** 定时任务ID */
  id: 0,
  /** 任务名称 */
  name: "",
  /** 调用目标 */
  invoke_target: "",
  /** 目标参数 */
  args: [],
  /** cron表达式 */
  cron_expression: "",
  /** 状态 */
  status: Status.STATUS_ENABLE,
  /** 定时任务名称多语言翻译。 */
  i18ns: []
});

const rules = computed(() => ({
  name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.job.field.name") }),
      trigger: "blur"
    },
    { max: 50, message: t("common.validation.max_length", { field: t("system.base.job.field.name"), max: 50 }), trigger: "blur" }
  ],
  cron_expression: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.job.field.cron") }),
      trigger: "blur"
    },
    { max: 50, message: t("common.validation.max_length", { field: t("system.base.job.field.cron"), max: 50 }), trigger: "blur" }
  ],
  invoke_target: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.job.field.invoke_target") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.base.job.field.invoke_target"), max: 100 }),
      trigger: "blur"
    }
  ],
  args: {
    validator: (rule: unknown, value: BaseJobArgs[], callback: (error?: Error) => void) => {
      if (value.some(arg => !arg.key)) callback(new Error(t("system.base.job.validation.arg_key_required")));
      else callback();
    },
    trigger: "blur"
  },
  status: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.status") }),
      trigger: "blur"
    }
  ]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

/**
 * 渲染任务参数标签，便于在列表中快速查看键值对。
 */
function renderArgsCell(scope: RenderScope<BaseJob>) {
  const args = scope.row.args ?? [];
  if (!args.length) return "--";
  return h(
    "div",
    null,
    args.map((arg, index) =>
      h(
        ElTag,
        {
          key: `${arg.key}-${arg.value}-${index}`,
          class: "mr-5"
        },
        () => `${arg.key}=${arg.value}`
      )
    )
  );
}

/** 渲染定时任务名称翻译预览。 */
function renderJobNameCell(scope: RenderScope<BaseJob>) {
  const row = scope.row;
  return h(DynamicI18nCell, {
    source: row.name,
    targetType: I18nTargetType.I18N_TARGET_TYPE_BASE_JOB_NAME,
    targetId: row.id,
    i18ns: row.i18ns
  });
}

/**
 * 渲染定时任务操作列。
 */
function renderOperationCell(scope: RenderScope<BaseJob>) {
  const row = scope.row;
  const actionNodes: VNode[] = [];

  if (BUTTONS.value["base:job:update"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `edit-${row.id}`,
          type: "primary",
          link: true,
          icon: EditPen,
          onClick: () => handleOpenDialog(row.id)
        },
        () => t("common.action.edit")
      )
    );
  }

  if (BUTTONS.value["base:job:delete"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `delete-${row.id}`,
          type: "danger",
          link: true,
          icon: Delete,
          onClick: () => handleDelete(row)
        },
        () => t("common.action.delete")
      )
    );
  }

  if (row.status === Status.STATUS_ENABLE && (row.entry_id === undefined || row.entry_id === 0) && BUTTONS.value["base:job:start"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `start-${row.id}`,
          type: "primary",
          link: true,
          icon: VideoPlay,
          class: "job-action job-action--start",
          onClick: () => handleStart(row.id, row.name)
        },
        () => t("system.base.job.action.start")
      )
    );
  }

  if (row.status === Status.STATUS_ENABLE && row.entry_id > 0 && BUTTONS.value["base:job:stop"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `stop-${row.id}`,
          type: "warning",
          link: true,
          icon: VideoPause,
          class: "job-action job-action--stop",
          onClick: () => handleStop(row.id, row.name)
        },
        () => t("system.base.job.action.stop")
      )
    );
  }

  if (row.status === Status.STATUS_ENABLE && (row.entry_id === undefined || row.entry_id === 0) && BUTTONS.value["base:job:exec"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `exec-${row.id}`,
          type: "success",
          link: true,
          icon: Promotion,
          class: "job-action job-action--exec",
          onClick: () => handleExec(row.id, row.name)
        },
        () => t("system.base.job.action.execute")
      )
    );
  }

  if (BUTTONS.value["base:job:log"]) {
    actionNodes.push(
      h(
        ElButton,
        {
          key: `log-${row.id}`,
          type: "primary",
          link: true,
          icon: Tickets,
          class: "job-action job-action--log",
          onClick: () => handleOpenBaseJob(row.id, row.name)
        },
        () => t("system.base.job.action.log")
      )
    );
  }

  if (!actionNodes.length) return "--";
  return h(
    "div",
    {
      class: "job-operation",
      key: `job-operation-${row.id}`
    },
    actionNodes
  );
}

/** 定时任务表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: t("system.base.job.field.name"),
    component: "input",
    props: { placeholder: t("system.base.job.placeholder.name") }
  },
  {
    prop: "i18ns",
    label: t("system.base.i18n.field.i18ns"),
    component: "slot",
    slotName: "i18ns",
    colSpan: 24
  },
  {
    prop: "invoke_target",
    label: t("system.base.job.field.invoke_target"),
    component: "input",
    props: { placeholder: t("system.base.job.placeholder.invoke_target") }
  },
  {
    prop: "cron_expression",
    label: t("system.base.job.field.cron"),
    component: "cron-expression",
    props: { placeholder: "0 0 0 * * *" }
  },
  {
    prop: "args",
    label: t("system.base.job.field.args"),
    component: "kv-list",
    props: {
      keyInputProps: { placeholder: t("system.base.job.field.arg_key") },
      valueInputProps: { placeholder: t("system.base.job.field.arg_value") },
      addText: t("system.base.job.action.add_arg")
    }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

/** 定时任务表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "name",
    label: t("system.base.job.field.name"),
    minWidth: 140,
    search: { el: "input" },
    showOverflowTooltip: false,
    render: scope => renderJobNameCell(scope as unknown as RenderScope<BaseJob>)
  },
  { prop: "invoke_target", label: t("system.base.job.field.invoke_target"), minWidth: 180, search: { el: "input" } },
  {
    prop: "args",
    label: t("system.base.job.field.args_short"),
    minWidth: 140,
    render: scope => renderArgsCell(scope as unknown as RenderScope<BaseJob>)
  },
  { prop: "cron_expression", label: t("system.base.job.field.cron"), minWidth: 150 },
  { prop: "entry_id", label: t("system.base.job.field.entry_id"), minWidth: 100, align: "right" },
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
      disabled: scope => (scope.row as BaseJob).entry_id === 0 || !BUTTONS.value["base:job:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseJob)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.action"),
    width: 380,
    fixed: "right",
    render: scope => renderOperationCell(scope as unknown as RenderScope<BaseJob>)
  }
]);

/** 定时任务顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:job:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:job:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseJob[])
  }
]);

/**
 * 请求定时任务列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseJobTable(params: PageBaseJobRequest) {
  await loadEnabledBaseLanguages();
  const data = await defBaseJobService.PageBaseJob(buildPageRequest(params));
  return { data: { list: data.base_jobs ?? [], total: data.total } };
}

/**
 * 刷新定时任务表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开定时任务弹窗。
 */
async function handleOpenDialog(jobId?: number) {
  await loadEnabledBaseLanguages();
  resetForm();
  dialog.editing = Boolean(jobId);
  dialog.visible = true;
  if (!jobId) return;

  const data = await defBaseJobService.GetBaseJob({ id: jobId });
  Object.assign(formData, data);
	i18nValues.value = normalizeDynamicI18ns(data.i18ns as DynamicI18nRecord[]);
}

/**
 * 关闭定时任务弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置定时任务表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.name = "";
  formData.invoke_target = "";
  formData.args = [];
  formData.cron_expression = "";
  formData.status = Status.STATUS_ENABLE;
  formData.i18ns = [];
	i18nValues.value = normalizeDynamicI18ns(undefined);
}

/**
 * 提交定时任务表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseJobForm;
    submitData.i18ns = serializeDynamicI18ns(
      i18nValues.value,
      I18nTargetType.I18N_TARGET_TYPE_BASE_JOB_NAME,
      submitData.id
    );
    const request = submitData.id
      ? defBaseJobService.UpdateBaseJob({ base_job: submitData })
      : defBaseJobService.CreateBaseJob({ base_job: submitData });
    request.then(() => {
      ElMessage.success(t(submitData.id ? "system.base.job.message.update_success" : "system.base.job.message.create_success"));
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在定时任务状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseJob) {
  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(nextStatus === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  const jobName = row.name || row.invoke_target || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(t("system.base.job.message.confirm_status", { action, name: jobName }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await defBaseJobService.SetBaseJobStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除定时任务，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseJob | BaseJob[]) {
  const jobList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseJob[])
    : selected && typeof selected === "object"
      ? [selected as BaseJob]
      : [];
  const jobIds = (
    jobList.length ? jobList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!jobIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const singleJobName = jobList[0]?.name || jobList[0]?.invoke_target || `ID:${jobList[0]?.id ?? ""}`;
  const confirmMessage = jobList.length
    ? jobList.length === 1
      ? t("system.base.job.message.confirm_delete_single", { name: singleJobName })
      : t("system.base.job.message.confirm_delete_batch", { count: jobList.length })
    : t("system.base.job.message.confirm_delete_selected");

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseJobService.DeleteBaseJob({ id: jobIds }).then(() => {
        ElMessage.success(t("system.base.job.message.delete_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.job.message.delete_canceled"));
    }
  );
}

/**
 * 启动定时任务。
 */
function handleStart(id: number, name: string) {
  ElMessageBox.confirm(t("system.base.job.message.confirm_start", { name }), t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StartBaseJob({ id }).then(() => {
        ElMessage.success(t("system.base.job.message.start_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.job.message.start_canceled"));
    }
  );
}

/**
 * 停止定时任务。
 */
function handleStop(id: number, name: string) {
  ElMessageBox.confirm(t("system.base.job.message.confirm_stop", { name }), t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StopBaseJob({ id }).then(() => {
        ElMessage.success(t("system.base.job.message.stop_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.job.message.stop_canceled"));
    }
  );
}

/**
 * 执行一次定时任务。
 */
function handleExec(id: number, name: string) {
  ElMessageBox.confirm(t("system.base.job.message.confirm_execute", { name }), t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseJobService.ExecuteBaseJob({ id }).then(() => {
        ElMessage.success(t("system.base.job.message.execute_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.job.message.execute_canceled"));
    }
  );
}

/**
 * 打开定时任务日志页面。
 */
function handleOpenBaseJob(id: number, name: string) {
  navigateTo(router, "/base/job/log", { jobId: id, title: t("system.base.job.title.log", { name }) });
}
</script>

<style scoped>
.job-operation {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  align-items: center;
  white-space: nowrap;
}
.job-action {
  margin-left: 0;
  font-weight: 500;
}
.job-action:deep(.el-icon) {
  margin-right: 4px;
}
.job-action--start {
  --el-button-text-color: var(--el-color-primary);
}
.job-action--stop {
  --el-button-text-color: var(--el-color-warning);
}
.job-action--exec {
  --el-button-text-color: var(--el-color-success);
}
.job-action--log {
  --el-button-text-color: var(--el-color-info);
}
</style>
