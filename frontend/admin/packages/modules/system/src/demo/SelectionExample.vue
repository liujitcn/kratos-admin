<template>
  <div v-if="active" class="selection-example">
    <template v-if="componentId === 'AiModelSelector'">
      <AiModelSelector :model-providers="providers" :provider-id="providerId" :model-name="modelName" @change="selectModel" />
      <p aria-live="polite">当前选择：{{ providerId }} / {{ modelName }}</p>
    </template>
    <template v-else-if="componentId === 'TenantProjectSelect'">
      <TenantProjectSelect v-model="project" :options="projectOptions" aria-label="选择演示租户项目" />
      <p aria-live="polite">{{ project ? JSON.stringify(project) : "尚未选择项目" }}</p>
    </template>
    <template v-else-if="componentId === 'ApiTransfer'">
      <ApiTransfer v-model="selectedApis" :options="apiOptions" :titles="['可选 API', '已选 API']" :panel-height="220" />
      <p aria-live="polite">已选：{{ selectedApis.join("、") || "无" }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onActivated, onDeactivated, ref } from "vue";
import AiModelSelector from "@liujitcn/kratos-admin-system/components/AiModelSelector/index.vue";
import TenantProjectSelect, {
  type TenantProjectSelection
} from "@liujitcn/kratos-admin-system/components/tenant-project/TenantProjectSelect.vue";
import ApiTransfer from "@liujitcn/kratos-admin-system/components/api/ApiTransfer.vue";
import type { ApiTransferOption } from "@liujitcn/kratos-admin-system/components/api/apiTransfer";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import type { TreeBaseTenantProjectResponse_Option } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineProps<{ componentId: string }>();
const active = ref(true);
const providerId = ref(1);
const modelName = ref("demo-chat");
const providers: AiProviderModelOption[] = [
  {
    provider_id: 1,
    name: "本地演示供应商",
    models: [
      { model_name: "demo-chat", display_name: "通用对话模型" },
      { model_name: "demo-reasoning", display_name: "推理模型" }
    ]
  }
];
const project = ref<TenantProjectSelection>({ value: "project:1:11", type: "project", tenant_id: 1, project_id: 11 });
const projectOptions: TreeBaseTenantProjectResponse_Option[] = [
  {
    value: "tenant:1",
    label: "演示租户",
    type: "tenant",
    tenant_id: 1,
    project_id: 0,
    disabled: false,
    children: [
      { value: "project:1:11", label: "校园服务", type: "project", tenant_id: 1, project_id: 11, disabled: false, children: [] },
      { value: "project:1:12", label: "数据分析", type: "project", tenant_id: 1, project_id: 12, disabled: false, children: [] }
    ]
  }
];
const selectedApis = ref<string[]>(["/demo.ReportService/ListReports"]);
const apiOptions: ApiTransferOption[] = ["ListReports", "GetReport", "ExportReport"].map((operation, index) => ({
  id: index + 1,
  tool_name: "",
  tool_prompts: [],
  service_name: "demo.ReportService",
  service_desc: "报表服务",
  desc: ["查询报表", "查看报表", "导出报表"][index],
  operation: `/demo.ReportService/${operation}`,
  method: "GET",
  path: `/demo/reports/${operation}`,
  mcp_status: Status.STATUS_DISABLE,
  agent_status: Status.STATUS_DISABLE,
  openapi_service_code: "",
  openapi_service_name: "",
  tenant_response: false,
  value: `/demo.ReportService/${operation}`,
  label: ["查询报表", "查看报表", "导出报表"][index]
}));

/** 将选择结果保存在当前示例中。 */
function selectModel(id: number, name: string) {
  providerId.value = id;
  modelName.value = name;
}

onActivated(() => {
  active.value = true;
});
// 隐藏缓存页面时卸载子组件，清理下拉面板的传送节点。
onDeactivated(() => {
  active.value = false;
});
</script>

<style scoped>
.selection-example {
  display: grid;
  gap: 16px;
  min-width: 0;
}
.selection-example p {
  margin: 0;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}
</style>
