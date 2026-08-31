<template>
  <div class="profile-security">
    <el-card class="security-card" shadow="never">
      <div class="security-intro">
        <div>
          <h3>{{ t("system.profile.security.title") }}</h3>
          <p>{{ t("system.profile.security.description") }}</p>
        </div>
        <el-tag effect="plain" type="success">{{ t("system.profile.security.value.account_normal") }}</el-tag>
      </div>
      <div class="security-list">
        <div class="security-item">
          <div class="security-item__content">
            <strong>{{ t("system.profile.security.field.login_password") }}</strong>
            <p>{{ t("system.profile.security.message.password_hint") }}</p>
          </div>
          <el-button type="primary" plain @click="emit('switchTab', 'password')">
            {{ t("system.profile.security.action.change_password") }}
          </el-button>
        </div>
        <div class="security-item">
          <div class="security-item__content">
            <strong>{{ t("system.profile.security.field.phone") }}</strong>
            <p>{{ mobileTip }}</p>
          </div>
          <el-button plain @click="openPhoneDialog">
            {{ profile.phone ? t("system.profile.security.action.change_phone") : t("system.profile.security.action.bind_now") }}
          </el-button>
        </div>
        <div class="security-item">
          <div class="security-item__content">
            <strong>{{ t("system.profile.security.field.mfa") }}</strong>
            <p>
              {{
                mfaEnabled
                  ? t("system.profile.security.message.mfa_enabled_method", { method: mfaMethodLabel })
                  : t("system.profile.security.message.mfa_disabled")
              }}
            </p>
            <p v-if="mfaEnabled && mfaMethod === 'webauthn'" class="security-item__hint">
              {{ t("system.profile.security.message.mfa_webauthn_manage_hint") }}
            </p>
          </div>
          <div class="security-item__actions">
            <el-button v-if="!mfaEnabled" plain @click="openMfaSetup">{{
              t("system.profile.security.action.mfa_enable")
            }}</el-button>
            <template v-else>
              <el-button v-if="mfaMethod === 'totp'" plain @click="openMfaRecovery">
                {{ t("system.profile.security.action.mfa_recovery") }}
              </el-button>
              <el-button plain type="danger" @click="openMfaDisable">{{
                t("system.profile.security.action.mfa_disable")
              }}</el-button>
            </template>
          </div>
        </div>
        <div v-for="item in oauthBindings" :key="item.provider" class="security-item">
          <div class="security-item__content security-item__content--oauth">
            <el-tooltip :content="oauthName(item)" placement="top" :trigger="['hover', 'focus']">
              <span class="oauth-icon" :aria-label="oauthName(item)" :title="oauthName(item)">
                <component :is="getOauthProviderIcon(item)" />
              </span>
            </el-tooltip>
            <div>
              <strong>{{ oauthName(item) }}</strong>
              <p>
                {{
                  item.bound
                    ? t("system.profile.security.message.oauth_bound")
                    : t("system.profile.security.message.oauth_unbound")
                }}
              </p>
            </div>
          </div>
          <el-button
            v-if="item.bound"
            plain
            type="danger"
            :loading="oauthLoadingProvider === item.provider"
            @click="handleUnbindOauth(item)"
          >
            {{ t("system.profile.security.action.unbind") }}
          </el-button>
          <el-button v-else plain :loading="oauthLoadingProvider === item.provider" @click="handleBindOauth(item)">{{
            t("system.profile.security.action.bind")
          }}</el-button>
        </div>
      </div>
    </el-card>

    <ProDialog
      v-model="phoneDialogVisible"
      :title="t('system.profile.security.dialog.bind_phone')"
      :width="520"
      @closed="handleDialogClosed"
    >
      <ProForm ref="phoneFormRef" :model="phoneForm" :fields="phoneFormFields" :rules="phoneFormRules">
        <template #mobileCodeInput>
          <el-input v-model="phoneForm.code" :placeholder="t('system.profile.security.placeholder.code')">
            <template #append>
              <el-button :disabled="countdown > 0" @click="handleSendCode">
                {{
                  countdown > 0
                    ? t("system.profile.security.action.retry_after", { seconds: countdown })
                    : t("system.profile.security.action.send_code")
                }}
              </el-button>
            </template>
          </el-input>
        </template>
      </ProForm>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="phoneDialogVisible = false">{{ t("common.action.cancel") }}</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmitPhone">
            {{ t("common.action.save") }}
          </el-button>
        </div>
      </template>
    </ProDialog>
    <ProDialog
      v-model="mfaDialogVisible"
      :title="mfaDialogTitle"
      :width="'min(560px, calc(100vw - 32px))'"
      @closed="resetMfaDialog"
    >
      <div class="mfa-dialog-content">
        <MfaSetupPanel v-if="mfaSetupUri" :uri="mfaSetupUri" />
        <ProForm
          ref="mfaFormRef"
          class="mfa-dialog-form"
          :model="mfaForm"
          :fields="mfaDialogFields"
          :rules="mfaDialogRules"
          label-width="auto"
          autocomplete="off"
          size="default"
        />
        <div v-if="recoveryCodes.length" class="mfa-dialog-recovery">
          <div class="mfa-copyable-value">
            <span class="mfa-copyable-value__text">{{ recoveryCodesText }}</span>
            <el-tooltip :content="t('system.profile.security.action.mfa_copy_recovery_codes')" placement="top">
              <el-button
                class="mfa-copy-button"
                text
                circle
                :aria-label="t('system.profile.security.action.mfa_copy_recovery_codes')"
                @click="copyRecoveryCodes"
              >
                <el-icon class="mfa-copy-button__icon"><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
        <p v-if="recoveryCodes.length" class="mfa-recovery-hint">
          {{ t("system.profile.security.message.mfa_recovery_codes_warning") }}
        </p>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="mfaDialogVisible = false">{{ t("common.action.close") }}</el-button>
          <el-button :type="mfaDialogActionType" :loading="mfaLoading" @click="handleMfaAction">
            {{ mfaDialogActionText }}
          </el-button>
        </div>
      </template>
    </ProDialog>
    <MfaRecoveryCodesDialog
      v-model="recoveryCodesDialogVisible"
      :codes="recoveryCodes"
      @confirm="handleRecoveryCodesClosed"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/auth";
import { defMfaService } from "@liujitcn/kratos-admin-core/api/base/mfa";
import { defProfileOauthService } from "@liujitcn/kratos-admin-system/api/base/oauth";
import type { OauthBinding } from "@liujitcn/kratos-admin-system/rpc/base/v1/oauth";
import type {
  SendPhoneCodeRequest,
  UserPhoneForm,
  UserProfileForm
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import MfaSetupPanel from "@liujitcn/kratos-admin-core/components/Mfa/MfaSetupPanel.vue";
import MfaRecoveryCodesDialog from "@liujitcn/kratos-admin-core/components/Mfa/MfaRecoveryCodesDialog.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { PasswordCrypto } from "@liujitcn/kratos-admin-core/rpc/common/v1/types";
import {
  getOauthProviderIcon,
  withOauthProviderDisplay,
  type OauthProviderDisplay,
  PASSWORD_CRYPTO_SCENE,
  encryptPassword,
  copyText,
  createWebAuthnCredential,
  getWebAuthnAssertion
} from "@liujitcn/kratos-admin-core/security";
import { resolveFrontendRouteURL } from "@liujitcn/kratos-admin-core/navigation";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";

/** 安全中心组件属性。 */
interface ProfileSecurityProps {
  /** 当前用户资料。 */
  profile: UserProfileForm;
}

const props = defineProps<ProfileSecurityProps>();

const emit = defineEmits<{
  refreshed: [];
  switchTab: [tab: "account" | "security" | "password"];
}>();

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const phoneFormRef = ref<ProFormInstance>();
const mfaFormRef = ref<ProFormInstance>();
const phoneDialogVisible = ref(false);
const submitLoading = ref(false);
/** 安全中心三方账号绑定展示项。 */
type SecurityOauthBinding = OauthBinding & OauthProviderDisplay;

const oauthBindings = ref<SecurityOauthBinding[]>([]);
const oauthLoadingProvider = ref("");
const countdown = ref(0);
const phoneTimer = ref<number | null>(null);
const phoneForm = reactive<UserPhoneForm>({
  phone: "",
  code: ""
});
const sendPhoneCodeForm = reactive<SendPhoneCodeRequest>({
  phone: ""
});
const mfaEnabled = ref(false);
const mfaPolicy = ref("");
const mfaMethod = ref("totp");
const mfaDialogVisible = ref(false);
const recoveryCodesDialogVisible = ref(false);
const mfaDialogMode = ref<"setup" | "disable" | "recovery">("setup");
const mfaLoading = ref(false);
/** MFA 操作弹窗表单。 */
interface MfaDialogForm {
  /** 当前密码，仅用于生成一次性加密请求。 */
  password: string;
  /** 动态口令。 */
  code: string;
  /** 一次性恢复码。 */
  recoveryCode: string;
}

const mfaForm = reactive<MfaDialogForm>({
  password: "",
  code: "",
  recoveryCode: ""
});
const mfaSetupTicket = ref("");
const mfaSetupUri = ref("");
const mfaSetupMethod = ref("totp");
const mfaSetupWebAuthnOptionsJson = ref("");
const recoveryCodes = ref<string[]>([]);
const recoveryCodesText = computed(() => recoveryCodes.value.join("\n"));
/** 禁用 MFA 时是否需要同时校验已绑定的 MFA 因子。 */
const mfaFactorEnabled = computed(() => mfaDialogMode.value === "disable" && mfaEnabled.value);
const mfaDialogTitle = computed(() => {
  if (mfaDialogMode.value === "disable") return t("system.profile.security.dialog.mfa_disable");
  if (mfaDialogMode.value === "recovery") return t("system.profile.security.dialog.mfa_recovery");
  if (!mfaSetupTicket.value) return t("core.password.verify_title");
  return t("system.profile.security.dialog.mfa_setup");
});
const mfaDialogActionText = computed(() => {
  if (mfaDialogMode.value === "disable") return t("system.profile.security.action.mfa_disable");
  if (mfaDialogMode.value === "recovery") return t("common.action.confirm");
  if (mfaSetupUri.value || (mfaSetupMethod.value === "webauthn" && mfaSetupTicket.value)) {
    return mfaSetupMethod.value === "webauthn"
      ? t("core.login.mfa_webauthn_action")
      : t("system.profile.security.action.mfa_confirm");
  }
  return t("system.profile.security.action.mfa_next");
});
const mfaDialogActionType = computed(() => (mfaDialogMode.value === "disable" ? "danger" : "primary"));
const mfaMethodLabel = computed(() =>
  mfaMethod.value === "webauthn"
    ? t("system.profile.security.message.mfa_method_webauthn")
    : t("system.profile.security.message.mfa_method_totp")
);

const mfaDialogFields = computed<ProFormField[]>(() => {
  const passwordField: ProFormField = {
    prop: "password",
    label: t("system.profile.security.field.login_password"),
    component: "password",
    props: {
      autocomplete: "current-password",
      name: "mfa-current-password",
      placeholder: t("system.profile.security.placeholder.current_password")
    }
  };
  const codeField: ProFormField = {
    prop: "code",
    label: t("system.profile.security.field.code"),
    component: "input",
    itemProps: { required: mfaDialogMode.value !== "disable" },
    props: {
      autocomplete: "one-time-code",
      inputmode: "numeric",
      maxlength: 8,
      name: "mfa-verification-code",
      placeholder: t("system.profile.security.placeholder.mfa_code")
    }
  };
  const recoveryCodeField: ProFormField = {
    prop: "recoveryCode",
    label: t("system.profile.security.field.mfa_recovery_code"),
    component: "input",
    props: {
      autocomplete: "off",
      name: "mfa-recovery-code",
      placeholder: t("system.profile.security.placeholder.mfa_recovery_code")
    }
  };
  if (mfaDialogMode.value === "setup" && !mfaSetupUri.value && !mfaSetupTicket.value) return [passwordField];
  if (mfaDialogMode.value === "setup" && mfaSetupMethod.value === "webauthn") return [];
  if (mfaDialogMode.value === "disable" && !mfaFactorEnabled.value) return [passwordField];
  if (mfaDialogMode.value === "disable" && mfaMethod.value === "webauthn") return [passwordField, recoveryCodeField];
  if (mfaDialogMode.value === "disable") return [passwordField, codeField, recoveryCodeField];
  return [codeField];
});

const mfaDialogRules = computed(() => {
  if (mfaDialogMode.value === "disable") {
    if (!mfaFactorEnabled.value) {
      return {
        password: [{ required: true, message: t("system.profile.security.placeholder.current_password"), trigger: "blur" }]
      };
    }
    if (mfaMethod.value === "webauthn") {
      return {
        password: [{ required: true, message: t("system.profile.security.placeholder.current_password"), trigger: "blur" }]
      };
    }
    return {
      password: [{ required: true, message: t("system.profile.security.placeholder.current_password"), trigger: "blur" }],
      code: [
        {
          validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
            if (value || mfaForm.recoveryCode) {
              callback();
              return;
            }
            callback(new Error(t("system.profile.security.message.mfa_disable_factor_required")));
          },
          trigger: "blur"
        }
      ]
    };
  }
  if (mfaDialogMode.value === "setup" && !mfaSetupUri.value && !mfaSetupTicket.value) {
    return {
      password: [{ required: true, message: t("system.profile.security.placeholder.current_password"), trigger: "blur" }]
    };
  }
  if (mfaDialogMode.value === "setup" && mfaSetupMethod.value === "webauthn") return {};
  return {
    code: [{ required: true, message: t("system.profile.security.placeholder.mfa_code"), trigger: "blur" }]
  };
});

const phoneFormFields = computed<ProFormField[]>(() => [
  {
    prop: "phone",
    label: t("system.profile.security.field.phone_number"),
    component: "input",
    props: { placeholder: t("system.profile.security.placeholder.phone") }
  },
  {
    prop: "code",
    label: t("system.profile.security.field.code"),
    component: "slot",
    slotName: "mobileCodeInput"
  }
]);

const phoneFormRules = computed(() => ({
  phone: [
    { required: true, message: t("system.profile.security.placeholder.phone"), trigger: "blur" },
    { pattern: /^1[3-9]\d{9}$/, message: t("system.profile.security.validation.phone"), trigger: "blur" }
  ],
  code: [{ required: true, message: t("system.profile.security.placeholder.code"), trigger: "blur" }]
}));

/** 根据当前绑定状态输出手机号说明文案。 */
const mobileTip = computed(() => {
  return props.profile.phone
    ? t("system.profile.security.message.phone_bound", { phone: props.profile.phone })
    : t("system.profile.security.message.phone_unbound");
});

/** 获取当前安全设置页绝对地址，供 OAuth 绑定完成后回跳到前端页面。 */
function getCurrentSecurityPath() {
  const query = { ...route.query };
  delete query.oauth_bind_provider;
  delete query.oauth_bind_success;
  delete query.oauth_bind_error;
  return resolveFrontendRouteURL(router, { path: route.path, query });
}

/** 拉取当前用户三方账号绑定状态。 */
async function loadOauthBindings() {
  const result = await defProfileOauthService.ListOauthBinding({});
  oauthBindings.value = result.bindings.map(withOauthProviderDisplay);
}

/** 拉取当前用户 MFA 状态。 */
async function loadMfaStatus() {
  const result = await defMfaService.GetMfaStatus({});
  mfaEnabled.value = result.enabled;
  mfaPolicy.value = result.policy;
  mfaMethod.value = result.method || "totp";
}

/** 打开 MFA 绑定弹窗。 */
function openMfaSetup() {
  resetMfaDialog();
  mfaDialogMode.value = "setup";
  mfaDialogVisible.value = true;
}

/** 打开 MFA 禁用弹窗。 */
function openMfaDisable() {
  resetMfaDialog();
  mfaDialogMode.value = "disable";
  mfaDialogVisible.value = true;
}

/** 打开恢复码重新生成弹窗。 */
function openMfaRecovery() {
  resetMfaDialog();
  mfaDialogMode.value = "recovery";
  mfaDialogVisible.value = true;
}

/** 开始绑定 MFA。 */
async function beginMfaSetup(password: PasswordCrypto) {
  mfaLoading.value = true;
  try {
    const result = await defMfaService.BeginMfaSetup({ password, setup_ticket: "" });
    mfaSetupTicket.value = result.setup_ticket;
    mfaSetupUri.value = result.otpauth_uri;
    mfaSetupMethod.value = result.method || mfaMethod.value || "totp";
    mfaSetupWebAuthnOptionsJson.value = result.webauthn_options_json || "";
  } finally {
    mfaLoading.value = false;
  }
}

/** 确认绑定 MFA。 */
async function confirmMfaSetup() {
  if (!mfaSetupTicket.value || (mfaSetupMethod.value !== "webauthn" && !mfaForm.code)) return;
  mfaLoading.value = true;
  try {
    const webauthnResponseJson =
      mfaSetupMethod.value === "webauthn" ? await createWebAuthnCredential(mfaSetupWebAuthnOptionsJson.value) : "";
    const result = await defMfaService.ConfirmMfaSetup({
      setup_ticket: mfaSetupTicket.value,
      code: mfaForm.code,
      webauthn_response_json: webauthnResponseJson
    });
    recoveryCodes.value = result.recovery_codes;
    mfaEnabled.value = true;
    mfaDialogVisible.value = false;
    recoveryCodesDialogVisible.value = true;
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t("common.message.request_error"));
  } finally {
    mfaLoading.value = false;
  }
}

/** 禁用 MFA。 */
async function disableMfa() {
  if (!mfaForm.password) return;
  if (mfaFactorEnabled.value && mfaMethod.value === "totp" && !mfaForm.code && !mfaForm.recoveryCode) {
    ElMessage.error(t("system.profile.security.message.mfa_disable_factor_required"));
    return;
  }
  mfaLoading.value = true;
  try {
    const password = await encryptPassword(mfaForm.password, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA);
    if (mfaForm.recoveryCode) {
      await defMfaService.DisableMfa({
        password,
        code: "",
        webauthn_challenge_id: "",
        webauthn_response_json: "",
        recovery_code: mfaForm.recoveryCode
      });
    } else if (mfaMethod.value === "webauthn") {
      const challenge = await defMfaService.BeginMfaDisable({});
      const webauthnResponseJson = await getWebAuthnAssertion(challenge.webauthn_options_json);
      await defMfaService.DisableMfa({
        password,
        code: "",
        webauthn_challenge_id: challenge.challenge_id,
        webauthn_response_json: webauthnResponseJson,
        recovery_code: ""
      });
    } else {
      await defMfaService.DisableMfa({
        password,
        code: mfaForm.code,
        webauthn_challenge_id: "",
        webauthn_response_json: "",
        recovery_code: ""
      });
    }
    mfaEnabled.value = false;
    ElMessage.success(t("system.profile.security.message.mfa_disabled"));
    await userStore.clearAuthData();
    await router.replace({ path: "/login" });
  } finally {
    mfaLoading.value = false;
  }
}

/** 重新生成恢复码。 */
async function regenerateRecoveryCodes() {
  if (!mfaForm.code) return;
  mfaLoading.value = true;
  try {
    const result = await defMfaService.RegenerateMfaRecoveryCodes({ code: mfaForm.code });
    recoveryCodes.value = result.recovery_codes;
  } finally {
    mfaLoading.value = false;
  }
}

/** 复制当前显示的 MFA 恢复码。 */
async function copyRecoveryCodes() {
  await copyText(recoveryCodesText.value);
  ElMessage.success(t("system.profile.security.message.mfa_recovery_codes_copied"));
}

/** 关闭绑定成功后的恢复码弹窗并重新登录。 */
async function handleRecoveryCodesClosed() {
  if (!recoveryCodes.value.length) return;
  ElMessage.success(t("system.profile.security.message.mfa_enabled"));
  recoveryCodes.value = [];
  await userStore.clearAuthData();
  await router.replace({ path: "/login" });
}

/** 校验当前 MFA 表单并执行对应步骤。 */
async function handleMfaAction() {
  if (!(await mfaFormRef.value?.validate())) return;
  if (mfaDialogMode.value === "disable") {
    await disableMfa();
    return;
  }
  if (mfaDialogMode.value === "recovery") {
    await regenerateRecoveryCodes();
    return;
  }
  if (mfaSetupUri.value || (mfaSetupMethod.value === "webauthn" && mfaSetupTicket.value)) {
    await confirmMfaSetup();
    return;
  }
  try {
    const password = await encryptPassword(mfaForm.password, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_MFA);
    await beginMfaSetup(password);
    mfaForm.password = "";
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t("common.message.request_error"));
  }
}

/** 重置 MFA 弹窗状态。 */
function resetMfaDialog() {
  mfaForm.password = "";
  mfaForm.code = "";
  mfaForm.recoveryCode = "";
  mfaFormRef.value?.clearValidate();
  mfaSetupTicket.value = "";
  mfaSetupUri.value = "";
  mfaSetupMethod.value = "totp";
  mfaSetupWebAuthnOptionsJson.value = "";
  // 绑定弹窗关闭与恢复码弹窗打开同时进行时，保留本次绑定返回的恢复码。
  if (!recoveryCodesDialogVisible.value) {
    recoveryCodes.value = [];
  }
}

/** 根据当前语言返回三方登录方式名称。 */
function oauthName(binding: SecurityOauthBinding) {
  return binding.nameKey.includes(".") ? t(binding.nameKey) : binding.nameKey;
}

/** 处理 OAuth 绑定回跳结果。 */
async function consumeOauthBindingResult() {
  const bindError = route.query.oauth_bind_error;
  const bindSuccess = route.query.oauth_bind_success;
  if (typeof bindError === "string" && bindError) {
    ElMessage.error(bindError);
  } else if (bindSuccess === "1") {
    ElMessage.success(t("system.profile.security.message.oauth_bind_success"));
  } else {
    return;
  }
  await router.replace({
    path: route.path,
    query: {
      ...route.query,
      oauth_bind_provider: undefined,
      oauth_bind_success: undefined,
      oauth_bind_error: undefined
    }
  });
}

/** 发起三方账号绑定授权。 */
async function handleBindOauth(binding: SecurityOauthBinding) {
  if (oauthLoadingProvider.value) return;
  oauthLoadingProvider.value = binding.provider;
  try {
    const result = await defProfileOauthService.CreateOauthBindingAuthorization({
      provider: binding.provider,
      redirect_url: getCurrentSecurityPath()
    });
    if (result.authorization_url) {
      window.location.href = result.authorization_url;
    }
  } finally {
    oauthLoadingProvider.value = "";
  }
}

/** 解绑三方账号并刷新绑定状态。 */
async function handleUnbindOauth(binding: SecurityOauthBinding) {
  await ElMessageBox.confirm(
    t("system.profile.security.dialog.confirm_unbind", { provider: oauthName(binding) }),
    t("common.title.warning"),
    {
      type: "warning",
      confirmButtonText: t("system.profile.security.action.unbind"),
      cancelButtonText: t("common.action.cancel")
    }
  );
  oauthLoadingProvider.value = binding.provider;
  try {
    await defProfileOauthService.UnbindOauthAccount({ provider: binding.provider });
    ElMessage.success(t("system.profile.security.message.oauth_unbind_success"));
    await loadOauthBindings();
  } finally {
    oauthLoadingProvider.value = "";
  }
}

/** 打开手机号绑定弹窗，并回填当前手机号。 */
function openPhoneDialog() {
  phoneForm.phone = props.profile.phone;
  phoneForm.code = "";
  phoneDialogVisible.value = true;
}

/** 发送手机验证码并启动倒计时。 */
async function handleSendCode() {
  if (!phoneForm.phone) {
    ElMessage.error(t("system.profile.security.placeholder.phone"));
    return;
  }
  if (!/^1[3-9]\d{9}$/.test(phoneForm.phone)) {
    ElMessage.error(t("system.profile.security.validation.phone"));
    return;
  }

  sendPhoneCodeForm.phone = phoneForm.phone;
  await defProfileAuthService.SendPhoneCode(sendPhoneCodeForm);
  ElMessage.success(t("system.profile.security.message.code_sent"));
  startCountdown();
}

/** 提交绑定手机号请求。 */
async function handleSubmitPhone() {
  if (!(await phoneFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defProfileAuthService.UpdateUserPhone({ user_phone: phoneForm });
    ElMessage.success(t("system.profile.security.message.phone_updated"));
    phoneDialogVisible.value = false;
    emit("refreshed");
  } finally {
    submitLoading.value = false;
  }
}

/** 启动验证码倒计时，并在重复发送前进行限制。 */
function startCountdown() {
  clearCountdown();
  countdown.value = 60;
  phoneTimer.value = window.setInterval(() => {
    if (countdown.value <= 1) {
      clearCountdown();
      return;
    }
    countdown.value -= 1;
  }, 1000);
}

/** 清理验证码倒计时。 */
function clearCountdown() {
  if (phoneTimer.value !== null) {
    window.clearInterval(phoneTimer.value);
    phoneTimer.value = null;
  }
  countdown.value = 0;
}

/** 弹窗关闭后重置临时表单状态。 */
function handleDialogClosed() {
  phoneFormRef.value?.resetFields();
  phoneFormRef.value?.clearValidate();
  phoneForm.phone = props.profile.phone;
  phoneForm.code = "";
}

onBeforeUnmount(() => {
  clearCountdown();
});

onMounted(async () => {
  await consumeOauthBindingResult();
  await loadOauthBindings();
  await loadMfaStatus();
});
</script>

<style scoped lang="scss">
.profile-security {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.security-card {
  border: 1px solid #ebeef5;
  border-radius: var(--admin-page-radius);
}
:deep(.security-card .el-card__body) {
  padding: 20px;
}
.security-intro {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
}
.security-intro h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}
.security-intro p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}
.security-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 16px;
}
.security-item {
  display: flex;
  gap: 18px;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #ffffff;
  border: 1px solid #f0f2f5;
  border-radius: var(--admin-page-radius);
}
.security-item__content {
  min-width: 0;
}
.security-item__content--oauth {
  display: flex;
  gap: 12px;
  align-items: center;
}
.oauth-icon {
  display: inline-flex;
  flex: 0 0 36px;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border-radius: 50%;

  svg,
  img {
    width: 28px;
    height: 28px;
    object-fit: contain;
  }
}
.security-item strong {
  display: block;
  margin-bottom: 6px;
  font-size: 15px;
  color: #303133;
}
.security-item p {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #909399;
}
.security-item__hint {
  margin-top: 4px !important;
  color: var(--el-color-warning) !important;
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
.security-item__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}
.mfa-dialog-content {
  min-width: 0;
}
.mfa-dialog-recovery {
  margin-top: 12px;
}
.mfa-copyable-value {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  overflow-wrap: anywhere;
}
.mfa-copyable-value__text {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
}
.mfa-copy-button {
  flex: 0 0 auto;
  margin-top: -4px;
}
.mfa-copy-button__icon {
  color: var(--el-color-primary);
  font-size: 18px;
}
.mfa-recovery-hint {
  line-height: 1.6;
  color: var(--el-text-color-regular);
}
.mfa-recovery-hint {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--el-color-warning);
}
.mfa-dialog-content :deep(.pro-form) {
  margin: 0;
}
.mfa-dialog-content :deep(.mfa-dialog-form .el-form-item) {
  margin-bottom: 16px;
}

@media screen and (width <= 768px) {
  .security-intro,
  .security-item {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
