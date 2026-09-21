<!-- 国际化自定义翻译 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      :key="isDefaultTenant ? 'default-tenant' : 'normal-tenant'"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseI18nCustomTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.i18n_custom.action.edit' : 'system.base.i18n_custom.action.create')"
      width="680px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import {
  getCurrentLocale,
  getDefaultLocaleText,
  normalizeLocale,
  refreshAdminRuntimeConfig,
  t
} from "@liujitcn/kratos-admin-core";
import { defBaseI18nCustomService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_i18n_custom";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import type {
  BaseI18nCustom,
  BaseI18nCustomForm,
  PageBaseI18nCustomRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n_custom";
import { BaseConfigSite } from "@liujitcn/kratos-admin-system/rpc/base/v1/config";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseI18nCustom", inheritAttrs: false });

/** 自定义翻译表单状态，平台管理员新增时选择目标租户。 */
type BaseI18nCustomFormState = Omit<BaseI18nCustomForm, "tenant_id"> & {
  /** 租户ID。 */
  tenant_id?: number;
};

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const enabledLanguages = ref<Array<{ language_code: string; native_name?: string; language_name?: string }>>([]);
const dialog = reactive({ editing: false, visible: false });
const formData = reactive<BaseI18nCustomFormState>({
  id: 0,
  tenant_id: undefined,
  site: BaseConfigSite.BASE_CONFIG_SITE_ADMIN,
  key: "",
  locale: getCurrentLocale(),
  value: "",
  status: Status.STATUS_ENABLE,
  remark: ""
});

const siteOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.i18n_custom.site.system"), value: BaseConfigSite.BASE_CONFIG_SITE_SYSTEM },
  { label: t("system.base.i18n_custom.site.admin"), value: BaseConfigSite.BASE_CONFIG_SITE_ADMIN },
  { label: t("system.base.i18n_custom.site.app"), value: BaseConfigSite.BASE_CONFIG_SITE_APP }
]);

const localeOptions = computed<ProFormOption[]>(() =>
  enabledLanguages.value.map(item => ({
    label: item.native_name || item.language_name || item.language_code,
    value: item.language_code
  }))
);

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const rules = computed(() => ({
  tenant_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.tenant") }),
      trigger: "change"
    }
  ],
  site: [{ required: true, message: t("system.base.i18n_custom.placeholder.site"), trigger: "change" }],
  key: [{ required: true, message: t("system.base.i18n_custom.placeholder.key"), trigger: "blur" }],
  locale: [{ required: true, message: t("system.base.i18n_custom.placeholder.locale"), trigger: "change" }],
  value: [{ required: true, message: t("system.base.i18n_custom.placeholder.value"), trigger: "blur" }]
}));

const { isDefaultTenant, tenantColumns, tenantFormField, toRequestTenantId, loadTenantOptions } = useTenantScope();
onMounted(() => {
  if (isDefaultTenant.value) void loadTenantOptions();
});

const formFields = computed<ProFormField[]>(() => [
  tenantFormField({ label: t("common.field.tenant"), disabledOnEdit: true }),
  {
    prop: "site",
    label: t("system.base.i18n_custom.field.site"),
    component: "select",
    options: siteOptions.value,
    props: { disabled: dialog.editing }
  },
  {
    prop: "key",
    label: t("system.base.i18n_custom.field.key"),
    component: "input",
    props: { maxlength: 255, placeholder: t("system.base.i18n_custom.placeholder.key"), disabled: dialog.editing }
  },
  {
    prop: "locale",
    label: t("system.base.i18n_custom.field.locale"),
    component: "select",
    options: localeOptions.value,
    props: { filterable: true, placeholder: t("system.base.i18n_custom.placeholder.locale"), disabled: dialog.editing }
  },
  {
    prop: "value",
    label: t("system.base.i18n_custom.field.value"),
    component: "input",
    props: { type: "textarea", rows: 4, maxlength: 10000, showWordLimit: true, placeholder: t("system.base.i18n_custom.placeholder.value") }
  },
  { prop: "status", label: t("system.base.i18n_custom.field.status"), component: "radio-group", options: statusOptions.value },
  {
    prop: "remark",
    label: t("system.base.i18n_custom.field.remark"),
    component: "input",
    props: { maxlength: 255, placeholder: t("system.base.i18n_custom.placeholder.remark") }
  }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...tenantColumns({ label: t("common.field.tenant"), order: 1 }),
  { prop: "key", label: t("system.base.i18n_custom.field.key"), minWidth: 250, search: { el: "input" } },
  {
    prop: "site",
    label: t("system.base.i18n_custom.field.site"),
    width: 110,
    search: { el: "select", enum: siteOptions.value },
    render: scope => siteOptions.value.find(item => item.value === (scope.row as BaseI18nCustom).site)?.label || "-"
  },
  {
    prop: "locale",
    label: t("system.base.i18n_custom.field.locale"),
    width: 120,
    search: { el: "select", enum: localeOptions },
    render: scope => getLanguageLabel((scope.row as BaseI18nCustom).locale)
  },
  {
    prop: "default_value",
    label: t("system.base.i18n_custom.field.default_value"),
    minWidth: 200,
    render: scope => getDefaultLocaleText(normalizeLocale((scope.row as BaseI18nCustom).locale), (scope.row as BaseI18nCustom).key) || "-"
  },
  { prop: "value", label: t("system.base.i18n_custom.field.value"), minWidth: 200 },
  {
    prop: "status",
    label: t("system.base.i18n_custom.field.status"),
    width: 100,
    search: { el: "select", enum: statusOptions.value },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:i18n-custom:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseI18nCustom)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:i18n-custom:update"],
        onClick: scope => handleOpenDialog((scope.row as BaseI18nCustom).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:i18n-custom:delete"],
        onClick: scope => handleDelete(scope.row as BaseI18nCustom)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:i18n-custom:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:i18n-custom:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseI18nCustom[])
  }
]);

/** 请求国际化自定义翻译分页列表。 */
async function requestBaseI18nCustomTable(params: PageBaseI18nCustomRequest) {
  enabledLanguages.value = await loadLanguages();
  const data = await defBaseI18nCustomService.PageBaseI18nCustom({
    ...buildPageRequest(params),
    tenant_id: toRequestTenantId(params.tenant_id)
  });
  return { data: { list: data.items ?? [], total: data.total } };
}

/** 加载启用语言选项。 */
async function loadLanguages() {
  return loadEnabledBaseLanguages();
}

/** 返回语言显示名称。 */
function getLanguageLabel(locale: string) {
  const language = enabledLanguages.value.find(item => item.language_code === locale);
  return language?.native_name || language?.language_name || locale;
}

/** 打开国际化自定义翻译编辑弹窗。 */
async function handleOpenDialog(id?: number) {
  resetForm();
  dialog.editing = Boolean(id);
  await formDialogRef.value?.open({
    load: async () => ({
      languages: await loadLanguages(),
      data: id ? await defBaseI18nCustomService.GetBaseI18nCustom({ id }) : undefined,
      tenants: await loadTenantOptions()
    }),
    commit: ({ languages, data }) => {
      enabledLanguages.value = languages;
      if (data) Object.assign(formData, data);
    }
  });
}

/** 关闭编辑弹窗并重置表单。 */
function handleCloseDialog() {
  formDialogRef.value?.close();
  resetForm();
}

/** 重置国际化自定义翻译表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, {
    id: 0,
    tenant_id: undefined,
    site: BaseConfigSite.BASE_CONFIG_SITE_ADMIN,
    key: "",
    locale: getCurrentLocale(),
    value: "",
    status: Status.STATUS_ENABLE,
    remark: ""
  });
}

/** 提交国际化自定义翻译。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;
  const payload = JSON.parse(JSON.stringify(formData)) as BaseI18nCustomForm;
  if (payload.id) {
    await defBaseI18nCustomService.UpdateBaseI18nCustom({ i18n_custom: payload });
  } else {
    await defBaseI18nCustomService.CreateBaseI18nCustom({ i18n_custom: payload });
  }
  await refreshAdminRuntimeConfig();
  ElMessage.success(t(payload.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.i18n_custom.resource") }));
  handleCloseDialog();
  proTable.value?.getTableList();
}

/** 设置国际化自定义翻译状态前确认。 */
function handleBeforeSetStatus(row: BaseI18nCustom) {
  return ElMessageBox.confirm(
    t("system.base.i18n_custom.message.confirm_status", {
      action: row.status === Status.STATUS_ENABLE ? t("common.status.disabled") : t("common.status.enabled"),
      name: row.key
    }),
    t("common.title.warning"),
    { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
  ).then(
    async () => {
      await defBaseI18nCustomService.SetBaseI18nCustomStatus({
        id: row.id,
        status: row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE
      });
      await refreshAdminRuntimeConfig();
      return true;
    },
    () => false
  );
}

/** 删除国际化自定义翻译，兼容单项和批量操作。 */
function handleDelete(selected?: BaseI18nCustom | BaseI18nCustom[] | number | string | Array<number | string>) {
  const rows = Array.isArray(selected) ? (selected.filter(item => typeof item === "object") as BaseI18nCustom[]) : selected && typeof selected === "object" ? [selected] : [];
  const ids = (rows.length ? rows.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)).join(",");
  if (!ids) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  ElMessageBox.confirm(
    t("system.base.i18n_custom.message.confirm_delete", { name: rows.length === 1 ? rows[0].key : ids }),
    t("common.title.warning"),
    { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
  ).then(async () => {
    await defBaseI18nCustomService.DeleteBaseI18nCustom({ id: ids });
    await refreshAdminRuntimeConfig();
    ElMessage.success(t("common.message.delete_success"));
    proTable.value?.getTableList();
  });
}

</script>
