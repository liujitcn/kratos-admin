<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="resolvedColumns" :request-api="requestTable" />

    <ProDialog v-model="dialog.visible" :title="config.detailTitle" width="1280px" @close="handleCloseDialog">
      <el-descriptions border :column="2" class="detail-container">
        <el-descriptions-item v-for="field in config.detailFields" :key="field.key" :label="field.label" :span="field.span ?? 1">
          <pre v-if="field.code" class="code-block">{{ formatField(field) }}</pre>
          <span v-else>{{ formatField(field) }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <section v-if="config.trace && (traceLoading || traceItems.length > 0)" class="trace-section">
        <h3>{{ t("system.base.log.trace.title") }}</h3>
        <el-table v-loading="traceLoading" :data="traceItems" max-height="300" border>
          <el-table-column :label="t('system.base.log.trace.log_type')" width="130">
            <template #default="scope">{{ formatLogType(scope.row.log_type) }}</template>
          </el-table-column>
          <el-table-column prop="resource" :label="t('system.base.log.trace.resource')" min-width="280" show-overflow-tooltip />
          <el-table-column prop="user_name" :label="t('system.base.log.field.user_name')" width="130" />
          <el-table-column :label="t('system.base.log.field.result')" width="100">
            <template #default="scope">{{ formatResult(scope.row.result) }}</template>
          </el-table-column>
          <el-table-column :label="t('system.base.log.trace.duration')" width="110" align="right">
            <template #default="scope">{{ formatDuration(scope.row.duration_ms) }}</template>
          </el-table-column>
          <el-table-column prop="occurred_at" :label="t('system.base.log.field.occurred_at')" width="190" />
        </el-table>
      </section>

      <template #footer>
        <el-button @click="handleCloseDialog">{{ config.closeText }}</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, unref } from "vue";
import type { ColumnProps, EnumProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { formatJson } from "@liujitcn/kratos-admin-core/format";
import { t } from "@liujitcn/kratos-admin-core";
import { BaseLogType, BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import type { BaseLogTraceItem } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import { logEnumLabel } from "./log";

/** 审计详情字段配置。 */
export interface LogDetailField {
  /** 详情对象中对应的字段名。 */
  key: string;
  /** 字段在详情弹窗中的显示名称。 */
  label: string;
  /** 字段占用的描述列数，未设置时使用一列。 */
  span?: number;
  /** 是否按格式化 JSON 的代码块展示。 */
  code?: boolean;
  /** 字段对应的枚举字典选项。 */
  enum?: EnumProps[];
  /** 字段自定义格式化函数。 */
  format?: (value: unknown) => string;
}

/** 审计列表页面配置。 */
export interface LogTableConfig {
  /** 审计列表表格的列定义。 */
  columns: ColumnProps[];
  /** 详情弹窗标题。 */
  detailTitle: string;
  /** 详情弹窗关闭按钮文案。 */
  closeText: string;
  /** 详情字段和展示格式定义。 */
  detailFields: LogDetailField[];
  /** 分页查询函数，返回列表和总数。 */
  request: (params: Record<string, unknown>) => Promise<{ list: Record<string, unknown>[]; total: number }>;
  /** 按日志编号读取详情。 */
  get: (id: string) => Promise<Record<string, unknown>>;
  /** 按请求编号和链路编号读取关联审计时间线。 */
  trace?: (requestID: string, traceID: string) => Promise<BaseLogTraceItem[]>;
}

const props = defineProps<{ config: LogTableConfig }>();
const proTable = ref<ProTableInstance>();
const detail = reactive<Record<string, unknown>>({});
const dialog = reactive({ visible: false });
const traceItems = ref<BaseLogTraceItem[]>([]);
const traceLoading = ref(false);
const resolvedColumns = computed(() => unref(props.config.columns));

/** 请求审计日志分页数据。 */
async function requestTable(params: Record<string, unknown>) {
  const result = await props.config.request(buildPageRequest(params));
  return { data: { list: result.list, total: result.total } };
}

/** 打开审计详情弹窗。 */
async function handleOpenDialog(id: string) {
  const value = await props.config.get(id);
  Object.keys(detail).forEach(key => delete detail[key]);
  Object.assign(detail, value);
  dialog.visible = true;
  traceItems.value = [];
  const requestID = String(detail.request_id ?? "");
  const traceID = String(detail.trace_id ?? "");
  if (!props.config.trace || (!requestID && !traceID)) return;
  traceLoading.value = true;
  try {
    traceItems.value = await props.config.trace(requestID, traceID);
  } catch {
    traceItems.value = [];
  } finally {
    traceLoading.value = false;
  }
}

/** 关闭并清理审计详情弹窗。 */
function handleCloseDialog() {
  dialog.visible = false;
  Object.keys(detail).forEach(key => delete detail[key]);
  traceItems.value = [];
  traceLoading.value = false;
}

/** 格式化详情字段，JSON 内容使用统一脱敏展示格式。 */
function formatField(field: LogDetailField) {
  const value = detail[field.key];
  if (value === undefined || value === null || value === "") return t("common.value.none");
  if (field.enum) return logEnumLabel(field.enum, value, t("common.value.unknown"));
  if (field.format) return field.format(value);
  if (field.code) return formatJson(String(value));
  return String(value);
}

/** 格式化关联日志耗时。 */
function formatDuration(value: unknown) {
  const duration = Number(value);
  return duration > 0 ? `${duration} ms` : t("common.value.none");
}

/** 格式化关联审计日志类型。 */
function formatLogType(value: BaseLogType) {
  const keys: Partial<Record<BaseLogType, string>> = {
    [BaseLogType.BASE_LOG_TYPE_LOGIN]: "system.base.log.trace.type.login",
    [BaseLogType.BASE_LOG_TYPE_API]: "system.base.log.trace.type.api",
    [BaseLogType.BASE_LOG_TYPE_OPERATION]: "system.base.log.trace.type.operation",
    [BaseLogType.BASE_LOG_TYPE_DATA_ACCESS]: "system.base.log.trace.type.data_access",
    [BaseLogType.BASE_LOG_TYPE_PERMISSION]: "system.base.log.trace.type.permission",
    [BaseLogType.BASE_LOG_TYPE_POLICY_EVALUATION]: "system.base.log.trace.type.policy"
  };
  return t(keys[value] ?? "system.base.log.trace.type.unknown");
}

/** 格式化关联审计结果。 */
function formatResult(value: BaseLogResult) {
  const keys: Partial<Record<BaseLogResult, string>> = {
    [BaseLogResult.BASE_LOG_RESULT_SUCCESS]: "system.base.log.result.success",
    [BaseLogResult.BASE_LOG_RESULT_FAILURE]: "system.base.log.result.failure",
    [BaseLogResult.BASE_LOG_RESULT_ERROR]: "system.base.log.result.error"
  };
  return t(keys[value] ?? "system.base.log.result.unspecified");
}

defineExpose({ handleOpenDialog, proTable });
</script>

<style scoped>
.detail-container {
  width: 100%;
}

.code-block {
  max-height: 360px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.trace-section {
  margin-top: 20px;
}

.trace-section h3 {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 600;
}
</style>
