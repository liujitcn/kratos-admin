<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestBaseScriptTable" />

    <FormDialog
      ref="formDialogRef"
      v-model="dialog.visible"
      :title="t(dialog.editing ? 'system.base.script.action.edit' : 'system.base.script.action.create')"
      width="min(1180px, calc(100vw - 32px))"
      top="3vh"
      label-width="130px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #source>
        <CodeEditor v-model="formData.source" :language="editorLanguage" height="360px" />
        <div v-if="formData.language === BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_WASM" class="source-hint">
          {{ t("system.base.script.placeholder.wasm_source") }}
        </div>
      </template>
    </FormDialog>

    <ProDialog ref="testDialogRef" v-model="testDialog.visible" :title="t('system.base.script.test.title')" width="760px">
      <el-descriptions :column="2" border>
        <el-descriptions-item :label="t('system.base.script.field.name')">{{ testScript?.name }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.field.language')">
          <DictLabel code="base_script_language" :model-value="testScript?.language" />
        </el-descriptions-item>
      </el-descriptions>
      <el-form class="test-form" label-position="top">
        <el-form-item :label="t('system.base.script.test.context')">
          <el-input v-model="testContext" type="textarea" :rows="8" spellcheck="false" />
        </el-form-item>
      </el-form>
      <el-alert v-if="testResult" :type="testResult.success ? 'success' : 'error'" :title="testResult.success ? t('system.base.script.test.success') : t('system.base.script.test.failed')" show-icon :closable="false">
        <template v-if="!testResult.success">{{ testResult.error }}</template>
      </el-alert>
      <template v-if="testResult?.success">
        <div class="result-title">{{ t('system.base.script.test.output') }} · {{ testResult.duration_ms }}ms</div>
        <pre class="result-block">{{ formatJson(testResult.output_json) }}</pre>
      </template>
      <template #footer>
        <el-button @click="testDialogRef?.close()">{{ t('common.action.close') }}</el-button>
        <el-button type="primary" :loading="testLoading" @click="handleTestRun">{{ t('system.base.script.action.run') }}</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, Promotion, Tickets } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import CodeEditor from "@liujitcn/kratos-admin-core/components/CodeEditor/index.vue";
import DictLabel from "@liujitcn/kratos-admin-core/components/Dict/DictLabel.vue";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { formatJson } from "@liujitcn/kratos-admin-core/format";
import { t } from "@liujitcn/kratos-admin-core";
import router, { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { defBaseScriptService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_script";
import type { BaseScript, BaseScriptForm, PageBaseScriptRequest, TestRunBaseScriptResponse } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script";
import { BaseScriptLanguage } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseScript", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const testDialogRef = ref<InstanceType<typeof ProDialog>>();
const testLoading = ref(false);
const testScript = ref<BaseScript>();
const testContext = ref("{}\n");
const testResult = ref<TestRunBaseScriptResponse>();

const defaultSource = "function execute(ctx) {\n  return ctx;\n}\n\nexecute(ctx);";
const sourceTemplates: Partial<Record<BaseScriptLanguage, string>> = {
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_JAVASCRIPT]: defaultSource,
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_LUA]: "return ctx",
	[BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_PYTHON]: "1 + 1",
	[BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_GO]: "1 + 1",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_CEL]: "ctx",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_EXPR]: "ctx",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_STARLARK]: "ctx",
	[BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_TCL]: "expr {1 + 1}"
};
const monacoLanguageMap: Partial<Record<BaseScriptLanguage, string>> = {
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_JAVASCRIPT]: "javascript",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_LUA]: "lua",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_PYTHON]: "python",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_GO]: "go",
  [BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_STARLARK]: "python"
};

const dialog = reactive({ visible: false, editing: false });
const testDialog = reactive({ visible: false });
const formData = reactive<BaseScriptForm>({
  id: 0,
  name: "",
  language: BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_JAVASCRIPT,
  hook_point: "",
  source: "function execute(ctx) {\n  return ctx;\n}\n\nexecute(ctx);",
  priority: 0,
  description: "",
  critical: false,
  status: Status.STATUS_ENABLE
});

const editorLanguage = computed(() => monacoLanguageMap[formData.language] ?? "plaintext");

watch(
  () => formData.language,
  (language, previousLanguage) => {
    const previousSource = sourceTemplates[previousLanguage];
    if (previousSource && formData.source === previousSource) {
      formData.source = sourceTemplates[language] ?? "";
    }
  }
);

const rules = computed(() => ({
  name: [{ required: true, message: t("system.base.script.validation.name"), trigger: "blur" }],
  source: [{ required: true, message: t("system.base.script.validation.source"), trigger: "blur" }]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const formFields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("system.base.script.field.name"), component: "input", props: { maxlength: 100, showWordLimit: true } },
  { prop: "language", label: t("system.base.script.field.language"), component: "dict", props: { code: "base_script_language", codeType: "number", style: { width: "100%" } } },
  { prop: "hook_point", label: t("system.base.script.field.hook_point"), component: "input", props: { maxlength: 160, placeholder: t("system.base.script.placeholder.hook_point") } },
  { prop: "priority", label: t("system.base.script.field.priority"), component: "input-number", props: { min: 0, max: 1000, precision: 0 } },
  { prop: "description", label: t("system.base.script.field.description"), component: "textarea", props: { maxlength: 512, rows: 2, showWordLimit: true } },
  { prop: "critical", label: t("system.base.script.field.critical"), component: "switch" },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "source", label: t("system.base.script.field.source"), component: "slot" }
]);

const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("system.base.script.field.name"), minWidth: 180, search: { el: "input" } },
  { prop: "language", label: t("system.base.script.field.language"), minWidth: 110, dictCode: "base_script_language", dictValueType: "number" },
  { prop: "hook_point", label: t("system.base.script.field.hook_point"), minWidth: 180, search: { el: "input" } },
  { prop: "priority", label: t("system.base.script.field.priority"), minWidth: 90, align: "right" },
  { prop: "version", label: t("system.base.script.field.version"), minWidth: 90, align: "right" },
  { prop: "status", label: t("common.field.status"), minWidth: 100, search: { el: "select" }, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), beforeChange: scope => handleBeforeSetStatus(scope.row as BaseScript) } },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
  { prop: "operation", label: t("common.field.action"), cellType: "actions", actions: [
    { label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:script:update"], onClick: scope => handleOpenDialog((scope.row as BaseScript).id) },
    { label: t("system.base.script.action.run"), type: "success", link: true, icon: Promotion, hidden: () => !BUTTONS.value["base:script:test-run"], onClick: scope => openTestRun(scope.row as BaseScript) },
    { label: t("system.base.script.action.log"), type: "info", link: true, icon: Tickets, hidden: () => !BUTTONS.value["base:script:log"], onClick: scope => openLogs(scope.row as BaseScript) },
    { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:script:delete"], onClick: scope => handleDelete(scope.row as BaseScript) }
  ] }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:script:create"], onClick: () => handleOpenDialog() }
]);

async function requestBaseScriptTable(params: PageBaseScriptRequest) {
  const data = await defBaseScriptService.PageBaseScript(buildPageRequest(params));
  return { data: { list: data.base_scripts ?? [], total: data.total } };
}

async function handleOpenDialog(id?: number) {
  resetForm();
  dialog.editing = Boolean(id);
  await formDialogRef.value?.open({
    load: () => (id ? defBaseScriptService.GetBaseScript({ id }) : undefined),
    commit: data => { if (data) Object.assign(formData, data); }
  });
}

function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, { id: 0, name: "", hook_point: "", priority: 0, description: "", critical: false, status: Status.STATUS_ENABLE, language: BaseScriptLanguage.BASE_SCRIPT_LANGUAGE_JAVASCRIPT, source: defaultSource });
}

function handleCloseDialog() {
  formDialogRef.value?.close();
  resetForm();
}

function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const payload = JSON.parse(JSON.stringify(formData)) as BaseScriptForm;
    const request = payload.id ? defBaseScriptService.UpdateBaseScript({ base_script: payload }) : defBaseScriptService.CreateBaseScript({ base_script: payload });
    request.then(() => { ElMessage.success(t("system.base.script.message.saved")); handleCloseDialog(); proTable.value?.getTableList(); });
  });
}

async function handleBeforeSetStatus(row: BaseScript) {
  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(t("system.base.script.message.confirm_status", { name: row.name }), t("common.title.notice"), { type: "warning" });
    await defBaseScriptService.SetBaseScriptStatus({ id: row.id, status: nextStatus });
    proTable.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

function handleDelete(row: BaseScript) {
  ElMessageBox.confirm(t("system.base.script.message.confirm_delete", { name: row.name }), t("common.title.warning"), { type: "warning" }).then(async () => {
    await defBaseScriptService.DeleteBaseScript({ id: String(row.id) });
    ElMessage.success(t("common.message.delete_success"));
    proTable.value?.getTableList();
  });
}

function openTestRun(row: BaseScript) {
  testScript.value = row;
  testContext.value = "{}\n";
  testResult.value = undefined;
  testDialog.visible = true;
}

async function handleTestRun() {
  if (!testScript.value) return;
  testLoading.value = true;
  try {
    testResult.value = await defBaseScriptService.TestRunBaseScript({ id: testScript.value.id, language: testScript.value.language, source: "", context_json: testContext.value });
  } finally {
    testLoading.value = false;
  }
}

function openLogs(row: BaseScript) {
  navigateTo(router, "/base/script/log", { scriptId: row.id, title: t("system.base.script.title.logs", { name: row.name }) });
}
</script>

<style scoped>
.test-form { margin-top: 16px; }
.result-title { margin-top: 16px; font-weight: 600; }
.result-block { max-height: 260px; padding: 12px; margin: 8px 0 0; overflow: auto; background: var(--el-fill-color-light); border-radius: var(--admin-page-radius); white-space: pre-wrap; }
.source-hint { margin-top: 8px; color: var(--el-text-color-secondary); font-size: 12px; }
</style>
