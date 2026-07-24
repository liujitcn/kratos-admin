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
      :title="dialog.title"
      width="1200px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="100px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #textValue>
        <el-input v-model="formData.value" placeholder="请输入配置值" />
      </template>
      <template #imageValue>
        <UploadImg v-model:image-url="formData.value" upload-type="config" />
      </template>
      <template #richTextValue>
        <WangEditor v-model:value="formData.value" />
      </template>
      <template #dictValue>
        <Dict v-if="dictValueCode" v-model="formData.value" :code="dictValueCode" code-type="string" placeholder="请选择字典配置值" />
        <el-input v-else v-model="formData.value" placeholder="请输入配置值" />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import UploadImg from "@/components/Upload/Img.vue";
import WangEditor from "@/components/WangEditor/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseConfigService } from "@/api/system/base_config";
import type { BaseConfig, BaseConfigForm, PageBaseConfigRequest } from "@/rpc/system/admin/v1/base_config";
import { BaseConfigSite, Status } from "@/rpc/common/v1/enum";
import { BaseConfigType } from "@/rpc/system/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseConfig",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
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

const rules = reactive({
  site: [{ required: true, message: "请选择系统配置位置", trigger: "change" }],
  name: [
    { required: true, message: "请输入系统配置名称", trigger: "blur" },
    { max: 50, message: "配置名称不能超过 50 个字符", trigger: "blur" }
  ],
  type: [{ required: true, message: "请选择系统配置类型", trigger: "change" }],
  key: [
    { required: true, message: "请输入系统配置编码", trigger: "blur" },
    { max: 50, message: "配置key不能超过 50 个字符", trigger: "blur" }
  ],
  value: [{ required: true, message: "配置值不能为空", trigger: "blur" }],
  status: [{ required: true, message: "请选择状态", trigger: "change" }]
});

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

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
    label: "配置名称",
    component: "input",
    props: { placeholder: "请输入配置名称", disabled: formData.id > 0 }
  },
  {
    prop: "site",
    label: "配置位置",
    component: "dict",
    props: { code: "base_config_site", disabled: formData.id > 0 }
  },
  {
    prop: "key",
    label: "配置键",
    component: "input",
    props: { placeholder: "请输入配置键", disabled: formData.id > 0 }
  },
  {
    prop: "type",
    label: "配置类型",
    component: "dict",
    props: { code: "base_config_type", disabled: formData.id > 0 }
  },
  {
    prop: "value",
    label: "配置值",
    component: "slot",
    slotName: "textValue",
    visible: model => model.type == BaseConfigType.TEXT
  },
  {
    prop: "value",
    label: "配置值",
    component: "slot",
    slotName: "imageValue",
    visible: model => model.type == BaseConfigType.IMAGE
  },
  {
    prop: "value",
    label: "配置值",
    component: "slot",
    slotName: "richTextValue",
    visible: model => model.type == BaseConfigType.RICH_TEXT,
    colSpan: 24
  },
  {
    prop: "value",
    label: "配置值",
    component: "slot",
    slotName: "dictValue",
    visible: model => model.type == BaseConfigType.DICT
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
]);

/** 系统配置表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "site", label: "配置位置", minWidth: 120, dictCode: "base_config_site", search: { el: "select" } },
  { prop: "name", label: "配置名称", minWidth: 140, search: { el: "input" } },
  { prop: "type", label: "配置类型", minWidth: 120, dictCode: "base_config_type", search: { el: "select" } },
  { prop: "key", label: "配置键", minWidth: 160, search: { el: "input" } },
  {
    prop: "status",
    label: "状态",
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:config:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseConfig)
    }
  },
  { prop: "created_at", label: "创建时间", minWidth: 180 },
  { prop: "updated_at", label: "更新时间", minWidth: 180 },
  {
    prop: "operation",
    label: "操作",
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:config:update"],
        params: scope => ({ configId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.configId as number | undefined) ?? (scope.row as BaseConfig).id)
      },
      {
        label: "删除",
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:config:delete"],
        onClick: scope => handleDelete(scope.row as BaseConfig)
      }
    ]
  }
];

/** 系统配置顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:config:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:config:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseConfig[])
  },
  {
    label: "刷新缓存",
    type: "primary",
    icon: RefreshLeft,
    hidden: () => !BUTTONS.value["base:config:refresh"],
    onClick: () => handleRefreshCache()
  }
];

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
  dialog.title = configId ? "修改系统配置" : "新增系统配置";
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
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseConfigForm;
    const request = submitData.id
      ? defBaseConfigService.UpdateBaseConfig({ base_config: submitData })
      : defBaseConfigService.CreateBaseConfig({ base_config: submitData });
    request.then(() => {
      ElMessage.success(submitData.id ? "修改系统配置成功" : "新增系统配置成功");
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
    ElMessage.success("刷新成功");
  });
}, 1000);

/**
 * 在系统配置状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseConfig) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const configName = row.name || row.key || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}配置？\n配置名称：${configName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseConfigService.SetBaseConfigStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
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
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = configList.length
    ? configList.length === 1
      ? `是否确定删除配置？\n配置名称：${configList[0].name || configList[0].key || `ID:${configList[0].id}`}`
      : `确认删除已选中的 ${configList.length} 项系统配置吗？`
    : "确认删除已选中的系统配置吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseConfigService.DeleteBaseConfig({ id: configIds }).then(() => {
        ElMessage.success("删除系统配置成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除系统配置");
    }
  );
}
</script>
