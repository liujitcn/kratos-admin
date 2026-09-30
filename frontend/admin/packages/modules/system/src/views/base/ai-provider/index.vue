<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.editing ? 'common.action.edit' : 'common.action.create')"
      width="min(980px, calc(100vw - 32px))"
      label-width="10em"
      :model="form"
      :fields="fields"
      :rules="rules"
      :form-props="{ disabled: testingModels }"
      :close-on-click-modal="!testingModels"
      :close-on-press-escape="!testingModels"
      @confirm="submit"
      @close="resetForm"
    >
      <template #models>
        <div class="provider-models">
          <div v-if="form.models.length === 0" class="provider-models__empty">
            {{ t("system.base.ai_provider.model.empty") }}
          </div>
          <div v-else class="provider-models__list">
            <div v-for="(model, index) in form.models" :key="`${model.model_name}-${index}`" class="provider-models__item">
              <div class="provider-models__content">
                <div class="provider-models__heading">
                  <strong>{{ model.model_name }}</strong>
                  <span v-if="model.display_name" class="provider-models__alias">✦ {{ model.display_name }}</span>
                  <el-tag size="small" effect="plain">{{ categoryLabel(model.category) }}</el-tag>
                  <el-tag size="small" effect="plain" :type="model.status === Status.STATUS_ENABLE ? 'success' : 'info'">
                    {{ t(model.status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled") }}
                  </el-tag>
                  <el-tooltip
                    v-if="modelTestResults[model.model_name]"
                    :content="resolvePersistedMessage(modelTestResults[model.model_name]?.message)"
                    :disabled="!modelTestResults[model.model_name]?.message"
                    placement="top"
                  >
                    <el-tag
                      size="small"
                      effect="plain"
                      :type="modelTestResults[model.model_name]?.success ? 'success' : 'danger'"
                    >
                      {{ t(modelTestResults[model.model_name]?.success
                        ? "system.base.ai.model.action.test_success"
                        : "system.base.ai.model.action.test_failed") }} · {{ modelTestResults[model.model_name]?.duration_ms }} ms
                    </el-tag>
                  </el-tooltip>
                </div>
                <div class="provider-models__details">
                  <template v-if="model.category === AiModelCategory.AI_MODEL_CATEGORY_CHAT">
                    <span>{{ t("system.base.ai.model.field.api_type") }}: {{ apiTypeLabel(model.api_type) }}</span>
                    <span>{{ t("system.base.ai.model.field.temperature") }}: {{ model.temperature }}</span>
                    <span>{{ t("system.base.ai.model.field.max_tokens") }}: {{ model.max_tokens }}</span>
                  </template>
                  <span v-if="model.category === AiModelCategory.AI_MODEL_CATEGORY_EMBEDDING && model.dimensions > 0">
                    {{ t("system.base.ai.model.field.dimensions") }}: {{ model.dimensions }}
                  </span>
                  <span>{{ t("system.base.ai.model.field.timeout_seconds") }}: {{ model.timeout_seconds }}</span>
                  <span>{{ t("system.base.ai.model.field.max_retries") }}: {{ model.max_retries }}</span>
                </div>
              </div>
              <div class="provider-models__actions">
                <el-tooltip :content="t('common.action.edit')" placement="top">
                  <el-button text circle :disabled="testingModels" :icon="EditPen" :aria-label="t('common.action.edit')" @click="openModelDialog(index)" />
                </el-tooltip>
                <el-tooltip :content="t('common.action.delete')" placement="top">
                  <el-button text circle :disabled="testingModels" :icon="Delete" :aria-label="t('common.action.delete')" @click="removeModel(index)" />
                </el-tooltip>
              </div>
            </div>
          </div>
          <el-button type="primary" plain :icon="CirclePlus" :disabled="testingModels" @click="openModelDialog()">
            {{ t("system.base.ai_provider.model.add") }}
          </el-button>
        </div>
      </template>
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="testingModels" @click="dialog.visible = false">
            {{ t("common.action.cancel") }}
          </el-button>
          <el-button type="primary" plain :icon="Connection" :loading="testingModels" :disabled="!form.models.length" @click="testProviderModels">
            {{ t("system.base.ai_provider.model.test") }}
          </el-button>
          <el-button type="primary" :disabled="testingModels" @click="submit">
            {{ t("common.action.confirm") }}
          </el-button>
        </div>
      </template>
    </FormDialog>

    <FormDialog
      v-model="modelDialog.visible"
      ref="modelDialogRef"
      :title="t(modelDialog.editing ? 'system.base.ai_provider.model.edit' : 'system.base.ai_provider.model.add')"
      width="min(720px, calc(100vw - 32px))"
      label-width="10em"
      :model="modelForm"
      :fields="modelFields"
      :rules="modelRules"
      @confirm="saveModel"
      @close="resetModelDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Connection, Delete, EditPen } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { encodeSecretFields } from "@liujitcn/kratos-admin-core/security";
import { resolvePersistedMessage } from "../../utils/persisted-message";
import { defAiProviderService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_provider";
import {
  type AiProvider,
  type AiProviderForm,
  AiModelCategory,
  type AiProviderModelForm,
  AiProviderModelTestResult,
  type PageAiProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_provider";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "AiProvider", inheritAttrs: false });

/** Provider个性化配置编辑项。 */
interface ConfigItem {
  key: string;
  value: string;
}

/** 单个AI模型的页面表单状态；分类个性化配置先拍平成兄弟字段，提交时再收敛进 config。 */
interface ModelFormState extends Omit<AiProviderModelForm, "config"> {
  api_type: string;
  temperature: number;
  max_tokens: number;
  dimensions: number;
  duration: number;
  size: string;
  quality: string;
  resolution: string;
  task: string;
  voice: string;
  speed: number;
  timeout_seconds: number;
  max_retries: number;
}

/** AI Provider页面表单状态。 */
interface FormState extends Omit<AiProviderForm, "models"> {
  models: ModelFormState[];
  config_items: ConfigItem[];
}

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const modelDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, editing: false });
const modelDialog = reactive({ visible: false, editing: false, index: -1 });
const form = reactive<FormState>(defaultForm());
const modelForm = reactive<ModelFormState>(defaultModelForm());
const testingModels = ref(false);
const modelTestResults = ref<Record<string, AiProviderModelTestResult>>({});

watch(
  () => [form.provider, form.base_url, form.api_key, JSON.stringify(form.config_items), JSON.stringify(form.models)],
  () => {
    modelTestResults.value = {};
  }
);

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const providerOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai_provider.provider.openai_compatible"), value: "openai_compatible" },
  { label: t("system.base.ai_provider.provider.ollama"), value: "ollama" }
]);

const categoryOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai.model.category.chat"), value: AiModelCategory.AI_MODEL_CATEGORY_CHAT },
  { label: t("system.base.ai.model.category.embedding"), value: AiModelCategory.AI_MODEL_CATEGORY_EMBEDDING },
  { label: t("system.base.ai.model.category.rerank"), value: AiModelCategory.AI_MODEL_CATEGORY_RERANK },
  { label: t("system.base.ai.model.category.image"), value: AiModelCategory.AI_MODEL_CATEGORY_IMAGE },
  { label: t("system.base.ai.model.category.video"), value: AiModelCategory.AI_MODEL_CATEGORY_VIDEO },
  { label: t("system.base.ai.model.category.audio"), value: AiModelCategory.AI_MODEL_CATEGORY_AUDIO }
]);

const apiTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai.model.api_type.chat_completions"), value: "CHAT_COMPLETIONS" },
  { label: t("system.base.ai.model.api_type.responses"), value: "RESPONSES" }
]);

const taskOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai.model.task.tts"), value: "tts" },
  { label: t("system.base.ai.model.task.asr"), value: "asr" }
]);

/** 分类个性化配置的表单字段，按分类过滤展示。 */
const categoryConfigFields: Partial<Record<AiModelCategory, string[]>> = {
  [AiModelCategory.AI_MODEL_CATEGORY_CHAT]: ["api_type", "temperature", "max_tokens", "timeout_seconds", "max_retries"],
  [AiModelCategory.AI_MODEL_CATEGORY_EMBEDDING]: ["dimensions", "timeout_seconds", "max_retries"],
  [AiModelCategory.AI_MODEL_CATEGORY_RERANK]: ["timeout_seconds", "max_retries"],
  [AiModelCategory.AI_MODEL_CATEGORY_IMAGE]: ["size", "quality", "timeout_seconds", "max_retries"],
  [AiModelCategory.AI_MODEL_CATEGORY_VIDEO]: ["duration", "resolution", "timeout_seconds", "max_retries"],
  [AiModelCategory.AI_MODEL_CATEGORY_AUDIO]: ["task", "voice", "speed", "timeout_seconds", "max_retries"]
};

/** 全部配置字段的表单项定义。 */
const configFieldMap: Record<string, ProFormField> = {
  api_type: { prop: "api_type", label: t("system.base.ai.model.field.api_type"), component: "select", colSpan: 12, options: apiTypeOptions.value },
  temperature: { prop: "temperature", label: t("system.base.ai.model.field.temperature"), component: "input-number", colSpan: 12, props: { min: 0, max: 2, precision: 2, step: 0.1, controlsPosition: "right" } },
  max_tokens: { prop: "max_tokens", label: t("system.base.ai.model.field.max_tokens"), component: "input-number", colSpan: 12, props: { min: 0, max: 200000, precision: 0, controlsPosition: "right" } },
  dimensions: { prop: "dimensions", label: t("system.base.ai.model.field.dimensions"), component: "input-number", colSpan: 12, props: { min: 0, max: 8192, precision: 0, controlsPosition: "right", placeholder: t("system.base.ai.model.placeholder.dimensions") } },
  duration: { prop: "duration", label: t("system.base.ai.model.field.duration"), component: "input-number", colSpan: 12, props: { min: 0, precision: 0, controlsPosition: "right" } },
  size: { prop: "size", label: t("system.base.ai.model.field.size"), component: "input", colSpan: 12, props: { maxlength: 50, placeholder: "1024x1024" } },
  quality: { prop: "quality", label: t("system.base.ai.model.field.quality"), component: "input", colSpan: 12, props: { maxlength: 50 } },
  resolution: { prop: "resolution", label: t("system.base.ai.model.field.resolution"), component: "input", colSpan: 12, props: { maxlength: 50, placeholder: "1920x1080" } },
  task: { prop: "task", label: t("system.base.ai.model.field.task"), component: "select", colSpan: 12, options: taskOptions.value },
  voice: { prop: "voice", label: t("system.base.ai.model.field.voice"), component: "input", colSpan: 12, props: { maxlength: 100 } },
  speed: { prop: "speed", label: t("system.base.ai.model.field.speed"), component: "input-number", colSpan: 12, props: { min: 0, max: 4, precision: 2, step: 0.1, controlsPosition: "right" } },
  timeout_seconds: { prop: "timeout_seconds", label: t("system.base.ai.model.field.timeout_seconds"), component: "input-number", colSpan: 12, props: { min: 0, max: 600, precision: 0, controlsPosition: "right" } },
  max_retries: { prop: "max_retries", label: t("system.base.ai.model.field.max_retries"), component: "input-number", colSpan: 12, props: { min: 0, max: 10, precision: 0, controlsPosition: "right" } }
};

const fields = computed<ProFormField[]>(() => [
  { prop: "provider", label: t("system.base.ai_provider.field.provider"), component: "select", colSpan: 12, options: providerOptions.value, props: { disabled: dialog.editing } },
  { prop: "name", label: t("system.base.ai_provider.field.name"), component: "input", colSpan: 12, props: { maxlength: 100, autocomplete: "off" } },
  { prop: "base_url", label: t("system.base.ai_provider.field.base_url"), component: "input", colSpan: 12, props: { maxlength: 512, autocomplete: "off", placeholder: t("system.base.ai_provider.placeholder.base_url") } },
  { prop: "api_key", label: t("system.base.ai_provider.field.api_key"), component: "input", colSpan: 12, secret: { resource: "ai_provider", field: "api_key" }, props: { type: "password", showPassword: true, maxlength: 1024, autocomplete: "new-password", placeholder: form.api_key_configured ? t("system.base.ai_provider.placeholder.keep_api_key") : t("system.base.ai_provider.placeholder.api_key") } },
  { prop: "models", label: t("system.base.ai_provider.field.models"), component: "slot", slotName: "models", colSpan: 24 },
  { prop: "config_items", label: t("system.base.ai_provider.field.config"), component: "kv-list", colSpan: 24, props: { keyInputProps: { maxlength: 128 } } },
  { prop: "sort", label: t("common.field.sort"), component: "input-number", colSpan: 12, props: { min: 0, precision: 0 } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", colSpan: 12, options: statusOptions.value }
]);

const modelFields = computed<ProFormField[]>(() => {
  const category = modelForm.category;
  const configFields = (categoryConfigFields[category] ?? []).map(key => configFieldMap[key]);
  return [
    { prop: "model_name", label: t("system.base.ai.model.field.model_name"), component: "input", colSpan: 12, props: { maxlength: 200, autocomplete: "off" } },
    { prop: "display_name", label: t("system.base.ai.model.field.display_name"), component: "input", colSpan: 12, props: { maxlength: 100, autocomplete: "off", placeholder: t("system.base.ai.model.placeholder.display_name") } },
    { prop: "category", label: t("system.base.ai.model.field.category"), component: "select", colSpan: 12, options: categoryOptions.value },
    { prop: "sort", label: t("common.field.sort"), component: "input-number", colSpan: 12, props: { min: 0, precision: 0 } },
    { prop: "status", label: t("common.field.status"), component: "radio-group", colSpan: 12, options: statusOptions.value },
    ...configFields
  ];
});

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t("system.base.ai_provider.validation.name"), trigger: "blur" }],
  provider: [{ required: true, message: t("system.base.ai_provider.validation.provider"), trigger: "change" }],
  models: [{
    validator: (_rule, value: ModelFormState[], callback) => {
      if (!Array.isArray(value) || value.length === 0) {
        callback(new Error(t("system.base.ai_provider.model.validation.required")));
        return;
      }
      const names = value.map(model => model.model_name);
      callback(names.some(name => !name)
        ? new Error(t("system.base.ai.model.validation.model_name"))
        : new Set(names).size !== names.length
          ? new Error(t("system.base.ai_provider.model.validation.duplicate"))
          : undefined);
    },
    trigger: "change"
  }]
}));

const modelRules = computed<FormRules>(() => ({
  model_name: [
    { required: true, message: t("system.base.ai.model.validation.model_name"), trigger: "blur" },
    { max: 200, message: t("system.base.ai.model.validation.model_name_length"), trigger: "blur" }
  ],
  category: [{ required: true, message: t("system.base.ai.model.validation.category"), trigger: "change" }]
}));

const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("system.base.ai_provider.field.name"), minWidth: 150, search: { el: "input" } },
  { prop: "provider", label: t("system.base.ai_provider.field.provider"), width: 190, search: { el: "select", enum: providerOptions.value }, enum: providerOptions.value },
  { prop: "base_url", label: t("system.base.ai_provider.field.base_url"), minWidth: 220 },
  { prop: "api_key_configured", label: t("system.base.ai_provider.field.api_key"), width: 140, formatter: row => row.api_key_configured ? t("system.base.ai_provider.value.configured") : t("system.base.ai_provider.value.not_configured") },
  { prop: "model_names", label: t("system.base.ai_provider.field.models"), minWidth: 220, formatter: row => (row.model_names ?? []).join(", ") },
  { prop: "sort", label: t("common.field.sort"), width: 80, align: "right" },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 110,
    search: { el: "select", enum: statusOptions.value },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !isDefaultTenant.value || !BUTTONS.value["base:ai-provider:status"],
      beforeChange: scope => handleSetStatus(scope.row as AiProvider)
    }
  },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:ai-provider:update"], onClick: scope => openDialog((scope.row as AiProvider).id) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:ai-provider:delete"], onClick: scope => handleDelete(scope.row as AiProvider) }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !isDefaultTenant.value || !BUTTONS.value["base:ai-provider:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !isDefaultTenant.value || !BUTTONS.value["base:ai-provider:delete"], disabled: scope => !scope.isSelected, onClick: scope => handleDelete(scope.selectedListIds as number[]) }
]);

/** 创建默认供应商表单。 */
function defaultForm(): FormState {
  return {
    id: 0,
    provider: "openai_compatible",
    name: "",
    base_url: "",
    api_key: undefined,
    api_key_configured: false,
    models: [],
    config: {},
    config_items: [],
    sort: 0,
    status: Status.STATUS_DISABLE
  };
}

/** 创建默认模型表单；各分类配置字段都先给默认值，切换分类时无需重置。 */
function defaultModelForm(): ModelFormState {
  return {
    id: 0,
    model_name: "",
    display_name: "",
    category: AiModelCategory.AI_MODEL_CATEGORY_CHAT,
    sort: 0,
    status: Status.STATUS_ENABLE,
    api_type: "CHAT_COMPLETIONS",
    temperature: 0.2,
    max_tokens: 1024,
    dimensions: 0,
    duration: 0,
    size: "",
    quality: "",
    resolution: "",
    task: "tts",
    voice: "",
    speed: 0,
    timeout_seconds: 60,
    max_retries: 2
  };
}

/** 模型分类显示名。 */
function categoryLabel(category: AiModelCategory) {
  const option = categoryOptions.value.find(item => item.value === category);
  return option?.label ?? String(category);
}

/** 接口类型显示名。 */
function apiTypeLabel(apiType: string) {
  const option = apiTypeOptions.value.find(item => item.value === apiType);
  return option?.label ?? apiType;
}

/** 请求AI供应商表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defAiProviderService.PageAiProvider(buildPageRequest<PageAiProviderRequest>(params as unknown as PageAiProviderRequest));
  return { data: { list: data.providers ?? [], total: data.total } };
}

/** 打开供应商编辑表单。 */
async function openDialog(id?: number) {
  await dialogRef.value?.open({
    load: () => (id ? defAiProviderService.GetAiProvider({ id }) : undefined),
    commit: value => {
      if (value) {
        Object.assign(form, defaultForm(), value, {
          models: (value.models ?? []).map(model => flattenModel(model)),
          config_items: configToItems(value.config)
        });
      } else {
        Object.assign(form, defaultForm());
      }
      dialog.editing = Boolean(id);
    }
  });
}

/** 提交AI供应商表单（供应商与模型两张表随同一请求保存）。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const payload = buildAiProviderPayload(await encodeSecretFields(fields.value, form));
  if (!payload) return;
  if (dialog.editing) await defAiProviderService.UpdateAiProvider({ ai_provider: payload });
  else await defAiProviderService.CreateAiProvider({ ai_provider: payload });
  ElMessage.success(t(dialog.editing ? "system.base.ai_provider.message.update_success" : "system.base.ai_provider.message.create_success"));
  dialog.visible = false;
  resetForm();
  table.value?.getTableList();
}

/** 测试当前草稿中的全部模型并展示逐项结果。 */
async function testProviderModels() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const payload = buildAiProviderPayload(await encodeSecretFields(fields.value, form));
  if (!payload) return;

  testingModels.value = true;
  modelTestResults.value = {};
  let response: Awaited<ReturnType<typeof defAiProviderService.TestAiProviderModels>>;
  try {
    response = await defAiProviderService.TestAiProviderModels({ ai_provider: payload });
  } finally {
    testingModels.value = false;
  }
  modelTestResults.value = Object.fromEntries(response.results.map(result => [result.model_name, result]));
}

/** 构造AI Provider保存或测试请求参数，source 为加密转换后的提交模型。 */
function buildAiProviderPayload(source: Record<string, any>): AiProviderForm | undefined {
  const config = configItemsToMap(source.config_items);
  if (!config) return undefined;
  return {
    id: source.id,
    provider: source.provider,
    name: source.name,
    base_url: source.base_url,
    api_key: source.api_key,
    api_key_configured: source.api_key_configured,
    models: source.models.map(model => buildModelPayload(model)),
    config,
    sort: source.sort,
    status: source.status
  };
}

/** 将页面模型表单收敛为保存或测试请求参数。 */
function buildModelPayload(model: ModelFormState): AiProviderModelForm {
  const category = model.category;
  const config: Record<string, unknown> = {};
  for (const key of categoryConfigFields[category] ?? []) {
    const value = model[key as keyof ModelFormState];
    if (value !== undefined && value !== "") config[key] = value;
  }
  return {
    id: model.id,
    model_name: model.model_name,
    display_name: model.display_name,
    category: model.category,
    config,
    sort: model.sort,
    status: model.status
  };
}

/** 将接口返回的模型表单拍平为页面状态。 */
function flattenModel(model: AiProviderModelForm): ModelFormState {
  const state = { ...defaultModelForm(), ...model, config: undefined } as unknown as ModelFormState;
  for (const [key, value] of Object.entries(model.config ?? {})) {
    if (key in state) (state as unknown as Record<string, unknown>)[key] = value;
  }
  return state;
}

/** 重置供应商表单。 */
function resetForm() {
  Object.assign(form, defaultForm());
  dialog.editing = false;
  modelDialog.visible = false;
  resetModelDialog();
  modelTestResults.value = {};
}

/** 打开模型新增或编辑表单。 */
function openModelDialog(index?: number) {
  modelDialog.index = index ?? -1;
  modelDialog.editing = index !== undefined;
  Object.assign(modelForm, index === undefined ? defaultModelForm() : { ...form.models[index] });
  modelDialog.visible = true;
}

/** 校验并保存当前模型表单。 */
async function saveModel() {
  const valid = await modelDialogRef.value?.validate();
  if (!valid) return;
  const duplicated = form.models.some((model, index) => index !== modelDialog.index && model.model_name === modelForm.model_name.trim());
  if (duplicated) {
    ElMessage.warning(t("system.base.ai_provider.model.validation.duplicate"));
    return;
  }
  if (modelDialog.index < 0) form.models.push({ ...modelForm });
  else form.models.splice(modelDialog.index, 1, { ...modelForm });
  modelDialog.visible = false;
}

/** 删除模型表单项。 */
async function removeModel(index: number) {
  try {
    await ElMessageBox.confirm(
      t("system.base.ai_provider.model.confirm_remove", { name: form.models[index].model_name }),
      t("common.action.delete"),
      { type: "warning" }
    );
    form.models.splice(index, 1);
  } catch {
    return;
  }
}

/** 重置模型表单。 */
function resetModelDialog() {
  Object.assign(modelForm, defaultModelForm());
  modelDialog.editing = false;
  modelDialog.index = -1;
}

/** 将供应商返回的配置对象转换为键值表单项。 */
function configToItems(config: Record<string, unknown> | undefined): ConfigItem[] {
  return Object.entries(config ?? {}).map(([key, value]) => ({ key, value: typeof value === "string" ? value : JSON.stringify(value) }));
}

/** 将键值表单项转换为Provider配置JSON对象。 */
function configItemsToMap(items: ConfigItem[]): Record<string, unknown> | undefined {
  const result: Record<string, unknown> = {};
  for (const item of items) {
    const key = item.key.trim();
    if (!key || Object.prototype.hasOwnProperty.call(result, key)) {
      ElMessage.error(t("system.base.ai_provider.validation.config_invalid"));
      return undefined;
    }
    const value = item.value.trim();
    if (!value) {
      result[key] = "";
      continue;
    }
    try {
      result[key] = JSON.parse(value);
    } catch {
      result[key] = value;
    }
  }
  return result;
}

/** 批量删除AI供应商（级联删除其下模型）。 */
async function handleDelete(target: AiProvider | number | Array<number>) {
  const row = typeof target === "object" && !Array.isArray(target) ? target : undefined;
  const ids = (row ? [row.id] : normalizeSelectedIds(target as number | number[])).map(Number);
  if (!ids.length) return;
  try {
    await ElMessageBox.confirm(t("system.base.ai_provider.message.confirm_delete", { name: row?.name ?? ids.join(", ") }), t("common.action.delete"), { type: "warning" });
  } catch {
    return;
  }
  for (const id of ids) await defAiProviderService.DeleteAiProvider({ id });
  ElMessage.success(t("system.base.ai_provider.message.delete_success"));
  table.value?.getTableList();
}

/** 更新AI供应商状态。 */
async function handleSetStatus(row: AiProvider) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"),
        resource: t("system.base.ai_provider.resource"),
        field: t("system.base.ai_provider.field.name"),
        value: row.name
      }),
      t("common.title.notice"),
      { type: "warning" }
    );
    await defAiProviderService.SetAiProviderStatus({ id: row.id, status });
    ElMessage.success(t("system.base.ai_provider.message.status_success"));
    table.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}
</script>

<style scoped lang="scss">
.provider-models {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}
.provider-models__empty {
  padding: 16px 0;
  color: var(--el-text-color-secondary);
  text-align: center;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
.provider-models__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-models__item {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
.provider-models__content {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.provider-models__heading {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.provider-models__alias {
  color: var(--el-text-color-secondary);
}
.provider-models__details {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.provider-models__actions {
  display: flex;
  flex-shrink: 0;
}
</style>
