<!-- API 管理 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestBaseApiTable" />

    <el-drawer v-model="detailDrawer.visible" title="API 详情" size="70%" @close="handleCloseDetail">
      <el-descriptions v-if="detailData" :column="1" border>
        <el-descriptions-item label="工具名">{{ detailData.tool_name }}</el-descriptions-item>
        <el-descriptions-item label="工具提示词">
          <div class="tool-prompts">
            <el-tag v-for="prompt in detailToolPrompts" :key="prompt" effect="plain">{{ prompt }}</el-tag>
            <span v-if="!detailToolPrompts.length">--</span>
          </div>
        </el-descriptions-item>
        <el-descriptions-item label="服务名">{{ detailData.service_name }}</el-descriptions-item>
        <el-descriptions-item label="服务描述">{{ detailData.service_desc }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ detailData.desc }}</el-descriptions-item>
        <el-descriptions-item label="操作方法">{{ detailData.operation }}</el-descriptions-item>
        <el-descriptions-item label="请求方法">{{ detailData.method }}</el-descriptions-item>
        <el-descriptions-item label="请求地址">{{ detailData.path }}</el-descriptions-item>
        <el-descriptions-item label="MCP工具">{{ formatStatus(detailData.mcp_status) }}</el-descriptions-item>
        <el-descriptions-item label="Agent工具">{{ formatStatus(detailData.agent_status) }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="detailDoc" class="api-doc">
        <section class="api-doc-section">
          <div class="api-doc-title">请求参数</div>
          <el-table
            v-if="detailParameters.length > 0"
            :data="detailParameters"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" label="字段" min-width="220" />
            <el-table-column prop="in" label="位置" width="90" />
            <el-table-column label="类型" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column label="必填" width="80" align="center">
              <template #default="{ row }">{{ row.required ? "是" : "否" }}</template>
            </el-table-column>
            <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
          </el-table>
          <el-empty v-else description="无请求参数" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">请求体</div>
          <el-table
            v-if="requestBodyRows.length > 0"
            :data="requestBodyRows"
            row-key="path"
            default-expand-all
            :tree-props="{ children: 'children' }"
          >
            <el-table-column prop="path" label="字段" min-width="220" />
            <el-table-column label="类型" min-width="180">
              <template #default="{ row }">{{ formatSchemaType(row) }}</template>
            </el-table-column>
            <el-table-column label="必填" width="80" align="center">
              <template #default="{ row }">{{ row.required ? "是" : "否" }}</template>
            </el-table-column>
            <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
          </el-table>
          <el-empty v-else description="无请求体" :image-size="72" />
        </section>

        <section class="api-doc-section">
          <div class="api-doc-title">返回值</div>
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
                <el-table-column prop="path" label="字段" min-width="220" />
                <el-table-column label="类型" min-width="180">
                  <template #default="{ row }">{{ formatSchemaType(row) }}</template>
                </el-table-column>
                <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
              </el-table>
              <el-empty v-else description="无响应体" :image-size="72" />
            </el-collapse-item>
          </el-collapse>
          <el-empty v-else description="无返回值" :image-size="72" />
        </section>
      </div>
    </el-drawer>

    <FormDialog
      v-model="editDialog.visible"
      ref="editDialogRef"
      title="编辑API"
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
import type { ColumnProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@/components/ProForm/interface";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseApiService } from "@/api/system/base_api";
import { Status } from "@/rpc/common/v1/enum";
import type { BaseApi, BaseApiDoc, BaseApiDocResponse, BaseApiDocSchema, PageBaseApiRequest } from "@/rpc/system/admin/v1/base_api";
import { buildPageRequest } from "@/utils/proTable";

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
const editFields: ProFormField[] = [
  {
    prop: "tool_name",
    label: "工具名",
    component: "input",
    props: { disabled: true }
  },
  {
    prop: "tool_prompts",
    label: "工具提示词",
    component: "dynamic-list",
    props: { inputProps: { placeholder: "请输入工具提示词" } }
  },
  {
    prop: "mcp_status",
    label: "MCP工具状态",
    component: "switch",
    props: { activeValue: Status.ENABLE, inactiveValue: Status.DISABLE, activeText: "启用", inactiveText: "禁用" }
  },
  {
    prop: "agent_status",
    label: "Agent工具状态",
    component: "switch",
    props: { activeValue: Status.ENABLE, inactiveValue: Status.DISABLE, activeText: "启用", inactiveText: "禁用" }
  }
];

const statusOptions = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** API 表格列配置。 */
const columns: ColumnProps[] = [
  { prop: "tool_name", label: "工具名", minWidth: 260, search: { el: "input" } },
  {
    prop: "tool_prompts",
    label: "工具提示词",
    minWidth: 240,
    search: { el: "input", key: "tool_prompt" },
    render: scope => formatToolPrompts((scope.row as BaseApi).tool_prompts)
  },
  { prop: "service_name", label: "服务名", minWidth: 180, search: { el: "input" } },
  { prop: "service_desc", label: "服务描述", minWidth: 180, search: { el: "input" } },
  { prop: "desc", label: "描述", minWidth: 180, search: { el: "input" } },
  { prop: "operation", label: "操作方法", minWidth: 260, search: { el: "input" } },
  { prop: "method", label: "请求方法", width: 110, search: { el: "input" } },
  { prop: "path", label: "请求地址", minWidth: 260, search: { el: "input" } },
  {
    prop: "mcp_status",
    label: "MCP工具状态",
    width: 120,
    enum: statusOptions,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:api:mcp-status"],
      beforeChange: scope => handleBeforeSetMcpStatus(scope.row as BaseApi)
    }
  },
  {
    prop: "agent_status",
    label: "Agent工具状态",
    width: 130,
    enum: statusOptions,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: () => !BUTTONS.value["base:api:agent-status"],
      beforeChange: scope => handleBeforeSetAgentStatus(scope.row as BaseApi)
    }
  },
  {
    prop: "operation",
    label: "操作",
    width: 210,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: "编辑",
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:api:update"],
        onClick: scope => handleOpenEditDialog(scope.row as BaseApi)
      },
      {
        label: "详情",
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:api:info"],
        onClick: scope => handleOpenDetail((scope.row as BaseApi).id)
      }
    ]
  }
];

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
  ElMessage.success("保存成功");
  handleCloseEditDialog();
  refreshTable();
}

/**
 * MCP 工具状态切换前进行二次确认，并调用状态接口完成持久化。
 */
async function handleBeforeSetMcpStatus(row: BaseApi) {
  const nextStatus = row.mcp_status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const apiName = row.desc || row.operation || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}该 API 的 MCP 工具能力？\nAPI：${apiName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseApiService.SetBaseApiMcpStatus({ id: row.id, mcp_status: nextStatus });
    ElMessage.success(`${text}成功`);
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
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const apiName = row.desc || row.operation || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}该 API 的 Agent 工具能力？\nAPI：${apiName}`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseApiService.SetBaseApiAgentStatus({ id: row.id, agent_status: nextStatus });
    ElMessage.success(`${text}成功`);
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
  return prompts.filter(Boolean).join("；");
}

/**
 * 格式化 API 工具状态。
 */
function formatStatus(status: Status) {
  if (status === Status.ENABLE) return "启用";
  if (status === Status.DISABLE) return "禁用";
  return "未知";
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
