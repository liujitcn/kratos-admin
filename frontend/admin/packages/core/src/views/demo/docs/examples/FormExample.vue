<template>
  <div v-if="active" class="component-form-example">
    <template v-if="componentId === 'DynamicList'">
      <DynamicList v-model="model.contacts" :input-props="{ placeholder: '输入通知收件人' }" />
      <pre class="example-result">{{ JSON.stringify(model.contacts, null, 2) }}</pre>
    </template>
    <template v-else-if="componentId === 'KvList'">
      <KvList v-model="model.variables" add-text="新增参数" />
      <pre class="example-result">{{ JSON.stringify(model.variables, null, 2) }}</pre>
    </template>
    <template v-else-if="componentId === 'CronExpression'">
      <CronExpression v-model="model.cron" />
      <p class="example-tip">当前表达式：{{ model.cron }}</p>
    </template>
    <template v-else>
      <ProForm ref="formRef" :model="model" :fields="fields" :col-span="12" label-width="7em" />
      <el-space>
        <el-button type="primary" @click="validateForm">校验表单</el-button>
        <el-button @click="resetForm">重置</el-button>
      </el-space>
      <p v-if="result" class="example-tip" role="status">{{ result }}</p>
      <pre class="example-result">{{ JSON.stringify(model, null, 2) }}</pre>
    </template>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import ProForm from "@/components/ProForm/index.vue";
import DynamicList from "@/components/ProForm/components/DynamicList.vue";
import KvList from "@/components/ProForm/components/KvList.vue";
import CronExpression from "@/components/CronExpression/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const formRef = ref<ProFormInstance>();
const result = ref("");
const model = reactive({
  name: "智慧校园平台",
  enabled: true,
  notify: true,
  channel: "email",
  contacts: ["admin@example.com"],
  variables: [{ key: "ENV", value: "demo" }],
  cron: "0 */5 * * * *"
});
const fields: ProFormField[] = [
  {
    prop: "name",
    label: "项目名称",
    component: "input",
    rules: [{ required: true, message: "请输入项目名称", trigger: "blur" }]
  },
  { prop: "enabled", label: "启用项目", component: "switch" },
  { prop: "notify", label: "发送通知", component: "checkbox", checkboxLabel: "启用完成通知" },
  {
    prop: "channel",
    label: "通知渠道",
    component: "select",
    visible: value => value.notify,
    options: [
      { label: "电子邮件", value: "email" },
      { label: "站内消息", value: "inbox" }
    ]
  },
  {
    prop: "contacts",
    label: "通知收件人",
    component: "dynamic-list",
    visible: value => value.notify && value.channel === "email",
    colSpan: 24,
    props: { inputProps: { placeholder: "输入邮箱地址" } }
  },
  { prop: "variables", label: "扩展参数", component: "kv-list", colSpan: 24 },
  { prop: "cron", label: "执行规则", component: "cron-expression", colSpan: 24 }
];

/** 仅校验本地表单并展示结果。 */
async function validateForm() {
  result.value = (await formRef.value?.validate()) ? "校验通过，表单数据已准备就绪。" : "请先补齐必填字段。";
}

/** 恢复示例初始数据并清理校验。 */
function resetForm() {
  Object.assign(model, {
    name: "智慧校园平台",
    enabled: true,
    notify: true,
    channel: "email",
    contacts: ["admin@example.com"],
    variables: [{ key: "ENV", value: "demo" }],
    cron: "0 */5 * * * *"
  });
  formRef.value?.clearValidate();
  result.value = "";
}
</script>

<style scoped lang="scss">
.component-form-example {
  max-width: 860px;
}
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.example-result {
  max-height: 260px;
  padding: 14px;
  margin: 20px 0 0;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg-muted);
  border-radius: var(--admin-page-radius);
}
</style>
