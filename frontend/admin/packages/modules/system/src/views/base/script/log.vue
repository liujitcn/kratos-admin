<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseScriptLogTable" />
    <ProDialog ref="dialogRef" v-model="dialog.visible" :title="t('system.base.script.log.title.detail')" width="900px">
      <el-descriptions border :column="2">
        <el-descriptions-item :label="t('system.base.script.log.field.script')">{{ detail.script_name }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.log.field.trigger')">{{ detail.trigger_type }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.log.field.version')">{{ detail.version }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.log.field.duration')">{{ detail.duration_ms }}ms</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.log.field.request_id')">{{ detail.request_id || "-" }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.script.log.field.trace_id')">{{ detail.trace_id || "-" }}</el-descriptions-item>
      </el-descriptions>
      <el-alert v-if="!detail.success" class="mt-4" type="error" :title="t('system.base.script.test.failed')" :description="detail.error" show-icon />
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch, computed } from "vue";
import { useRoute } from "vue-router";
import { InfoFilled } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseScriptLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_script_log";
import type { BaseScriptLog, PageBaseScriptLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script_log";

defineOptions({ name: "BaseScriptLog", inheritAttrs: false });

const route = useRoute();
const proTable = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof ProDialog>>();
const initialScriptId = Number(route.query.scriptId ?? 0);
const dialog = reactive({ visible: false });
const detail = reactive<BaseScriptLog>({ id: 0, script_id: 0, script_name: "", language: "", trigger_type: "", hook_point: "", version: 0, success: false, duration_ms: 0, error: "", request_id: "", trace_id: "", created_at: "" });

const columns = computed<ColumnProps[]>(() => [
  { prop: "script_name", label: t("system.base.script.log.field.script"), minWidth: 180, search: { el: "input" } },
  { prop: "language", label: t("system.base.script.field.language"), minWidth: 110 },
  { prop: "trigger_type", label: t("system.base.script.log.field.trigger"), minWidth: 120, search: { el: "select" } },
  { prop: "success", label: t("common.field.status"), minWidth: 100, cellType: "status", statusProps: { activeValue: true, inactiveValue: false, activeText: t("system.base.script.test.success"), inactiveText: t("system.base.script.test.failed") } },
  { prop: "duration_ms", label: t("system.base.script.log.field.duration"), minWidth: 100, align: "right" },
  { prop: "created_at", label: t("system.base.script.log.field.created_at"), minWidth: 180, align: "center" },
  { prop: "action", label: t("common.field.action"), cellType: "actions", actions: [{ label: t("common.action.view"), type: "primary", link: true, icon: InfoFilled, onClick: scope => handleOpenDialog((scope.row as BaseScriptLog).id) }] }
]);

async function requestBaseScriptLogTable(params: PageBaseScriptLogRequest) {
  const data = await defBaseScriptLogService.PageBaseScriptLog(buildPageRequest(params));
  return { data: { list: data.base_script_logs ?? [], total: data.total } };
}

async function handleOpenDialog(id: number) {
  const data = await defBaseScriptLogService.GetBaseScriptLog({ id });
  Object.assign(detail, data);
  dialog.visible = true;
}

watch(
  () => route.query.scriptId,
  value => {
    const nextId = Number(value ?? 0);
    if (!proTable.value) return;
    proTable.value.searchParam.script_id = nextId || undefined;
    proTable.value.searchInitParam.script_id = nextId || undefined;
    proTable.value.search();
  }
);
</script>
