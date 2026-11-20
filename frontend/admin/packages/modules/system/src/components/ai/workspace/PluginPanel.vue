<template>
  <section class="assistant-workspace">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <el-tabs v-model="tab">
      <el-tab-pane :label="t('system.ai.workspace.catalog')" name="catalog" />
      <el-tab-pane :label="t('system.ai.workspace.connections')" name="connections" />
      <el-tab-pane v-if="canManage" :label="t('system.ai.workspace.tools')" name="tools" />
    </el-tabs>
    <el-alert v-if="mobile" :title="t('system.ai.workspace.desktop_hint')" :closable="false" type="info" />
    <div v-if="tab === 'catalog'" v-loading="loading">
      <div class="workspace-toolbar">
        <el-input v-model="keyword" clearable :placeholder="t('system.ai.workspace.search')" />
        <el-select v-model="scope" :aria-label="t('system.ai.workspace.scope')"
          ><el-option :label="t('system.ai.workspace.all')" value="" /><el-option
            v-for="item in ['system', 'tenant', 'private']"
            :key="item"
            :label="t(`system.ai.workspace.${item}`)"
            :value="item"
        /></el-select>
        <el-button v-if="!mobile && (canCreatePrivate || canManageCatalog)" type="primary" @click="openDefinition()">{{
          t("system.ai.workspace.add_mcp")
        }}</el-button>
      </div>
      <el-empty v-if="!loading && !error && !visiblePlugins.length" :description="t('system.ai.workspace.no_matches')" />
      <div class="plugin-grid">
        <article v-for="plugin in visiblePlugins" :key="plugin.id" class="plugin-card">
          <div class="plugin-card-title">
            <el-text tag="h3">{{ plugin.name }}</el-text>
            <el-tag size="small">{{ plugin.source }}</el-tag>
          </div>
          <el-text tag="p">{{ plugin.description }}</el-text>
          <el-tag :type="plugin.unavailable_reason ? 'warning' : 'info'">{{
            plugin.unavailable_reason ||
            (plugin.installation?.enabled ? t("system.ai.workspace.enabled") : t("system.ai.workspace.disabled"))
          }}</el-tag>
          <div class="workspace-toolbar">
            <el-button @click="openDetail(plugin)">{{ t("system.ai.workspace.details") }}</el-button>
            <el-button
              v-if="plugin.source !== 'MCP'"
              :disabled="!!plugin.unavailable_reason || !plugin.enabled || !plugin.assembled || !plugin.installation?.enabled"
              type="primary"
              @click="emit('select', { plugin_id: plugin.id, connection_id: '' })"
              >{{ t("system.ai.workspace.use_chat") }}</el-button
            >
            <el-button
              v-else-if="!mobile"
              :disabled="!plugin.enabled || !plugin.installation?.enabled || connectedPluginIDs.has(plugin.id)"
              type="primary"
              @click="openConnection(plugin)"
              >{{
                connectedPluginIDs.has(plugin.id)
                  ? t("system.ai.workspace.connection_connected")
                  : t("system.ai.workspace.connect")
              }}</el-button
            >
            <el-button v-if="!mobile && canManageTools" @click="openInstallation(plugin)">{{
              t("system.ai.workspace.installation")
            }}</el-button>
            <template v-if="!mobile && plugin.can_manage">
              <el-button @click="openDefinition(plugin)">{{ t("common.action.edit") }}</el-button>
              <el-button type="danger" @click="removePlugin(plugin)">{{ t("common.action.delete") }}</el-button>
            </template>
          </div>
        </article>
      </div>
    </div>
    <div v-else-if="tab === 'connections'">
      <el-input v-model="connectionKeyword" class="workspace-query" clearable :placeholder="t('system.ai.workspace.search')" />
      <ProTable row-key="id" :columns="connectionColumns" :data="visibleConnections" :pagination="false" :tool-button="false" />
    </div>
    <div v-else-if="canManage">
      <div class="workspace-toolbar">
        <el-select v-model="toolPluginID" :placeholder="t('system.ai.workspace.plugins')" @change="reloadToolsTable"
          ><el-option v-for="plugin in managedPlugins" :key="plugin.id" :label="plugin.name" :value="plugin.id"
        /></el-select>
        <el-input
          v-model="toolKeyword"
          clearable
          :placeholder="t('system.ai.workspace.search')"
          @keyup.enter="reloadToolsTable"
          @clear="reloadToolsTable"
        />
      </div>
      <el-alert v-if="toolError" :title="toolError" type="error" :closable="false" />
      <ProTable
        ref="toolsTable"
        row-key="id"
        :columns="toolColumns"
        :request-api="requestToolsTable"
        :init-param="toolTableInitParam"
        :request-error="handleToolsError"
        :tool-button="false"
      />
    </div>
    <el-drawer
      v-model="detailOpen"
      class="plugin-detail-drawer"
      :title="detail?.name"
      :size="mobile ? '100%' : 'min(860px, 94vw)'"
      destroy-on-close
    >
      <template #header>
        <div v-if="detail" class="plugin-detail-heading">
          <div class="plugin-detail-heading__title">
            <strong>{{ detail.name }}</strong>
            <el-tag size="small">{{ detail.source }}</el-tag>
          </div>
          <el-text>{{ detail.description }}</el-text>
        </div>
      </template>
      <div v-if="detail" class="plugin-detail-content">
        <div class="plugin-detail-meta">
          <span>{{ t("system.ai.workspace.code") }}：{{ detail.code }}</span>
          <span>{{ t("system.ai.workspace.endpoint") }}：{{ detail.endpoint || "-" }}</span>
        </div>
        <el-alert v-if="detail.unavailable_reason" :title="detail.unavailable_reason" type="warning" :closable="false" />
        <el-alert v-if="detailError" :title="detailError" type="error" :closable="false" />
        <div v-loading="detailLoading" class="plugin-detail-layout">
          <aside class="plugin-tool-nav">
            <div class="plugin-tool-nav__header">
              <strong>{{ t("system.ai.workspace.tools") }}</strong>
              <el-text type="info">{{ detailTotal }}</el-text>
            </div>
            <el-input
              v-model="detailToolKeyword"
              clearable
              :placeholder="t('system.ai.workspace.search')"
              @keyup.enter="searchDetailTools"
              @clear="searchDetailTools"
            />
            <nav class="plugin-tool-nav__items" :aria-label="t('system.ai.workspace.tools')">
              <button
                v-for="item in detailTools"
                :key="item.id"
                type="button"
                class="plugin-tool-nav__item"
                :class="{ 'is-active': selectedDetailTool?.id === item.id }"
                @click="selectedDetailToolId = item.id"
              >
                <strong>{{ item.source_name }}</strong>
                <span>{{ item.description || "-" }}</span>
              </button>
            </nav>
            <el-pagination
              v-if="detailTotal > detailPageSize"
              v-model:current-page="detailPageNum"
              :page-size="detailPageSize"
              :total="detailTotal"
              layout="prev, pager, next"
              background
              small
              @current-change="loadDetailTools"
            />
            <el-empty v-if="!detailLoading && !detailTools.length" :description="t('system.ai.workspace.no_matches')" />
          </aside>
          <main v-if="selectedDetailTool" class="plugin-tool-preview">
            <div class="plugin-tool-preview__heading">
              <strong>{{ selectedDetailTool.source_name }}</strong>
              <el-tag size="small">
                {{
                  selectedDetailTool.effect === "read"
                    ? t("system.ai.workspace.read")
                    : t("system.ai.workspace.requires_confirmation")
                }}
              </el-tag>
            </div>
            <el-text class="plugin-tool-preview__description">{{ selectedDetailTool.description || "-" }}</el-text>
            <span class="plugin-tool-preview__label">{{ t("system.ai.workspace.schema") }}</span>
            <pre>{{ selectedDetailTool.input_schema || "-" }}</pre>
          </main>
          <el-empty v-else-if="!detailLoading" :description="t('system.ai.workspace.no_matches')" />
        </div>
      </div>
    </el-drawer>
    <FormDialog
      ref="formRef"
      v-model="formOpen"
      class="plugin-form-dialog"
      :title="dialogTitle"
      :model="form"
      :fields="fields"
      :confirm-loading="saving || authorizing"
      label-width="auto"
      width="min(760px, 96vw)"
      @confirm="saveForm"
      @closed="resetForm"
    >
      <template #schema>
        <pre class="tool-schema">{{ editingTool?.input_schema }}</pre>
      </template>
    </FormDialog>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ChatDotRound, Key, Refresh, Share, SwitchButton, View } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { encryptSecret } from "@liujitcn/kratos-admin-core/security";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defAiPluginService as api } from "@liujitcn/kratos-admin-system/api/base/v1/ai_plugin";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import type {
  AiPlugin,
  AiPluginConnection,
  AiPluginForm,
  AiPluginSelection,
  AiPluginTool
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";
import { canConfigureAssistant, hasConnectedPluginConnection } from "./state";
const props = defineProps<{
  plugins: AiPlugin[];
  connections: AiPluginConnection[];
  mobile: boolean;
  loading: boolean;
  error: string;
  canManageTools: boolean;
  canManageCatalog: boolean;
  canCreatePrivate: boolean;
}>();
const emit = defineEmits<{ refresh: []; select: [value: AiPluginSelection] }>();
const tab = ref("catalog"),
  keyword = ref(""),
  connectionKeyword = ref(""),
  toolKeyword = ref(""),
  scope = ref("");
const managedPlugins = computed(() => props.plugins);
const canManage = computed(() => canConfigureAssistant(props.mobile, props.canManageTools));
const connectedPluginIDs = computed(
  () =>
    new Set(props.plugins.filter(plugin => hasConnectedPluginConnection(plugin.id, props.connections)).map(plugin => plugin.id))
);
const visiblePlugins = computed(() =>
  props.plugins.filter(
    item =>
      (!scope.value || item.visibility === scope.value) &&
      `${item.name} ${item.description}`.toLowerCase().includes(keyword.value.toLowerCase())
  )
);
const toolPluginID = ref(""),
  toolError = ref("");
const detail = ref<AiPlugin>(),
  detailOpen = ref(false),
  detailToolKeyword = ref(""),
  selectedDetailToolId = ref("");
const detailTools = ref<AiPluginTool[]>([]),
  detailTotal = ref(0),
  detailPageNum = ref(1),
  detailLoading = ref(false),
  detailError = ref("");
const toolsTable = ref<InstanceType<typeof ProTable>>();
const toolTableInitParam = { page_size: 20 };
const detailPageSize = 20;
const formOpen = ref(false),
  formRef = ref<InstanceType<typeof FormDialog>>(),
  saving = ref(false);
const kind = ref<"definition" | "connection" | "credential" | "policy" | "installation">("connection");
const editingConnection = ref<AiPluginConnection>();
const editingPlugin = ref<AiPlugin>(),
  editingTool = ref<AiPluginTool>();
const form = ref<Record<string, any>>({});
const authorizing = ref(false);
let authorizationPopup: Window | null = null;
let authorizationTimer: ReturnType<typeof setInterval> | undefined;
let detailRequestSerial = 0;

const dialogTitle = computed(() => t(`system.ai.workspace.${kind.value}`));
const visibleConnections = computed(() =>
  props.connections.filter(item =>
    `${item.label} ${props.plugins.find(plugin => plugin.id === item.plugin_id)?.name ?? ""}`
      .toLowerCase()
      .includes(connectionKeyword.value.toLowerCase())
  )
);
const selectedDetailTool = computed(
  () => detailTools.value.find(item => item.id === selectedDetailToolId.value) ?? detailTools.value[0]
);
const connectionColumns = computed<ColumnProps[]>(() => [
  {
    prop: "plugin_id",
    label: t("system.ai.workspace.plugins"),
    render: ({ row }) => props.plugins.find(item => item.id === row.plugin_id)?.name || row.plugin_id
  },
  { prop: "label", label: t("system.ai.workspace.account") },
  { prop: "sharing", label: t("system.ai.workspace.scope"), render: ({ row }) => t(`system.ai.workspace.${row.sharing}`) },
  {
    prop: "state",
    label: t("system.ai.workspace.connection_state"),
    render: ({ row }) => t(`system.ai.workspace.connection_${row.state}`)
  },
  { prop: "scopes", label: t("system.ai.workspace.authorization_scope") },
  { prop: "last_error", label: t("system.ai.workspace.reason") },
  {
    cellType: "actions",
    label: t("common.field.operation"),
    width: 300,
    actions: [
      {
        label: t("system.ai.workspace.use_chat"),
        icon: ChatDotRound,
        hidden: ({ row }) => row.state !== "connected",
        onClick: ({ row }) => emit("select", { plugin_id: row.plugin_id, connection_id: row.id })
      },
      {
        label: t("system.ai.workspace.refresh_tools"),
        icon: Refresh,
        hidden: ({ row }) => props.mobile || !row.can_manage,
        onClick: ({ row }) => refreshConnection(row as AiPluginConnection)
      },
      {
        label: t("system.ai.workspace.authorize"),
        icon: Key,
        hidden: ({ row }) => props.mobile || !row.can_manage || !["oauth", "token"].includes(row.auth_type),
        onClick: ({ row }) =>
          row.auth_type === "oauth" ? authorize(row as AiPluginConnection) : openCredential(row as AiPluginConnection)
      },
      {
        label: t("system.ai.workspace.share"),
        icon: Share,
        hidden: ({ row }) => props.mobile || !props.canManageTools || !row.can_manage || row.sharing === "tenant",
        onClick: ({ row }) => setSharing(row as AiPluginConnection, "tenant")
      },
      {
        label: t("system.ai.workspace.unshare"),
        icon: Share,
        hidden: ({ row }) => props.mobile || !props.canManageTools || !row.can_manage || row.sharing !== "tenant",
        onClick: ({ row }) => setSharing(row as AiPluginConnection, "private")
      },
      {
        label: t("system.ai.workspace.disconnect"),
        icon: SwitchButton,
        type: "danger",
        hidden: ({ row }) => props.mobile || !row.can_manage,
        onClick: ({ row }) => disconnect(row as AiPluginConnection)
      }
    ]
  }
]);
const toolColumns = computed<ColumnProps[]>(() => [
  { prop: "source_name", label: t("system.ai.workspace.name") },
  { prop: "description", label: t("system.ai.workspace.description") },
  {
    prop: "allowed",
    label: t("system.ai.workspace.configured"),
    render: ({ row }) => (row.allowed ? t("system.ai.workspace.enabled") : t("system.ai.workspace.disabled"))
  },
  {
    prop: "effective_enabled",
    label: t("system.ai.workspace.effective"),
    render: ({ row }) => (row.effective_enabled ? t("system.ai.workspace.enabled") : t("system.ai.workspace.disabled"))
  },
  {
    prop: "effect",
    label: t("system.ai.workspace.effect"),
    render: ({ row }) => (row.revision === 0 ? t("system.ai.workspace.pending_review") : t(`system.ai.workspace.${row.effect}`))
  },
  { prop: "unavailable_reason", label: t("system.ai.workspace.reason") },
  {
    cellType: "actions",
    label: t("common.field.operation"),
    actions: [{ label: t("system.ai.workspace.review"), icon: View, onClick: ({ row }) => openPolicy(row as AiPluginTool) }]
  }
]);
const fields = computed<ProFormField[]>(() => {
  const required = [{ required: true, message: t("system.ai.workspace.required"), trigger: "blur" }];
  if (kind.value === "credential")
    return [
      {
        prop: "token",
        label: t("system.ai.workspace.token"),
        component: "password",
        props: { autocomplete: "new-password" },
        rules: required
      }
    ];
  if (kind.value === "connection")
    return [
      { prop: "label", label: t("system.ai.workspace.account"), component: "input", rules: required },
      {
        prop: "auth_type",
        label: t("system.ai.workspace.auth_type"),
        component: "select",
        options: ["none", "token", "oauth"].map(value => ({ label: value, value }))
      },
      {
        prop: "token",
        label: t("system.ai.workspace.token"),
        component: "password",
        visible: model => model.auth_type === "token",
        props: { autocomplete: "new-password" },
        rules: required
      }
    ];
  if (kind.value === "installation")
    return [
      { prop: "enabled", label: t("system.ai.workspace.enabled"), component: "switch" },
      {
        prop: "terminals",
        label: t("system.ai.workspace.terminals"),
        component: "checkbox-group",
        options: ["admin", "app"].map(value => ({ label: value, value }))
      }
    ];
  if (kind.value === "policy")
    return [
      { prop: "schema", label: t("system.ai.workspace.schema"), component: "slot", slotName: "schema" },
      { prop: "allowed", label: t("system.ai.workspace.enabled"), component: "switch" },
      {
        prop: "effect",
        label: t("system.ai.workspace.effect"),
        component: "select",
        options: ["read", "write", "unknown"].map(value => ({ label: t(`system.ai.workspace.${value}`), value }))
      },
      {
        prop: "prompts",
        label: t("system.ai.workspace.prompts"),
        component: "dynamic-list",
        props: { inputProps: { maxlength: 2000, type: "textarea" } }
      }
    ];
  return [
    {
      prop: "code",
      label: t("system.ai.workspace.code"),
      component: "input",
      rules: required,
      props: { disabled: !!editingPlugin.value }
    },
    { prop: "name", label: t("system.ai.workspace.name"), component: "input", rules: required },
    { prop: "description", label: t("system.ai.workspace.description"), component: "textarea" },
    {
      prop: "visibility",
      label: t("system.ai.workspace.scope"),
      component: "select",
      options: ["private", "tenant", "system"].map(value => ({ label: t(`system.ai.workspace.${value}`), value })),
      props: { disabled: !props.canManageCatalog }
    },
    {
      prop: "endpoint",
      label: t("system.ai.workspace.endpoint"),
      component: "input",
      visible: model => model.source === "MCP",
      rules: required
    },
    {
      prop: "config_json",
      label: t("system.ai.workspace.public_config"),
      component: "textarea",
      visible: () => props.canManageCatalog,
      props: { rows: 4 }
    },
    { prop: "enabled", label: t("system.ai.workspace.enabled"), component: "switch" }
  ];
});
/** 读取工具配置页当前分页，搜索和分页均由服务端处理。 */
async function requestToolsTable(params: Record<string, unknown>) {
  if (!toolPluginID.value) return { data: { list: [], total: 0 } };
  const pageNum = Number(params.page_num) || 1;
  const pageSize = Number(params.page_size) || toolTableInitParam.page_size;
  const response = await api.ListAiPluginTool({
    plugin_id: toolPluginID.value,
    terminal: Terminal.TERMINAL_ADMIN,
    keyword: toolKeyword.value,
    page_num: pageNum,
    page_size: pageSize
  });
  return { data: { list: response.tools ?? [], total: response.total } };
}
/** 重新加载工具配置页并回到第一页。 */
function reloadToolsTable() {
  toolError.value = "";
  toolsTable.value?.reset();
}
/** 展示工具配置页请求错误。 */
function handleToolsError(error: unknown) {
  toolError.value = error instanceof Error ? error.message : t("system.ai.workspace.failed");
}
/** 读取详情抽屉当前页工具，避免一次加载全部 Schema。 */
async function loadDetailTools(pageNum = detailPageNum.value) {
  if (!detail.value) return;
  const requestSerial = ++detailRequestSerial;
  detailPageNum.value = pageNum;
  detailLoading.value = true;
  detailError.value = "";
  try {
    const response = await api.ListAiPluginTool({
      plugin_id: detail.value.id,
      terminal: Terminal.TERMINAL_ADMIN,
      keyword: detailToolKeyword.value,
      page_num: detailPageNum.value,
      page_size: detailPageSize
    });
    if (requestSerial !== detailRequestSerial) return;
    detailTools.value = response.tools ?? [];
    detailTotal.value = response.total;
  } catch (error) {
    if (requestSerial !== detailRequestSerial) return;
    detailTools.value = [];
    detailTotal.value = 0;
    detailError.value = error instanceof Error ? error.message : t("system.ai.workspace.failed");
  } finally {
    if (requestSerial === detailRequestSerial) detailLoading.value = false;
  }
}
/** 重置详情搜索并请求第一页。 */
function searchDetailTools() {
  detailPageNum.value = 1;
  void loadDetailTools();
}
/** 打开插件详情并按页加载工具快照。 */
async function openDetail(plugin: AiPlugin) {
  detail.value = plugin;
  detailTools.value = [];
  detailTotal.value = 0;
  detailPageNum.value = 1;
  detailToolKeyword.value = "";
  selectedDetailToolId.value = "";
  detailError.value = "";
  detailOpen.value = true;
  await loadDetailTools();
}
/** 打开本人账号连接表单，已有连接时不重复打开。 */
function openConnection(plugin: AiPlugin) {
  if (props.mobile || connectedPluginIDs.value.has(plugin.id)) return;
  editingPlugin.value = plugin;
  kind.value = "connection";
  form.value = { label: "", auth_type: "none", token: "" };
  formOpen.value = true;
}
/** 打开插件定义表单，个人新增固定私有MCP。 */
function openDefinition(plugin?: AiPlugin) {
  if (props.mobile) return;
  editingPlugin.value = plugin;
  kind.value = "definition";
  form.value = plugin
    ? {
        code: plugin.code,
        name: plugin.name,
        description: plugin.description,
        source: plugin.source,
        visibility: plugin.visibility,
        endpoint: plugin.endpoint,
        enabled: plugin.enabled,
        config_json: plugin.config_json
      }
    : { code: "", name: "", description: "", source: "MCP", visibility: "private", endpoint: "", enabled: true, config_json: "" };
  formOpen.value = true;
}
/** 打开当前租户安装策略。 */
function openInstallation(plugin: AiPlugin) {
  editingPlugin.value = plugin;
  kind.value = "installation";
  form.value = { enabled: plugin.installation?.enabled ?? false, terminals: [...(plugin.installation?.terminals ?? ["admin"])] };
  formOpen.value = true;
}
/** 冻结待审查的工具契约与版本。 */
function openPolicy(tool: AiPluginTool) {
  editingTool.value = tool;
  kind.value = "policy";
  form.value = { allowed: tool.allowed, effect: tool.effect || "unknown", prompts: [...(tool.prompts ?? [])] };
  formOpen.value = true;
}
/** 提交配置只采用真实接口结果，版本冲突由服务端返回。 */
async function saveForm() {
  if (props.mobile || !(await formRef.value?.validate())) return;
  saving.value = true;
  try {
    if (kind.value === "credential") {
      const connection = editingConnection.value!;
      await api.ReauthorizeAiPluginConnection({
        id: connection.id,
        revision: connection.revision,
        token: await encryptSecret(form.value.token)
      });
    } else if (kind.value === "connection") {
      const connection = await api.CreateAiPluginConnection({
        plugin_id: editingPlugin.value!.id,
        label: form.value.label,
        auth_type: form.value.auth_type,
        token: form.value.auth_type === "token" ? await encryptSecret(form.value.token) : undefined
      });
      if (connection.auth_type === "oauth") await authorize(connection);
    } else if (kind.value === "definition") {
      const definition = form.value as AiPluginForm;
      if (editingPlugin.value)
        await api.UpdateAiPlugin({ id: editingPlugin.value.id, revision: editingPlugin.value.revision, form: definition });
      else await api.CreateAiPlugin({ form: definition });
    } else if (kind.value === "installation") {
      const plugin = editingPlugin.value!;
      await api.SetAiPluginInstallation({
        plugin_id: plugin.id,
        revision: plugin.installation?.revision ?? 0,
        enabled: form.value.enabled,
        terminals: form.value.terminals
      });
    } else {
      const tool = editingTool.value!;
      await api.SetAiPluginToolPolicy({
        plugin_id: tool.plugin_id,
        tool_id: tool.id,
        contract_hash: tool.contract_hash,
        revision: tool.revision,
        allowed: form.value.allowed,
        effect: form.value.effect,
        prompts: form.value.prompts
      });
      await toolsTable.value?.getTableList();
    }
    formOpen.value = false;
    emit("refresh");
  } finally {
    saving.value = false;
  }
}
/** 关闭时清除凭据、编辑对象及校验状态。 */
function resetForm() {
  form.value = {};
  editingPlugin.value = undefined;
  editingConnection.value = undefined;
  editingTool.value = undefined;
  formRef.value?.clearValidate();
}
/** 检测连接并重新获取完整工具快照。 */
async function refreshConnection(connection: AiPluginConnection) {
  await api.RefreshAiPluginConnection({ id: connection.id, revision: connection.revision });
  emit("refresh");
}
/** 在原连接上更换令牌，保留标签和会话引用。 */
function openCredential(connection: AiPluginConnection) {
  if (props.mobile || !connection.can_manage) return;
  editingConnection.value = connection;
  kind.value = "credential";
  form.value = { token: "" };
  formOpen.value = true;
}

/** 共享仅允许有管理权限的账号拥有者显式确认。 */
async function setSharing(connection: AiPluginConnection, sharing: string) {
  if (props.mobile || !props.canManageTools || !connection.can_manage) return;
  await ElMessageBox.confirm(
    t("system.ai.workspace.sharing_confirm", { name: connection.label }),
    t("system.ai.workspace.scope"),
    { type: "warning" }
  );
  await api.SetAiPluginConnectionSharing({ id: connection.id, revision: connection.revision, sharing });
  emit("refresh");
}

/** 弹窗承接真实OAuth回调，主窗口轮询服务端连接状态。 */
async function authorize(connection: AiPluginConnection) {
  if (props.mobile || authorizing.value) return;
  authorizationPopup = window.open("", "_blank", "popup,width=600,height=740");
  if (!authorizationPopup) {
    ElMessage.warning(t("system.ai.workspace.popup_blocked"));
    return;
  }
  authorizationPopup.sessionStorage.setItem("ai-plugin-oauth-popup", "1");
  authorizationPopup.opener = null;
  authorizing.value = true;
  try {
    const result = await api.StartAiPluginOAuth({ id: connection.id, revision: connection.revision });
    authorizationPopup.location.assign(result.authorization_url);
    const startedAt = Date.now();
    let checking = false;
    authorizationTimer = setInterval(async () => {
      if (checking) return;
      if (!authorizationPopup || authorizationPopup.closed || Date.now() - startedAt > 5 * 60 * 1000) {
        stopAuthorization();
        emit("refresh");
        return;
      }
      checking = true;
      try {
        const response = await api.ListAiPluginConnection({ plugin_id: connection.plugin_id });
        const current = response.connections?.find(item => item.id === connection.id);
        if (current && current.revision > connection.revision && current.state === "connected") {
          authorizationPopup.close();
          stopAuthorization();
          formOpen.value = false;
          emit("refresh");
        } else if (current && current.revision > connection.revision && current.last_error) {
          ElMessage.warning(current.last_error);
          stopAuthorization();
          emit("refresh");
        }
      } catch {
        stopAuthorization();
        emit("refresh");
      } finally {
        checking = false;
      }
    }, 2000);
  } catch (error) {
    authorizationPopup?.close();
    stopAuthorization();
    throw error;
  }
}

/** 结束轮询，不在客户端伪造连接成功状态。 */
function stopAuthorization() {
  clearInterval(authorizationTimer);
  authorizationTimer = undefined;
  authorizing.value = false;
  authorizationPopup = null;
}
/** 断开连接前说明历史消息仍然保留。 */
async function disconnect(connection: AiPluginConnection) {
  await ElMessageBox.confirm(
    t("system.ai.workspace.disconnect_confirm", { name: connection.label }),
    t("system.ai.workspace.disconnect"),
    { type: "warning" }
  );
  await api.DisconnectAiPluginConnection({ id: connection.id, revision: connection.revision });
  emit("refresh");
}
/** 删除定义保留服务端引用保护。 */
async function removePlugin(plugin: AiPlugin) {
  await ElMessageBox.confirm(t("system.ai.workspace.delete_confirm", { name: plugin.name }), t("common.action.delete"), {
    type: "warning"
  });
  await api.DeleteAiPlugin({ id: plugin.id, revision: plugin.revision });
  emit("refresh");
}
onBeforeUnmount(() => {
  clearInterval(authorizationTimer);
});

watch(
  [tab, managedPlugins],
  ([currentTab, plugins]) => {
    if (currentTab !== "tools") return;
    if (!plugins.length) {
      toolPluginID.value = "";
      toolError.value = "";
      return;
    }
    const selectedPluginID = plugins.some(plugin => plugin.id === toolPluginID.value) ? toolPluginID.value : plugins[0].id;
    const changed = selectedPluginID !== toolPluginID.value;
    toolPluginID.value = selectedPluginID;
    if (changed) reloadToolsTable();
  },
  { immediate: true }
);

watch(detailTools, items => {
  if (!items.some(item => item.id === selectedDetailToolId.value)) selectedDetailToolId.value = items[0]?.id ?? "";
});

watch(
  () => props.mobile,
  mobile => {
    if (!mobile) return;
    formOpen.value = false;
    if (tab.value === "tools") tab.value = "catalog";
  }
);
</script>
<style scoped lang="scss">
.assistant-workspace {
  min-width: 0;
  min-height: 0;
  padding: 20px;
  overflow: auto;
}
.workspace-toolbar,
.plugin-card-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
h3 {
  margin: 0;
  font-size: 16px;
}
p {
  color: var(--admin-page-text-secondary);
  line-height: 1.7;
  white-space: pre-wrap;
}
.workspace-query {
  max-width: 320px;
  margin: 12px 0;
}
.workspace-toolbar {
  margin: 14px 0;
}
.workspace-toolbar > .el-input {
  width: 240px;
}
.workspace-toolbar > .el-select {
  width: 150px;
}
.plugin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(300px, 100%), 1fr));
  gap: 14px;
}
.plugin-card {
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  padding: 16px;
  background: var(--admin-page-card-bg);
}
.plugin-card .workspace-toolbar > .el-button {
  margin-left: 0;
}
.plugin-card-title {
  justify-content: space-between;
}
:global(.plugin-form-dialog.el-dialog) {
  max-height: calc(100dvh - 32px);
  margin: 16px auto;
  overflow: hidden;
}
:global(.plugin-form-dialog.el-dialog .el-dialog__body) {
  max-height: calc(100dvh - 160px);
  overflow-y: auto;
}
.plugin-detail-heading {
  display: grid;
  gap: 4px;
  min-width: 0;
}
.plugin-detail-heading__title,
.plugin-tool-nav__header,
.plugin-tool-preview__heading {
  display: flex;
  align-items: center;
  gap: 8px;
}
.plugin-detail-heading__title strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.plugin-detail-content {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 14px;
  min-height: 100%;
}
.plugin-detail-meta {
  display: grid;
  gap: 8px;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}
.plugin-detail-layout {
  display: grid;
  grid-template-columns: minmax(220px, 30%) minmax(0, 1fr);
  position: relative;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
}
.plugin-tool-nav {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: var(--el-fill-color-light);
  border-right: 1px solid var(--el-border-color);
}
.plugin-tool-nav__header {
  justify-content: space-between;
}
.plugin-tool-nav :deep(.el-pagination) {
  justify-content: center;
}
.plugin-tool-nav__items {
  display: grid;
  min-height: 0;
  gap: 4px;
  overflow-y: auto;
  align-content: start;
}
.plugin-tool-nav__item {
  display: grid;
  gap: 4px;
  width: 100%;
  padding: 10px;
  color: var(--admin-page-text-regular);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--admin-page-radius);
  cursor: pointer;
}
.plugin-tool-nav__item:hover,
.plugin-tool-nav__item.is-active {
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  border-color: var(--el-color-primary-light-5);
}
.plugin-tool-nav__item:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: -2px;
}
.plugin-tool-nav__item span {
  display: -webkit-box;
  overflow: hidden;
  color: var(--admin-page-text-secondary);
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.plugin-tool-preview {
  min-width: 0;
  padding: 20px;
  overflow: auto;
}
.plugin-tool-preview__heading {
  justify-content: space-between;
}
.plugin-tool-preview__description {
  display: block;
  margin-top: 12px;
  white-space: pre-wrap;
}
.plugin-tool-preview__label {
  display: block;
  margin-top: 22px;
  color: var(--admin-page-text-secondary);
}
pre {
  max-height: none;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--admin-page-text-regular);
  background: var(--el-fill-color-light);
  padding: 12px;
  border-radius: var(--admin-page-radius);
}
:global(.plugin-detail-drawer .el-drawer__body) {
  min-height: 0;
  overflow: hidden;
}
@media (width <= 768px) {
  .assistant-workspace {
    padding: 12px;
  }
  .plugin-detail-layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(220px, 42%) minmax(0, 1fr);
  }
  .plugin-tool-nav {
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
}
</style>
