<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseJobLogTable" />

    <ProDialog v-model="dialog.visible" :title="t('system.jobLog.title.detail')" width="1200px" @close="handleCloseDialog">
      <div class="detail-container">
        <el-descriptions :title="t('system.jobLog.section.basic')" border :column="2">
          <el-descriptions-item :label="t('system.common.field.status')">
            <DictLabel v-model="detail.status" code="base_job_log_status" />
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.jobLog.field.processTime')">{{ detail.process_time }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.jobLog.field.executeTime')">{{ detail.execute_time }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :title="t('system.jobLog.section.execution')" border :column="1" class="mt-4">
          <el-descriptions-item :label="t('system.jobLog.field.input')">
            <pre class="code-block">{{ formatJson(detail.input) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.jobLog.field.output')">
            <pre class="code-block">{{ formatJson(detail.output) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="detail.status === BaseJobLogStatus.FAIL"
          :title="t('system.jobLog.field.error')"
          type="error"
          :description="detail.error"
          class="mt-4"
          show-icon
        />
      </div>

      <template #footer>
        <el-button @click="handleCloseDialog">{{ t("common.action.close") }}</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { InfoFilled } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { defBaseJobService } from "@liujitcn/kratos-admin-system/api/system/base_job";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { formatJson } from "@liujitcn/kratos-admin-core/format";
import type { BaseJobLog, PageBaseJobLogRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_job";
import { BaseJobLogStatus } from "@liujitcn/kratos-admin-system/rpc/system/common/v1/enum";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseJobLog",
  inheritAttrs: false
});

const route = useRoute();
const proTable = ref<ProTableInstance>();
const jobId = ref(Number(route.query.jobId ?? 0));

const dialog = reactive({
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
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "status",
    label: t("system.common.field.status"),
    minWidth: 120,
    dictCode: "base_job_log_status",
    search: { el: "select" }
  },
  {
    prop: "execute_time",
    label: t("system.jobLog.field.executeTime"),
    minWidth: 180,
    search: {
      el: "date-picker",
      props: {
        type: "daterange",
        editable: false,
        class: "!w-[240px]",
        rangeSeparator: "~",
        startPlaceholder: t("system.common.placeholder.startDate"),
        endPlaceholder: t("system.common.placeholder.endDate"),
        valueFormat: "YYYY-MM-DD"
      }
    }
  },
  { prop: "process_time", label: t("system.jobLog.field.processTimeMs"), minWidth: 130, align: "right" },
  {
    prop: "detailAction",
    label: t("system.common.field.action"),
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: InfoFilled,
        onClick: scope => handleOpenDialog((scope.row as BaseJobLog).id)
      }
    ]
  }
]);

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
