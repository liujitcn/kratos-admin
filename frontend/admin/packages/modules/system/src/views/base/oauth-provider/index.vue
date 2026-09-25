<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'common.action.edit' : 'common.action.create')"
      class="oauth-provider-dialog"
      width="min(1280px, calc(100vw - 32px))"
      top="3vh"
      label-width="9em"
      :col-span="12"
      :gutter="24"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleClose"
    >
      <template #nameI18ns>
        <DynamicI18nEditor v-model="nameI18nValues" :source="formData.name" :maxlength="50" />
      </template>
      <template #descriptionI18ns>
        <DynamicI18nEditor v-model="descriptionI18nValues" :source="formData.description" :maxlength="255" multiline />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseOauthProviderService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_oauth_provider";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import type {
  BaseOauthProvider,
  BaseOauthProviderForm,
  PageBaseOauthProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_oauth_provider";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/i18n/DynamicI18nEditor.vue";
import {
  normalizeDynamicI18ns,
  serializeDynamicI18ns,
  type DynamicI18nValue
} from "@liujitcn/kratos-admin-system/components/i18n/dynamicI18n";

defineOptions({ name: "BaseOauthProvider", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ editing: false, visible: false });
const formData = reactive<OauthProviderFormState>(defaultForm());
const nameI18nValues = ref<DynamicI18nValue[]>(normalizeDynamicI18ns(undefined));
const descriptionI18nValues = ref<DynamicI18nValue[]>(normalizeDynamicI18ns(undefined));

/** OAuth Provider 个性化配置键值编辑项。 */
interface OauthProviderConfigItem {
  key: string;
  value: string;
}

/** OAuth Provider 页面表单状态。 */
interface OauthProviderFormState extends BaseOauthProviderForm {
  config_items: OauthProviderConfigItem[];
}

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const rules = computed(() => ({
  provider: [{ required: true, message: t("system.base.oauth_provider.placeholder.provider"), trigger: "blur" }],
  name: [{ required: true, message: t("system.base.oauth_provider.placeholder.name"), trigger: "blur" }],
  icon: [{ required: true, message: t("system.base.oauth_provider.placeholder.icon"), trigger: "blur" }],
  client_id: [{ required: true, message: t("system.base.oauth_provider.placeholder.client_id"), trigger: "blur" }],
  client_secret: [
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        if (!formData.id && !value) callback(new Error(t("system.base.oauth_provider.placeholder.client_secret")));
        else callback();
      },
      trigger: "blur"
    }
  ]
}));

const formFields = computed<ProFormField[]>(() => [
  { prop: "provider", label: t("system.base.oauth_provider.field.provider"), component: "input", colSpan: 12, props: { disabled: dialog.editing, maxlength: 32, placeholder: t("system.base.oauth_provider.placeholder.provider") } },
  { prop: "name", suffixSlotName: "nameI18ns", label: t("system.base.oauth_provider.field.name"), component: "input", colSpan: 12, props: { maxlength: 50, placeholder: t("system.base.oauth_provider.placeholder.name") } },
  { prop: "description", suffixSlotName: "descriptionI18ns", label: t("system.base.oauth_provider.field.description"), component: "input", colSpan: 24, props: { maxlength: 255, placeholder: t("system.base.oauth_provider.placeholder.description") } },
  { prop: "icon", label: t("system.base.oauth_provider.field.icon"), component: "input", colSpan: 12, props: { maxlength: 255, placeholder: t("system.base.oauth_provider.placeholder.icon") } },
  { prop: "client_id", label: t("system.base.oauth_provider.field.client_id"), component: "input", colSpan: 12, props: { maxlength: 255, placeholder: t("system.base.oauth_provider.placeholder.client_id") } },
  { prop: "client_secret", label: t("system.base.oauth_provider.field.client_secret"), component: "input", colSpan: 12, props: { type: "password", showPassword: true, maxlength: 512, placeholder: dialog.editing ? t("system.base.oauth_provider.placeholder.keep_secret") : t("system.base.oauth_provider.placeholder.client_secret") } },
  { prop: "redirect_uri", label: t("system.base.oauth_provider.field.redirect_uri"), component: "input", colSpan: 12, props: { maxlength: 512, placeholder: t("system.base.oauth_provider.placeholder.redirect_uri") } },
  { prop: "scopes", label: t("system.base.oauth_provider.field.scopes"), component: "dynamic-list", colSpan: 12, props: { inputProps: { maxlength: 128, placeholder: t("system.base.oauth_provider.placeholder.scopes") } } },
  { prop: "config_items", label: t("system.base.oauth_provider.field.config"), component: "kv-list", colSpan: 12, props: { keyInputProps: { maxlength: 128 } } },
  { prop: "sort", label: t("common.field.sort"), component: "input-number", colSpan: 12, props: { min: 0, precision: 0 } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", colSpan: 12, options: statusOptions.value }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "name", label: t("system.base.oauth_provider.field.name"), minWidth: 150, search: { el: "input" } },
  { prop: "provider", label: t("system.base.oauth_provider.field.provider"), minWidth: 130, search: { el: "input" } },
  { prop: "client_id", label: t("system.base.oauth_provider.field.client_id"), minWidth: 180 },
  { prop: "redirect_uri", label: t("system.base.oauth_provider.field.redirect_uri"), minWidth: 240 },
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
      disabled: () => !isDefaultTenant.value || !BUTTONS.value["base:oauth-provider:status"],
      beforeChange: scope => handleSetStatus(scope.row as BaseOauthProvider)
    }
  },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:oauth-provider:update"], onClick: scope => openDialog((scope.row as BaseOauthProvider).id) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:oauth-provider:delete"], onClick: scope => handleDelete(scope.row as BaseOauthProvider) }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !isDefaultTenant.value || !BUTTONS.value["base:oauth-provider:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !isDefaultTenant.value || !BUTTONS.value["base:oauth-provider:delete"], disabled: scope => !scope.isSelected, onClick: scope => handleDelete(scope.selectedListIds as number[]) }
]);

/** 创建默认 OAuth 登录方式表单。 */
function defaultForm(): OauthProviderFormState {
  return { id: 0, provider: "", name: "", description: "", icon: "", client_id: "", client_secret: "", secret_configured: false, redirect_uri: "", scopes: [], config: {}, config_items: [], sort: 0, status: Status.STATUS_DISABLE, name_i18ns: [], description_i18ns: [] };
}

/** 请求 OAuth 登录方式表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defBaseOauthProviderService.PageBaseOauthProvider(buildPageRequest<PageBaseOauthProviderRequest>(params as unknown as PageBaseOauthProviderRequest));
  return { data: { list: data.base_oauth_providers ?? [], total: data.total } };
}

/** 打开 OAuth 登录方式表单。 */
async function openDialog(id?: number) {
  await loadEnabledBaseLanguages();
  await formDialogRef.value?.open({
    load: () => (id ? defBaseOauthProviderService.GetBaseOauthProvider({ id }) : undefined),
    commit: data => {
      Object.assign(formData, defaultForm(), data ?? {});
      dialog.editing = Boolean(id);
      formData.config_items = configToItems(formData.config);
      nameI18nValues.value = normalizeDynamicI18ns(formData.name_i18ns);
      descriptionI18nValues.value = normalizeDynamicI18ns(formData.description_i18ns);
    }
  });
}

/** 提交 OAuth 登录方式表单。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;
  const config = configItemsToMap(formData.config_items);
  if (!config) return;
  const payload = JSON.parse(JSON.stringify(formData)) as BaseOauthProviderForm & { config_items?: OauthProviderConfigItem[] };
  delete payload.config_items;
  payload.config = config;
  payload.scopes = [...new Set(formData.scopes.map(value => value.trim()).filter(Boolean))];
  payload.name_i18ns = serializeDynamicI18ns(nameI18nValues.value, I18nTargetType.I18N_TARGET_TYPE_BASE_OAUTH_PROVIDER_NAME, payload.id, payload.name);
  payload.description_i18ns = serializeDynamicI18ns(descriptionI18nValues.value, I18nTargetType.I18N_TARGET_TYPE_BASE_OAUTH_PROVIDER_DESCRIPTION, payload.id, payload.description);
  if (payload.id) await defBaseOauthProviderService.UpdateBaseOauthProvider({ base_oauth_provider: payload });
  else await defBaseOauthProviderService.CreateBaseOauthProvider({ base_oauth_provider: payload });
  ElMessage.success(t("common.message.operation_success"));
  dialog.visible = false;
  await proTable.value?.getTableList();
}

/** 删除 OAuth 登录方式。 */
async function handleDelete(value: BaseOauthProvider | number | number[]) {
  const ids = (Array.isArray(value) ? value : typeof value === "object" ? [value.id] : normalizeSelectedIds(value)).join(",");
  if (!ids) return;
  await ElMessageBox.confirm(t("common.confirm.delete"), t("common.title.warning"), { type: "warning" });
  await defBaseOauthProviderService.DeleteBaseOauthProvider({ id: ids });
  await proTable.value?.getTableList();
  ElMessage.success(t("common.message.operation_success"));
}

/** 设置 OAuth 登录方式状态。 */
async function handleSetStatus(row: BaseOauthProvider) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"),
        resource: t("system.base.oauth_provider.resource"),
        field: t("system.base.oauth_provider.field.name"),
        value: row.name
      }),
      t("common.title.notice"),
      { type: "warning" }
    );
    await defBaseOauthProviderService.SetBaseOauthProviderStatus({ id: row.id, status });
    return true;
  } catch {
    return false;
  }
}

/** 关闭并重置 OAuth 登录方式表单。 */
function handleClose() {
  Object.assign(formData, defaultForm());
  nameI18nValues.value = normalizeDynamicI18ns(undefined);
  descriptionI18nValues.value = normalizeDynamicI18ns(undefined);
  formDialogRef.value?.resetFields();
}

/** 将结构化配置转换为键值编辑列表。 */
function configToItems(config: Record<string, any> | undefined): OauthProviderConfigItem[] {
  return Object.entries(config ?? {}).map(([key, value]) => ({
    key,
    value: typeof value === "string" ? value : JSON.stringify(value)
  }));
}

/** 将键值编辑列表转换为结构化配置，并校验空键和重复键。 */
function configItemsToMap(items: OauthProviderConfigItem[]): Record<string, unknown> | undefined {
  const result: Record<string, unknown> = {};
  for (const item of items) {
    const key = item.key.trim();
    if (!key || Object.prototype.hasOwnProperty.call(result, key)) {
      ElMessage.error(t("system.base.oauth_provider.message.config_invalid"));
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
</script>

<style scoped lang="scss">
:global(.oauth-provider-dialog .el-dialog__body) {
  max-height: calc(88vh - 130px);
  overflow-y: auto;
}
</style>
