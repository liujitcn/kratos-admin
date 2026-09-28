<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'common.action.edit' : 'common.action.create')"
      width="min(1000px, calc(100vw - 32px))"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleClose"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
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
import { defBaseMessageCategoryService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message_category";
import { defBaseMessageProviderService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message_provider";
import { defBaseMessageTemplateService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_message_template";
import type { BaseMessageCategory } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_category";
import type { BaseMessageProvider } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_provider";
import type { BaseMessageTemplate, BaseMessageTemplateForm, PageBaseMessageTemplateRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_template";
import { MessageContentFormat } from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";

defineOptions({ name: "BaseMessageTemplate", inheritAttrs: false });

interface ConfigItem { key: string; value: string; }
interface FormState extends Omit<BaseMessageTemplateForm, "category_id" | "provider_id"> {
  category_id: number | undefined;
  provider_id: number | undefined;
  config_items: ConfigItem[];
}

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, editing: false });
const formData = reactive<FormState>(defaultForm());
const categories = ref<BaseMessageCategory[]>([]);
const providers = ref<BaseMessageProvider[]>([]);

const contentFormatOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.message_template.content_format.plain_text"), value: MessageContentFormat.MESSAGE_CONTENT_FORMAT_PLAIN_TEXT },
  { label: t("system.base.message_template.content_format.safe_markdown"), value: MessageContentFormat.MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN },
  { label: t("system.base.message_template.content_format.rich_text"), value: MessageContentFormat.MESSAGE_CONTENT_FORMAT_RICH_TEXT }
]);
const categoryOptions = computed<ProFormOption[]>(() => categories.value.map(item => ({ label: item.name, value: item.id })));
const providerOptions = computed<ProFormOption[]>(() => providers.value.map(item => ({ label: item.name, value: item.id, disabled: item.status !== 1 })));
const rules = computed(() => ({ category_id: [{ required: true, message: t("system.base.message_template.validation.category"), trigger: "change" }], provider_id: [{ required: true, message: t("system.base.message_template.validation.provider"), trigger: "change" }], title: [{ required: true, message: t("system.base.message_template.validation.title"), trigger: "blur" }] }));
const formFields = computed<ProFormField[]>(() => [
  { prop: "category_id", label: t("system.base.message_template.field.category"), component: "select", options: categoryOptions.value, props: { disabled: dialog.editing, filterable: true } },
  { prop: "provider_id", label: t("system.base.message_template.field.provider"), component: "select", options: providerOptions.value, props: { disabled: dialog.editing, filterable: true } },
  { prop: "title", label: t("system.base.message_template.field.title"), component: "input", props: { maxlength: 200 } },
  { prop: "content", label: t("system.base.message_template.field.content"), component: "textarea", colSpan: 24, props: { maxlength: 20000, rows: 8, showWordLimit: true } },
  { prop: "content_format", label: t("system.base.message_template.field.content_format"), component: "select", options: contentFormatOptions.value },
  { prop: "config_items", label: t("system.base.message_template.field.config"), component: "kv-list", props: { keyInputProps: { maxlength: 128 } } }
]);
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "category_name", label: t("system.base.message_template.field.category"), minWidth: 150 },
  { prop: "provider_name", label: t("system.base.message_template.field.provider"), minWidth: 150 },
  { prop: "title", label: t("system.base.message_template.field.title"), minWidth: 180 },
  { prop: "content_format", label: t("system.base.message_template.field.content_format"), minWidth: 120, render: scope => contentFormatLabel((scope.row as BaseMessageTemplate).content_format) },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180, align: "center" },
  { prop: "operation", label: t("common.field.operation"), cellType: "actions", actions: [
    { label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-template:update"], onClick: scope => openDialog((scope.row as BaseMessageTemplate).id) },
    { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:message-template:delete"], onClick: scope => handleDelete(scope.row as BaseMessageTemplate) }
  ] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-template:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !isDefaultTenant.value || !BUTTONS.value["base:message-template:delete"], disabled: scope => !scope.isSelected, onClick: scope => handleDelete(scope.selectedListIds as number[]) }
]);

onMounted(loadOptions);

function defaultForm(): FormState { return { id: 0, category_id: undefined, provider_id: undefined, title: "", content: "", content_format: MessageContentFormat.MESSAGE_CONTENT_FORMAT_PLAIN_TEXT, config: {}, config_items: [] }; }
async function loadOptions() {
  const [categoryData, providerData] = await Promise.all([defBaseMessageCategoryService.OptionBaseMessageCategory({}), defBaseMessageProviderService.OptionBaseMessageProvider({})]);
  categories.value = (categoryData.list ?? []).map(item => ({ id: Number(item.value), name: item.label } as BaseMessageCategory));
  providers.value = (providerData.list ?? []).map(item => ({ id: Number(item.value), name: item.label, status: item.disabled ? 2 : 1 } as BaseMessageProvider));
}
async function requestTable(params: Record<string, unknown>) { const data = await defBaseMessageTemplateService.PageBaseMessageTemplate(buildPageRequest<PageBaseMessageTemplateRequest>(params as unknown as PageBaseMessageTemplateRequest)); return { data: { list: data.base_message_templates ?? [], total: data.total } }; }
async function openDialog(id?: number) { await loadOptions(); await formDialogRef.value?.open({ load: () => (id ? defBaseMessageTemplateService.GetBaseMessageTemplate({ id }) : undefined), commit: data => { Object.assign(formData, defaultForm(), data ?? {}); dialog.editing = Boolean(id); formData.config_items = configToItems(formData.config); } }); }
async function handleSubmit() { if (!(await formDialogRef.value?.validate())) return; const config = configItemsToMap(formData.config_items); if (!config) return; const payload = JSON.parse(JSON.stringify(formData)) as BaseMessageTemplateForm & { config_items?: ConfigItem[] }; delete payload.config_items; payload.config = config; if (payload.id) await defBaseMessageTemplateService.UpdateBaseMessageTemplate({ base_message_template: payload }); else await defBaseMessageTemplateService.CreateBaseMessageTemplate({ base_message_template: payload }); ElMessage.success(t("common.message.operation_success")); dialog.visible = false; await proTable.value?.getTableList(); }
async function handleDelete(value: BaseMessageTemplate | number | number[]) { const ids = (Array.isArray(value) ? value : typeof value === "object" ? [value.id] : normalizeSelectedIds(value)).join(","); if (!ids) return; try { await ElMessageBox.confirm(t("common.confirm.delete"), t("common.title.warning"), { type: "warning" }); } catch { return; } await defBaseMessageTemplateService.DeleteBaseMessageTemplate({ id: ids }); await proTable.value?.getTableList(); ElMessage.success(t("common.message.operation_success")); }
function handleClose() { Object.assign(formData, defaultForm()); formDialogRef.value?.resetFields(); }
function contentFormatLabel(value: MessageContentFormat) { return contentFormatOptions.value.find(item => item.value === value)?.label ?? ""; }
function configToItems(config: Record<string, any> | undefined): ConfigItem[] { return Object.entries(config ?? {}).map(([key, value]) => ({ key, value: typeof value === "string" ? value : JSON.stringify(value) })); }
function configItemsToMap(items: ConfigItem[]): Record<string, unknown> | undefined { const result: Record<string, unknown> = {}; for (const item of items) { const key = item.key.trim(); if (!key || Object.prototype.hasOwnProperty.call(result, key)) { ElMessage.error(t("system.base.message_template.message.config_invalid")); return undefined; } const value = item.value.trim(); if (!value) { result[key] = ""; continue; } try { result[key] = JSON.parse(value); } catch { result[key] = value; } } return result; }
</script>
