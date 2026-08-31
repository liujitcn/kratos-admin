<template>
  <ProDialog
    :model-value="modelValue"
    :title="dialogTitle"
    :width="width"
    :top="top"
    :confirm-text="confirmText"
    :cancel-text="cancelText"
    :confirm-loading="submitting"
    :destroy-on-close="destroyOnClose"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    @update:model-value="handleVisibleChange"
    @confirm="handleConfirm"
    @cancel="handleCancel"
    @close="handleClose"
    @closed="handleClosed"
  >
    <p v-if="description" class="password-verify-dialog__description">{{ description }}</p>
    <ProForm
      ref="formRef"
      :model="form"
      :fields="fields"
      :rules="rules"
      :label-width="labelWidth"
      @keyup.enter.prevent="handleConfirm"
    />
  </ProDialog>
</template>

<script setup lang="ts" name="PasswordVerifyDialog">
import { computed, reactive, ref } from "vue";
import type { FormRules } from "element-plus";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import ProForm from "@/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import { useLocaleStore } from "@/locales";
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from "@/utils/passwordCrypto";
import type { PasswordCryptoScene } from "@/utils/passwordCrypto";
import type { PasswordCrypto } from "@/rpc/common/v1/types";

const { t } = useLocaleStore();

/** 通用密码验证弹窗属性。 */
interface PasswordVerifyDialogProps {
  /** 弹窗显示状态。 */
  modelValue: boolean;
  /** 弹窗标题。 */
  title?: string;
  /** 密码输入前的可选补充说明。 */
  description?: string;
  /** 弹窗宽度。 */
  width?: string | number;
  /** 弹窗顶部距离。 */
  top?: string;
  /** 密码字段标题。 */
  passwordLabel?: string;
  /** 密码输入提示。 */
  passwordPlaceholder?: string;
  /** 确认按钮文案。 */
  confirmText?: string;
  /** 取消按钮文案。 */
  cancelText?: string;
  /** 外部业务提交状态。 */
  confirmLoading?: boolean;
  /** 密码加密场景。 */
  scene?: PasswordCryptoScene;
  /** 是否关闭时销毁内容。 */
  destroyOnClose?: boolean;
  /** 是否允许点击遮罩关闭。 */
  closeOnClickModal?: boolean;
  /** 是否允许按 Esc 关闭。 */
  closeOnPressEscape?: boolean;
  /** 表单标签宽度。 */
  labelWidth?: string;
}

const props = withDefaults(defineProps<PasswordVerifyDialogProps>(), {
  title: "",
  description: "",
  width: "500px",
  top: "8vh",
  passwordLabel: "",
  passwordPlaceholder: "",
  confirmText: "",
  cancelText: "",
  confirmLoading: false,
  scene: PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA,
  destroyOnClose: false,
  closeOnClickModal: false,
  closeOnPressEscape: true,
  labelWidth: "auto"
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  confirm: [password: PasswordCrypto];
  cancel: [];
  close: [];
  closed: [];
}>();

const formRef = ref<ProFormInstance>();
const form = reactive({ password: "" });
const encrypting = ref(false);
const submitting = computed(() => props.confirmLoading || encrypting.value);
const dialogTitle = computed(() => props.title || t("core.password.verify_title"));
const passwordLabel = computed(() => props.passwordLabel || t("core.password.verify_label"));
const passwordPlaceholder = computed(() => props.passwordPlaceholder || t("core.password.required"));
const confirmText = computed(() => props.confirmText || t("common.action.confirm"));
const cancelText = computed(() => props.cancelText || t("common.action.cancel"));

const fields = computed<ProFormField[]>(() => [
  {
    prop: "password",
    label: passwordLabel.value,
    component: "password",
    props: {
      autocomplete: "current-password",
      placeholder: passwordPlaceholder.value,
      showPassword: true
    }
  }
]);

const rules = computed<FormRules>(() => ({
  password: [{ required: true, message: t("core.password.required"), trigger: "blur" }]
}));

/** 同步弹窗显示状态到外部。 */
function handleVisibleChange(value: boolean) {
  emit("update:modelValue", value);
}

/** 校验密码并向业务方返回加密结果。 */
async function handleConfirm() {
  if (submitting.value) return;
  if (!(await formRef.value?.validate())) return;
  encrypting.value = true;
  try {
    const password = await encryptPassword(form.password, props.scene);
    emit("confirm", password);
  } finally {
    encrypting.value = false;
  }
}

/** 对外透传取消事件。 */
function handleCancel() {
  emit("cancel");
}

/** 对外透传关闭事件。 */
function handleClose() {
  emit("close");
}

/** 关闭弹窗后清理密码和校验状态。 */
function handleClosed() {
  form.password = "";
  formRef.value?.clearValidate();
  emit("closed");
}
</script>

<style scoped lang="scss">
.password-verify-dialog__description {
  margin: 0 0 16px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.password-verify-dialog__description + :deep(.pro-form) {
  margin-top: 0;
}
</style>
