<template>
  <div class="change-password-form">
    <ProForm
      ref="passwordFormRef"
      :model="passwordForm"
      :fields="passwordFormFields"
      :rules="passwordFormRules"
    />
    <PasswordStrength :password="passwordForm.new_pwd" class="change-password-form__strength" />
    <div class="change-password-form__footer">
      <el-button @click="resetPasswordForm">{{ t("common.action.reset") }}</el-button>
      <el-button type="primary" :loading="submitLoading" @click="handleSubmitPassword">
        {{ t("system.profile.password.action.update") }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/auth";
import type { UserPasswordForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import PasswordStrength from "@liujitcn/kratos-admin-core/components/PasswordStrength/index.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { LOGIN_URL } from "@liujitcn/kratos-admin-core/config";
import { clearPasswordChangeRequired } from "@liujitcn/kratos-admin-core/request";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import {
  PASSWORD_CRYPTO_SCENE,
  encryptPassword
} from "@liujitcn/kratos-admin-core/security";

/** 修改密码表单状态，明文只在前端校验和加密前短暂保存。 */
interface UserPasswordFormState {
  /** 原密码明文只保留在前端表单中，提交前转换为密码密文。 */
  old_pwd: string;
  /** 新密码明文只保留在前端表单中，提交前转换为密码密文。 */
  new_pwd: string;
  /** 确认密码只用于前端一致性校验，不提交后端。 */
  confirm_pwd: string;
}

const router = useRouter();
const userStore = useUserStore();
const passwordFormRef = ref<ProFormInstance>();
const submitLoading = ref(false);
const passwordForm = reactive<UserPasswordFormState>({
  old_pwd: "",
  new_pwd: "",
  confirm_pwd: ""
});

const passwordFormFields = computed<ProFormField[]>(() => [
  {
    prop: "old_pwd",
    label: t("system.profile.password.field.old_password"),
    component: "password",
    props: { placeholder: t("system.profile.password.placeholder.old_password") }
  },
  {
    prop: "new_pwd",
    label: t("system.profile.password.field.new_password"),
    component: "password",
    props: { placeholder: t("system.profile.password.placeholder.new_password") }
  },
  {
    prop: "confirm_pwd",
    label: t("system.profile.password.field.confirm_password"),
    component: "password",
    props: { placeholder: t("system.profile.password.placeholder.confirm_password") }
  }
]);

const passwordFormRules = computed(() => ({
  old_pwd: [{ required: true, message: t("system.profile.password.placeholder.old_password"), trigger: "blur" }],
  new_pwd: [
    { required: true, message: t("system.profile.password.placeholder.new_password"), trigger: "blur" }
  ],
  confirm_pwd: [{ required: true, message: t("system.profile.password.placeholder.confirm_password"), trigger: "blur" }]
}));

/** 提交修改密码请求，并校验两次输入的一致性。 */
async function handleSubmitPassword() {
  if (!(await passwordFormRef.value?.validate())) return;
  if (passwordForm.new_pwd !== passwordForm.confirm_pwd) {
    ElMessage.error(t("system.profile.password.message.mismatch"));
    return;
  }
  submitLoading.value = true;
  try {
    const oldPwd = await encryptPassword(passwordForm.old_pwd, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_UPDATE_USER_PASSWORD);
    const newPwd = await encryptPassword(passwordForm.new_pwd, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_UPDATE_USER_PASSWORD);
    const userPassword: UserPasswordForm = {
      old_pwd: oldPwd,
      new_pwd: newPwd
    };
    await defProfileAuthService.UpdateUserPassword({ user_password: userPassword });
    clearPasswordChangeRequired();
    ElMessage.success(t("system.profile.password.message.updated"));
    resetPasswordForm();
    await forceRelogin();
  } finally {
    submitLoading.value = false;
  }
}

/** 重置密码表单内容与校验状态。 */
function resetPasswordForm() {
  passwordFormRef.value?.resetFields();
  passwordFormRef.value?.clearValidate();
  passwordForm.old_pwd = "";
  passwordForm.new_pwd = "";
  passwordForm.confirm_pwd = "";
}

/** 修改密码成功后强制重新登录，避免旧登录态继续使用。 */
async function forceRelogin() {
  try {
    await userStore.logout();
  } catch (_error) {
    // 若后端已提前让旧令牌失效，前端仍需保证本地登录态被清空。
    userStore.clearAuthData();
  }
  await router.replace(LOGIN_URL);
}

</script>

<style scoped lang="scss">
.change-password-form__strength {
  margin-top: 16px;
}

.change-password-form__footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}
</style>
