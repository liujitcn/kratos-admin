<template>
  <div v-if="active" class="log-example">
    <LogTable ref="table" :config="config" />
  </div>
</template>

<script setup lang="ts">
import { onActivated, onDeactivated, ref } from "vue";
import LogTable, { type LogTableConfig } from "@liujitcn/kratos-admin-system/components/log/LogTable.vue";

defineProps<{ componentId: string }>();
const active = ref(true);
const table = ref<InstanceType<typeof LogTable>>();
const rows = [
  {
    id: "log-1",
    user_name: "张明",
    resource: "经营日报",
    action: "查看",
    occurred_at: "2026-11-13 09:20:00",
    payload: '{"report":"daily","result":"success"}'
  },
  {
    id: "log-2",
    user_name: "李华",
    resource: "服务概览",
    action: "导出",
    occurred_at: "2026-11-13 10:05:00",
    payload: '{"report":"overview","result":"success"}'
  },
  {
    id: "log-3",
    user_name: "王芳",
    resource: "数据卡片",
    action: "查看",
    occurred_at: "2026-11-13 11:10:00",
    payload: '{"report":"card","result":"success"}'
  }
];
const config: LogTableConfig = {
  columns: [
    { prop: "user_name", label: "操作人", minWidth: 100, search: { el: "input" } },
    { prop: "resource", label: "资源", minWidth: 140 },
    { prop: "action", label: "操作", width: 100 },
    { prop: "occurred_at", label: "操作时间", minWidth: 190 },
    {
      prop: "operation",
      label: "操作",
      width: 90,
      actions: [{ label: "详情", type: "primary", onClick: ({ row }) => table.value?.handleOpenDialog(String(row.id)) }]
    }
  ],
  detailTitle: "本地日志详情",
  closeText: "关闭",
  detailFields: [
    { key: "id", label: "日志编号" },
    { key: "user_name", label: "操作人" },
    { key: "resource", label: "资源" },
    { key: "action", label: "操作" },
    { key: "payload", label: "内容", code: true, span: 2 }
  ],
  request: async params => {
    const filtered = rows.filter(row => row.user_name.includes(String(params.user_name ?? "")));
    const page = Number(params.page_num ?? 1);
    const size = Number(params.page_size ?? 10);
    return { list: filtered.slice((page - 1) * size, page * size), total: filtered.length };
  },
  get: async id => {
    const row = rows.find(item => item.id === id);
    if (!row) throw new Error("日志不存在");
    return row;
  }
};

onActivated(() => {
  active.value = true;
});
onDeactivated(() => {
  active.value = false;
});
</script>

<style scoped>
.log-example {
  min-height: 370px;
}
.log-example :deep(.table-box) {
  padding: 0;
}
</style>
