<template>
  <div v-if="active">
    <el-button type="primary" @click="openDialog">{{ componentId === "FormDialog" ? "打开表单弹窗" : "打开通用弹窗" }}</el-button>
    <p class="example-tip" role="status">{{ result || "打开后可以确认、取消或关闭弹窗。" }}</p>
    <FormDialog
      v-if="componentId === 'FormDialog'"
      ref="formRef"
      v-model="visible"
      title="编辑项目"
      :model="model"
      :fields="fields"
      @confirm="saveForm"
      @closed="resetForm"
    />
    <ProDialog v-else v-model="visible" title="项目预览" @confirm="confirmDialog">
      <p class="dialog-copy">{{ model.name }}</p>
      <p class="dialog-copy">弹窗内容使用默认插槽，底部操作由组件统一管理。</p>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { onDeactivated, reactive, ref } from "vue";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const visible = ref(false);
const result = ref("");
const formRef = ref<ProFormInstance>();
const model = reactive({ name: "智慧校园平台", remark: "统一校园服务入口" });
const fields: ProFormField[] = [
  {
    prop: "name",
    label: "项目名称",
    component: "input",
    rules: [{ required: true, message: "请输入项目名称", trigger: "blur" }]
  },
  { prop: "remark", label: "项目说明", component: "textarea", props: { rows: 3 } }
];
onDeactivated(() => {
  visible.value = false;
});

/** 打开示例弹窗。 */
function openDialog() {
  visible.value = true;
  result.value = "";
}

/** 接收通过校验后的本地表单。 */
function saveForm() {
  result.value = `已保存：${model.name}`;
  visible.value = false;
}

/** 确认通用弹窗。 */
function confirmDialog() {
  result.value = "已确认项目预览。";
  visible.value = false;
}

/** 关闭后重置示例表单与校验。 */
function resetForm() {
  Object.assign(model, { name: "智慧校园平台", remark: "统一校园服务入口" });
  formRef.value?.clearValidate();
}
</script>

<style scoped lang="scss">
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.dialog-copy {
  line-height: 1.7;
  color: var(--admin-page-text-primary);
}
</style>
