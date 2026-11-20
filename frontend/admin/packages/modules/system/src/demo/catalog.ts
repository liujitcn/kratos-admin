import type { AdminComponentApi, AdminComponentDoc } from "@liujitcn/kratos-admin-core";
import componentApi from "./component-api.json";

const selectionExample = () => import("./SelectionExample.vue");
const docs: Omit<AdminComponentDoc, "owner" | "group" | "api">[] = [
  {
    id: "AiModelSelector",
    label: "AI 模型选择",
    file: "AiModelSelector/index.vue",
    description: "按供应商展示可用模型，并同步当前供应商编号和模型名称。",
    usage:
      "传入供应商模型目录，监听 change(providerId, modelName) 更新父页面状态。组件本身不加载模型，也不发送对话；管理入口随当前账号的菜单权限显示。示例目录保存在本地。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import AiModelSelector from "@liujitcn/kratos-admin-system/components/AiModelSelector/index.vue";

const providerId = ref(1);
const modelName = ref("demo-chat");
const providers = [{ provider_id: 1, name: "演示供应商", models: [
  { model_name: "demo-chat", display_name: "通用对话模型" }
] }];
function changeModel(id: number, name: string) {
  providerId.value = id;
  modelName.value = name;
}
</script>

<template>
  <AiModelSelector :model-providers="providers" :provider-id="providerId"
    :model-name="modelName" @change="changeModel" />
</template>`,
    example: selectionExample,
    related: ["AiSessionPanel", "Ai"]
  },
  {
    id: "AiSessionPanel",
    label: "AI 会话列表",
    file: "AiSessionPanel/index.vue",
    description: "展示会话搜索、活动会话、新建入口和编辑菜单，由父页面处理数据。",
    usage:
      "active、keyword 使用具名 v-model；sessions 应由父页面按关键词过滤。create、action、change、toggleCollapse 仅通知父页面，组件不请求服务。editable=false 可隐藏编辑菜单，hideCollapse 可隐藏折叠按钮。示例的重命名与删除只更新内存列表。",
    code: `<script setup lang="ts">
import { computed, ref } from "vue";
import AiSessionPanel from "@liujitcn/kratos-admin-system/components/AiSessionPanel/index.vue";

const active = ref("demo-1");
const keyword = ref("");
const sessions = ref([{ id: "demo-1", title: "本周服务概览", summary: "分析业务变化" }]);
const filtered = computed(() => sessions.value.filter(item => item.title.includes(keyword.value)));
</script>

<template>
  <AiSessionPanel v-model:active="active" v-model:keyword="keyword"
    :sessions="filtered" :editable="false" hide-collapse />
</template>`,
    example: () => import("./SessionExample.vue"),
    related: ["AiModelSelector"]
  },
  {
    id: "Ai",
    label: "AI 入口",
    file: "ai/Ai.vue",
    description: "管理端顶部的 AI 助手入口，按功能开关和当前账号菜单权限显示。",
    usage:
      "System 模块已经将该组件注册为 headerTools，无需在业务页面重复挂载。组件没有业务 props；依赖 configStore.aiEnabled 与名为 AiChat 的授权菜单，点击后进入真实助手页面，因此这里提供注册说明。",
    code: `import { systemAdminModule } from "@liujitcn/kratos-admin-system";

// 将 System 模块交给应用的模块配置，即可注册 AI 顶部入口。
const modules = [systemAdminModule];`,
    related: ["AiModelSelector", "AiSessionPanel"]
  },
  {
    id: "AiFlowBlocks",
    label: "AI 流程块",
    file: "ai/AiFlowBlocks.vue",
    description: "根据模块注册的渲染器展示助手消息中的结构化业务流程块。",
    usage:
      "由聊天面板传入 message、activeFlowMessageId 与 renderers；按渲染器声明的类型筛选 blocks，flow-action 交回父页面处理。组件只负责展示和转发动作，不执行工具或提交业务请求。",
    code: `<script setup lang="ts">
import AiFlowBlocks from "@liujitcn/kratos-admin-system/components/ai/AiFlowBlocks.vue";

const props = defineProps<InstanceType<typeof AiFlowBlocks>["$props"]>();
</script>

<template>
  <AiFlowBlocks v-bind="props" />
</template>`,
    related: ["ChatPanel"]
  },
  {
    id: "ChatPanel",
    label: "AI 聊天面板",
    file: "ai/ChatPanel.vue",
    description: "管理端 AI 聊天展示入口，组合消息、模型选择、知识库选择和输入框。",
    usage:
      "从公开 components 路径导入，契约与内部聊天面板完全相同。父页面维护会话、消息、草稿和发送状态，并处理 submit、messageAction、messageEdit、approvalDecision 等事件；面板本身不发起模型请求。示例透传父级提供的 props 与事件处理器，不创建真实会话。",
    code: `<script setup lang="ts">
import ChatPanel from "@liujitcn/kratos-admin-system/components/ai/ChatPanel.vue";

const props = defineProps<InstanceType<typeof ChatPanel>["$props"]>();
</script>

<template>
  <ChatPanel v-bind="props" />
</template>`,
    related: ["AiModelSelector", "AiSessionPanel", "AiFlowBlocks"]
  },
  {
    id: "AutomationPanel",
    label: "AI 自动化工作区",
    file: "ai/workspace/AutomationPanel.vue",
    description: "管理本人 AI 自动化任务、执行记录、审批和结果会话。",
    usage:
      "父页面提供 active、mobile、当前模型、知识库和插件选择。激活后读取真实任务；运行、停止、审批和计划修改会调用业务接口，移动模式限制计划管理。pending、result、session 事件交由父页面更新徽标、定位结果或切换会话，因此文档不自动挂载。",
    code: `<script setup lang="ts">
import AutomationPanel from "@liujitcn/kratos-admin-system/components/ai/workspace/AutomationPanel.vue";

const props = defineProps<InstanceType<typeof AutomationPanel>["$props"]>();
</script>

<template>
  <AutomationPanel v-bind="props" />
</template>`,
    related: ["ChatPanel", "PluginPicker"]
  },
  {
    id: "FilePanel",
    label: "AI 文件工作区",
    file: "ai/workspace/FilePanel.vue",
    description: "展示当前账号可访问的 AI 文件，提供预览、下载和回收站操作。",
    usage:
      "无需 props，挂载后会读取真实文件列表。重命名、移入回收站、恢复和彻底删除都会修改后端数据，服务端继续校验租户及文件访问边界；仅在真实 AI 工作区挂载，文档不重复加载业务文件。",
    code: `<script setup lang="ts">
import FilePanel from "@liujitcn/kratos-admin-system/components/ai/workspace/FilePanel.vue";
</script>

<template>
  <FilePanel />
</template>`,
    related: ["ChatPanel", "ProTable"]
  },
  {
    id: "PluginPanel",
    label: "AI 插件工作区",
    file: "ai/workspace/PluginPanel.vue",
    description: "展示插件目录和连接账号，并提供授权、工具刷新及使用入口。",
    usage:
      "父页面传入目录、连接、加载状态和当前管理能力；refresh 重新读取数据，select 将插件选择交回聊天页。canManageTools、canManageCatalog、canCreatePrivate 只控制界面，不能替代服务端鉴权；授权和共享等操作会修改真实连接，文档不自动挂载。",
    code: `<script setup lang="ts">
import PluginPanel from "@liujitcn/kratos-admin-system/components/ai/workspace/PluginPanel.vue";

const props = defineProps<InstanceType<typeof PluginPanel>["$props"]>();
</script>

<template>
  <PluginPanel v-bind="props" />
</template>`,
    related: ["PluginPicker", "ChatPanel"]
  },
  {
    id: "PluginPicker",
    label: "AI 插件选择器",
    file: "ai/workspace/PluginPicker.vue",
    description: "按插件与连接账号选择聊天使用的工具集，显示不可用原因。",
    usage:
      "v-model 保存 plugin_id 与可选 connection_id；plugins、connections 由父页面加载。组件不自行授权或执行工具；失效的已有选择会保留并标记为不可用，便于用户手动移除。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import PluginPicker from "@liujitcn/kratos-admin-system/components/ai/workspace/PluginPicker.vue";
import type { AiPlugin, AiPluginConnection, AiPluginSelection } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";

const props = defineProps<{ plugins: AiPlugin[]; connections: AiPluginConnection[] }>();
const selected = ref<AiPluginSelection[]>([]);
</script>

<template>
  <PluginPicker v-model="selected" :plugins="props.plugins" :connections="props.connections" />
</template>`,
    related: ["PluginPanel", "AutomationPanel"]
  },
  {
    id: "DynamicI18nEditor",
    label: "动态多语言编辑",
    file: "i18n/DynamicI18nEditor.vue",
    description: "在字段旁打开译文草稿弹窗，确认后把译文交给父表单。",
    usage:
      "v-model 的每项包含 locale、id、text；source 是原文，multiline 与 maxlength 控制输入。组件读取共享语言缓存，需要宿主先加载已启用的非主语言目录，否则入口不显示。翻译按钮会调用草稿翻译服务，确认只更新本地值，持久化由主表单负责。因语言缓存及翻译服务依赖，这里提供集成示例。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import DynamicI18nEditor from "@liujitcn/kratos-admin-system/components/i18n/DynamicI18nEditor.vue";
import type { DynamicI18nValue } from "@liujitcn/kratos-admin-system/components/i18n/dynamicI18n";

const values = ref<DynamicI18nValue[]>([]);
</script>

<template>
  <DynamicI18nEditor v-model="values" source="经营日报" :maxlength="100" />
</template>`,
    related: ["DynamicI18nCell", "ProForm"]
  },
  {
    id: "DynamicI18nCell",
    label: "多语言单元格",
    file: "i18n/DynamicI18nCell.vue",
    description: "在列表中显示当前语言的字段译文，并按资源字段和记录编号保存修改。",
    usage:
      "source 为原文，i18ns 使用 BaseI18n 记录数组，targetKey 为资源字段键，targetId 为实际记录编号。editable=false 只读查看；可编辑状态确认会直接调用 UpdateBaseI18n。内部复用 DynamicI18nEditor，依赖宿主初始化的共享语言缓存，这里提供接入代码。",
    code: `<DynamicI18nCell
  :source="row.name"
  target-key="base_role.name"
  :target-id="row.id"
  :i18ns="row.i18ns"
  :editable="canEdit"
/>

<!-- 组件从 @liujitcn/kratos-admin-system/components/i18n/DynamicI18nCell.vue 导入。 -->`,
    related: ["DynamicI18nEditor", "ProTable"]
  },
  {
    id: "TenantProjectSelect",
    label: "租户项目选择",
    file: "tenant-project/TenantProjectSelect.vue",
    description: "从租户项目树中选择范围，返回节点值与租户、项目编号。",
    usage:
      "v-model 接收 TenantProjectSelection，节点值采用 tenant:租户ID 或 project:租户ID:项目ID。显式传 options 时不加载目录；不传则在挂载时请求项目树。组件向 el-tree-select 透传属性，支持清空、筛选和禁用。示例完整传入本地 options。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import TenantProjectSelect, { type TenantProjectSelection } from "@liujitcn/kratos-admin-system/components/tenant-project/TenantProjectSelect.vue";

const selection = ref<TenantProjectSelection>();
const options = [{ value: "tenant:1", label: "演示租户", type: "tenant",
  tenant_id: 1, project_id: 0, disabled: false, children: [
    { value: "project:1:11", label: "校园服务", type: "project",
      tenant_id: 1, project_id: 11, disabled: false, children: [] }
  ] }];
</script>

<template>
  <TenantProjectSelect v-model="selection" :options="options" />
</template>`,
    example: selectionExample,
    related: ["TenantProjectText", "TenantProjectManager"]
  },
  {
    id: "TenantProjectText",
    label: "租户项目文本",
    file: "tenant-project/TenantProjectText.vue",
    description: "按当前租户上下文展示项目名称，默认租户下补充所属租户。",
    usage:
      "优先传 row 中的 project_name、tenant_name，也可用 projectField、tenantField 映射字段；仅有编号时可传 projectId、tenantId。组件挂载时会主动加载共享项目目录，即使行内已有名称仍会加载，因此此处使用集成代码说明。",
    code: `<script setup lang="ts">
import { TenantProjectText } from "@liujitcn/kratos-admin-system";

const row = { tenant_id: 1, tenant_name: "演示租户", project_id: 11, project_name: "校园服务" };
</script>

<template>
  <TenantProjectText :row="row" />
</template>`,
    related: ["TenantProjectSelect", "TenantProjectManager"]
  },
  {
    id: "TenantProjectManager",
    label: "租户项目管理",
    file: "tenant-project/TenantProjectManager.vue",
    description: "封装项目查询、创建、编辑、删除和状态操作，并提供业务扩展列与操作。",
    usage:
      "该组件自带真实项目服务与权限判断。extraColumns 可通过 after 指定基础列之后的位置；loadExtraData 接收当前页项目键和查询参数，extraActions 接收单条 TenantProjectContext。自定义具名插槽附带 tenant-project 上下文。这里仅展示接入，不执行项目维护。",
    code: `<script setup lang="ts">
import { TenantProjectManager } from "@liujitcn/kratos-admin-system";
</script>

<template>
  <TenantProjectManager />
</template>`,
    related: ["TenantProjectSelect", "TenantProjectText", "ProjectGrantDialog", "ProTable"]
  },
  {
    id: "ProjectGrantDialog",
    label: "项目授权弹窗",
    file: "tenant-project-grant/ProjectGrantDialog.vue",
    description: "按岗位、角色、部门或用户主体配置指定租户下的直接项目授权。",
    usage:
      "通过实例 open(subjectType, subjectId, tenantId) 打开，不使用 v-model。打开会加载真实租户、项目及已有授权，确认直接保存授权；“全部项目”以 [0] 表示。只应从已完成权限检查的业务入口调用，文档页不挂载授权表单。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import ProjectGrantDialog from "@liujitcn/kratos-admin-system/components/tenant-project-grant/ProjectGrantDialog.vue";
import { BaseTenantProjectGrantSubjectType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project_grant";

const grant = ref<InstanceType<typeof ProjectGrantDialog>>();
// 在授权业务入口，用实际角色编号和租户编号调用：
// await grant.value?.open(BaseTenantProjectGrantSubjectType.BASE_TENANT_PROJECT_GRANT_SUBJECT_TYPE_ROLE, roleId, tenantId);
</script>

<template>
  <ProjectGrantDialog ref="grant" />
</template>`,
    related: ["TenantProjectManager", "FormDialog"]
  },
  {
    id: "ApiTransfer",
    label: "API 穿梭选择",
    file: "api/ApiTransfer.vue",
    description: "以双面板选择 API 操作标识，支持筛选、禁用和选项内容插槽。",
    usage:
      "v-model 为 operation 字符串数组；options 使用 ApiTransferOption（BaseApi 与 ProFormOption 的组合），value 对应 operation。titles 提供两个面板标题，panelHeight 单位为像素。组件只修改选择值，实际授权由父表单完成。示例使用本地 API 目录。",
    code: `<ApiTransfer
  v-model="selectedOperations"
  :options="catalog.options"
  :titles="['可选 API', '已选 API']"
  :panel-height="260"
>
  <template #default="{ option }">{{ option.label }}</template>
</ApiTransfer>

<!-- ApiTransfer 从 System 的 components/api/ApiTransfer.vue 导入。
catalog 可由 components/api/apiTransfer 中的 loadApiTransferCatalog() 加载。 -->`,
    example: selectionExample,
    related: ["ProjectGrantDialog"]
  },
  {
    id: "LogTable",
    label: "日志表格",
    file: "log/LogTable.vue",
    description: "按配置组合日志查询、分页、详情与关联审计时间线。",
    usage:
      "config.request 接收规范化分页条件并返回 { list, total }，config.get 按日志编号返回详情对象；trace 为可选的关联日志查询。columns 配置详情操作时调用实例 handleOpenDialog(id)。defineExpose 还提供内部 proTable。示例 request/get 都读取本地数组。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import LogTable, { type LogTableConfig } from "@liujitcn/kratos-admin-system/components/log/LogTable.vue";

const page = ref<InstanceType<typeof LogTable>>();
const rows = [{ id: "log-1", user_name: "张明", action: "查看" }];
const config: LogTableConfig = {
  columns: [
    { prop: "user_name", label: "操作人" },
    { prop: "action", label: "操作" },
    { prop: "operation", label: "详情", actions: [
      { label: "查看", onClick: ({ row }) => page.value?.handleOpenDialog(String(row.id)) }
    ] }
  ],
  detailTitle: "日志详情", closeText: "关闭",
  detailFields: [{ key: "user_name", label: "操作人" }, { key: "action", label: "操作" }],
  request: async () => ({ list: rows, total: rows.length }),
  get: async id => {
    const row = rows.find(item => item.id === id);
    if (!row) throw new Error("日志不存在");
    return row;
  }
};
</script>

<template>
  <LogTable ref="page" :config="config" />
</template>`,
    example: () => import("./LogExample.vue"),
    related: ["ProTable", "ProDialog"]
  },
  {
    id: "Notification",
    label: "消息通知",
    file: "notification/Notification.vue",
    description: "顶部未读消息入口，组合实时更新、消息详情与收件箱跳转。",
    usage:
      "System 模块已经注册该顶部工具。挂载会查询未读汇总和最近消息并订阅 SSE；打开消息会标记已读，“全部已读”会更新真实消息状态。消息动作按当前路由权限校验后跳转，富文本详情复用 RichTextPreview。文档页不重复创建通知订阅。",
    code: `import { systemAdminModule } from "@liujitcn/kratos-admin-system";

// System 模块内的 headerTools 已包含 Notification。
const modules = [systemAdminModule];`,
    related: ["RichTextPreview", "ProDialog"]
  },
  {
    id: "ChangePasswordForm",
    label: "修改密码表单",
    file: "account/ChangePasswordForm.vue",
    description: "完成当前密码、新密码校验和加密提交，修改成功后要求重新登录。",
    usage:
      "该表单不接受 props，也不向外透传提交事件；内部加载密码策略、读取加密公钥并调用当前账号改密接口，成功会退出登录。它用于个人中心和强制改密流程，文档页只展示接入代码，不采集密码。",
    code: `<script setup lang="ts">
import ChangePasswordForm from "@liujitcn/kratos-admin-system/components/account/ChangePasswordForm.vue";
</script>

<template>
  <!-- 仅在真实账号安全页面挂载，确认后修改当前账号密码并退出登录。 -->
  <ChangePasswordForm />
</template>`,
    related: ["ForcedPasswordDialog", "PasswordStrength", "ProForm"]
  },
  {
    id: "ForcedPasswordDialog",
    label: "强制改密弹窗",
    file: "account/ForcedPasswordDialog.vue",
    description: "由请求层密码状态控制的强制改密入口，要求完成密码更新后继续使用。",
    usage:
      "System 模块已注册为顶部工具，由 passwordChangeState.required 控制显示，没有 v-model 或业务 props。弹窗禁止关闭、遮罩关闭与 Esc 关闭，内部使用 ChangePasswordForm。不要在业务页面再次注册，也不要为演示修改全局强制改密状态。",
    code: `import { systemAdminModule } from "@liujitcn/kratos-admin-system";

// ForcedPasswordDialog 已随 System 模块注册，由请求层驱动。
const modules = [systemAdminModule];`,
    related: ["ChangePasswordForm", "ProDialog"]
  }
];

const apis: Record<string, AdminComponentApi> = componentApi;

/** System 模块提供的组件说明与本地示例。 */
export const systemComponentDocs: AdminComponentDoc[] = docs.map(doc => ({
  ...doc,
  group: "系统组件",
  owner: "System",
  api: apis[doc.file]
}));
