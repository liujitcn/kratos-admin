<template>
  <div class="script-select">
    <el-select
      :model-value="modelValue"
      clearable
      filterable
      :disabled="disabled"
      @update:model-value="emit('update:modelValue', $event || undefined)"
    >
      <el-option v-for="script in scripts" :key="script.id" :label="script.name" :value="script.id" />
      <el-option
        v-if="modelValue && !scripts.some(item => item.id === modelValue)"
        :value="modelValue"
        :label="`ID: ${modelValue}`"
        disabled
      />
    </el-select>
    <el-button v-if="modelValue && !disabled" link @click="testVisible = true">{{ t("system.workflow.script.test") }}</el-button>
    <ProDialog
      v-model="testVisible"
      :title="t('system.workflow.script.test')"
      width="680px"
      destroy-on-close
      @confirm="testVisible = false"
      @closed="resetTestRun"
    >
      <p class="script-help">{{ t("system.workflow.script.help") }}</p>
      <el-input v-model="context" type="textarea" :rows="12" :aria-label="t('system.workflow.script.input')" />
      <el-button :loading="testing" @click="testRun">{{ t("system.workflow.script.test") }}</el-button>
      <el-input :model-value="result" type="textarea" :rows="8" readonly :aria-label="t('system.workflow.script.output')" />
    </ProDialog>
  </div>
</template>
<script setup lang="ts">
import { onMounted, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import type { BaseScript } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_script";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

/** 工作流脚本选择器，复用系统级共享脚本库。 */
interface ScriptSelectProps {
  modelValue?: number;
  disabled?: boolean;
}
const props = defineProps<ScriptSelectProps>();
const emit = defineEmits<{ "update:modelValue": [value: number | undefined] }>();
const scripts = ref<BaseScript[]>([]);
const testVisible = ref(false);
const testing = ref(false);
const result = ref("");
const defaultContext = JSON.stringify(
  {
    form: {},
    variables: {},
    workflow: { instance_id: 0, tenant_id: 0, applicant_user_id: 0, business_type: "", business_id: 0, status: 0 },
    node: { id: "", type: "SCRIPT" },
    idempotency_key: "preview"
  },
  null,
  2
);
const context = ref(defaultContext);
onMounted(loadScripts);
/** 关闭试跑后清空结果并恢复默认输入，避免下一次打开沿用旧数据。 */
function resetTestRun() {
  context.value = defaultContext;
  result.value = "";
}
/** 按页加载可用于工作流的启用脚本，不截断下拉选项。 */
async function loadScripts() {
  const items: BaseScript[] = [];
  for (let page = 1; ; page += 1) {
    const data = await defWorkflowDefinitionService.PageWorkflowScript({
      page_num: page,
      page_size: 100,
      name: "",
      hook_point: "",
      status: Status.STATUS_ENABLE
    });
    items.push(...data.base_scripts);
    if (items.length >= data.total || !data.base_scripts.length) break;
  }
  scripts.value = items;
}
/** 用已保存脚本与用户输入的统一上下文试运行，不改变流程实例。 */
async function testRun() {
  if (!props.modelValue) return;
  try {
    JSON.parse(context.value);
  } catch {
    ElMessage.error(t("system.workflow.script.invalid_json"));
    return;
  }
  testing.value = true;
  try {
    const data = await defWorkflowDefinitionService.TestWorkflowScript({
      script_id: props.modelValue,
      context_json: context.value
    });
    result.value = JSON.stringify(data, null, 2);
  } finally {
    testing.value = false;
  }
}
</script>
<style scoped>
.script-select {
  display: flex;
  gap: 8px;
  width: 100%;
}
.script-help {
  color: var(--el-text-color-secondary);
  line-height: 1.7;
}
.el-select {
  flex: 1;
  min-width: 0;
}
</style>
