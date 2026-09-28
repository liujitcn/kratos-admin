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
                  <el-tag size="small" effect="plain">{{ apiTypeLabel(model.api_type) }}</el-tag>
                </div>
                <div class="provider-models__details">
                  <span>{{ t("system.base.ai_provider.field.temperature") }}: {{ model.temperature }}</span>
                  <span>{{ t("system.base.ai_provider.field.max_tokens") }}: {{ model.max_tokens }}</span>
                  <span>{{ t("system.base.ai_provider.field.timeout_seconds") }}: {{ model.timeout_seconds }}</span>
                  <span>{{ t("system.base.ai_provider.field.max_retries") }}: {{ model.max_retries }}</span>
                </div>
              </div>
              <div class="provider-models__actions">
                <el-tooltip :content="t('common.action.edit')" placement="top">
                  <el-button text circle :icon="EditPen" :aria-label="t('common.action.edit')" @click="openModelDialog(index)" />
                </el-tooltip>
                <el-tooltip :content="t('common.action.delete')" placement="top">
                  <el-button text circle :icon="Delete" :aria-label="t('common.action.delete')" @click="removeModel(index)" />
                </el-tooltip>
              </div>
            </div>
          </div>
          <el-button type="primary" plain :icon="CirclePlus" @click="openModelDialog()">
            {{ t("system.base.ai_provider.model.add") }}
          </el-button>
        </div>
      </template>
    </FormDialog>

    <FormDialog
      v-model="modelDialog.visible"
      ref="modelDialogRef"
      :title="t(modelDialog.editing ? 'system.base.ai_provider.model.edit' : 'system.base.ai_provider.model.add')"
      width="min(640px, calc(100vw - 32px))"
      label-width="9em"
      label-position="left"
      :model="modelForm"
      :fields="modelFields"
      :rules="modelRules"
      @confirm="saveModel"
      @close="resetModelDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiProviderService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_provider";
import type {
  AiProvider,
  AiProviderForm,
  PageAiProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_provider";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "AiProvider", inheritAttrs: false });

/** Provider个性化配置编辑项。 */
interface ConfigItem {
  key: string;
  value: string;
}

/** 模型请求接口类型。 */
type ModelApiType = "CHAT_COMPLETIONS" | "RESPONSES";

/** 单个AI模型的可编辑配置。 */
interface AiModelConfig {
  /** 服务商侧的模型名称。 */
  model_name: string;
  /** 模型请求接口类型。 */
  api_type: ModelApiType;
  /** 文本生成温度。 */
  temperature: number;
  /** 单次请求最大输出Token数。 */
  max_tokens: number;
  /** 模型请求超时秒数。 */
  timeout_seconds: number;
  /** 模型请求失败后的最大重试次数。 */
  max_retries: number;
}

/** AI Provider页面表单状态。 */
interface FormState extends Omit<AiProviderForm, "models_json"> {
  models: AiModelConfig[];
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
const modelForm = reactive<AiModelConfig>(defaultModelConfig());

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const providerOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai_provider.provider.openai_compatible"), value: "openai_compatible" },
  { label: t("system.base.ai_provider.provider.ollama"), value: "ollama" }
]);

const modelApiTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.ai_provider.model.api_type.chat_completions"), value: "CHAT_COMPLETIONS" },
  { label: t("system.base.ai_provider.model.api_type.responses"), value: "RESPONSES" }
]);

const fields = computed<ProFormField[]>(() => [
  { prop: "provider", label: t("system.base.ai_provider.field.provider"), component: "select", colSpan: 12, options: providerOptions.value, props: { disabled: dialog.editing } },
  { prop: "name", label: t("system.base.ai_provider.field.name"), component: "input", colSpan: 12, props: { maxlength: 100 } },
  { prop: "base_url", label: t("system.base.ai_provider.field.base_url"), component: "input", colSpan: 12, props: { maxlength: 512, placeholder: t("system.base.ai_provider.placeholder.base_url") } },
  { prop: "api_key", label: t("system.base.ai_provider.field.api_key"), component: "input", colSpan: 12, props: { type: "password", showPassword: true, maxlength: 1024, placeholder: form.api_key_configured ? t("system.base.ai_provider.placeholder.keep_api_key") : t("system.base.ai_provider.placeholder.api_key") } },
  { prop: "models", label: t("system.base.ai_provider.field.models"), component: "slot", slotName: "models", colSpan: 24 },
  { prop: "config_items", label: t("system.base.ai_provider.field.config"), component: "kv-list", colSpan: 24, props: { keyInputProps: { maxlength: 128 } } },
  { prop: "sort", label: t("common.field.sort"), component: "input-number", colSpan: 12, props: { min: 0, precision: 0 } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", colSpan: 12, options: statusOptions.value }
]);

const modelFields = computed<ProFormField[]>(() => [
  { prop: "model_name", label: t("system.base.ai_provider.field.model_name"), component: "input", colSpan: 12, props: { maxlength: 100 } },
  { prop: "api_type", label: t("system.base.ai_provider.field.api_type"), component: "select", colSpan: 12, options: modelApiTypeOptions.value },
  { prop: "temperature", label: t("system.base.ai_provider.field.temperature"), component: "input-number", colSpan: 12, props: { min: 0, max: 2, precision: 2, step: 0.1, controlsPosition: "right" } },
  { prop: "max_tokens", label: t("system.base.ai_provider.field.max_tokens"), component: "input-number", colSpan: 12, props: { min: 0, max: 200000, precision: 0, controlsPosition: "right" } },
  { prop: "timeout_seconds", label: t("system.base.ai_provider.field.timeout_seconds"), component: "input-number", colSpan: 12, props: { min: 0, max: 600, precision: 0, controlsPosition: "right" } },
  { prop: "max_retries", label: t("system.base.ai_provider.field.max_retries"), component: "input-number", colSpan: 12, props: { min: 0, max: 10, precision: 0, controlsPosition: "right" } }
]);

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t("system.base.ai_provider.validation.name"), trigger: "blur" }],
  provider: [{ required: true, message: t("system.base.ai_provider.validation.provider"), trigger: "change" }],
  models: [{
    validator: (_rule, value: AiModelConfig[], callback) => {
      if (!Array.isArray(value) || value.length === 0) {
        callback(new Error(t("system.base.ai_provider.model.validation.required")));
        return;
      }
      const names = value.map(model => model.model_name);
      callback(names.some(name => !name)
        ? new Error(t("system.base.ai_provider.validation.model_name"))
        : new Set(names).size !== names.length
          ? new Error(t("system.base.ai_provider.model.validation.duplicate"))
          : undefined);
    },
    trigger: "change"
  }]
}));

const modelRules = computed<FormRules>(() => ({
  model_name: [
    { required: true, message: t("system.base.ai_provider.validation.model_name"), trigger: "blur" },
    { max: 100, message: t("system.base.ai_provider.validation.model_name_length"), trigger: "blur" }
  ],
  api_type: [{ required: true, message: t("system.base.ai_provider.validation.api_type"), trigger: "change" }],
  temperature: [{ type: "number", required: true, min: 0, max: 2, message: t("system.base.ai_provider.model.validation.range"), trigger: "change" }],
  max_tokens: [{ type: "number", required: true, min: 0, max: 200000, message: t("system.base.ai_provider.model.validation.range"), trigger: "change" }],
  timeout_seconds: [{ type: "number", required: true, min: 0, max: 600, message: t("system.base.ai_provider.model.validation.range"), trigger: "change" }],
  max_retries: [{ type: "number", required: true, min: 0, max: 10, message: t("system.base.ai_provider.model.validation.range"), trigger: "change" }]
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
    api_key: "",
    api_key_configured: false,
    models: [],
    config: {},
    config_items: [],
    sort: 0,
    status: Status.STATUS_DISABLE
  };
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
        const { models_json, ...provider } = value;
        Object.assign(form, defaultForm(), provider, {
          models: parseModelConfigs(models_json),
          config_items: configToItems(value.config)
        });
      } else {
        Object.assign(form, defaultForm());
      }
      dialog.editing = Boolean(id);
    }
  });
}

/** 提交AI供应商表单。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const config = configItemsToMap(form.config_items);
  if (!config) return;
  const payload: AiProviderForm = {
    id: form.id,
    provider: form.provider,
    name: form.name,
    base_url: form.base_url,
    api_key: form.api_key,
    api_key_configured: form.api_key_configured,
    models_json: JSON.stringify(form.models),
    config,
    sort: form.sort,
    status: form.status
  };
  if (dialog.editing) await defAiProviderService.UpdateAiProvider({ ai_provider: payload });
  else await defAiProviderService.CreateAiProvider({ ai_provider: payload });
  ElMessage.success(t(dialog.editing ? "system.base.ai_provider.message.update_success" : "system.base.ai_provider.message.create_success"));
  dialog.visible = false;
  resetForm();
  table.value?.getTableList();
}

/** 重置供应商表单。 */
function resetForm() {
  Object.assign(form, defaultForm());
  dialog.editing = false;
  modelDialog.visible = false;
  resetModelDialog();
}

/** 打开模型新增或编辑表单。 */
function openModelDialog(index?: number) {
  modelDialog.index = index ?? -1;
  modelDialog.editing = index !== undefined;
  Object.assign(modelForm, index === undefined ? defaultModelConfig() : { ...form.models[index] });
  modelDialog.visible = true;
}

/** 校验并保存当前模型表单。 */
async function saveModel() {
  const valid = await modelDialogRef.value?.validate();
  if (!valid) return;
  const duplicated = form.models.some((model, index) => index !== modelDialog.index && model.model_name === modelForm.model_name);
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

/** 重置模型编辑弹窗状态。 */
function resetModelDialog() {
  modelDialog.editing = false;
  modelDialog.index = -1;
}

/** 返回模型接口类型的本地化名称。 */
function apiTypeLabel(apiType: ModelApiType) {
  return t(apiType === "RESPONSES"
    ? "system.base.ai_provider.model.api_type.responses"
    : "system.base.ai_provider.model.api_type.chat_completions");
}

/** 将供应商返回的模型配置数组转换为表单数据。 */
function parseModelConfigs(raw: string): AiModelConfig[] {
  const values = JSON.parse(raw) as Partial<AiModelConfig>[];
  return values.map(value => ({
    model_name: value.model_name ?? "",
    api_type: value.api_type === "RESPONSES" ? "RESPONSES" : "CHAT_COMPLETIONS",
    temperature: value.temperature ?? 0,
    max_tokens: value.max_tokens ?? 0,
    timeout_seconds: value.timeout_seconds ?? 0,
    max_retries: value.max_retries ?? 0
  }));
}

/** 创建单个模型的默认配置。 */
function defaultModelConfig(): AiModelConfig {
  return { model_name: "", api_type: "CHAT_COMPLETIONS", temperature: 0.2, max_tokens: 1024, timeout_seconds: 60, max_retries: 2 };
}

/** 将Provider配置对象转换为键值表单项。 */
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

/** 批量删除AI供应商。 */
async function handleDelete(target: AiProvider | number | Array<number>) {
  const row = typeof target === "object" && !Array.isArray(target) ? target : undefined;
  const ids = (row ? [row.id] : normalizeSelectedIds(target as number | number[])).map(Number);
  if (!ids.length) return;
  await ElMessageBox.confirm(t("system.base.ai_provider.message.confirm_delete", { name: row?.name ?? ids.join(", ") }), t("common.action.delete"), { type: "warning" });
  for (const id of ids) await defAiProviderService.DeleteAiProvider({ id });
  ElMessage.success(t("system.base.ai_provider.message.delete_success"));
  table.value?.getTableList();
}

/** 更新AI供应商状态。 */
async function handleSetStatus(row: AiProvider) {
  await defAiProviderService.SetAiProviderStatus({
    id: row.id,
    status: row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE
  });
  ElMessage.success(t("system.base.ai_provider.message.status_success"));
  table.value?.getTableList();
  return true;
}
</script>

<style scoped>
.provider-models {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
}

.provider-models__empty {
  width: 100%;
  padding: 16px;
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.provider-models__list {
  width: 100%;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
}

.provider-models__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-width: 0;
  padding: 12px 14px;
}

.provider-models__item + .provider-models__item {
  border-top: 1px solid var(--el-border-color-lighter);
}

.provider-models__content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.provider-models__heading,
.provider-models__details,
.provider-models__actions {
  display: flex;
  align-items: center;
}

.provider-models__heading {
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.provider-models__heading strong {
  overflow-wrap: anywhere;
}

.provider-models__details {
  flex-wrap: wrap;
  gap: 4px 14px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.provider-models__actions {
  flex: none;
  gap: 4px;
}

@media (max-width: 640px) {
  .provider-models__item {
    align-items: flex-start;
    padding: 10px;
  }
}
</style>
