<template>
  <div class="profile-password">
    <el-card class="password-card" shadow="never">
      <template #header>
        <div class="panel-header">
          <div>
            <h3>{{ t("system.profile.password.title") }}</h3>
            <p>{{ t("system.profile.password.description") }}</p>
          </div>
          <el-tag type="warning" effect="plain">{{ t("system.profile.password.value.strong_recommended") }}</el-tag>
        </div>
      </template>
      <div class="password-layout">
        <div class="password-form-wrap">
          <ProForm
            ref="passwordFormRef"
            :model="passwordForm"
            :fields="passwordFormFields"
            :rules="passwordFormRules"
            label-width="96px"
          />
          <PasswordStrength :password="passwordForm.new_pwd" class="password-form-wrap__strength" />
          <div class="password-footer">
            <el-button @click="resetPasswordForm">{{ t("common.action.reset") }}</el-button>
            <el-button type="primary" :loading="submitLoading" @click="handleSubmitPassword">
              {{ t("system.profile.password.action.update") }}
            </el-button>
          </div>
        </div>
        <div class="password-tips">
          <div class="tip-card">
            <span class="tip-badge">01</span>
            <strong>{{ t("system.profile.password.tip.unique_title") }}</strong>
            <p>{{ t("system.profile.password.tip.unique_description") }}</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">02</span>
            <strong>{{ t("system.profile.password.tip.strength_title") }}</strong>
            <p>{{ t("system.profile.password.tip.strength_description") }}</p>
          </div>
          <div class="tip-card">
            <span class="tip-badge">03</span>
            <strong>{{ t("system.profile.password.tip.record_title") }}</strong>
            <p>{{ t("system.profile.password.tip.record_description") }}</p>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/auth";
import type { UserPasswordForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import PasswordStrength from "@liujitcn/kratos-admin-core/components/PasswordStrength/index.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { LOGIN_URL } from "@liujitcn/kratos-admin-core/config";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import {
  PASSWORD_CRYPTO_SCENE,
  encryptPassword,
  getPasswordStrength,
  validatePasswordStrengthValue
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
    { required: true, message: t("system.profile.password.placeholder.new_password"), trigger: "blur" },
    { validator: validatePasswordStrength, trigger: "blur" }
  ],
  confirm_pwd: [{ required: true, message: t("system.profile.password.placeholder.confirm_password"), trigger: "blur" }]
}));

/** 统一计算当前新密码强度，供展示和提交校验复用。 */
const passwordStrength = computed(() => getPasswordStrength(passwordForm.new_pwd));

/** 提交修改密码请求，并校验两次输入的一致性。 */
async function handleSubmitPassword() {
  if (!(await passwordFormRef.value?.validate())) return;
  if (passwordForm.new_pwd !== passwordForm.confirm_pwd) {
    ElMessage.error(t("system.profile.password.message.mismatch"));
    return;
  }
  if (!passwordStrength.value.isValid) {
    ElMessage.error(t("system.profile.password.message.strength_insufficient"));
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

/**
 * 校验新密码强度，必须达到最高强度才允许提交。
 *
 * @param _rule 表单规则对象
 * @param value 当前输入的新密码
 * @param callback 校验回调
 */
function validatePasswordStrength(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value) {
    callback();
    return;
  }
  const result = validatePasswordStrengthValue(value);
  if (!result.valid) {
    callback(new Error(t("core.password.error")));
    return;
  }
  callback();
}
</script>

<style scoped lang="scss">
.password-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
}
:deep(.password-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}
:deep(.password-card .el-card__body) {
  padding: 20px;
}
.panel-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
}
.panel-header h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}
.panel-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}
.password-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(260px, 0.9fr);
  gap: 16px;
}
.password-form-wrap,
.tip-card {
  padding: 18px;
  background: #ffffff;
  border: 1px solid #f0f2f5;
  border-radius: 10px;
}
.password-form-wrap__strength {
  margin-top: 16px;
}
.password-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}
.password-tips {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tip-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 700;
  color: #409eff;
  background: #ecf5ff;
  border-radius: 8px;
}
.tip-card strong {
  display: block;
  margin-bottom: 8px;
  font-size: 15px;
  color: #303133;
}
.tip-card p {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #909399;
}

@media screen and (width <= 960px) {
  .password-layout {
    grid-template-columns: 1fr;
  }
}

@media screen and (width <= 640px) {
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
