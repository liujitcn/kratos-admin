<!-- 字典数据 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseDictItemTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.dict.item.action.edit' : 'system.base.dict.item.action.create')"
      width="820px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #tagTypeField>
        <div class="dict-item-tag-type">
          <el-tag v-if="formData.tag_type" :type="formatTagType(formData.tag_type)" class="mr-2">
            {{ formData.label || t("system.base.dict.item.value.tag_preview") }}
          </el-tag>
          <el-radio-group v-model="formData.tag_type">
            <el-radio value="success" border size="small">success</el-radio>
            <el-radio value="warning" border size="small">warning</el-radio>
            <el-radio value="info" border size="small">info</el-radio>
            <el-radio value="primary" border size="small">primary</el-radio>
            <el-radio value="danger" border size="small">danger</el-radio>
            <el-radio value="" border size="small">{{ t("system.base.dict.item.action.clear_tag") }}</el-radio>
          </el-radio-group>
        </div>
      </template>
      <template #translations>
        <DynamicTranslationEditor
          v-model="translationValues"
          :source="formData.label"
          :source-locale="formData.id > 0 ? undefined : locale"
          :maxlength="100"
        />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
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
import { defBaseDictService } from "@liujitcn/kratos-admin-system/api/system/base_dict";
import { loadEnabledBaseLanguages } from "@liujitcn/kratos-admin-system/api/system/base_language";
import type {
  BaseDictItem,
  BaseDictItemForm,
  PageBaseDictItemRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dict";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { TranslationTargetType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_translation";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import DynamicTranslationEditor from "@liujitcn/kratos-admin-system/components/DynamicTranslationEditor.vue";
import DynamicTranslationCell from "@liujitcn/kratos-admin-system/components/DynamicTranslationCell.vue";
import {
  normalizeDynamicTranslations,
  serializeDynamicTranslations,
  type DynamicTranslationRecord,
  type DynamicTranslationValue
} from "@liujitcn/kratos-admin-system/components/dynamicTranslation";

defineOptions({
  name: "BaseDictItem",
  inheritAttrs: false
});

const route = useRoute();
const { BUTTONS } = useAuthButtons();
const { locale } = useLocaleStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const translationValues = ref<DynamicTranslationValue[]>(normalizeDynamicTranslations(undefined, "label"));

const dictId = ref(Number(route.query.dictId ?? 0));

const dialog = reactive({
  editing: false,
  visible: false
});

const formData = reactive<BaseDictItemForm>({
  /** 字典项ID */
  id: 0,
  /** 字典ID */
  dict_id: dictId.value,
  /** 字典值 */
  value: "",
  /** 字典项标签 */
  label: "",
  /** 非主语言翻译 */
  translations: [],
  /** 标签类型 */
  tag_type: "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  value: [{ required: true, max: 50, message: t("system.base.dict.item.validation.value"), trigger: "blur" }],
  label: [{ required: true, max: 100, message: t("system.base.dict.item.validation.label"), trigger: "blur" }],
  tag_type: [
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.dict.item.field.tag_type"), max: 50 }),
      trigger: "blur"
    }
  ],
  sort: [{ required: true, message: t("common.validation.sort_positive"), trigger: "blur" }],
  status: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.status") }),
      trigger: "change"
    }
  ]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

/** 字典项表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "label",
    label: t("system.base.dict.item.field.label"),
    component: "input",
    props: { placeholder: t("system.base.dict.item.placeholder.label") }
  },
  {
    prop: "value",
    label: t("system.base.dict.item.field.value"),
    component: "input",
    props: { placeholder: t("system.base.dict.item.placeholder.value") }
  },
  {
    prop: "translations",
    label: t("system.base.translation.field.translations"),
    component: "slot",
    slotName: "translations",
    colSpan: 24
  },
  {
    prop: "sort",
    label: t("common.field.sort"),
    component: "input-number",
    props: { min: 1, precision: 0, step: 1, controlsPosition: "right", style: { width: "100%" } }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "tag_type", label: t("system.base.dict.item.field.tag_type"), component: "slot", slotName: "tagTypeField", colSpan: 24 }
]);

/**
 * 规范化标签类型，兼容 Element Plus Tag 的可选值。
 */
function formatTagType(tag_type: string) {
  if (["success", "info", "warning", "primary", "danger"].includes(tag_type)) {
    return tag_type as "success" | "info" | "warning" | "primary" | "danger";
  }
  return undefined;
}

/**
 * 渲染标签类型列，统一复用字典项标签的展示方式。
 */
function renderTagTypeCell(scope: RenderScope<BaseDictItem>) {
  if (!scope.row.tag_type) return t("common.value.none");
  return h(
    ElTag,
    {
      type: formatTagType(scope.row.tag_type),
      effect: "plain"
    },
    () => scope.row.label
  );
}

/** 渲染字典项标签翻译预览，并复用当前页面的编辑弹窗。 */
function renderDictItemLabelCell(scope: RenderScope<BaseDictItem>) {
  const row = scope.row;
  return h(DynamicTranslationCell, {
    source: row.label,
    targetType: TranslationTargetType.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM,
    targetId: row.id,
    translations: row.translations
  });
}

/** 字典项表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "label",
    label: t("system.base.dict.item.field.label"),
    minWidth: 140,
    search: { el: "input" },
    showOverflowTooltip: false,
    render: scope => renderDictItemLabelCell(scope as unknown as RenderScope<BaseDictItem>)
  },
  { prop: "value", label: t("system.base.dict.item.field.value"), minWidth: 140 },
  { prop: "sort", label: t("common.field.sort"), minWidth: 90, align: "right" },
  {
    prop: "tag_type",
    label: t("system.base.dict.item.field.tag_type"),
    minWidth: 120,
    render: scope => renderTagTypeCell(scope as unknown as RenderScope<BaseDictItem>)
  },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:dict-item:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseDictItem)
    }
  },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.action"),
    width: 180,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:dict-item:update"],
        params: scope => ({ dictItemId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.dictItemId as number | undefined) ?? (scope.row as BaseDictItem).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:dict-item:delete"],
        onClick: scope => handleDelete(scope.row as BaseDictItem)
      }
    ]
  }
]);

/** 字典项顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:dict-item:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:dict-item:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseDictItem[])
  }
]);

watch(
  () => route.query.dictId,
  value => {
    dictId.value = Number(value ?? 0);
    formData.dict_id = dictId.value;
    proTable.value?.search();
  }
);

/**
 * 请求字典项分页列表，并补充当前路由上的字典 ID。
 */
async function requestBaseDictItemTable(params: PageBaseDictItemRequest) {
  await loadEnabledBaseLanguages();
  const data = await defBaseDictService.PageBaseDictItem({ ...buildPageRequest(params), dict_id: dictId.value });
  return { data: { list: data.base_dict_items ?? [], total: data.total } };
}

/**
 * 刷新字典项表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开字典项编辑弹窗。
 */
async function handleOpenDialog(dictItemId?: number) {
  await loadEnabledBaseLanguages();
  resetForm();
  dialog.editing = Boolean(dictItemId);
  dialog.visible = true;
  if (!dictItemId) return;

  const data = await defBaseDictService.GetBaseDictItem({ id: dictItemId });
  Object.assign(formData, data);
  translationValues.value = normalizeDynamicTranslations(data.translations as DynamicTranslationRecord[], "label");
}

/**
 * 关闭字典项弹窗并恢复默认值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置字典项表单，避免弹窗之间相互污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.dict_id = dictId.value;
  formData.value = "";
  formData.label = "";
  formData.translations = [];
  formData.tag_type = "";
  formData.sort = 1;
  formData.status = Status.ENABLE;
  translationValues.value = normalizeDynamicTranslations(undefined, "label");
}

/**
 * 提交字典项表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(isValid => {
    if (!isValid) return;

    formData.dict_id = dictId.value;
    const submitData = JSON.parse(JSON.stringify(formData)) as BaseDictItemForm;
    submitData.translations = serializeDynamicTranslations(
      translationValues.value,
      TranslationTargetType.TRANSLATION_TARGET_TYPE_BASE_DICT_ITEM,
      submitData.id
    );
    const request = submitData.id
      ? defBaseDictService.UpdateBaseDictItem({ base_dict_item: submitData })
      : defBaseDictService.CreateBaseDictItem({ base_dict_item: submitData });
    request.then(() => {
      ElMessage.success(
        t(submitData.id ? "system.base.dict.item.message.update_success" : "system.base.dict.item.message.create_success")
      );
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在字典项状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseDictItem) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const action = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const itemName = row.label || row.value || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(
      t("system.base.dict.item.message.confirm_status", { action, name: itemName }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseDictService.SetBaseDictItemStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除字典项，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseDictItem | BaseDictItem[]) {
  const dictItemList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseDictItem[])
    : selected && typeof selected === "object"
      ? [selected as BaseDictItem]
      : [];
  const dictItemIds = (
    dictItemList.length
      ? dictItemList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!dictItemIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const singleItemName = dictItemList[0]?.label || dictItemList[0]?.value || `ID:${dictItemList[0]?.id ?? ""}`;
  const confirmMessage = dictItemList.length
    ? dictItemList.length === 1
      ? t("system.base.dict.item.message.confirm_delete_single", { name: singleItemName })
      : t("system.base.dict.item.message.confirm_delete_batch", { count: dictItemList.length })
    : t("system.base.dict.item.message.confirm_delete_selected");

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseDictService.DeleteBaseDictItem({ id: dictItemIds }).then(() => {
        ElMessage.success(t("system.base.dict.item.message.delete_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.dict.item.message.delete_canceled"));
    }
  );
}
</script>

<style scoped>
.dict-item-tag-type {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
