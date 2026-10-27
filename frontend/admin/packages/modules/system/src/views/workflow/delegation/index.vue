<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestDelegationTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t('system.workflow.delegation.title.set')"
      width="480px"
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
import { CirclePlus, Delete } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defWorkflowDelegationService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_delegation";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import type {
  PageWorkflowDelegationRequest,
  WorkflowDelegation
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_delegation";

defineOptions({
  name: "WorkflowDelegation",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({ visible: false });
const userOptions = ref<ProFormOption[]>([]);
const formData = reactive({
  /** 被委托人用户ID。 */
  delegate_user_id: undefined as number | undefined,
  /** 备注。 */
  remark: ""
});
const rules = computed(() => ({
  delegate_user_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.workflow.delegation.field.delegate") }),
      trigger: "change"
    }
  ],
  remark: [
    { max: 500, message: t("common.validation.max_length", { field: t("common.field.remark"), max: 500 }), trigger: "blur" }
  ]
}));

onMounted(() => {
  void loadUserOptions();
});

/** 预加载用户下拉。 */
async function loadUserOptions() {
  const data = await defBaseUserService.OptionBaseUser({ keyword: "" });
  userOptions.value = data.list ?? [];
}

/** 表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "delegate_user_id",
    label: t("system.workflow.delegation.field.delegate"),
    component: "select",
    props: { filterable: true, placeholder: t("system.workflow.delegation.placeholder.delegate") },
    options: userOptions.value
  },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { placeholder: t("common.placeholder.remark") }
  }
]);

/** 表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { prop: "delegator_name", label: t("system.workflow.delegation.field.delegator"), minWidth: 140 },
  { prop: "delegate_name", label: t("system.workflow.delegation.field.delegate"), minWidth: 140 },
  { prop: "remark", label: t("common.field.remark"), minWidth: 180 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["workflow:delegation:delete"],
        onClick: scope => handleDelete(scope.row as WorkflowDelegation)
      }
    ]
  }
]);

/** 顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("system.workflow.delegation.action.set"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["workflow:delegation:set"],
    onClick: () => handleOpenDialog()
  }
]);

/** 请求审批委托分页列表。 */
async function requestDelegationTable(params: PageWorkflowDelegationRequest) {
  const data = await defWorkflowDelegationService.PageWorkflowDelegation(buildPageRequest(params));
  return { data: { list: data.delegations ?? [], total: data.total } };
}

/** 打开设置委托弹窗。 */
async function handleOpenDialog() {
  resetForm();
  await formDialogRef.value?.open({ load: async () => undefined, commit: () => undefined });
}

/** 关闭弹窗并清理。 */
function handleCloseDialog() {
  formDialogRef.value?.close();
  resetForm();
}

/** 重置表单。 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.delegate_user_id = undefined;
  formData.remark = "";
}

/** 提交委托设置。 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    defWorkflowDelegationService
      .SetWorkflowDelegation({
        delegation: {
          delegate_user_id: formData.delegate_user_id ?? 0,
          remark: formData.remark
        }
      })
      .then(() => {
        ElMessage.success(t("system.workflow.delegation.message.set_success"));
        handleCloseDialog();
        proTable.value?.getTableList();
      });
  });
}

/** 删除审批委托。 */
function handleDelete(selected?: number | string | Array<number | string> | WorkflowDelegation | WorkflowDelegation[]) {
  const list = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as WorkflowDelegation[])
    : selected && typeof selected === "object"
      ? [selected as WorkflowDelegation]
      : [];
  const ids = (
    list.length ? list.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!ids) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  ElMessageBox.confirm(
    list.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("system.workflow.delegation.resource") })}\n${t("common.dialog.resource_field", {
          field: t("system.workflow.delegation.field.delegator"),
          value: list[0].delegator_name || `ID:${list[0].id}`
        })}`
      : t("common.dialog.delete_batch", { count: list.length, unit: "", resource: t("system.workflow.delegation.resource") }),
    t("common.title.warning"),
    { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
  ).then(
    () =>
      defWorkflowDelegationService.DeleteWorkflowDelegation({ id: ids }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.workflow.delegation.resource") }));
        proTable.value?.getTableList();
      }),
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.workflow.delegation.resource") }))
  );
}
</script>
