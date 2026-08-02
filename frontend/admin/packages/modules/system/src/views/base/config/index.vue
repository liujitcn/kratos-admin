<!-- 系统配置 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestBaseConfigTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey, { resource: t('system.config.resource') })"
      width="1200px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #textValue>
        <el-input
          v-model="formData.value"
          :placeholder="t('system.common.validation.requiredInput', { field: t('system.config.field.value') })"
        />
      </template>
      <template #imageValue>
        <UploadImg v-model:image-url="formData.value" upload-type="config" />
      </template>
      <template #richTextValue>
        <WangEditor v-model:value="formData.value" />
      </template>
      <template #dictValue>
        <Dict
          v-if="dictValueCode"
          v-model="formData.value"
          :code="dictValueCode"
          code-type="string"
          :placeholder="t('system.common.validation.requiredSelect', { field: t('system.config.field.value') })"
        />
        <el-input
          v-else
          v-model="formData.value"
          :placeholder="t('system.common.validation.requiredInput', { field: t('system.config.field.value') })"
        />
      </template>
      <template #booleanValue>
        <el-switch
          v-model="formData.value"
          active-value="true"
          inactive-value="false"
          active-text="true"
          inactive-text="false"
          inline-prompt
        />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { ElImage, ElMessage, ElMessageBox, ElTag, ElTooltip } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import DictLabel from "@liujitcn/kratos-admin-core/components/Dict/DictLabel.vue";
import UploadImg from "@liujitcn/kratos-admin-core/components/Upload/Img.vue";
import WangEditor from "@liujitcn/kratos-admin-core/components/WangEditor/index.vue";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseConfigService } from "@liujitcn/kratos-admin-system/api/system/base_config";
import type {
  BaseConfig,
  BaseConfigForm,
  PageBaseConfigRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_config";
import { BaseConfigSite } from "@liujitcn/kratos-admin-system/rpc/base/v1/enum";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { BaseConfigType } from "@liujitcn/kratos-admin-system/rpc/system/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseConfig",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  titleKey: "system.common.action.createResource",
  visible: false
});

const formData = reactive<BaseConfigForm>({
  /** 配置ID */
  id: 0,
  /** 位置：枚举【BaseConfigSite】 */
  site: BaseConfigSite.UNKNOWN_BCS,
  /** 配置名称 */
  name: "",
  /** 配置类型：枚举【BaseConfigType】 */
  type: BaseConfigType.UNKNOWN_BCT,
  /** 配置key */
  key: "",
  /** 配置value */
  value: "",
  /** 状态 */
  status: Status.ENABLE
});

const rules = computed(() => ({
  site: [
    {
      required: true,
      message: t("system.common.validation.requiredSelect", { field: t("system.config.field.site") }),
      trigger: "change"
    }
  ],
  name: [
    {
      required: true,
      message: t("system.common.validation.requiredInput", { field: t("system.config.field.name") }),
      trigger: "blur"
    },
    {
      max: 50,
      message: t("system.common.validation.maxLength", { field: t("system.config.field.name"), max: 50 }),
      trigger: "blur"
    }
  ],
  type: [
    {
      required: true,
      message: t("system.common.validation.requiredSelect", { field: t("system.config.field.type") }),
      trigger: "change"
    }
  ],
  key: [
    {
      required: true,
      message: t("system.common.validation.requiredInput", { field: t("system.config.field.key") }),
      trigger: "blur"
    },
    {
      max: 50,
      message: t("system.common.validation.maxLength", { field: t("system.config.field.key"), max: 50 }),
      trigger: "blur"
    }
  ],
  value: [
    {
      required: true,
      message: t("system.common.validation.requiredInput", { field: t("system.config.field.value") }),
      trigger: "blur"
    }
  ],
  status: [
    {
      required: true,
      message: t("system.common.validation.requiredSelect", { field: t("system.common.field.status") }),
      trigger: "change"
    }
  ]
}));

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

/** 字典类系统配置与字典编码的映射关系。 */
const BASE_CONFIG_DICT_CODE_MAP: Record<string, string> = {
  captchaType: "captcha_type"
};

/** 当前字典类配置对应的字典编码，未配置映射时允许退回手动输入。 */
const dictValueCode = computed(() => BASE_CONFIG_DICT_CODE_MAP[formData.key] ?? "");

/** 系统配置表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "name",
    label: t("system.config.field.name"),
    component: "input",
    props: {
      placeholder: t("system.common.validation.requiredInput", { field: t("system.config.field.name") }),
      disabled: formData.id > 0
    }
  },
  {
    prop: "site",
    label: t("system.config.field.site"),
    component: "dict",
    props: { code: "base_config_site", disabled: formData.id > 0 }
  },
  {
    prop: "key",
    label: t("system.config.field.key"),
    component: "input",
    props: {
      placeholder: t("system.common.validation.requiredInput", { field: t("system.config.field.key") }),
      disabled: formData.id > 0
    }
  },
  {
    prop: "type",
    label: t("system.config.field.type"),
    component: "dict",
    props: { code: "base_config_type", disabled: formData.id > 0 }
  },
  {
    prop: "value",
    label: t("system.config.field.value"),
    component: "slot",
    slotName: "textValue",
    visible: model => model.type == BaseConfigType.TEXT
  },
  {
    prop: "value",
    label: t("system.config.field.value"),
    component: "slot",
    slotName: "imageValue",
    visible: model => model.type == BaseConfigType.IMAGE
  },
  {
    prop: "value",
    label: t("system.config.field.value"),
    component: "slot",
    slotName: "richTextValue",
    visible: model => model.type == BaseConfigType.RICH_TEXT,
    colSpan: 24
  },
  {
    prop: "value",
    label: t("system.config.field.value"),
    component: "slot",
    slotName: "dictValue",
    visible: model => model.type == BaseConfigType.DICT
  },
  {
    prop: "value",
    label: t("system.config.field.value"),
    component: "slot",
    slotName: "booleanValue",
    visible: model => model.type == BaseConfigType.BOOLEAN
  },
  { prop: "status", label: t("system.common.field.status"), component: "radio-group", options: statusOptions.value }
]);

/** 系统配置表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "site", label: t("system.config.field.site"), minWidth: 120, dictCode: "base_config_site", search: { el: "select" } },
  { prop: "name", label: t("system.config.field.name"), minWidth: 140, search: { el: "input" } },
  { prop: "type", label: t("system.config.field.type"), minWidth: 120, dictCode: "base_config_type", search: { el: "select" } },
  {
    prop: "key",
    label: t("system.config.field.key"),
    minWidth: 160,
    search: { el: "input" },
    render: scope => renderConfigKeyCell(scope.row as BaseConfig)
  },
  {
    prop: "status",
    label: t("system.common.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:config:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseConfig)
    }
  },
  { prop: "created_at", label: t("system.common.field.createdAt"), minWidth: 180 },
  { prop: "updated_at", label: t("system.common.field.updatedAt"), minWidth: 180 },
  {
    prop: "operation",
    label: t("system.common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:config:update"],
        params: scope => ({ configId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.configId as number | undefined) ?? (scope.row as BaseConfig).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:config:delete"],
        onClick: scope => handleDelete(scope.row as BaseConfig)
      }
    ]
  }
]);

/**
 * 将配置键渲染为可悬停查看配置值的单元格。
 */
function renderConfigKeyCell(row: BaseConfig) {
  return h(
    ElTooltip,
    {
      placement: "top-start",
      effect: "light",
      showAfter: 200,
      enterable: true,
      maxWidth: 420
    },
    {
      default: () => h("span", { class: "config-key-cell" }, row.key),
      content: () => renderConfigValuePreview(row)
    }
  );
}

/**
 * 根据配置类型渲染悬停预览内容。
 */
function renderConfigValuePreview(row: BaseConfig) {
  if (row.type === BaseConfigType.IMAGE) {
    return row.value
      ? h(ElImage, {
          src: row.value,
          previewSrcList: [row.value],
          previewTeleported: true,
          fit: "contain",
          style: { width: "180px", height: "120px", borderRadius: "4px" }
        })
      : h("span", { class: "config-value-preview" }, t("system.config.message.imageMissing"));
  }

  if (row.type === BaseConfigType.BOOLEAN) {
    const value = row.value === "true" ? "true" : "false";
    return h(ElTag, { type: value === "true" ? "success" : "info" }, () => value);
  }

  if (row.type === BaseConfigType.DICT && BASE_CONFIG_DICT_CODE_MAP[row.key]) {
    return h(DictLabel, {
      code: BASE_CONFIG_DICT_CODE_MAP[row.key],
      modelValue: row.value,
      class: "config-value-preview"
    });
  }

  if (row.type === BaseConfigType.RICH_TEXT) {
    return row.value
      ? h("div", { class: "config-rich-text-preview", innerHTML: row.value })
      : h("span", { class: "config-value-preview" }, t("system.config.message.richTextMissing"));
  }

  const value = row.value;
  return h("span", { class: "config-value-preview" }, value || t("system.config.message.valueMissing"));
}

/** 系统配置顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:config:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:config:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseConfig[])
  },
  {
    label: t("system.common.action.refreshCache"),
    type: "primary",
    icon: RefreshLeft,
    hidden: () => !BUTTONS.value["base:config:refresh"],
    onClick: () => handleRefreshCache()
  }
]);

/**
 * 请求系统配置列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseConfigTable(params: PageBaseConfigRequest) {
  const data = await defBaseConfigService.PageBaseConfig(buildPageRequest(params));
  return { data: { list: data.base_configs ?? [], total: data.total } };
}

/**
 * 刷新系统配置表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开系统配置弹窗。
 */
function handleOpenDialog(configId?: number) {
  resetForm();
  dialog.titleKey = configId ? "system.common.action.editResource" : "system.common.action.createResource";
  dialog.visible = true;
  if (!configId) return;

  defBaseConfigService.GetBaseConfig({ id: configId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭系统配置弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置系统配置表单，避免新增时保留旧值。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.site = BaseConfigSite.UNKNOWN_BCS;
  formData.name = "";
  formData.type = BaseConfigType.UNKNOWN_BCT;
  formData.key = "";
  formData.value = "";
  formData.status = Status.ENABLE;
}

/**
 * 提交系统配置表单。
 */
function handleSubmit() {
  if (formData.type === BaseConfigType.BOOLEAN) {
    formData.value = formData.value === "true" ? "true" : "false";
  }
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseConfigForm;
    const request = submitData.id
      ? defBaseConfigService.UpdateBaseConfig({ base_config: submitData })
      : defBaseConfigService.CreateBaseConfig({ base_config: submitData });
    request.then(() => {
      ElMessage.success(
        t(submitData.id ? "system.common.message.updateSuccess" : "system.common.message.createSuccess", {
          resource: t("system.config.resource")
        })
      );
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 刷新服务端配置缓存，使用防抖避免重复点击。
 */
const handleRefreshCache = useDebounceFn(() => {
  defBaseConfigService.RefreshBaseConfigCache({}).then(() => {
    ElMessage.success(t("system.common.message.refreshSuccess"));
  });
}, 1000);

/**
 * 在系统配置状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseConfig) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const configName = row.name || row.key || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(
      t("system.common.dialog.statusChange", {
        action: text,
        resource: t("system.config.resource"),
        field: t("system.config.field.name"),
        value: configName
      }),
      t("common.title.notice"),
      {
        confirmButtonText: t("common.action.confirm"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defBaseConfigService.SetBaseConfigStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("system.common.message.statusSuccess", { action: text }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除系统配置，兼容单项删除与多选删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseConfig | BaseConfig[]) {
  const configList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseConfig[])
    : selected && typeof selected === "object"
      ? [selected as BaseConfig]
      : [];
  const configIds = (
    configList.length
      ? configList.map(item => item.id)
      : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!configIds) {
    ElMessage.warning(t("system.common.message.selectDeleteItem"));
    return;
  }

  const confirmMessage = configList.length
    ? configList.length === 1
      ? `${t("system.common.dialog.deleteSingle", { resource: t("system.config.resource") })}\n${t("system.common.dialog.resourceField", { field: t("system.config.field.name"), value: configList[0].name || configList[0].key || `ID:${configList[0].id}` })}`
      : t("system.common.dialog.deleteBatch", { count: configList.length, unit: "", resource: t("system.config.resource") })
    : t("system.common.dialog.deleteSelected", { resource: t("system.config.resource") });

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseConfigService.DeleteBaseConfig({ id: configIds }).then(() => {
        ElMessage.success(t("system.common.message.deleteSuccess", { resource: t("system.config.resource") }));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.common.dialog.cancelDelete", { resource: t("system.config.resource") }));
    }
  );
}
</script>

<style scoped lang="scss">
.config-key-cell {
  cursor: pointer;
  border-bottom: 1px dashed var(--el-color-info);
}

.config-value-preview {
  display: inline-block;
  max-width: 380px;
  white-space: pre-wrap;
  word-break: break-word;
}

.config-rich-text-preview {
  max-width: 520px;
  max-height: 360px;
  overflow: auto;
  line-height: 1.6;
  word-break: break-word;
}

.config-rich-text-preview :deep(p) {
  margin: 0 0 8px;
}

.config-rich-text-preview :deep(h1),
.config-rich-text-preview :deep(h2),
.config-rich-text-preview :deep(h3),
.config-rich-text-preview :deep(h4),
.config-rich-text-preview :deep(h5),
.config-rich-text-preview :deep(h6) {
  margin: 0 0 8px;
  line-height: 1.35;
}

.config-rich-text-preview :deep(ul),
.config-rich-text-preview :deep(ol) {
  margin: 0 0 8px;
  padding-left: 24px;
}

.config-rich-text-preview :deep(img) {
  max-width: 100%;
  height: auto;
}
</style>
