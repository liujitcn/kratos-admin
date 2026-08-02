<template>
  <div class="profile-security">
    <el-card class="security-card" shadow="never">
      <div class="security-intro">
        <div>
          <h3>{{ t("system.profile.security.title") }}</h3>
          <p>{{ t("system.profile.security.description") }}</p>
        </div>
        <el-tag effect="plain" type="success">{{ t("system.profile.security.value.accountNormal") }}</el-tag>
      </div>
      <div class="security-list">
        <div class="security-item">
          <div class="security-item__content">
            <strong>{{ t("system.profile.security.field.loginPassword") }}</strong>
            <p>{{ t("system.profile.security.message.passwordHint") }}</p>
          </div>
          <el-button type="primary" plain @click="emit('switchTab', 'password')">
            {{ t("system.profile.security.action.changePassword") }}
          </el-button>
        </div>
        <div class="security-item">
          <div class="security-item__content">
            <strong>{{ t("system.profile.security.field.phone") }}</strong>
            <p>{{ mobileTip }}</p>
          </div>
          <el-button plain @click="openPhoneDialog">
            {{ profile.phone ? t("system.profile.security.action.changePhone") : t("system.profile.security.action.bindNow") }}
          </el-button>
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
                  item.bound ? t("system.profile.security.message.oauthBound") : t("system.profile.security.message.oauthUnbound")
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

    <el-card class="status-card" shadow="never">
      <template #header>
        <div class="status-header">
          <div>
            <h3>{{ t("system.profile.security.status.title") }}</h3>
            <p>{{ t("system.profile.security.status.description") }}</p>
          </div>
        </div>
      </template>
      <div class="status-grid">
        <div class="status-item">
          <span>{{ t("system.profile.security.field.phoneVerification") }}</span>
          <strong>{{ profile.phone ? t("common.status.enabled") : t("common.status.disabled") }}</strong>
        </div>
        <div class="status-item">
          <span>{{ t("system.profile.security.field.profileCompletion") }}</span>
          <strong>{{ profileCompletion }}</strong>
        </div>
      </div>
    </el-card>

    <ProDialog
      v-model="phoneDialogVisible"
      :title="t('system.profile.security.dialog.bindPhone')"
      :width="520"
      @closed="handleDialogClosed"
    >
      <ProForm ref="phoneFormRef" :model="phoneForm" :fields="phoneFormFields" :rules="phoneFormRules" label-width="96px">
        <template #mobileCodeInput>
          <el-input v-model="phoneForm.code" :placeholder="t('system.profile.security.placeholder.code')">
            <template #append>
              <el-button :disabled="countdown > 0" @click="handleSendCode">
                {{
                  countdown > 0
                    ? t("system.profile.security.action.retryAfter", { seconds: countdown })
                    : t("system.profile.security.action.sendCode")
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
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/auth";
import { defProfileOauthService } from "@liujitcn/kratos-admin-system/api/base/oauth";
import type { OauthBinding } from "@liujitcn/kratos-admin-system/rpc/base/v1/oauth";
import type {
  SendPhoneCodeRequest,
  UserPhoneForm,
  UserProfileForm
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { getOauthProviderIcon, withOauthProviderDisplay, type OauthProviderDisplay } from "@liujitcn/kratos-admin-core/security";
import { resolveFrontendRouteURL } from "@liujitcn/kratos-admin-core/navigation";

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
const phoneFormRef = ref<ProFormInstance>();
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

const phoneFormFields = computed<ProFormField[]>(() => [
  {
    prop: "phone",
    label: t("system.profile.security.field.phoneNumber"),
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
    ? t("system.profile.security.message.phoneBound", { phone: props.profile.phone })
    : t("system.profile.security.message.phoneUnbound");
});

/** 根据关键资料估算当前资料完成度。 */
const profileCompletion = computed(() => {
  const fieldList = [props.profile.nick_name, props.profile.phone, props.profile.role_name, props.profile.dept_name];
  const completedCount = fieldList.filter(item => Boolean(item)).length;
  return `${Math.round((completedCount / fieldList.length) * 100)}%`;
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
    ElMessage.success(t("system.profile.security.message.oauthBindSuccess"));
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
    t("system.profile.security.dialog.confirmUnbind", { provider: oauthName(binding) }),
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
    ElMessage.success(t("system.profile.security.message.oauthUnbindSuccess"));
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
  ElMessage.success(t("system.profile.security.message.codeSent"));
  startCountdown();
}

/** 提交绑定手机号请求。 */
async function handleSubmitPhone() {
  if (!(await phoneFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defProfileAuthService.UpdateUserPhone({ user_phone: phoneForm });
    ElMessage.success(t("system.profile.security.message.phoneUpdated"));
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
});
</script>

<style scoped lang="scss">
.profile-security {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.security-card,
.status-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
}
:deep(.security-card .el-card__body),
:deep(.status-card .el-card__body) {
  padding: 20px;
}
:deep(.status-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}
.security-intro,
.status-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
}
.security-intro h3,
.status-header h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}
.security-intro p,
.status-header p {
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
.security-item,
.status-item {
  display: flex;
  gap: 18px;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #ffffff;
  border: 1px solid #f0f2f5;
  border-radius: 10px;
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
.security-item strong,
.status-item strong {
  display: block;
  margin-bottom: 6px;
  font-size: 15px;
  color: #303133;
}
.security-item p,
.status-item span {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #909399;
}
.status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

@media screen and (width <= 768px) {
  .security-intro,
  .status-header,
  .security-item,
  .status-item {
    flex-direction: column;
    align-items: flex-start;
  }
  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
