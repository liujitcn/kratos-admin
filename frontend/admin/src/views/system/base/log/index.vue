<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseLogTable" />

    <ProDialog v-model="dialog.visible" :title="dialog.title" width="1500px" @close="handleCloseDialog">
      <div class="detail-container">
        <el-descriptions title="基础信息" border :column="2">
          <el-descriptions-item label="操作结果">
            <el-tag :type="detail.is_success ? 'success' : 'danger'" effect="light">
              {{ detail.is_success ? "成功" : "失败" }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态码">
            <el-tag :type="statusCodeColor" effect="light">{{ detail.status_code || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detail.cost_time || "--" }}</el-descriptions-item>
          <el-descriptions-item label="操作时间">{{ detail.request_time || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="请求信息" border :column="2" direction="vertical" class="mt-4 compact-descriptions">
          <el-descriptions-item label="请求ID">{{ detail.request_id || "--" }}</el-descriptions-item>
          <el-descriptions-item label="操作方法">
            <el-tag effect="plain">{{ detail.operation || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求方法">
            <el-tag effect="plain">{{ detail.method || "--" }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="请求路径">{{ detail.path || "--" }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.request_uri" label="请求 URI" :span="2">{{
            detail.request_uri
          }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.referer" label="来源页" :span="2">{{ detail.referer }}</el-descriptions-item>
          <el-descriptions-item label="请求头" :span="2">
            <pre class="code-block">{{ formatPayload(detail.request_header) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求体" :span="2">
            <pre class="code-block">{{ formatPayload(detail.request_body) }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="请求结果" :span="2">
            <pre class="code-block">{{ formatPayload(detail.response) }}</pre>
          </el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="用户信息" border :column="2" class="mt-4">
          <el-descriptions-item label="用户ID">{{ detail.user_id || "--" }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ detail.user_name || "--" }}</el-descriptions-item>
          <el-descriptions-item label="客户端IP">{{ detail.client_ip || "--" }}</el-descriptions-item>
          <el-descriptions-item label="地理位置">{{ detail.location || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="客户端信息" border :column="2" direction="vertical" class="mt-4 compact-descriptions">
          <el-descriptions-item label="浏览器">
            {{ [detail.browser_name, detail.browser_version].filter(Boolean).join(" ") || "--" }}
          </el-descriptions-item>
          <el-descriptions-item label="操作系统">
            {{ [detail.os_name, detail.os_version].filter(Boolean).join(" ") || "--" }}
          </el-descriptions-item>
          <el-descriptions-item label="客户端名称">{{ detail.client_name || "--" }}</el-descriptions-item>
          <el-descriptions-item label="客户端ID">{{ detail.client_id || "--" }}</el-descriptions-item>
          <el-descriptions-item label="User Agent" :span="2">{{ detail.user_agent || "--" }}</el-descriptions-item>
        </el-descriptions>

        <el-alert v-if="!detail.is_success" title="失败原因" type="error" :description="detail.reason" class="mt-4" show-icon />
      </div>

      <template #footer>
        <el-button @click="handleCloseDialog">关闭</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { InfoFilled } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { defBaseLogService } from "@/api/system/base_log";
import { buildPageRequest } from "@/utils/proTable";
import { formatJson } from "@/utils/utils";
import type { BaseLog, PageBaseLogRequest } from "@/rpc/system/admin/v1/base_log";

defineOptions({
  name: "BaseLog",
  inheritAttrs: false
});

const proTable = ref<ProTableInstance>();

const dialog = reactive({
  title: "",
  visible: false
});

/** 状态码颜色计算。 */
const statusCodeColor = computed(() => {
  const code = detail.status_code;
  if (code >= 200 && code < 300) return "success";
  if (code >= 400 && code < 500) return "warning";
  if (code >= 500) return "danger";
  return "info";
});

/** 创建默认日志详情，避免多次查看时残留上一条数据。 */
function createDefaultDetail(): BaseLog {
  return {
    /** 日志ID */
    id: 0,
    /** 请求ID */
    request_id: "",
    /** 请求时间 */
    request_time: "",
    /** 请求方法 */
    method: "",
    /** 操作方法 */
    operation: "",
    /** 请求路径 */
    path: "",
    /** 请求源 */
    referer: "",
    /** 请求URI */
    request_uri: "",
    /** 请求头 */
    request_header: "",
    /** 请求体 */
    request_body: "",
    /** 响应信息 */
    response: "",
    /** 操作耗时 */
    cost_time: "",
    /** 操作是否成功 */
    is_success: false,
    /** 状态码 */
    status_code: 0,
    /** 操作失败原因 */
    reason: "",
    /** 操作地理位置 */
    location: "",
    /** 操作者用户ID */
    user_id: 0,
    /** 操作者账号名 */
    user_name: "",
    /** 操作者IP */
    client_ip: "",
    /** 浏览器的用户代理信息 */
    user_agent: "",
    /** 浏览器名称 */
    browser_name: "",
    /** 浏览器版本 */
    browser_version: "",
    /** 客户端ID */
    client_id: "",
    /** 客户端名称 */
    client_name: "",
    /** 操作系统名称 */
    os_name: "",
    /** 操作系统版本 */
    os_version: ""
  };
}

const detail = reactive<BaseLog>(createDefaultDetail());

function normalizeBoolean(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    return normalized === "true" || normalized === "1" || normalized === "success";
  }
  return false;
}

function normalizeNumber(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function normalizeDetail(data: BaseLog): BaseLog {
  return {
    ...data,
    is_success: normalizeBoolean((data as BaseLog & { is_success?: unknown }).is_success),
    status_code: normalizeNumber(data.status_code)
  };
}

function formatPayload(value: string): string {
  return value ? formatJson(value) : "--";
}

/** 日志表格列配置。 */
const columns: ColumnProps[] = [
  {
    prop: "operation",
    label: "操作方法",
    minWidth: 140,
    search: { el: "input" }
  },
  {
    prop: "status_code",
    label: "状态码",
    minWidth: 100,
    search: { el: "input-number", props: { min: 0, controlsPosition: "right" } }
  },
  {
    prop: "request_time",
    label: "操作时间",
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
  { prop: "user_name", label: "操作人", minWidth: 100 },
  { prop: "client_ip", label: "IP 地址", minWidth: 130 },
  { prop: "location", label: "地区", minWidth: 120 },
  { prop: "browser_name", label: "浏览器", minWidth: 120 },
  { prop: "os_name", label: "终端系统", minWidth: 200 },
  { prop: "cost_time", label: "执行时间(ms)", minWidth: 130, align: "right" },
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
        onClick: scope => handleOpenDialog((scope.row as BaseLog).id)
      }
    ]
  }
];

/**
 * 请求系统日志列表，并通过 buildPageRequest 统一处理分页参数。
 */
async function requestBaseLogTable(params: PageBaseLogRequest) {
  const data = await defBaseLogService.PageBaseLog(buildPageRequest(params));
  return { data: { list: data.base_logs ?? [], total: data.total } };
}

/**
 * 打开系统日志详情弹窗。
 */
function handleOpenDialog(logId?: number) {
  resetDetail();
  dialog.title = "系统日志详情";
  dialog.visible = true;
  if (!logId) return;

  defBaseLogService.GetBaseLog({ id: logId }).then(data => {
    Object.assign(detail, normalizeDetail(data));
  });
}

/**
 * 关闭系统日志弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetDetail();
}

/**
 * 重置日志详情，避免关闭后保留旧数据。
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
  max-height: 240px;
  padding: 12px;
  margin: 0;
  overflow: auto;
  line-height: 1.6;
  word-break: break-all;
  white-space: pre-wrap;
  background: #f5f7fa;
  border-radius: 4px;
}
:deep(.compact-descriptions .el-descriptions__label) {
  font-weight: 600;
}
:deep(.compact-descriptions .el-descriptions__content) {
  vertical-align: top;
}
</style>
