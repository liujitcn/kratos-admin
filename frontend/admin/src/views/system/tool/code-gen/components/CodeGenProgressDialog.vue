<template>
  <ProDialog
    :model-value="modelValue"
    title="代码生成进度"
    width="min(1080px, 94vw)"
    top="4vh"
    :close-on-click-modal="false"
    :show-footer="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-loading="loading" class="progress-dialog">
      <template v-if="task">
        <div class="task-heading">
          <div>
            <strong>{{ task.message }}</strong>
            <span v-if="task.current_table_name">当前：{{ task.current_table_name }}</span>
          </div>
          <el-tag :type="statusMeta(task.status).type" effect="plain">{{ statusMeta(task.status).label }}</el-tag>
        </div>
        <el-progress
          :percentage="percentage(task.completed_steps, task.total_steps)"
          :status="progressStatus(task.status)"
          :stroke-width="12"
        />
        <div class="task-meta">
          <span>{{ task.completed_steps }} / {{ task.total_steps }} 步</span>
          <span>创建：{{ formatTime(task.created_at) }}</span>
          <span v-if="task.finished_at">完成：{{ formatTime(task.finished_at) }}</span>
        </div>

        <el-collapse v-model="expandedTables" class="table-progress-list">
          <el-collapse-item v-for="table in task.tables" :key="table.table_id" :name="table.table_id">
            <template #title>
              <div class="table-heading">
                <strong>{{ table.table_name }}</strong>
                <span>{{ table.completed_steps }} / {{ table.total_steps }}</span>
                <el-tag :type="statusMeta(table.status).type" size="small" effect="plain">
                  {{ statusMeta(table.status).label }}
                </el-tag>
              </div>
            </template>
            <div v-if="table.message" class="table-message">
              <span>{{ table.message }}</span>
              <el-tooltip content="复制错误信息" placement="top">
                <el-button :icon="CopyDocument" link aria-label="复制错误信息" @click.stop="copyMessage(table.message)" />
              </el-tooltip>
            </div>
            <el-table :data="table.steps" row-key="id" size="small" border>
              <el-table-column label="状态" width="86" align="center">
                <template #default="{ row }">
                  <el-icon :class="{ 'is-loading': row.status === CodeGenTaskStepStatus.CODE_GEN_TASK_STEP_STATUS_RUNNING }">
                    <Loading v-if="row.status === CodeGenTaskStepStatus.CODE_GEN_TASK_STEP_STATUS_RUNNING" />
                    <Check v-else-if="row.status === CodeGenTaskStepStatus.CODE_GEN_TASK_STEP_STATUS_SUCCEEDED" />
                    <CircleClose v-else-if="row.status === CodeGenTaskStepStatus.CODE_GEN_TASK_STEP_STATUS_FAILED" />
                    <Minus v-else-if="row.status === CodeGenTaskStepStatus.CODE_GEN_TASK_STEP_STATUS_SKIPPED" />
                    <Clock v-else />
                  </el-icon>
                </template>
              </el-table-column>
              <el-table-column prop="label" label="步骤" min-width="130" />
              <el-table-column prop="path" label="文件路径" min-width="260" show-overflow-tooltip />
              <el-table-column prop="message" label="结果" min-width="180" show-overflow-tooltip />
              <el-table-column label="输出" width="90" align="center">
                <template #default="{ row }">
                  <el-button v-if="row.output" :icon="Document" link @click="showOutput(row.label, row.output)" />
                </template>
              </el-table-column>
            </el-table>
          </el-collapse-item>
        </el-collapse>
      </template>
      <el-result v-else-if="errorMessage" icon="error" title="任务不可用" :sub-title="errorMessage">
        <template #extra>
          <el-button type="primary" @click="startTracking">重新加载</el-button>
        </template>
      </el-result>
    </div>

  </ProDialog>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { Check, CircleClose, Clock, CopyDocument, Document, Loading, Minus } from "@element-plus/icons-vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defCodeGenService } from "@/api/system/code_gen";
import { subscribeCodeGenProgress, type SseStop } from "@/api/base/sse";
import type { CodeGenTask } from "@/rpc/system/admin/v1/code_gen";
import { CodeGenTaskStatus, CodeGenTaskStepStatus } from "@/rpc/system/admin/v1/code_gen";

defineOptions({ name: "CodeGenProgressDialog", inheritAttrs: false });

/** 代码生成进度弹窗属性。 */
const props = defineProps<{ modelValue: boolean; taskId: string }>();

/** 代码生成进度弹窗事件。 */
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  completed: [];
  unavailable: [];
}>();

/** 状态展示信息。 */
type StatusMeta = { label: string; type: "success" | "warning" | "danger" | "info" };

const task = ref<CodeGenTask>();
const loading = ref(false);
const errorMessage = ref("");
const expandedTables = ref<number[]>([]);
let stopSse: SseStop | undefined;
let refreshTimer: ReturnType<typeof setInterval> | undefined;
let completedTaskId = "";

/** 启动SSE订阅与三秒轮询兜底。 */
async function startTracking() {
  stopTracking();
  const taskId = props.taskId;
  await loadTask(false, taskId);
  if (taskId !== props.taskId || !task.value || isFinished(task.value.status)) return;
  stopSse = subscribeCodeGenProgress(taskId, latest => {
    if (taskId === props.taskId) applyTask(latest);
  });
  refreshTimer = setInterval(() => void loadTask(true, taskId), 3000);
}

/** 查询当前任务快照。 */
async function loadTask(silent = false, taskId = props.taskId) {
  if (!taskId) {
    markUnavailable();
    return;
  }
  if (!silent) loading.value = true;
  try {
    const latest = await defCodeGenService.GetCodeGenTask({ task_id: taskId });
    if (taskId !== props.taskId) return;
    if (!latest.task_id) {
      markUnavailable();
      return;
    }
    applyTask(latest);
    errorMessage.value = "";
  } catch {
    if (taskId === props.taskId && (!silent || !task.value)) markUnavailable();
  } finally {
    if (!silent && taskId === props.taskId) loading.value = false;
  }
}

/** 应用实时或轮询返回的任务快照。 */
function applyTask(latest: CodeGenTask) {
  task.value = latest;
  const runningTable = latest.tables.find(table => table.status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_RUNNING);
  if (runningTable && !expandedTables.value.includes(runningTable.table_id)) {
    expandedTables.value = [runningTable.table_id];
  }
  if (isFinished(latest.status)) {
    stopTracking();
    if (completedTaskId !== latest.task_id) {
      completedTaskId = latest.task_id;
      emit("completed");
    }
  }
}

/** 标记任务已过期或无权访问。 */
function markUnavailable() {
  stopTracking();
  task.value = undefined;
  errorMessage.value = "任务不存在、已过期或无权访问";
  emit("unavailable");
}

/** 停止当前任务跟踪。 */
function stopTracking() {
  stopSse?.();
  stopSse = undefined;
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = undefined;
}

/** 返回状态标签配置。 */
function statusMeta(status: CodeGenTaskStatus): StatusMeta {
  if (status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_RUNNING) return { label: "执行中", type: "warning" };
  if (status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_SUCCEEDED) return { label: "已完成", type: "success" };
  if (status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_FAILED) return { label: "失败", type: "danger" };
  return { label: "等待中", type: "info" };
}

/** 返回进度条状态。 */
function progressStatus(status: CodeGenTaskStatus): "success" | "exception" | undefined {
  if (status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_SUCCEEDED) return "success";
  if (status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_FAILED) return "exception";
  return undefined;
}

/** 计算任务完成百分比。 */
function percentage(completed: number, total: number) {
  return total > 0 ? Math.round((completed / total) * 100) : 0;
}

/** 判断任务是否进入终态。 */
function isFinished(status: CodeGenTaskStatus) {
  return status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_SUCCEEDED || status === CodeGenTaskStatus.CODE_GEN_TASK_STATUS_FAILED;
}

/** 格式化任务时间。 */
function formatTime(value: string) {
  const date = new Date(value);
  return !value || Number.isNaN(date.getTime()) ? "-" : date.toLocaleString();
}

/** 弹出展示命令输出。 */
function showOutput(title: string, output: string) {
  ElMessageBox.alert(`<pre class="code-gen-command-output">${escapeHTML(output)}</pre>`, title, {
    dangerouslyUseHTMLString: true,
    customClass: "code-gen-output-dialog",
    confirmButtonText: "关闭"
  });
}

/** 复制生成任务的错误信息，便于定位生成命令问题。 */
async function copyMessage(message: string) {
	try {
		await navigator.clipboard.writeText(message);
		ElMessage.success("错误信息已复制");
	} catch {
		ElMessage.error("复制失败，请手动复制");
	}
}

/** 转义命令输出中的HTML字符。 */
function escapeHTML(value: string) {
  return value.replace(
    /[&<>"']/g,
    character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character]!
  );
}

watch(
  () => props.taskId,
  taskId => {
    if (taskId) {
      // 任务运行期间即使弹窗隐藏，也继续跟踪终态并通知父页面解除生成锁定。
      void startTracking();
      return;
    }
    stopTracking();
    task.value = undefined;
    errorMessage.value = "";
  },
  { immediate: true }
);

onBeforeUnmount(stopTracking);
</script>

<style scoped lang="scss">
.progress-dialog {
  min-height: 220px;
}

.task-heading,
.table-heading,
.task-meta {
  display: flex;
  gap: 12px;
  align-items: center;
}

.task-heading {
  justify-content: space-between;
  margin-bottom: 14px;
}

.task-heading > div {
  display: flex;
  gap: 12px;
  align-items: baseline;
}

.task-meta {
  flex-wrap: wrap;
  justify-content: space-between;
  margin: 10px 0 18px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.table-progress-list {
  max-height: 54vh;
  overflow: auto;
}

.table-heading {
  width: 100%;
  padding-right: 16px;
}

.table-heading strong {
  margin-right: auto;
}

.table-message {
	display: flex;
	gap: 6px;
	align-items: flex-start;
	margin: 0 0 10px;
	color: var(--el-text-color-secondary);
}

.table-message span {
	min-width: 0;
	white-space: pre-wrap;
	word-break: break-word;
}
</style>

<style lang="scss">
.code-gen-command-output {
  max-height: 56vh;
  margin: 0;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
  text-align: left;
  white-space: pre-wrap;
}
</style>
