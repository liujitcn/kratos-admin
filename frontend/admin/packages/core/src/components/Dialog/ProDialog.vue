<template>
  <el-dialog
    class="pro-dialog"
    :model-value="modelValue"
    :title="title"
    :width="width"
    :top="top"
    :destroy-on-close="destroyOnClose"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    v-bind="$attrs"
    @update:model-value="handleVisibleChange"
    @close="handleClose"
    @closed="handleClosed"
  >
    <template v-if="$slots.header" #header="headerProps">
      <slot name="header" v-bind="headerProps" />
    </template>

    <slot />

    <template v-if="showFooter && $slots.footer" #footer>
      <slot name="footer" />
    </template>
    <template v-else-if="showFooter" #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">{{ cancelText || t("common.action.cancel") }}</el-button>
        <el-button type="primary" :loading="confirmLoading" @click="handleConfirm">
          {{ confirmText || t("common.action.confirm") }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" name="ProDialog">
import { watch } from "vue";
import { useLocaleStore } from "@/locales";
import { createDialogRequestController, type DialogOpenOptions } from "./interface";

const { t } = useLocaleStore();

/** 通用弹窗组件属性。 */
interface ProDialogProps {
  modelValue: boolean;
  title?: string;
  width?: string | number;
  top?: string;
  confirmText?: string;
  cancelText?: string;
  confirmLoading?: boolean;
  destroyOnClose?: boolean;
  closeOnClickModal?: boolean;
  closeOnPressEscape?: boolean;
  /** 是否展示弹窗底部操作区。 */
  showFooter?: boolean;
}

const props = withDefaults(defineProps<ProDialogProps>(), {
  title: "",
  width: "500px",
  top: "8vh",
  confirmText: "",
  cancelText: "",
  confirmLoading: false,
  destroyOnClose: false,
  closeOnClickModal: true,
  closeOnPressEscape: true,
  showFooter: true
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  confirm: [];
  cancel: [];
  close: [];
  closed: [];
}>();

const dialogRequestController = createDialogRequestController();

watch(
  () => props.modelValue,
  value => {
    if (!value) dialogRequestController.invalidate();
  }
);

/** 异步加载弹窗数据，并只提交最新一次打开请求。 */
async function open<T>(options: DialogOpenOptions<T>) {
  const opened = await dialogRequestController.open(options);
  if (opened) emit("update:modelValue", true);
  return opened;
}

/** 关闭弹窗并使未完成的打开请求失效。 */
function close() {
  dialogRequestController.invalidate();
  emit("update:modelValue", false);
}

/** 同步弹窗显示状态到外部。 */
function handleVisibleChange(value: boolean) {
  if (!value) dialogRequestController.invalidate();
  emit("update:modelValue", value);
}

/** 处理点击确定按钮后的回调。 */
function handleConfirm() {
  emit("confirm");
}

/** 处理点击取消按钮后的回调，并主动关闭弹窗。 */
function handleCancel() {
  close();
  emit("cancel");
}

/** 处理弹窗关闭时的回调。 */
function handleClose() {
  dialogRequestController.invalidate();
  emit("close");
}

/** 处理弹窗完全关闭后的回调。 */
function handleClosed() {
  emit("closed");
}

defineExpose({ open, close });
</script>
