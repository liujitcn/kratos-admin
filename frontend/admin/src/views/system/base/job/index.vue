<!-- 定时任务 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestBaseJobTable" />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="1000px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="150px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, type VNode } from "vue";
import { ElButton, ElMessage, ElMessageBox, ElTag } from "element-plus";
import { CirclePlus, Delete, EditPen, Promotion, Tickets, VideoPause, VideoPlay } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance, RenderScope } from "@/components/ProTable/interface";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseJobService } from "@/api/system/base_job";
import type { BaseJob, BaseJobArgs, BaseJobForm, PageBaseJobRequest } from "@/rpc/system/admin/v1/base_job";
import router from "@/routers";
import { Status } from "@/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";
import { navigateTo } from "@/utils/router";

defineOptions({
  name: "BaseJob",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
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
  status: Status.ENABLE
});

const rules = computed(() => ({
  name: [
    { required: true, message: "请输入任务名称", trigger: "blur" },
    { max: 50, message: "任务名称不能超过 50 个字符", trigger: "blur" }
  ],
  cron_expression: [
    { required: true, message: "请输入cron表达式", trigger: "blur" },
    { max: 50, message: "cron表达式不能超过 50 个字符", trigger: "blur" }
  ],
  invoke_target: [
    { required: true, message: "请输入调用目标", trigger: "blur" },
    { max: 100, message: "调用目标不能超过 100 个字符", trigger: "blur" }
  ],
  args: {
    validator: (rule: unknown, value: BaseJobArgs[], callback: (error?: Error) => void) => {
      if (value.some(arg => !arg.key)) callback(new Error("所有参数必须填写key"));
      else callback();
    },
    trigger: "blur"
  },
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

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
        () => "编辑"
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
        () => "删除"
      )
    );
  }

  if (row.status === Status.ENABLE && (row.entry_id === undefined || row.entry_id === 0) && BUTTONS.value["base:job:start"]) {
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
        () => "启动"
      )
    );
  }

  if (row.status === Status.ENABLE && row.entry_id > 0 && BUTTONS.value["base:job:stop"]) {
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
        () => "停止"
      )
    );
  }

  if (row.status === Status.ENABLE && (row.entry_id === undefined || row.entry_id === 0) && BUTTONS.value["base:job:exec"]) {
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
        () => "执行一次"
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
        () => "日志"
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
const formFields: ProFormField[] = [
  { prop: "name", label: "任务名称", component: "input", props: { placeholder: "请输入任务名称" } },
  { prop: "invoke_target", label: "调用目标", component: "input", props: { placeholder: "请输入调用目标" } },
  { prop: "cron_expression", label: "cron表达式", component: "cron-expression", props: { placeholder: "0 0 0 * * *" } },
  {
    prop: "args",
    label: "目标参数",
    component: "kv-list",
    props: {
      keyInputProps: { placeholder: "参数" },
      valueInputProps: { placeholder: "值" },
      addText: "添加参数"
    }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 定时任务表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "任务名称", minWidth: 140, search: { el: "input" } },
  { prop: "invoke_target", label: "调用目标", minWidth: 180, search: { el: "input" } },
  { prop: "args", label: "参数", minWidth: 140, render: scope => renderArgsCell(scope as unknown as RenderScope<BaseJob>) },
  { prop: "cron_expression", label: "cron表达式", minWidth: 150 },
  { prop: "entry_id", label: "任务id", minWidth: 100, align: "right" },
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
      disabled: scope => (scope.row as BaseJob).entry_id === 0 || !BUTTONS.value["base:job:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseJob)
    }
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 380,
    fixed: "right",
    render: scope => renderOperationCell(scope as unknown as RenderScope<BaseJob>)
  }
];

/** 定时任务顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:job:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:job:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseJob[])
  }
];

/**
 * 请求定时任务列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseJobTable(params: PageBaseJobRequest) {
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
function handleOpenDialog(jobId?: number) {
  resetForm();
  dialog.title = jobId ? "修改定时任务" : "新增定时任务";
  dialog.visible = true;
  if (!jobId) return;

  defBaseJobService.GetBaseJob({ id: jobId }).then(data => {
    Object.assign(formData, data);
  });
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
  formData.status = Status.ENABLE;
}

/**
 * 提交定时任务表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseJobForm;
    const request = submitData.id
      ? defBaseJobService.UpdateBaseJob({ base_job: submitData })
      : defBaseJobService.CreateBaseJob({ base_job: submitData });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改定时任务成功" : "新增定时任务成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在定时任务状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseJob) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const jobName = row.name || row.invoke_target || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}定时任务？\n任务名称：${jobName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseJobService.SetBaseJobStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
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
    ElMessage.warning("请勾选删除项");
    return;
  }

  const singleJobName = jobList[0]?.name || jobList[0]?.invoke_target || `ID:${jobList[0]?.id ?? ""}`;
  const confirmMessage = jobList.length
    ? jobList.length === 1
      ? `是否确定删除定时任务？\n任务名称：${singleJobName}`
      : `确认删除已选中的 ${jobList.length} 个定时任务吗？`
    : "确认删除已选中的定时任务吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.DeleteBaseJob({ id: jobIds }).then(() => {
        ElMessage.success("删除定时任务成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除定时任务");
    }
  );
}

/**
 * 启动定时任务。
 */
function handleStart(id: number, name: string) {
  ElMessageBox.confirm(`是否确定启动定时任务？\n任务名称：${name}`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StartBaseJob({ id }).then(() => {
        ElMessage.success("启动成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消启动定时任务");
    }
  );
}

/**
 * 停止定时任务。
 */
function handleStop(id: number, name: string) {
  ElMessageBox.confirm(`是否确定停止定时任务？\n任务名称：${name}`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StopBaseJob({ id }).then(() => {
        ElMessage.success("停止成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消停止定时任务");
    }
  );
}

/**
 * 执行一次定时任务。
 */
function handleExec(id: number, name: string) {
  ElMessageBox.confirm(`是否确定执行一次定时任务？\n任务名称：${name}`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.ExecuteBaseJob({ id }).then(() => {
        ElMessage.success("执行成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消执行定时任务");
    }
  );
}

/**
 * 打开定时任务日志页面。
 */
function handleOpenBaseJob(id: number, name: string) {
  navigateTo(router, "/base/job/log", { jobId: id, title: `【${name}】定时任务日志` });
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
