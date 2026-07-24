<template>
  <ProDialog
    :model-value="modelValue"
    :title="title"
    :width="width"
    :top="top"
    :confirm-text="confirmText"
    :cancel-text="cancelText"
    :confirm-loading="confirmLoading"
    :destroy-on-close="destroyOnClose"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    v-bind="$attrs"
    @update:model-value="handleVisibleChange"
    @confirm="handleConfirm"
    @cancel="handleCancel"
    @close="handleClose"
    @closed="handleClosed"
  >
    <ProForm
      ref="proFormRef"
      :model="model"
      :fields="fields"
      :rules="rules"
      :label-width="labelWidth"
      :gutter="gutter"
      :col-span="colSpan"
      v-bind="formProps"
    >
      <template v-for="slotName in slotNames" #[slotName]="slotScope">
        <slot :name="slotName" v-bind="slotScope" />
      </template>
    </ProForm>

    <template #footer>
      <slot name="footer">
        <div class="dialog-footer">
          <el-button @click="handleCancel">{{ cancelText }}</el-button>
          <el-button type="primary" :loading="confirmLoading" @click="handleConfirm">{{ confirmText }}</el-button>
        </div>
      </slot>
    </template>
  </ProDialog>
</template>

<script setup lang="ts" name="FormDialog">
import { computed, ref, useSlots } from "vue";
import type { FormRules } from "element-plus";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";

/** 表单弹窗组件属性。 */
interface FormDialogProps {
  modelValue: boolean;
  title?: string;
  width?: string | number;
  top?: string;
  model: Record<string, any>;
  fields: ProFormField[];
  rules?: FormRules;
  labelWidth?: string;
  gutter?: number;
  colSpan?: number;
  confirmText?: string;
  cancelText?: string;
  confirmLoading?: boolean;
  destroyOnClose?: boolean;
  closeOnClickModal?: boolean;
  closeOnPressEscape?: boolean;
  formProps?: Record<string, any>;
}

withDefaults(defineProps<FormDialogProps>(), {
  title: "",
  width: "500px",
  top: "8vh",
  rules: () => ({}),
  labelWidth: "110px",
  gutter: 20,
  colSpan: 24,
  confirmText: "确定",
  cancelText: "取消",
  confirmLoading: false,
  destroyOnClose: false,
  closeOnClickModal: true,
  closeOnPressEscape: true,
  formProps: () => ({})
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  confirm: [];
  cancel: [];
  close: [];
  closed: [];
}>();

const slots = useSlots();
const proFormRef = ref<ProFormInstance>();

/** 汇总外部透传的插槽名称，统一转发到内部 ProForm。 */
const slotNames = computed(() => Object.keys(slots).filter(slotName => slotName !== "footer"));

/** 同步弹窗显示状态到外部。 */
function handleVisibleChange(value: boolean) {
  emit("update:modelValue", value);
}

/** 对外透传确定事件，保持页面业务提交流程不变。 */
function handleConfirm() {
  emit("confirm");
}

/** 关闭弹窗并对外透传取消事件。 */
function handleCancel() {
  emit("update:modelValue", false);
  emit("cancel");
}

/** 对外透传弹窗关闭事件。 */
function handleClose() {
  emit("close");
}

/** 对外透传弹窗完全关闭事件。 */
function handleClosed() {
  emit("closed");
}

/** 校验内部 ProForm。 */
async function validate() {
  return proFormRef.value?.validate();
}

/** 重置内部 ProForm。 */
function resetFields() {
  proFormRef.value?.resetFields();
}

/** 清理内部 ProForm 校验状态。 */
function clearValidate(props?: string | string[]) {
  proFormRef.value?.clearValidate(props);
}

defineExpose({
  validate,
  resetFields,
  clearValidate
});
</script>
