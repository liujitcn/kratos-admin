<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestDefinitionTable"
    />

    <WorkflowDefinitionDialog ref="definitionDialogRef" @saved="refreshTable" />
    <WorkflowDefinitionView ref="definitionViewRef" />
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref } from "vue";
import { ElMessage, ElMessageBox, ElSwitch } from "element-plus";
import { CirclePlus, Delete, EditPen, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import type {
  PageWorkflowDefinitionRequest,
  WorkflowDefinition
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_definition";
import { WorkflowDefinitionStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/workflow_common";
import WorkflowDefinitionDialog from "./definition-dialog.vue";
import WorkflowDefinitionView from "./definition-view.vue";

defineOptions({
  name: "WorkflowDefinition",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const definitionDialogRef = ref<InstanceType<typeof WorkflowDefinitionDialog>>();
const definitionViewRef = ref<InstanceType<typeof WorkflowDefinitionView>>();

/** 流程定义表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("system.workflow.definition.field.name"), minWidth: 160, search: { el: "input" } },
  { prop: "code", label: t("system.workflow.definition.field.code"), minWidth: 140, search: { el: "input" } },
  { prop: "version", label: t("system.workflow.definition.field.version"), minWidth: 90, align: "center" },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 110,
    align: "center",
    search: {
      el: "select",
      options: [
        { label: t("system.workflow.definition.status.draft"), value: WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DRAFT },
        {
          label: t("system.workflow.definition.status.enabled"),
          value: WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_ENABLED
        },
        {
          label: t("system.workflow.definition.status.disabled"),
          value: WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DISABLED
        }
      ]
    },
    render: scope => {
      const row = scope.row as WorkflowDefinition;
      const enabled = row.status === WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_ENABLED;
      return h(ElSwitch, {
        modelValue: enabled,
        inlinePrompt: true,
        activeValue: true,
        inactiveValue: false,
        activeText: t("system.workflow.definition.status.enabled"),
        inactiveText: t(
          row.status === WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DRAFT
            ? "system.workflow.definition.status.draft"
            : "system.workflow.definition.status.disabled"
        ),
        disabled: !BUTTONS.value["workflow:definition:status"],
        beforeChange: () => handleToggleStatus(row)
      });
    }
  },
  { prop: "remark", label: t("common.field.remark"), minWidth: 160 },
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
        hidden: () => !BUTTONS.value["workflow:definition:update"],
        disabled: scope => (scope.row as WorkflowDefinition).status !== WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DRAFT,
        onClick: scope => handleOpenDialog((scope.row as WorkflowDefinition).id)
      },
      {
        label: t("system.workflow.definition.action.view"),
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenView((scope.row as WorkflowDefinition).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["workflow:definition:delete"],
        disabled: scope => (scope.row as WorkflowDefinition).status !== WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DRAFT,
        onClick: scope => handleDelete(scope.row as WorkflowDefinition)
      }
    ]
  }
]);

/** 顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["workflow:definition:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["workflow:definition:delete"],
    disabled: scope =>
      !scope.selectedList.length ||
      (scope.selectedList as WorkflowDefinition[]).some(
        item => item.status !== WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DRAFT
      ),
    onClick: scope => handleDelete(scope.selectedList as WorkflowDefinition[])
  }
]);

/** 请求流程定义分页列表。 */
async function requestDefinitionTable(params: PageWorkflowDefinitionRequest) {
  const data = await defWorkflowDefinitionService.PageWorkflowDefinition(params);
  return { data: { list: data.definitions ?? [], total: data.total } };
}

/** 刷新表格。 */
function refreshTable() {
  proTable.value?.getTableList();
}

/** 打开流程定义编辑器，id 为空表示新增。 */
async function handleOpenDialog(id?: number) {
  await definitionDialogRef.value?.open(id);
}

/** 打开流程定义只读查看器。 */
async function handleOpenView(id: number) {
  await definitionViewRef.value?.open(id);
}

/** 启停流程定义：草稿/停用转启用，启用转停用。 */
async function handleToggleStatus(row: WorkflowDefinition): Promise<boolean> {
  const next =
    row.status === WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_ENABLED
      ? WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_DISABLED
      : WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_ENABLED;
  const statusLabel = t(
    next === WorkflowDefinitionStatus.WORKFLOW_DEFINITION_STATUS_ENABLED
      ? "system.workflow.definition.status.enabled"
      : "system.workflow.definition.status.disabled"
  );
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: statusLabel,
        resource: t("system.workflow.definition.resource"),
        field: t("system.workflow.definition.field.name"),
        value: row.name || row.code || `ID:${row.id}`
      }),
      t("common.title.notice"),
      { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
    );
    await defWorkflowDefinitionService.SetWorkflowDefinitionStatus({ id: row.id, status: next });
    ElMessage.success(t("common.message.status_success", { action: statusLabel }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/** 删除流程定义，仅草稿状态可用。 */
function handleDelete(selected?: number | string | Array<number | string> | WorkflowDefinition | WorkflowDefinition[]) {
  const list = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as WorkflowDefinition[])
    : selected && typeof selected === "object"
      ? [selected as WorkflowDefinition]
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
      ? `${t("common.dialog.delete_single", { resource: t("system.workflow.definition.resource") })}\n${t("common.dialog.resource_field", { field: t("system.workflow.definition.field.name"), value: list[0].name || list[0].code || `ID:${list[0].id}` })}`
      : t("common.dialog.delete_batch", { count: list.length, unit: "", resource: t("system.workflow.definition.resource") }),
    t("common.title.warning"),
    { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
  ).then(
    () =>
      defWorkflowDefinitionService.DeleteWorkflowDefinition({ id: ids }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t("system.workflow.definition.resource") }));
        refreshTable();
      }),
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("system.workflow.definition.resource") }))
  );
}
</script>
