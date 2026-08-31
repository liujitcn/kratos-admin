<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :request-api="requestBaseFileTable"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseFileService } from "@liujitcn/kratos-admin-system/api/system/base_file";
import type { BaseFile, PageBaseFileRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_file";

defineOptions({ name: "BaseFile", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();

const columns = computed<ColumnProps[]>(() => [
  { prop: "file_name", label: t("system.base.file.field.name"), minWidth: 220, search: { el: "input" } },
  { prop: "extension", label: t("system.base.file.field.extension"), width: 100, search: { el: "input" } },
  { prop: "mime_type", label: t("system.base.file.field.mime"), minWidth: 180 },
  { prop: "size", label: t("system.base.file.field.size"), width: 120, render: scope => formatFileSize((scope.row as BaseFile).size) },
  { prop: "content_hash", label: t("system.base.file.field.hash"), minWidth: 220 },
  { prop: "tenant_id", label: t("system.base.file.field.tenant"), width: 100 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:file:info"],
        onClick: scope => handleView(scope.row as BaseFile)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:file:delete"],
        onClick: scope => handleDelete(scope.row as BaseFile)
      }
    ]
  }
]);

/** 请求文件资产分页列表。 */
async function requestBaseFileTable(params: PageBaseFileRequest) {
  const data = await defBaseFileService.PageBaseFile(buildPageRequest(params));
  return { data: { list: data.base_files ?? [], total: data.total } };
}

/** 查看文件资产详情。 */
async function handleView(row: BaseFile) {
  const detail = await defBaseFileService.GetBaseFile({ id: row.id });
  const lines = [
    `${t("system.base.file.field.name")}：${detail.file_name}`,
    `${t("system.base.file.field.path")}：${detail.file_directory}/${detail.save_file_name}`,
    `${t("system.base.file.field.mime")}：${detail.mime_type}`,
    `${t("system.base.file.field.size")}：${formatFileSize(detail.size)}`,
    `${t("system.base.file.field.hash")}：${detail.content_hash}`,
    `${t("system.base.file.field.url")}：${detail.link_url}`
  ];
  await ElMessageBox.alert(h("pre", { class: "file-detail" }, lines.join("\n")), t("system.base.file.detail"), { dangerouslyUseHTMLString: false });
}

/** 删除文件资产。 */
async function handleDelete(row: BaseFile) {
  try {
    await ElMessageBox.confirm(
      t("system.base.file.confirm_delete", { name: row.file_name }),
      t("common.title.warning"),
      { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
    );
    await defBaseFileService.DeleteBaseFile({ id: row.id });
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.file.resource") }));
    proTable.value?.getTableList();
  } catch {
    // 用户取消确认或接口失败时由请求层展示错误。
  }
}

function formatFileSize(size: number) {
  if (!size) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = size;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 2)} ${units[index]}`;
}
</script>

<style scoped>
.file-detail {
  max-width: 720px;
  margin: 0;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  text-align: left;
}
</style>
