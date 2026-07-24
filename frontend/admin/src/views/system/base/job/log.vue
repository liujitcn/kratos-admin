<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseJobLogTable" />

    <ProDialog v-model="dialog.visible" :title="dialog.title" width="1200px" @close="handleCloseDialog">
      <div class="detail-container">
        <el-descriptions title="基础信息" border :column="2">
          <el-descriptions-item label="状态">
            <DictLabel v-model="detail.status" code="base_job_log_status" />
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.process_time }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.execute_time }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="请求信息" border :column="1" class="mt-4">
          <el-descriptions-item label="执行参数">
            <pre class="code-block">{{ formatJson(detail.input) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="输出结果">
            <pre class="code-block">{{ formatJson(detail.output) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="detail.status === BaseJobLogStatus.FAIL"
          title="失败原因"
          type="error"
          :description="detail.error"
          class="mt-4"
          show-icon
        />
      </div>

      <template #footer>
        <el-button @click="handleCloseDialog">关闭</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { InfoFilled } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defBaseJobService } from "@/api/system/base_job";
import { buildPageRequest } from "@/utils/proTable";
import { formatJson } from "@/utils/utils";
import type { BaseJobLog, PageBaseJobLogRequest } from "@/rpc/system/admin/v1/base_job";
import { BaseJobLogStatus } from "@/rpc/system/common/v1/enum";

defineOptions({
  name: "BaseJobLog",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const jobId = ref(Number(route.query.jobId ?? 0));

const dialog = reactive({
  title: "",
  visible: false
});

/** 创建默认任务日志详情，避免弹窗切换时残留上一条记录。 */
function createDefaultDetail(): BaseJobLog {
  return {
    /** 任务日志ID */
    id: 0,
    /** 任务ID */
    job_id: 0,
    /** 执行参数 */
    input: "",
    /** 输出结果 */
    output: "",
    /** 错误信息 */
    error: "",
    /** 状态 */
    status: BaseJobLogStatus.UNKNOWN_BJLS,
    /** 消耗时间 */
    process_time: "",
    /** 执行时间 */
    execute_time: ""
  };
}

const detail = reactive<BaseJobLog>(createDefaultDetail());

/** 定时任务日志表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "status",
    label: "状态",
    minWidth: 120,
    dictCode: "base_job_log_status",
    search: { el: "select" }
  },
  {
    prop: "execute_time",
    label: "执行时间",
    minWidth: 180,
    search: {
      el: "date-picker",
      props: {
        type: "daterange",
        editable: false,
        class: "!w-[240px]",
        rangeSeparator: "~",
        startPlaceholder: "开始时间",
        endPlaceholder: "截止时间",
        valueFormat: "YYYY-MM-DD"
      }
    }
  },
  { prop: "process_time", label: "消耗时间(ms)", minWidth: 130, align: "right" },
  {
    prop: "detailAction",
    label: "操作",
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: InfoFilled,
        onClick: scope => handleOpenDialog((scope.row as BaseJobLog).id)
      }
    ]
  }
];

watch(
  () => route.query.jobId,
  value => {
    jobId.value = Number(value ?? 0);
    proTable.value?.search();
  }
);

/**
 * 请求定时任务日志列表，并补充当前任务 ID。
 */
async function requestBaseJobLogTable(params: PageBaseJobLogRequest) {
  const data = await defBaseJobService.PageBaseJobLog({ ...buildPageRequest(params), job_id: jobId.value });
  return { data: { list: data.base_job_logs ?? [], total: data.total } };
}

/**
 * 打开定时任务日志详情弹窗。
 */
function handleOpenDialog(logId?: number) {
  resetDetail();
  dialog.title = "定时任务日志详情";
  dialog.visible = true;
  if (!logId) return;

  defBaseJobService.GetBaseJobLog({ id: logId }).then(data => {
    Object.assign(detail, data);
  });
}

/**
 * 关闭定时任务日志详情弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetDetail();
}

/**
 * 重置任务日志详情，避免关闭后残留旧数据。
 */
function resetDetail() {
  Object.assign(detail, createDefaultDetail());
}
</script>

<style scoped>
.detail-container {
  max-height: 70vh;
  padding: 20px;
  overflow-y: auto;
  background: #ffffff;
  border-radius: 4px;
}
.mt-4 {
  margin-top: 16px;
}
.code-block {
  max-height: 200px;
  padding: 12px;
  margin: 0;
  overflow: auto;
  background: #f5f7fa;
  border-radius: 4px;
}
</style>
