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

      <template #footer>
        <el-button @click="handleCloseDialog">{{ config.closeText }}</el-button>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, unref } from "vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { formatJson } from "@liujitcn/kratos-admin-core/format";

/** 审计详情字段配置。 */
export interface AuditLogDetailField {
  /** 详情对象中对应的字段名。 */
  key: string;
  /** 字段在详情弹窗中的显示名称。 */
  label: string;
  /** 字段占用的描述列数，未设置时使用一列。 */
  span?: number;
  /** 是否按格式化 JSON 的代码块展示。 */
  code?: boolean;
}

/** 审计列表页面配置。 */
export interface AuditLogTableConfig {
  /** 审计列表表格的列定义。 */
  columns: ColumnProps[];
  /** 详情弹窗标题。 */
  detailTitle: string;
  /** 详情弹窗关闭按钮文案。 */
  closeText: string;
  /** 详情字段和展示格式定义。 */
  detailFields: AuditLogDetailField[];
  /** 分页查询函数，返回列表和总数。 */
  request: (params: Record<string, unknown>) => Promise<{ list: Record<string, unknown>[]; total: number }>;
  /** 按日志编号读取详情。 */
  get: (id: number) => Promise<Record<string, unknown>>;
}

const props = defineProps<{ config: AuditLogTableConfig }>();
const proTable = ref<ProTableInstance>();
const detail = reactive<Record<string, unknown>>({});
const dialog = reactive({ visible: false });
const resolvedColumns = computed(() => unref(props.config.columns));

/** 请求审计日志分页数据。 */
async function requestTable(params: Record<string, unknown>) {
  const result = await props.config.request(buildPageRequest(params));
  return { data: { list: result.list, total: result.total } };
}

/** 打开审计详情弹窗。 */
async function handleOpenDialog(id: number) {
  const value = await props.config.get(id);
  Object.keys(detail).forEach(key => delete detail[key]);
  Object.assign(detail, value);
  dialog.visible = true;
}

/** 关闭并清理审计详情弹窗。 */
function handleCloseDialog() {
  dialog.visible = false;
  Object.keys(detail).forEach(key => delete detail[key]);
}

/** 格式化详情字段，JSON 内容使用统一脱敏展示格式。 */
function formatField(field: AuditLogDetailField) {
  const value = detail[field.key];
  if (value === undefined || value === null || value === "") return "--";
  if (field.code) return formatJson(String(value));
  return String(value);
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
</style>
