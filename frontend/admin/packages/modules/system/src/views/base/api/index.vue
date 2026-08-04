<!-- API 管理 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseApiTable" />

    <el-drawer v-model="detailDrawer.visible" :title="t('system.base.api.title.detail')" size="70%" @close="handleCloseDetail">
      <el-descriptions v-if="detailData" :column="1" border>
        <el-descriptions-item :label="t('system.base.api.field.tool_name')">{{ detailData.tool_name }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.tool_prompts')">
          <div class="tool-prompts">
            <el-tag v-for="prompt in detailToolPrompts" :key="prompt" effect="plain">{{ prompt }}</el-tag>
            <span v-if="!detailToolPrompts.length">--</span>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.service_name')">{{ detailData.service_name }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.service_description')">{{
          detailData.service_desc
        }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.description')">{{ detailData.desc }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.operation')">{{ detailData.operation }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.method')">{{ detailData.method }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.path')">{{ detailData.path }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.mcp_tool')">{{
          formatStatus(detailData.mcp_status)
        }}</el-descriptions-item>
        <el-descriptions-item :label="t('system.base.api.field.agent_tool')">{{
          formatStatus(detailData.agent_status)
        }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="detailDoc" class="api-doc">
        <section class="api-doc-section">
          <div class="api-doc-title">{{ t("system.base.api.section.parameters") }}</div>
          <el-table
            v-if="detailParameters.length > 0"
            :data="detailParameters"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" :label="t('system.base.api.doc.field.name')" min-width="220" />
            <el-table-column prop="in" :label="t('system.base.api.doc.field.location')" width="90" />
            <el-table-column :label="t('system.base.api.doc.field.type')" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column :label="t('system.base.api.doc.field.required')" width="80" align="center">
              <template #default="{ row }">{{ t(row.required ? "common.value.yes" : "common.value.no") }}</template>
            </el-table-column>
            <el-table-column
              prop="description"
              :label="t('system.base.api.doc.field.description')"
              min-width="240"
              show-overflow-tooltip
            />
          </el-table>
          <el-empty v-else :description="t('system.base.api.message.no_parameters')" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">{{ t("system.base.api.section.request_body") }}</div>
          <el-table
            v-if="requestBodyRows.length > 0"
            :data="requestBodyRows"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" :label="t('system.base.api.doc.field.name')" min-width="220" />
            <el-table-column :label="t('system.base.api.doc.field.type')" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column :label="t('system.base.api.doc.field.required')" width="80" align="center">
              <template #default="{ row }">{{ t(row.required ? "common.value.yes" : "common.value.no") }}</template>
            </el-table-column>
            <el-table-column
              prop="description"
              :label="t('system.base.api.doc.field.description')"
              min-width="240"
              show-overflow-tooltip
            />
          </el-table>
          <el-empty v-else :description="t('system.base.api.message.no_request_body')" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">{{ t("system.base.api.section.responses") }}</div>
          <el-collapse v-if="detailResponses.length > 0">
            <el-collapse-item v-for="response in detailResponses" :key="response.status" :name="response.status">
              <template #title>
                <span class="api-doc-response-title">{{ response.status }} {{ response.description }}</span>
              </template>
              <el-table
                v-if="responseBodyRows(response).length > 0"
                :data="responseBodyRows(response)"
                row-key="path"
                default-expand-all
                :tree-props="{ children: 'children' }"
              >
                <el-table-column prop="path" :label="t('system.base.api.doc.field.name')" min-width="220" />
                <el-table-column :label="t('system.base.api.doc.field.type')" min-width="180">
                  <template #default="{ row }">{{ formatSchemaType(row) }}</template>
                </el-table-column>
                <el-table-column
                  prop="description"
                  :label="t('system.base.api.doc.field.description')"
                  min-width="240"
                  show-overflow-tooltip
                />
              </el-table>
              <el-empty v-else :description="t('system.base.api.message.no_response_body')" :image-size="72" />
            </el-collapse-item>
          </el-collapse>
          <el-empty v-else :description="t('system.base.api.message.no_responses')" :image-size="72" />
        </section>
      </div>
    </el-drawer>

    <FormDialog
      v-model="editDialog.visible"
      ref="editDialogRef"
      :title="t('system.base.api.title.edit')"
      width="760px"
      :model="editForm"
      :fields="editFields"
      label-width="120px"
      @confirm="handleSubmitEdit"
      @close="handleCloseEditDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { EditPen, View } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/base_api";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import type {
  BaseApi,
  BaseApiDoc,
  BaseApiDocResponse,
  BaseApiDocSchema,
  PageBaseApiRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseApi",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const editDialogRef = ref<InstanceType<typeof FormDialog>>();
const detailData = ref<BaseApi>();
const detailDoc = ref<BaseApiDoc>();

const detailDrawer = reactive({
  visible: false
});

const editDialog = reactive({
  visible: false
});

const editForm = reactive({
  id: 0,
  tool_name: "",
  tool_prompts: [] as string[],
  mcp_status: Status.ENABLE,
  agent_status: Status.ENABLE
});

const requestBodyRows = computed(() => schemaRows(detailDoc.value?.request_body));

/** 将 ProtoJSON 省略的空重复字段固定为数组，供详情区域安全渲染。 */
const detailParameters = computed(() => detailDoc.value?.parameters ?? []);
const detailResponses = computed(() => detailDoc.value?.responses ?? []);
const detailToolPrompts = computed(() => detailData.value?.tool_prompts ?? []);

/** API 编辑表单字段配置。 */
const editFields = computed<ProFormField[]>(() => [
  {
    prop: "tool_name",
    label: t("system.base.api.field.tool_name"),
    component: "input",
    props: { disabled: true }
  },
  {
    prop: "tool_prompts",
    label: t("system.base.api.field.tool_prompts"),
    component: "dynamic-list",
    props: { inputProps: { placeholder: t("system.base.api.placeholder.tool_prompt") } }
  },
  {
    prop: "mcp_status",
    label: t("system.base.api.field.mcp_status"),
    component: "switch",
    props: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled")
    }
  },
  {
    prop: "agent_status",
    label: t("system.base.api.field.agent_status"),
    component: "switch",
    props: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled")
    }
  }
]);

const statusOptions = computed(() => [
  { label: t("common.status.enabled"), value: Status.ENABLE },
  { label: t("common.status.disabled"), value: Status.DISABLE }
]);

/** API 表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { prop: "tool_name", label: t("system.base.api.field.tool_name"), minWidth: 260, search: { el: "input" } },
  {
    prop: "tool_prompts",
    label: t("system.base.api.field.tool_prompts"),
    minWidth: 240,
    search: { el: "input", key: "tool_prompt" },
    render: scope => formatToolPrompts((scope.row as BaseApi).tool_prompts)
  },
  { prop: "service_name", label: t("system.base.api.field.service_name"), minWidth: 180, search: { el: "input" } },
  { prop: "service_desc", label: t("system.base.api.field.service_description"), minWidth: 180, search: { el: "input" } },
  { prop: "desc", label: t("system.base.api.field.description"), minWidth: 180, search: { el: "input" } },
  { prop: "operation", label: t("system.base.api.field.operation"), minWidth: 260, search: { el: "input" } },
  { prop: "method", label: t("system.base.api.field.method"), width: 110, search: { el: "input" } },
  { prop: "path", label: t("system.base.api.field.path"), minWidth: 260, search: { el: "input" } },
  {
    prop: "mcp_status",
    label: t("system.base.api.field.mcp_status"),
    width: 120,
    enum: statusOptions.value,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:api:mcp-status"],
      beforeChange: scope => handleBeforeSetMcpStatus(scope.row as BaseApi)
    }
  },
  {
    prop: "agent_status",
    label: t("system.base.api.field.agent_status"),
    width: 130,
    enum: statusOptions.value,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:api:agent-status"],
      beforeChange: scope => handleBeforeSetAgentStatus(scope.row as BaseApi)
    }
  },
  {
    prop: "operation",
    label: t("common.field.action"),
    width: 210,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:api:update"],
        onClick: scope => handleOpenEditDialog(scope.row as BaseApi)
      },
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:api:info"],
        onClick: scope => handleOpenDetail((scope.row as BaseApi).id)
      }
    ]
  }
]);

/**
 * 请求 API 分页列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseApiTable(params: PageBaseApiRequest) {
  const data = await defBaseApiService.PageBaseApi(buildPageRequest(params));
  return { data: { list: data.base_apis ?? [], total: data.total } };
}

/**
 * 刷新 API 表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开 API 详情抽屉。
 */
async function handleOpenDetail(apiId: number) {
  const [baseApi, baseApiDoc] = await Promise.all([
    defBaseApiService.GetBaseApi({ id: apiId }),
    defBaseApiService.GetBaseApiDoc({ id: apiId })
  ]);
  detailData.value = baseApi;
  detailDoc.value = baseApiDoc;
  detailDrawer.visible = true;
}

/**
 * 关闭详情抽屉并清空旧数据，避免下次打开时短暂展示旧详情。
 */
function handleCloseDetail() {
  detailDrawer.visible = false;
  detailData.value = undefined;
  detailDoc.value = undefined;
}

/**
 * 打开 API 编辑弹窗并回填当前行数据。
 */
function handleOpenEditDialog(row: BaseApi) {
  editForm.id = row.id;
  editForm.tool_name = row.tool_name;
  editForm.tool_prompts = [...(row.tool_prompts ?? [])];
  editForm.mcp_status = row.mcp_status;
  editForm.agent_status = row.agent_status;
  editDialog.visible = true;
}

/**
 * 关闭 API 编辑弹窗并清空表单状态。
 */
function handleCloseEditDialog() {
  editDialog.visible = false;
  editForm.id = 0;
  editForm.tool_name = "";
  editForm.tool_prompts = [];
  editForm.mcp_status = Status.ENABLE;
  editForm.agent_status = Status.ENABLE;
  editDialogRef.value?.clearValidate();
}

/**
 * 提交 API 编辑配置。
 */
async function handleSubmitEdit() {
  await editDialogRef.value?.validate();
  await defBaseApiService.UpdateBaseApi({
    id: editForm.id,
    tool_prompts: editForm.tool_prompts.filter(Boolean),
    mcp_status: editForm.mcp_status,
    agent_status: editForm.agent_status
  });
  ElMessage.success(t("system.base.api.message.save_success"));
  handleCloseEditDialog();
  refreshTable();
}

/**
 * MCP 工具状态切换前进行二次确认，并调用状态接口完成持久化。
 */
async function handleBeforeSetMcpStatus(row: BaseApi) {
  const nextStatus = row.mcp_status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const action = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const apiName = row.desc || row.operation || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(t("system.base.api.message.confirm_mcp_status", { action, api: apiName }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await defBaseApiService.SetBaseApiMcpStatus({ id: row.id, mcp_status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * Agent 工具状态切换前进行二次确认，并调用状态接口完成持久化。
 */
async function handleBeforeSetAgentStatus(row: BaseApi) {
  const nextStatus = row.agent_status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const action = t(nextStatus === Status.ENABLE ? "common.status.enabled" : "common.status.disabled");
  const apiName = row.desc || row.operation || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(t("system.base.api.message.confirm_agent_status", { action, api: apiName }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await defBaseApiService.SetBaseApiAgentStatus({ id: row.id, agent_status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 格式化工具提示词列表。
 */
function formatToolPrompts(prompts: string[]) {
  if (!prompts?.length) return "--";
  return prompts.filter(Boolean).join(", ");
}

/**
 * 格式化 API 工具状态。
 */
function formatStatus(status: Status) {
  if (status === Status.ENABLE) return t("common.status.enabled");
  if (status === Status.DISABLE) return t("common.status.disabled");
  return t("common.value.unknown");
}

/**
 * 将可选 Schema 转成表格行。
 */
function schemaRows(schema?: BaseApiDocSchema) {
  return schema ? [schema] : [];
}

/**
 * 获取响应体表格行。
 */
function responseBodyRows(response: BaseApiDocResponse) {
  return schemaRows(response.body);
}

/**
 * 格式化 Schema 类型，补充格式、引用类型与枚举值。
 */
function formatSchemaType(schema: BaseApiDocSchema) {
  const values = [schema.type];
  if (schema.format) values.push(`<${schema.format}>`);
  if (schema.ref) values.push(schema.ref);
  // ProtoJSON 会省略空 repeated 字段，枚举缺失时按空数组展示。
  if (schema.enum?.length > 0) values.push(schema.enum.join(" | "));
  return values.filter(Boolean).join(" ");
}
</script>

<style scoped lang="scss">
.api-doc {
  margin-top: 18px;
}
.api-doc-section {
  margin-top: 20px;
}
.api-doc-title {
  margin-bottom: 10px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.api-doc-response-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}
.tool-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
