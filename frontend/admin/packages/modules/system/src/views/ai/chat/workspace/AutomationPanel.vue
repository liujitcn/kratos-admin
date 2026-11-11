<template>
  <section class="automation-workspace">
    <el-alert v-if="loadError" :title="loadError" :closable="false" type="error" show-icon />
    <el-alert v-if="mobile" :title="t('system.ai.workspace.desktop_hint')" :closable="false" type="info" />
    <div class="workspace-toolbar">
      <el-input
        v-model="keyword"
        clearable
        :placeholder="t('system.ai.workspace.search')"
        @keyup.enter="loadAutomations"
        @clear="loadAutomations"
      />
      <el-select v-model="status" :aria-label="t('system.ai.workspace.state')" @change="loadAutomations"
        ><el-option
          v-for="item in ['', 'active', 'paused', 'attention', 'archived']"
          :key="item"
          :value="item"
          :label="t(`system.ai.workspace.${item || 'all'}`)"
      /></el-select>
      <el-button v-if="!mobile" type="primary" @click="openForm()">{{ t("common.action.create") }}</el-button>
      <el-button @click="openRuns()">{{ t("system.ai.workspace.runs") }}</el-button>
    </div>
    <div v-loading="loading">
      <ProTable row-key="id" :columns="columns" :data="automations" :pagination="false" :tool-button="false" />
    </div>
    <FormDialog
      ref="formRef"
      v-model="formOpen"
      :title="editing ? t('common.action.edit') : t('system.ai.workspace.create_automation')"
      :model="form"
      :fields="fields"
      :confirm-loading="saving"
      width="min(720px, 96vw)"
      @confirm="saveAutomation"
      @closed="resetForm"
    >
      <template #model
        ><AiModelSelector
          :model-providers="modelProviders"
          :provider-id="form.provider_id"
          :model-name="form.model_name"
          @change="
            (id, name) => {
              form.provider_id = id;
              form.model_name = name;
            }
          "
      /></template>
      <template #plugins><PluginPicker v-model="form.plugins" :plugins="plugins" :connections="connections" /></template>
      <template #preview
        ><div v-loading="previewLoading">
          <el-alert v-if="previewError" :title="previewError" type="error" :closable="false" />
          <ol v-else>
            <li v-for="time in previewTimes" :key="time">{{ time }}</li>
          </ol>
          <p>{{ t("system.ai.workspace.schedule_hint") }}</p>
        </div></template
      >
    </FormDialog>
    <ProDialog v-model="runsOpen" :title="t('system.ai.workspace.runs')" :show-footer="false" width="min(1100px, 96vw)">
      <el-button :loading="runsLoading" @click="loadRuns">{{ t("common.action.refresh") }}</el-button>
      <el-alert v-if="runsError" :title="runsError" type="error" :closable="false" />
      <div v-loading="runsLoading">
        <ProTable row-key="id" :columns="runColumns" :data="runs" :pagination="false" :tool-button="false" />
      </div>
    </ProDialog>
    <ProDialog v-model="runOpen" :title="t('system.ai.workspace.run_details')" :show-footer="false" width="min(680px, 96vw)">
      <template v-if="runDetail"
        ><p>{{ runStatus(runDetail.status) }}</p>
        <el-alert
          v-if="runDetail.result_unknown"
          :title="t('system.ai.workspace.result_unknown')"
          :closable="false"
          type="warning"
        />
        <p>{{ runDetail.summary }}</p>
        <p>{{ runDetail.error_code }}</p>
        <h4>{{ t("system.ai.workspace.frozen_instruction") }}</h4>
        <pre>{{ runDetail.configuration?.prompt }}</pre>
        <p>{{ runDetail.configuration?.schedule?.timezone }} · {{ runDetail.configuration?.model_name }}</p>
        <el-button :disabled="!runDetail.session_id || runDetail.session_deleted" @click="openResult(runDetail)">{{
          t("system.ai.workspace.result")
        }}</el-button></template
      >
    </ProDialog>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import AiModelSelector from "@liujitcn/kratos-admin-system/components/AiModelSelector/index.vue";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import type { AiKnowledgeOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_knowledge";
import type { AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiPlugin, AiPluginConnection, AiPluginSelection } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";
import type { AiAutomation, AiAutomationForm, AiAutomationRun } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_automation";
import type { Timestamp } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/timestamp";
import { defAiAutomationService as api } from "@liujitcn/kratos-admin-system/api/base/v1/ai_automation";
import PluginPicker from "./PluginPicker.vue";
const props = defineProps<{
  mobile: boolean;
  active: boolean;
  modelProviders: AiProviderModelOption[];
  knowledgeOptions: AiKnowledgeOption[];
  plugins: AiPlugin[];
  connections: AiPluginConnection[];
  providerId: number;
  modelName: string;
  knowledgeIds: number[];
  selectedPlugins: AiPluginSelection[];
}>();
const emit = defineEmits<{ pending: [value: number]; result: [id: string]; session: [value: AiSession] }>();
const automations = ref<AiAutomation[]>([]),
  keyword = ref(""),
  status = ref(""),
  loading = ref(false),
  loadError = ref("");
const busyIDs = ref<string[]>([]);
const formOpen = ref(false),
  formRef = ref<InstanceType<typeof FormDialog>>(),
  editing = ref<AiAutomation>(),
  saving = ref(false);
const form = ref<AiAutomationForm>(defaultForm());
const previewTimes = ref<string[]>([]),
  previewError = ref(""),
  previewLoading = ref(false);
let previewTimer: ReturnType<typeof setTimeout> | undefined,
  previewSequence = 0;
const runsOpen = ref(false),
  runsLoading = ref(false),
  runsError = ref(""),
  runs = ref<AiAutomationRun[]>([]),
  runAutomationID = ref("");
const runOpen = ref(false),
  runDetail = ref<AiAutomationRun>();
let pollTimer: ReturnType<typeof setInterval> | undefined;
const columns = computed<ColumnProps[]>(() => [
  { prop: "configuration.name", label: t("system.ai.workspace.name"), minWidth: 160 },
  { prop: "configuration.prompt", label: t("system.ai.workspace.instruction"), minWidth: 220, showOverflowTooltip: true },
  { prop: "status", label: t("system.ai.workspace.state"), render: ({ row }) => t(`system.ai.workspace.${row.status}`) },
  { prop: "configuration.schedule.timezone", label: t("system.ai.workspace.timezone"), width: 160 },
  {
    prop: "next_fire_at",
    label: t("system.ai.workspace.next_fire"),
    width: 190,
    render: ({ row }) => formatTime(row.next_fire_at, row.configuration?.schedule?.timezone)
  },
  { prop: "problem", label: t("system.ai.workspace.reason"), minWidth: 160 },
  {
    cellType: "actions",
    label: t("common.field.operation"),
    width: props.mobile ? 170 : 350,
    actions: [
      {
        label: t("system.ai.workspace.run_once"),
        disabled: ({ row }) => busyIDs.value.includes(row.id) || !!row.active_run_id || row.status === "archived",
        onClick: ({ row }) => runOnce(row as AiAutomation)
      },
      { label: t("system.ai.workspace.runs"), onClick: ({ row }) => openRuns(row.id) },
      {
        label: t("common.action.edit"),
        hidden: ({ row }) => props.mobile || row.status === "archived",
        onClick: ({ row }) => openForm(row as AiAutomation)
      },
      {
        label: t("system.ai.workspace.pause"),
        hidden: ({ row }) => props.mobile || row.status !== "active",
        onClick: ({ row }) => changeStatus(row as AiAutomation, "paused")
      },
      {
        label: t("system.ai.workspace.resume"),
        hidden: ({ row }) => props.mobile || row.status !== "paused",
        onClick: ({ row }) => changeStatus(row as AiAutomation, "active")
      },
      {
        label: t("system.ai.workspace.archive"),
        hidden: ({ row }) => props.mobile || row.status === "archived",
        onClick: ({ row }) => changeStatus(row as AiAutomation, "archived")
      }
    ]
  }
]);
const runColumns = computed<ColumnProps[]>(() => [
  { prop: "configuration.name", label: t("system.ai.workspace.name"), minWidth: 140 },
  { prop: "status", label: t("system.ai.workspace.state"), render: ({ row }) => runStatus(row.status) },
  { prop: "created_at", label: t("system.ai.workspace.time"), width: 190, render: ({ row }) => formatTime(row.created_at) },
  { prop: "summary", label: t("system.ai.workspace.summary"), minWidth: 200 },
  {
    cellType: "actions",
    label: t("common.field.operation"),
    width: 280,
    actions: [
      { label: t("system.ai.workspace.details"), onClick: ({ row }) => openRun(row.id) },
      {
        label: t("system.ai.workspace.result"),
        disabled: ({ row }) => !row.session_id || row.session_deleted,
        onClick: ({ row }) => openResult(row as AiAutomationRun)
      },
      {
        label: t("system.ai.workspace.stop"),
        hidden: ({ row }) => !["queued", "running", "waiting_approval", "canceling"].includes(row.status),
        disabled: ({ row }) => row.cancel_requested || busyIDs.value.includes(row.id),
        onClick: ({ row }) => stopRun(row as AiAutomationRun)
      },
      {
        label: t("system.ai.workspace.continue_discussion"),
        disabled: ({ row }) => !row.session_id || row.session_deleted,
        onClick: ({ row }) => continueRun(row as AiAutomationRun)
      }
    ]
  }
]);
const fields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: t("system.ai.workspace.name"),
    component: "input",
    rules: [{ required: true, message: t("system.ai.workspace.required"), trigger: "blur" }],
    props: { maxlength: 100 }
  },
  {
    prop: "prompt",
    label: t("system.ai.workspace.instruction"),
    component: "textarea",
    rules: [{ required: true, message: t("system.ai.workspace.required"), trigger: "blur" }],
    props: { rows: 4, maxlength: 16000 }
  },
  {
    prop: "schedule.frequency",
    label: t("system.ai.workspace.frequency"),
    component: "select",
    options: ["daily", "weekdays", "weekly", "once"].map(value => ({ label: t(`system.ai.workspace.${value}`), value }))
  },
  {
    prop: "schedule.weekdays",
    label: t("system.ai.workspace.week_days"),
    component: "checkbox-group",
    options: Array.from({ length: 7 }, (_, index) => ({
      value: index + 1,
      label: t(`system.ai.workspace.weekday_${index + 1}`)
    })),
    visible: model => model.schedule.frequency === "weekly"
  },
  {
    prop: "schedule.time",
    label: t("system.ai.workspace.time"),
    component: "input",
    props: { type: "time" },
    visible: model => model.schedule.frequency !== "once"
  },
  {
    prop: "schedule.once_at",
    label: t("system.ai.workspace.time"),
    component: "input",
    props: { type: "datetime-local" },
    visible: model => model.schedule.frequency === "once"
  },
  {
    prop: "schedule.timezone",
    label: t("system.ai.workspace.timezone"),
    component: "input",
    rules: [{ required: true, message: t("system.ai.workspace.required"), trigger: "blur" }],
    props: { placeholder: "Asia/Shanghai" }
  },
  { prop: "model", label: t("system.ai.model.title"), component: "slot", slotName: "model" },
  {
    prop: "knowledge_base_ids",
    label: t("system.ai.knowledge.title"),
    component: "select",
    props: { multiple: true },
    options: props.knowledgeOptions.map(item => ({ value: item.id, label: item.name }))
  },
  { prop: "plugins", label: t("system.ai.workspace.plugins"), component: "slot", slotName: "plugins" },
  { prop: "preview", label: t("system.ai.workspace.preview"), component: "slot", slotName: "preview" }
]);
/** 创建表单只带入确定的模型、知识库和插件引用。 */
function defaultForm(): AiAutomationForm {
  return {
    name: "",
    prompt: "",
    schedule: {
      frequency: "daily",
      weekdays: [],
      time: "09:00",
      once_at: "",
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai"
    },
    provider_id: props.providerId,
    model_name: props.modelName,
    knowledge_base_ids: [...props.knowledgeIds],
    plugins: props.selectedPlugins.map(item => ({ ...item }))
  };
}
/** 打开草稿或从服务端读取最新版本再编辑。 */
async function openForm(item?: AiAutomation) {
  if (props.mobile) return;
  const latest = item ? await api.GetAiAutomation({ id: item.id }) : undefined;
  editing.value = latest;
  form.value = latest?.configuration ? structuredClone(latest.configuration) : defaultForm();
  formOpen.value = true;
  await previewSchedule();
}
/** 查询当前身份真实计划与待确认数量。 */
async function loadAutomations() {
  if (loading.value) return;
  loading.value = true;
  loadError.value = "";
  try {
    const result = await api.ListAiAutomation({ status: status.value, keyword: keyword.value });
    automations.value = result.automations ?? [];
    emit("pending", result.pending_count ?? 0);
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t("system.ai.workspace.failed");
  } finally {
    loading.value = false;
  }
}
/** 服务端统一算法预览，忽略晚到的旧计划响应。 */
async function previewSchedule() {
  const sequence = ++previewSequence;
  previewLoading.value = true;
  previewError.value = "";
  try {
    const result = await api.PreviewAiAutomationSchedule({ schedule: form.value.schedule });
    if (sequence !== previewSequence) return;
    previewTimes.value = (result.times ?? []).map(time => formatTime(time, result.timezone));
    if (!previewTimes.value.length) previewError.value = t("system.ai.workspace.no_future");
  } catch (error) {
    if (sequence === previewSequence) {
      previewTimes.value = [];
      previewError.value = error instanceof Error ? error.message : t("system.ai.workspace.failed");
    }
  } finally {
    if (sequence === previewSequence) previewLoading.value = false;
  }
}
/** 保存前再次校验预览，固定模型不能为空。 */
async function saveAutomation() {
  if (props.mobile || !(await formRef.value?.validate())) return;
  if (!form.value.provider_id || !form.value.model_name) {
    ElMessage.warning(t("system.ai.workspace.model_required"));
    return;
  }
  saving.value = true;
  try {
    await previewSchedule();
    if (previewError.value) return;
    if (editing.value)
      await api.UpdateAiAutomation({ id: editing.value.id, revision: editing.value.revision, automation: form.value });
    else await api.CreateAiAutomation({ automation: form.value });
    formOpen.value = false;
    await loadAutomations();
  } finally {
    saving.value = false;
  }
}
/** 弹窗关闭后清除表单与校验。 */
function resetForm() {
  editing.value = undefined;
  form.value = defaultForm();
  previewTimes.value = [];
  previewError.value = "";
  formRef.value?.clearValidate();
}
/** 暂停恢复与归档均提交当前版本，旧审批由服务端撤销。 */
async function changeStatus(item: AiAutomation, value: string) {
  if (props.mobile) return;
  await ElMessageBox.confirm(
    t("system.ai.workspace.status_confirm", { name: item.configuration?.name || item.id }),
    t(`system.ai.workspace.${value}`),
    { type: "warning" }
  );
  await api.SetAiAutomationStatus({ id: item.id, revision: item.revision, status: value });
  await loadAutomations();
}
/** 每次明确运行产生一个幂等键，双击期间复用并禁用按钮。 */
async function runOnce(item: AiAutomation) {
  if (busyIDs.value.includes(item.id)) return;
  busyIDs.value.push(item.id);
  try {
    await ElMessageBox.confirm(
      t("system.ai.workspace.run_confirm", { name: item.configuration?.name || item.id }),
      t("system.ai.workspace.run_once")
    );
    await api.RunAiAutomation({ id: item.id, revision: item.revision, request_key: crypto.randomUUID() });
    await loadAutomations();
    await openRuns(item.id);
  } finally {
    busyIDs.value = busyIDs.value.filter(id => id !== item.id);
  }
}
/** 打开全部或单任务执行记录。 */
async function openRuns(id = "") {
  runAutomationID.value = id;
  runsOpen.value = true;
  await loadRuns();
}
/** 刷新真实执行记录，失败不显示为成功空态。 */
async function loadRuns() {
  runsLoading.value = true;
  runsError.value = "";
  try {
    runs.value = (await api.ListAiAutomationRun({ automation_id: runAutomationID.value, limit: 100 })).runs ?? [];
  } catch (error) {
    runsError.value = error instanceof Error ? error.message : t("system.ai.workspace.failed");
  } finally {
    runsLoading.value = false;
  }
}
/** 读取执行的冻结配置与问题详情。 */
async function openRun(id: string) {
  runDetail.value = await api.GetAiAutomationRun({ id });
  runOpen.value = true;
}
/** 结果会话只读打开，审批仍在原结果中进行。 */
function openResult(run: AiAutomationRun) {
  runsOpen.value = false;
  runOpen.value = false;
  emit("result", run.session_id);
}
/** 请求停止本次，不承诺撤回外部已经完成的写入。 */
async function stopRun(run: AiAutomationRun) {
  if (busyIDs.value.includes(run.id)) return;
  busyIDs.value.push(run.id);
  try {
    await api.StopAiAutomationRun({ id: run.id });
    await loadRuns();
    await loadAutomations();
  } finally {
    busyIDs.value = busyIDs.value.filter(id => id !== run.id);
  }
}
/** 让服务端创建普通会话，避免修改冻结结果。 */
async function continueRun(run: AiAutomationRun) {
  const response = await api.ContinueAiAutomationRun({ id: run.id });
  if (response.session) {
    runsOpen.value = false;
    emit("session", response.session);
  }
}
/** 按指定IANA时区展示服务端UTC时间。 */
function formatTime(time?: Timestamp | string, timezone?: string) {
  if (!time) return "—";
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: timezone || undefined
  }).format(new Date(typeof time === "string" ? time : Number(time.seconds) * 1000));
}
/** 执行状态使用四语言固定映射。 */
function runStatus(value: string) {
  return t(`system.ai.workspace.run_${value}`);
}
watch(
  () => form.value.schedule,
  () => {
    if (!formOpen.value) return;
    clearTimeout(previewTimer);
    previewTimer = setTimeout(() => void previewSchedule(), 300);
  },
  { deep: true }
);
watch(
  () => props.mobile,
  mobile => {
    if (mobile) formOpen.value = false;
  }
);
watch(
  () => props.active,
  active => {
    if (active) void loadAutomations();
  }
);
onMounted(() => {
  void loadAutomations();
  pollTimer = setInterval(() => {
    if (document.visibilityState !== "visible") return;
    if (props.active || automations.value.some(item => item.active_run_id)) void loadAutomations();
    if (runsOpen.value && !runsLoading.value) void loadRuns();
  }, 15000);
});
onBeforeUnmount(() => {
  clearTimeout(previewTimer);
  clearInterval(pollTimer);
});
defineExpose({ refresh: loadAutomations });
</script>
<style scoped lang="scss">
.automation-workspace {
  min-width: 0;
  min-height: 0;
  padding: 20px;
  overflow: auto;
}
.workspace-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.workspace-toolbar {
  margin: 16px 0;
}
.workspace-toolbar > .el-input {
  width: 240px;
}
.workspace-toolbar > .el-select {
  width: 160px;
}
p,
li {
  color: var(--admin-page-text-secondary);
  line-height: 1.7;
}
pre {
  white-space: pre-wrap;
  color: var(--admin-page-text-regular);
  background: var(--el-fill-color-light);
  padding: 12px;
  border-radius: var(--admin-page-radius);
}
@media (width <= 768px) {
  .automation-workspace {
    padding: 12px;
  }
}
</style>
