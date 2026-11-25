<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestAiShortcutTable" />
    <ShortcutDialog ref="dialog" @saved="refreshTable" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { defAiShortcutService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_shortcut";
import type { AiShortcutForm, PageAiShortcutRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_shortcut";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import ShortcutDialog from "./shortcut-dialog.vue";
import { sources, terminals } from "./shortcut-options";

defineOptions({ name: "AiShortcut", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialog = ref<InstanceType<typeof ShortcutDialog>>();

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "title", label: t("system.ai.shortcut.field.title"), minWidth: 155, search: { el: "input" } },
  { prop: "shortcut_key", label: t("system.ai.shortcut.field.key"), minWidth: 145, search: { el: "input" } },
  { prop: "source_type", label: t("system.ai.shortcut.field.source"), minWidth: 90, enum: sources(), search: { el: "select" } },
  { prop: "terminal", label: t("system.ai.shortcut.field.terminal"), minWidth: 90, enum: terminals(), search: { el: "select" } },
  {
    prop: "tool_name",
    label: t("system.base.api.field.tool_name"),
    minWidth: 200,
    showOverflowTooltip: true,
    search: { el: "input" }
  },
  { prop: "group_name", label: t("system.ai.shortcut.field.group"), minWidth: 90, showOverflowTooltip: true },
  { prop: "sort", label: t("common.field.sort"), width: 78, align: "center" },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    align: "center",
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["ai:shortcut:status"],
      beforeChange: ({ row }) => toggleStatus(row as AiShortcutForm),
      onChange: refreshTable
    }
  },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 170, align: "center" },
  {
    prop: "operation",
    label: t("common.field.action"),
    cellType: "actions",
    align: "center",
    fixed: "right",
    width: 205,
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["ai:shortcut:update"],
        onClick: ({ row }) => dialog.value?.open(row.id)
      },
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: View,
        onClick: ({ row }) => dialog.value?.open(row.id, true)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["ai:shortcut:delete"],
        onClick: ({ row }) => handleDelete([row as AiShortcutForm])
      }
    ]
  }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["ai:shortcut:create"],
    onClick: () => dialog.value?.open()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["ai:shortcut:delete"],
    disabled: ({ isSelected }) => !isSelected,
    onClick: ({ selectedList }) => handleDelete(selectedList as AiShortcutForm[])
  }
]);

/** 查询快捷入口分页并适配标准表格。 */
async function requestAiShortcutTable(params: PageAiShortcutRequest) {
  const result = await defAiShortcutService.PageAiShortcut(buildPageRequest(params));
  return { data: { list: result.shortcuts ?? [], total: result.total } };
}

/** 重新读取列表中的状态和修订号。 */
function refreshTable() {
  void table.value?.getTableList();
}

/** 按当前修订提交启停操作。 */
async function toggleStatus(row: AiShortcutForm) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(status === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        resource: t("system.ai.shortcut.resource"),
        action,
        field: t("system.ai.shortcut.field.title"),
        value: row.title
      }),
      t("common.title.notice"),
      { type: "warning" }
    );
    await defAiShortcutService.SetAiShortcutStatus({ id: row.id, status, revision: row.revision });
    return true;
  } catch {
    return false;
  }
}

/** 确认后提交整批删除并刷新列表。 */
async function handleDelete(rows: AiShortcutForm[]) {
  if (!rows.length) return;
  const name = rows.map(row => row.title).join("、");
  try {
    await ElMessageBox.confirm(t("system.ai.shortcut.confirm.delete", { name }), t("common.title.notice"), { type: "warning" });
  } catch {
    return;
  }
  await defAiShortcutService.DeleteAiShortcut({ id: rows.map(row => row.id).join(",") });
  ElMessage.success(t("common.message.delete_success", { resource: t("system.ai.shortcut.resource") }));
  refreshTable();
}
</script>
