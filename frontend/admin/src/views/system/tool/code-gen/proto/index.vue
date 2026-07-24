<!-- 代码生成 Proto 能力配置 -->
<template>
  <div v-loading="loading" class="app-container code-gen-sub-page">
    <el-card class="code-gen-sub-card" shadow="never">
      <template v-if="formData.id">
        <div class="code-gen-toolbar">
          <!-- 展示当前 Proto 配置对应的业务表。 -->
          <div class="code-gen-proto-toolbar__meta">
            <span class="code-gen-proto-toolbar__table-name" :title="formData.name">表名：{{ formData.name }}</span>
            <span class="code-gen-proto-toolbar__table-comment" :title="formData.comment || '--'">
              表注释：{{ formData.comment || "--" }}
            </span>
          </div>
          <el-button type="primary" :icon="Document" :disabled="!canEdit" @click="handleSaveProtoMethods()">保存</el-button>
        </div>

        <ProTable
          row-key="sort"
          :data="protoChecks"
          :columns="protoColumns"
          :pagination="false"
          :tool-button="false"
        >
          <template #exists="{ row }">
            <div class="code-gen-proto-status">
              <el-tag :type="row.exists ? 'success' : 'warning'">{{ row.exists ? "已存在" : "缺失" }}</el-tag>
              <span class="code-gen-proto-cell__secondary" :title="row.message">{{ row.message }}</span>
            </div>
          </template>
          <template #proto_info="{ row }">
            <el-popover trigger="hover" placement="top-start" :width="720" :show-after="150">
              <template #reference>
                <div class="code-gen-proto-capability-trigger">
                  <span class="code-gen-proto-capability-trigger__comment" :title="row.method_comment || '--'">
                    {{ row.method_comment || "--" }}
                  </span>
                </div>
              </template>
              <div class="code-gen-proto-capability-popover">
                <div class="code-gen-proto-capability-popover__item">
                  <span>方法描述</span>
                  <span class="code-gen-proto-capability-popover__value">{{ row.method_comment || "--" }}</span>
                </div>
                <div class="code-gen-proto-capability-popover__item">
                  <span>方法名</span>
                  <code>{{ row.method_name || "--" }}</code>
                </div>
                <div class="code-gen-proto-capability-popover__item">
                  <span>Proto路径</span>
                  <code class="code-gen-proto-capability-popover__path">{{ row.proto_file_path || "--" }}</code>
                </div>
                <div class="code-gen-proto-capability-popover__item">
                  <span>服务描述</span>
                  <span class="code-gen-proto-capability-popover__value">{{ row.service_comment || "--" }}</span>
                </div>
                <div class="code-gen-proto-capability-popover__item">
                  <span>服务名</span>
                  <code>{{ row.service_name || "--" }}</code>
                </div>
                <pre class="code-gen-proto-capability-popover__preview"><code>{{ resolveProtoDefinition(row) }}</code></pre>
              </div>
            </el-popover>
          </template>
          <template #generate_when_missing="{ row }">
            <div class="code-gen-proto-generate">
              <el-checkbox v-model="row.generate_when_missing" :disabled="row.exists || !canEdit">生成接口</el-checkbox>
              <el-button
                v-if="showProtoConfigButton(row)"
                type="primary"
                size="small"
                link
                :icon="Setting"
                :disabled="!canEdit"
                @click="openProtoConfigDialog(row)"
              >
                配置
              </el-button>
            </div>
          </template>
        </ProTable>
      </template>
      <el-empty v-else description="请先选择生成记录" />
    </el-card>

    <ProDialog
      v-model="configDialog.visible"
      :title="`${configDialog.methodName} - ${resolveAPIKindLabel(configDialog.apiKind)}配置`"
      width="min(520px, calc(100vw - 32px))"
      destroy-on-close
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-footer="false"
      @closed="handleProtoConfigDialogClosed"
    >
      <template #header="{ titleId, titleClass }">
        <div class="code-gen-config-dialog__header">
          <span :id="titleId" :class="titleClass">
            {{ `${configDialog.methodName} - ${resolveAPIKindLabel(configDialog.apiKind)}配置` }}
          </span>
          <el-button
            type="primary"
            :icon="Document"
            :disabled="!canEdit"
            aria-label="保存并关闭"
            @click="handleSaveProtoConfigDialog"
          >
            保存
          </el-button>
        </div>
      </template>

      <ProForm
        :model="protoConfigFormModel"
        :fields="protoConfigFields"
        label-position="top"
        class="code-gen-proto-config-form"
      />
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { Document, Setting } from "@element-plus/icons-vue";
import { useRoute, useRouter } from "vue-router";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import type { ColumnProps } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useTabsStore } from "@/stores/modules/tabs";
import { defCodeGenColumnService } from "@/api/system/code_gen_column";
import { defCodeGenProtoService } from "@/api/system/code_gen_proto";
import { defCodeGenTableService } from "@/api/system/code_gen_table";
import type { CodeGenProto, CodeGenProtoCheck, CodeGenProtoConfig } from "@/rpc/system/admin/v1/code_gen_proto";
import type { CodeGenDatabaseTable, CodeGenTableForm } from "@/rpc/system/admin/v1/code_gen_table";
import { createDefaultCodeGenTableForm } from "../config";

defineOptions({
  name: "CodeGenProto",
  inheritAttrs: false
});

const route = useRoute();
const router = useRouter();
const tabsStore = useTabsStore();
const { BUTTONS } = useAuthButtons();

const loading = ref(false);
const protoChecks = ref<CodeGenProtoCheck[]>([]);
const databaseTables = ref<CodeGenDatabaseTable[]>([]);
const targetColumnOptions = reactive<Record<string, ProFormOption[]>>({});
const loadingTargetColumns = reactive(new Set<string>());
const formData = reactive<CodeGenTableForm>(createDefaultCodeGenTableForm());

const triggerTypeLabels: Record<string, string> = {
  crud: "基础能力",
  page_tree: "树形页面",
  left_tree: "左树数据",
  entity_option: "实体选项",
  field_option: "字段选项",
  field_status: "字段状态"
};

const apiKindLabels: Record<string, string> = {
  crud: "增删改查",
  list: "分页列表",
  option: "选项",
  tree: "树形",
  status: "状态"
};

/** Proto 类型配置弹窗上下文。 */
interface CodeGenProtoConfigDialog {
  visible: boolean;
  row: CodeGenProtoCheck | undefined;
  methodName: string;
  apiKind: string;
  config: CodeGenProtoConfig;
}

/** Proto 配置中引用数据库字段的键。 */
type CodeGenProtoColumnKey = "parent_column" | "label_column" | "value_column" | "status_column";

const configDialog = reactive<CodeGenProtoConfigDialog>({
  visible: false,
  row: undefined,
  methodName: "",
  apiKind: "",
  config: createDefaultCodeGenProtoConfig()
});

/** Proto 检查结果表格列配置。 */
const protoColumns: ColumnProps[] = [
  { prop: "trigger_type", label: "触发来源", minWidth: 150, render: scope => resolveTriggerTypeLabel(String(scope.row.trigger_type)) },
  { prop: "api_kind", label: "接口类型", minWidth: 150, render: scope => resolveAPIKindLabel(String(scope.row.api_kind)) },
  { prop: "proto_info", label: "接口能力", minWidth: 290 },
  { prop: "exists", label: "状态", minWidth: 210 },
  { prop: "generate_when_missing", label: "生成设置", minWidth: 230 }
];

/** 将 Proto 配置对象适配为 ProForm 所需的通用表单模型。 */
const protoConfigFormModel = computed<Record<string, any>>(() => configDialog.config as Record<string, any>);

/** Proto 配置弹窗的动态表单字段。 */
const protoConfigFields = computed<ProFormField[]>(() => [
  {
    prop: "parent_column",
    label: "父节点字段",
    component: "select",
    options: () => configColumnOptions.value,
    props: () => ({ loading: configDialogLoading.value, filterable: true, clearable: true, placeholder: "请选择父节点字段" }),
    visible: () => configDialog.apiKind === "tree"
  },
  {
    prop: "label_column",
    label: "显示字段",
    component: "select",
    options: () => configColumnOptions.value,
    props: () => ({ loading: configDialogLoading.value, filterable: true, clearable: true, placeholder: "请选择显示字段" }),
    visible: () => ["option", "tree"].includes(configDialog.apiKind)
  },
  {
    prop: "value_column",
    label: "值字段",
    component: "select",
    options: () => configColumnOptions.value,
    props: () => ({ loading: configDialogLoading.value, filterable: true, clearable: true, placeholder: "请选择值字段" }),
    visible: () => ["option", "tree"].includes(configDialog.apiKind)
  },
  {
    prop: "lazy",
    label: "加载方式",
    component: "checkbox",
    checkboxLabel: "懒加载子节点",
    props: () => ({ disabled: configDialog.apiKind !== "tree" }),
    visible: () => configDialog.apiKind === "tree"
  },
  {
    prop: "status_column",
    label: "状态字段",
    component: "select",
    options: () => configColumnOptions.value,
    props: () => ({ loading: configDialogLoading.value, filterable: true, clearable: true, placeholder: "请选择状态字段" }),
    visible: () => configDialog.apiKind === "status"
  }
]);

/** 当前生成对象 ID。 */
const tableId = computed(() => {
  const value = route.params.tableId ?? route.query.tableId;
  const id = Number(Array.isArray(value) ? value[0] : value);
  return Number.isFinite(id) && id > 0 ? id : 0;
});

/** 是否可以维护 Proto 配置。 */
const canEdit = computed(() => !!BUTTONS.value["tool:code-gen-table:proto"]);

/** 当前配置弹窗可选的数据库字段。 */
const configColumnOptions = computed(() =>
  configDialog.row ? resolveColumnNameOptions(configDialog.row) : ([] as ProFormOption[])
);

/** 当前配置弹窗是否正在加载目标表字段。 */
const configDialogLoading = computed(() => {
  const tableName = configDialog.row ? resolveTargetTableName(configDialog.row) : "";
  return !!tableName && loadingTargetColumns.has(tableName);
});

// 路由生成对象变化时重新加载 Proto 配置。
watch(tableId, () => {
  void handleQuery();
});

/**
 * 查询生成对象字段与 Proto 配置。
 */
async function handleQuery() {
  loading.value = true;
  try {
    Object.assign(formData, createDefaultCodeGenTableForm());
    protoChecks.value = [];
    databaseTables.value = [];
    Object.keys(targetColumnOptions).forEach(key => delete targetColumnOptions[key]);
    if (!tableId.value) return;
    const [table, tableResponse] = await Promise.all([
      defCodeGenTableService.GetCodeGenTable({ id: tableId.value }),
      defCodeGenTableService.ListCodeGenDatabaseTable({})
    ]);
    Object.assign(formData, table);
    databaseTables.value = tableResponse.tables ?? [];
    const columnResponse = await defCodeGenColumnService.ListCodeGenDatabaseColumn({ table_name: formData.name });
    targetColumnOptions[formData.name] = createColumnOptions(columnResponse.columns ?? []);
    await loadProtoChecks();
    syncWorkspaceTitle();
  } finally {
    loading.value = false;
  }
}

/**
 * 同步当前页签和浏览器标题。
 */
function syncWorkspaceTitle() {
  const tableTitle = formData.comment || formData.name;
  const title = tableTitle ? `${tableTitle} Proto 能力` : "Proto 能力";
  tabsStore.setTabsTitle(title);
  document.title = `${title} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

/**
 * 自动检查当前生成对象需要的 Proto 能力。
 */
async function loadProtoChecks() {
  if (!formData.id) return;
  const data = await defCodeGenProtoService.ListCodeGenProto({ table_id: formData.id });
  protoChecks.value = (data.code_gen_protos ?? []).map(item => ({
    ...item,
    config: normalizeCodeGenProtoConfig(item.config)
  }));
  const targetTableNames = new Set(protoChecks.value.map(resolveTargetTableName).filter(Boolean));
  await Promise.all(Array.from(targetTableNames, loadTargetColumnOptions));
}

/** 根据检查项目标实体返回真实数据库表名。 */
function resolveTargetTableName(row: CodeGenProtoCheck) {
  if (row.target_entity_name === toPascalCase(formData.name)) return formData.name;
  return databaseTables.value.find(item => toPascalCase(item.name) === row.target_entity_name)?.name ?? "";
}

/** 返回检查项目标表对应的字段选项。 */
function resolveColumnNameOptions(row: CodeGenProtoCheck) {
  return targetColumnOptions[resolveTargetTableName(row)] ?? [];
}

/** 返回触发来源的展示文案。 */
function resolveTriggerTypeLabel(triggerType: string) {
  return triggerTypeLabels[triggerType] ?? triggerType;
}

/** 返回接口类型的展示文案。 */
function resolveAPIKindLabel(apiKind: string) {
  return apiKindLabels[apiKind] ?? apiKind;
}

/** 返回生成 RPC 的请求与响应类型签名。 */
function resolveProtoRPCSignature(row: CodeGenProtoCheck) {
  const methodName = row.method_name || "--";
  const entity = row.target_entity_name || toPascalCase(formData.name);
  let responseType = "google.protobuf.Empty";
  switch (row.api_kind) {
    case "list":
    case "tree":
      responseType = row.api_kind === "tree" && ["entity_option", "field_option", "left_tree"].includes(row.trigger_type)
        ? ".common.v1.TreeOptionResponse"
        : `${methodName}Response`;
      break;
    case "option":
      responseType = ".common.v1.SelectOptionResponse";
      break;
    case "crud":
      responseType = methodName === `Get${entity}` ? `${entity}Form` : "google.protobuf.Empty";
      break;
  }
  return `rpc ${methodName}(${methodName}Request) returns (${responseType})`;
}

/** 返回准备生成的 Proto 服务和 RPC 片段。 */
function resolveProtoDefinition(row: CodeGenProtoCheck) {
  const serviceName = row.service_name || "--";
  return [
    `// ${row.service_comment || "--"}`,
    `service ${serviceName} {`,
    `  // ${row.method_comment || "--"}`,
    `  ${resolveProtoRPCSignature(row)}`,
    "}"
  ].join("\n");
}

/** 判断接口类型是否需要额外配置。 */
function needsProtoConfig(apiKind: string) {
  return ["option", "tree", "status"].includes(apiKind);
}

/** 仅在勾选生成且类型需要配置时展示配置入口。 */
function showProtoConfigButton(row: CodeGenProtoCheck) {
  return !row.exists && row.generate_when_missing && needsProtoConfig(row.api_kind);
}

/** 创建完整的 Proto 类型配置默认值。 */
function createDefaultCodeGenProtoConfig(): CodeGenProtoConfig {
  return {
    parent_column: "",
    label_column: "",
    value_column: "",
    status_column: "",
    lazy: false
  };
}

/** 将接口返回的可选配置补齐为前端编辑模型。 */
function normalizeCodeGenProtoConfig(config: CodeGenProtoConfig | undefined) {
  return { ...createDefaultCodeGenProtoConfig(), ...(config ?? {}) };
}

/** 打开弹窗时仅为目标表存在的空配置字段填入约定默认值。 */
function applyProtoConfigDefaults(row: CodeGenProtoCheck, config: CodeGenProtoConfig) {
  const availableColumns = new Set(resolveColumnNameOptions(row).map(item => String(item.value)));
  const applyDefault = (key: CodeGenProtoColumnKey, value: string) => {
    if (!config[key] && availableColumns.has(value)) config[key] = value;
  };
  // 不同接口类型只填充其生成模板固定消费的字段。
  switch (row.api_kind) {
    case "option":
      applyDefault("label_column", "name");
      applyDefault("value_column", "id");
      break;
    case "tree":
      applyDefault("parent_column", "parent_id");
      applyDefault("label_column", "name");
      applyDefault("value_column", "id");
      break;
    case "status":
      applyDefault("status_column", "status");
      break;
  }
}

/** 打开当前接口的类型配置弹窗。 */
function openProtoConfigDialog(row: CodeGenProtoCheck) {
  const config = normalizeCodeGenProtoConfig(row.config);
  applyProtoConfigDefaults(row, config);
  configDialog.row = row;
  configDialog.methodName = row.method_name;
  configDialog.apiKind = row.api_kind;
  Object.assign(configDialog.config, config);
  configDialog.visible = true;
}

/** 保存 Proto 配置草稿并关闭弹窗。 */
function handleSaveProtoConfigDialog() {
  if (configDialog.row) configDialog.row.config = normalizeCodeGenProtoConfig(configDialog.config);
  configDialog.visible = false;
}

/** 清理 Proto 配置弹窗上下文。 */
function handleProtoConfigDialogClosed() {
  configDialog.row = undefined;
  configDialog.methodName = "";
  configDialog.apiKind = "";
  Object.assign(configDialog.config, createDefaultCodeGenProtoConfig());
}

/** 判断勾选生成的接口是否已具备完整类型配置。 */
function hasCompleteProtoConfig(row: CodeGenProtoCheck) {
  const config = normalizeCodeGenProtoConfig(row.config);
  // 每种接口类型检查其生成模板消费的固定字段。
  switch (row.api_kind) {
    case "option":
      return !!config.label_column && !!config.value_column;
    case "tree":
      return !!config.parent_column && !!config.label_column && !!config.value_column;
    case "status":
      return !!config.status_column;
    default:
      return true;
  }
}

/** 保存前定位第一个缺少必要配置的接口。 */
function validateProtoConfigs() {
  const row = protoChecks.value.find(
    item => !item.exists && item.generate_when_missing && needsProtoConfig(item.api_kind) && !hasCompleteProtoConfig(item)
  );
  if (!row) return true;
  ElMessage.warning(`请先配置接口 ${row.method_name}`);
  openProtoConfigDialog(row);
  return false;
}

/** 加载并缓存目标数据库表字段。 */
async function loadTargetColumnOptions(tableName: string) {
  if (!tableName || targetColumnOptions[tableName] || loadingTargetColumns.has(tableName)) return;
  loadingTargetColumns.add(tableName);
  try {
    const data = await defCodeGenColumnService.ListCodeGenDatabaseColumn({ table_name: tableName });
    targetColumnOptions[tableName] = createColumnOptions(data.columns ?? []);
  } finally {
    loadingTargetColumns.delete(tableName);
  }
}

/** 将数据库字段或生成字段转换成下拉选项。 */
function createColumnOptions(columns: Array<{ name: string; comment: string }>) {
  return columns
    .filter(item => item.name)
    .map(item => ({
      label: item.comment ? `${item.name}（${item.comment}）` : item.name,
      value: item.name
    }));
}

/**
 * 保存缺失 Proto 接口的生成选择。
 */
async function handleSaveProtoMethods(showMessage = true) {
  if (!formData.id) return false;
  if (!validateProtoConfigs()) return false;
  const codeGenProtos: CodeGenProto[] = protoChecks.value.map((item, index) => ({
    id: 0,
    table_id: formData.id,
    trigger_type: item.trigger_type,
    api_kind: item.api_kind,
    config: normalizeCodeGenProtoConfig(item.config),
    generate_when_missing: !item.exists && item.generate_when_missing,
    sort: index + 1
  }));
  await defCodeGenProtoService.SaveCodeGenProto({
    table_id: formData.id,
    code_gen_protos: codeGenProtos
  });
  if (showMessage) ElMessage.success("保存Proto生成选择成功");
  // 路由切换前保留当前页签地址，避免误删目标页签。
  const currentPath = route.fullPath;
  await router.push("/code/gen/table");
  await tabsStore.removeTabs(currentPath, false);
  return true;
}

/** 将数据库表名转换为生成器使用的实体名。 */
function toPascalCase(value: string) {
  return value
    .split("_")
    .filter(Boolean)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

onMounted(() => {
  syncWorkspaceTitle();
  void handleQuery();
});
</script>

<style scoped lang="scss">
.code-gen-sub-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.code-gen-config-dialog__header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

:deep(.code-gen-toolbar) {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.code-gen-proto-toolbar__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  color: var(--admin-page-text-secondary);
}
.code-gen-proto-toolbar__meta span {
  padding: 3px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 999px;
}
.code-gen-proto-toolbar__table-name {
  max-width: 220px;
}
.code-gen-proto-toolbar__table-comment {
  max-width: 280px;
}
.code-gen-proto-cell {
  /* 将名称、标签和说明保持在同一行，长文本优先收缩并通过 title 保留完整内容。 */
  display: flex;
  gap: 6px;
  align-items: center;
  justify-content: center;
  min-width: 0;
}
.code-gen-proto-cell__primary,
.code-gen-proto-cell__path,
.code-gen-proto-status .code-gen-proto-cell__secondary {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.code-gen-proto-cell__primary {
  flex: 0 1 auto;
  color: var(--admin-page-text-primary);
}
.code-gen-proto-cell__tags,
.code-gen-proto-status,
.code-gen-proto-generate {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  /* 多行单元格与列标题保持居中，避免文本继承居中而 Flex 子项贴左。 */
  justify-content: center;
  min-width: 0;
}
.code-gen-proto-cell__tags {
  flex: 0 0 auto;
  flex-wrap: nowrap;
}
.code-gen-proto-cell__secondary,
.code-gen-proto-cell__path {
  flex: 0 1 auto;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}
.code-gen-proto-status {
  flex-wrap: nowrap;
}
.code-gen-proto-status .code-gen-proto-cell__secondary {
  flex: 0 1 auto;
}
.code-gen-proto-capability-trigger {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  cursor: pointer;
  text-align: left;
}
.code-gen-proto-capability-trigger__comment {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--admin-page-text-primary);
}
.code-gen-proto-capability-popover {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.code-gen-proto-capability-popover__preview {
  max-width: 100%;
  margin: 0;
  padding: 10px;
  overflow-x: auto;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-primary);
  white-space: pre;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 4px;
}
.code-gen-proto-capability-popover__item {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-width: 0;
  color: var(--admin-page-text-secondary);
}
.code-gen-proto-capability-popover__item code {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--admin-page-text-primary);
}
.code-gen-proto-capability-popover__item .code-gen-proto-capability-popover__path {
  display: block;
  max-width: 100%;
  overflow-x: auto;
  overflow-wrap: normal;
  white-space: nowrap;
}
.code-gen-proto-capability-popover__value {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--admin-page-text-primary);
}
.code-gen-proto-config-form :deep(.el-select) {
  width: 100%;
}
</style>
