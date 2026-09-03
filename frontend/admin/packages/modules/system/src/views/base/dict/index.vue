<!-- 字典 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDictTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.dict.action.edit' : 'system.base.dict.action.create')"
      width="700px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #i18ns>
        <DynamicI18nEditor
          v-model="i18nValues"
          :source="formData.name"
          :maxlength="50"
        />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, List } from "@element-plus/icons-vue";
import type {
  ColumnProps,
  HeaderActionProps,
  ProTableInstance,
  RenderScope
} from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseDictService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_dict";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_language";
import type { BaseDict, BaseDictForm, PageBaseDictRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dict";
import router, { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { I18nTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_i18n";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/DynamicI18nEditor.vue";
import DynamicI18nCell from "@liujitcn/kratos-admin-system/components/DynamicI18nCell.vue";
import {
  normalizeDynamicI18ns,
  serializeDynamicI18ns,
  type DynamicI18nRecord,
  type DynamicI18nValue
} from "@liujitcn/kratos-admin-system/components/dynamicI18n";

defineOptions({
  name: "BaseDict",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const i18nValues = ref<DynamicI18nValue[]>(normalizeDynamicI18ns(undefined));
const dialog = reactive({
  editing: false,
  visible: false
});

const formData = reactive<BaseDictForm>({
  /** 字典ID */
  id: 0,
  /** 字典编号 */
  code: "",
  /** 字典名称 */
  name: "",
  /** 非主语言翻译 */
  i18ns: [],
  /** 状态 */
  status: Status.STATUS_ENABLE
});

const rules = computed(() => ({
  name: [
    { required: true, message: t("system.base.dict.placeholder.name"), trigger: "blur" },
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.dict.field.name"), max: 50 }),
      trigger: "blur"
    }
  ],
  code: [
    { required: true, message: t("system.base.dict.placeholder.code"), trigger: "blur" },
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.dict.field.code"), max: 50 }),
      trigger: "blur"
    }
  ],
  status: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.status") }),
      trigger: "change"
    }
  ]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

/** 字典表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: t("system.base.dict.field.name"),
    component: "input",
    props: { placeholder: t("system.base.dict.placeholder.name") }
  },
  {
    prop: "code",
    label: t("system.base.dict.field.code"),
    component: "input",
    props: { placeholder: t("system.base.dict.placeholder.code") }
  },
  { prop: "i18ns", label: t("system.base.i18n.field.i18ns"), component: "slot", slotName: "i18ns" },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

/** 渲染字典名称翻译预览，并复用当前页面的编辑弹窗。 */
function renderDictNameCell(scope: RenderScope<BaseDict>) {
  const row = scope.row;
  return h(DynamicI18nCell, {
    source: row.name,
    targetType: I18nTargetType.I18N_TARGET_TYPE_BASE_DICT_NAME,
    targetId: row.id,
    i18ns: row.i18ns
  });
}

/** 字典表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "name",
    label: t("system.base.dict.field.name"),
    minWidth: 140,
    search: { el: "input" },
    showOverflowTooltip: false,
    render: scope => renderDictNameCell(scope as unknown as RenderScope<BaseDict>)
  },
  { prop: "code", label: t("system.base.dict.field.code"), minWidth: 160, search: { el: "input" } },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:dict:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDict)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.action"),
    width: 240,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("system.base.dict.action.items"),
        type: "primary",
        link: true,
        icon: List,
        hidden: () => !BUTTONS.value["base:dict:items"],
        onClick: scope => handleOpenBaseDictItem(scope.row as BaseDict)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dict:update"],
        params: scope => ({ dictId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.dictId as number | undefined) ?? (scope.row as BaseDict).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dict:delete"],
        onClick: scope => handleDelete(scope.row as BaseDict)
      }
    ]
  }
]);

/** 字典顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dict:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dict:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDict[])
  }
]);

/**
 * 请求字典列表，并由 ProTable 统一管理分页搜索。
 */
async function requestBaseDictTable(params: PageBaseDictRequest) {
  await loadEnabledBaseLanguages();
  const data = await defBaseDictService.PageBaseDict(buildPageRequest(params));
  return { data: { list: data.base_dicts ?? [], total: data.total } };
}

/**
 * 刷新字典表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开字典编辑弹窗。
 */
async function handleOpenDialog(dictId?: number) {
  await loadEnabledBaseLanguages();
  resetForm();
  dialog.editing = Boolean(dictId);
  dialog.visible = true;
  if (!dictId) return;

  const data = await defBaseDictService.GetBaseDict({ id: dictId });
  Object.assign(formData, data);
	i18nValues.value = normalizeDynamicI18ns(data.i18ns as DynamicI18nRecord[]);
}

/**
 * 关闭字典弹窗并恢复表单初始值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置字典表单，避免新增时带入上次编辑结果。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.code = "";
  formData.name = "";
  formData.i18ns = [];
  formData.status = Status.STATUS_ENABLE;
	i18nValues.value = normalizeDynamicI18ns(undefined);
}

/**
 * 提交字典表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDictForm;
    submitData.i18ns = serializeDynamicI18ns(
      i18nValues.value,
      I18nTargetType.I18N_TARGET_TYPE_BASE_DICT_NAME,
      submitData.id
    );
    const request = submitData.id
      ? defBaseDictService.UpdateBaseDict({ base_dict: submitData })
      : defBaseDictService.CreateBaseDict({ base_dict: submitData });
    request.then(() => {
      ElMessage.success(t(submitData.id ? "system.base.dict.message.update_success" : "system.base.dict.message.create_success"));
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDict) {
  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(nextStatus === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  const dictName = row.name || row.code || String(row.id);
  try {
    await ElMessageBox.confirm(
      t("system.base.dict.message.confirm_status", { action, name: dictName }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseDictService.SetBaseDictStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDict | BaseDict[]) {
  const dictList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDict[])
    : selected && typeof selected === "object"
      ? [selected as BaseDict]
      : [];
  const dictIds = (
    dictList.length ? dictList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = dictList.length
    ? dictList.length === 1
      ? t("system.base.dict.message.confirm_delete_single", {
          name: dictList[0].name || dictList[0].code || `ID:${dictList[0].id}`
        })
      : t("system.base.dict.message.confirm_delete_batch", { count: dictList.length })
    : t("system.base.dict.message.confirm_delete_selected");

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDict({ id: dictIds }).then(() => {
        ElMessage.success(t("system.base.dict.message.delete_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.dict.message.delete_canceled"));
    }
  );
}

/**
 * 打开字典数据页面。
 */
function handleOpenBaseDictItem(row: BaseDict) {
  navigateTo(router, "/base/dict/item", { dictId: row.id, title: t("system.base.dict.title.items", { name: row.name }) });
}
</script>
