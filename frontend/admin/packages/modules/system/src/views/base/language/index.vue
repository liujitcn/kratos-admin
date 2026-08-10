<!-- 语言管理 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseLanguageTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.language.action.edit' : 'system.base.language.action.create')"
      width="620px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { applyLanguageConfig, defLanguageService, t } from "@liujitcn/kratos-admin-core";
import { defBaseLanguageService, invalidateEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";
import type { BaseLanguage, BaseLanguageForm, PageBaseLanguageRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_language";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseLanguage", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ editing: false, visible: false });
const formData = reactive<BaseLanguageForm>({
  id: 0,
  language_code: "",
  language_name: "",
  native_name: "",
  sort: 100,
  status: Status.STATUS_ENABLE
});

const rules = computed(() => ({
  language_code: [
    { required: true, message: t("system.base.language.placeholder.code"), trigger: "blur" },
    { max: 16, message: t("common.validation.max_length", { field: t("system.base.language.field.code"), max: 16 }), trigger: "blur" }
  ],
  language_name: [
    { required: true, message: t("system.base.language.placeholder.name"), trigger: "blur" },
    { max: 50, message: t("common.validation.max_length", { field: t("system.base.language.field.name"), max: 50 }), trigger: "blur" }
  ],
  native_name: [
    { required: true, message: t("system.base.language.placeholder.native_name"), trigger: "blur" },
    { max: 50, message: t("common.validation.max_length", { field: t("system.base.language.field.native_name"), max: 50 }), trigger: "blur" }
  ],
  sort: [{ required: true, message: t("system.base.language.placeholder.sort"), trigger: "change" }]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const formFields = computed<ProFormField[]>(() => [
  { prop: "language_code", label: t("system.base.language.field.code"), component: "input", props: { placeholder: t("system.base.language.placeholder.code"), disabled: dialog.editing } },
  { prop: "language_name", label: t("system.base.language.field.name"), component: "input", props: { placeholder: t("system.base.language.placeholder.name") } },
  { prop: "native_name", label: t("system.base.language.field.native_name"), component: "input", props: { placeholder: t("system.base.language.placeholder.native_name") } },
  { prop: "sort", label: t("system.base.language.field.sort"), component: "input-number", props: { min: 0, controlsPosition: "right", style: { width: "100%" } } },
  { prop: "status", label: t("system.base.language.field.status"), component: "radio-group", options: statusOptions.value }
]);

const columns = computed<ColumnProps[]>(() => [
  { prop: "native_name", label: t("system.base.language.field.native_name"), minWidth: 130 },
  { prop: "language_name", label: t("system.base.language.field.name"), minWidth: 130, search: { el: "input" } },
  { prop: "language_code", label: t("system.base.language.field.code"), minWidth: 120, search: { el: "input" } },
  {
    prop: "is_primary",
    label: t("system.base.language.field.primary"),
    minWidth: 100,
    cellType: "status",
    statusProps: {
      activeValue: true,
      inactiveValue: false,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:language:primary"],
      beforeChange: scope => handleBeforeSetPrimary(scope.row as BaseLanguage)
    }
  },
  {
    prop: "status",
    label: t("system.base.language.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:language:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseLanguage)
    }
  },
  { prop: "sort", label: t("system.base.language.field.sort"), width: 80 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:language:update"], onClick: scope => handleOpenDialog((scope.row as BaseLanguage).id) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:language:delete"], onClick: scope => handleDelete(scope.row as BaseLanguage) }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:language:create"], onClick: () => handleOpenDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:language:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => handleDelete(scope.selectedList as BaseLanguage[]) }
]);

/** 请求语言分页列表。 */
async function requestBaseLanguageTable(params: PageBaseLanguageRequest) {
  const data = await defBaseLanguageService.PageBaseLanguage(buildPageRequest(params));
  return { data: { list: data.base_languages ?? [], total: data.total } };
}

/** 打开语言编辑弹窗。 */
async function handleOpenDialog(id?: number) {
  resetForm();
  dialog.editing = Boolean(id);
  dialog.visible = true;
  if (!id) return;
  Object.assign(formData, await defBaseLanguageService.GetBaseLanguage({ id }));
}

/** 关闭语言编辑弹窗。 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/** 重置语言表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, { id: 0, language_code: "", language_name: "", native_name: "", sort: 100, status: Status.STATUS_ENABLE });
}

/** 提交语言表单。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const payload = JSON.parse(JSON.stringify(formData)) as BaseLanguageForm;
    const request = payload.id
      ? defBaseLanguageService.UpdateBaseLanguage({ base_language: payload })
      : defBaseLanguageService.CreateBaseLanguage({ base_language: payload });
    request.then(() => {
      invalidateEnabledBaseLanguages();
      void refreshAdminLanguageConfig();
      ElMessage.success(t(payload.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.language.resource") }));
      handleCloseDialog();
      proTable.value?.getTableList();
    });
  });
}

/** 设置语言状态前保护主语言。 */
function handleBeforeSetStatus(row: BaseLanguage) {
  if (row.is_primary && row.status === Status.STATUS_ENABLE) {
    ElMessage.warning(t("system.base.language.message.primary_cannot_disable"));
    return Promise.resolve(false);
  }
  return ElMessageBox.confirm(
    t("system.base.language.message.confirm_status", { action: row.status === Status.STATUS_ENABLE ? t("common.status.disabled") : t("common.status.enabled"), name: row.native_name }),
    t("common.title.warning"),
    { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
  ).then(
    () =>
      defBaseLanguageService
        .SetBaseLanguageStatus({ id: row.id, status: row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE })
        .then(() => {
          invalidateEnabledBaseLanguages();
          void refreshAdminLanguageConfig();
          return true;
        }),
    () => false
  );
}

/** 设置主语言前确认切换。 */
function handleBeforeSetPrimary(row: BaseLanguage) {
  if (row.is_primary) return Promise.resolve(false);
  return ElMessageBox.confirm(
    t("system.base.language.message.confirm_primary", { name: row.native_name }),
    t("common.title.warning"),
    { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
  ).then(
    () =>
      defBaseLanguageService
        .SetBaseLanguagePrimary({ id: row.id })
        .then(() => {
          invalidateEnabledBaseLanguages();
          void refreshAdminLanguageConfig();
          proTable.value?.getTableList();
          return true;
        }),
    () => false
  );
}

/** 删除语言，兼容单项和批量操作。 */
function handleDelete(selected?: BaseLanguage | BaseLanguage[] | number | string | Array<number | string>) {
  const rows = Array.isArray(selected) ? (selected.filter(item => typeof item === "object") as BaseLanguage[]) : selected && typeof selected === "object" ? [selected] : [];
  if (rows.some(item => item.is_primary)) {
    ElMessage.warning(t("system.base.language.message.primary_cannot_delete"));
    return;
  }
  const ids = (rows.length ? rows.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)).join(",");
  if (!ids) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  ElMessageBox.confirm(t("system.base.language.message.confirm_delete", { name: rows.length === 1 ? rows[0].native_name : ids }), t("common.title.warning"), {
    type: "warning",
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel")
  }).then(() => defBaseLanguageService.DeleteBaseLanguage({ id: ids })).then(() => {
    invalidateEnabledBaseLanguages();
    void refreshAdminLanguageConfig();
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.language.resource") }));
    proTable.value?.getTableList();
  });
}

/** refreshAdminLanguageConfig 刷新核心语言切换器的运行时配置。 */
async function refreshAdminLanguageConfig() {
  try {
    applyLanguageConfig(await defLanguageService.OptionLanguage({}));
  } catch {
    // 语言管理操作成功但公共语言查询失败时，保留当前切换器配置。
  }
}
</script>
