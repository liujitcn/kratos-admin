<template>
  <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" size="large">
    <el-form-item prop="tenant_code">
      <el-input v-model="loginForm.tenant_code" placeholder="请输入租户编码">
        <template #prefix>
          <el-icon class="el-input__icon">
            <office-building />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="user_name">
      <el-input v-model="loginForm.user_name" placeholder="请输入用户名">
        <template #prefix>
          <el-icon class="el-input__icon">
            <user />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item prop="password">
      <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" show-password autocomplete="new-password">
        <template #prefix>
          <el-icon class="el-input__icon">
            <lock />
          </el-icon>
        </template>
      </el-input>
    </el-form-item>
    <el-form-item v-if="!isBehaviorCaptcha" prop="captcha_code">
      <div class="captcha-row">
        <el-input v-model="loginForm.captcha_code" placeholder="请输入验证码" @keyup.enter="handleLogin(loginFormRef)">
          <template #prefix>
            <el-icon class="el-input__icon">
              <Key />
            </el-icon>
          </template>
        </el-input>
        <img
          v-if="captcha_base64"
          class="captcha-image"
          :style="{ width: captchaImageWidth }"
          :src="captcha_base64"
          alt="验证码"
          @load="handleCaptchaImageLoad"
          @click="getCaptcha"
        />
      </div>
    </el-form-item>
  </el-form>
  <div class="login-btn">
    <el-button :icon="CircleClose" round size="large" @click="resetForm(loginFormRef)"> 重置 </el-button>
    <el-button :icon="UserFilled" round size="large" type="primary" :loading="loading || oauthTicketLoading" @click="handleLogin(loginFormRef)">
      登录
    </el-button>
  </div>
  <div v-if="oauthProviders.length" class="oauth-login">
    <div class="oauth-divider">
      <span>其他登录方式</span>
    </div>
    <div class="oauth-provider-list">
      <el-tooltip
        v-for="provider in oauthProviders"
        :key="provider.provider"
        :content="provider.name"
        placement="top"
        :trigger="['hover', 'focus']"
      >
        <button
          class="oauth-provider-button"
          type="button"
          :aria-label="provider.name"
          :title="provider.name"
          :disabled="oauthTicketLoading || oauthLoadingProvider === provider.provider"
          @click="handleOauthLogin(provider)"
        >
          <component :is="getOauthProviderIcon(provider)" />
        </button>
      </el-tooltip>
    </div>
  </div>
  <el-dialog
    v-model="behaviorDialogVisible"
    width="364px"
    top="16vh"
    :show-close="false"
    append-to-body
    class="behavior-captcha-dialog"
  >
    <div v-loading="behaviorLoading" class="behavior-captcha-body">
      <GoCaptchaSlide
        v-if="currentCaptchaType === 'slide'"
        :config="slideCaptchaConfig"
        :data="behaviorCaptchaData"
        :events="slideCaptchaEvents"
      />
      <GoCaptchaClick
        v-else-if="currentCaptchaType === 'click'"
        :config="clickCaptchaConfig"
        :data="behaviorCaptchaData"
        :events="clickCaptchaEvents"
      />
      <GoCaptchaRotate
        v-else-if="currentCaptchaType === 'rotate'"
        :config="rotateCaptchaConfig"
        :data="behaviorCaptchaData"
        :events="rotateCaptchaEvents"
      />
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { HOME_URL } from "@/config";
import { getTimeState } from "@/utils";
import { defLoginService } from "@/api/base/login";
import { defOauthService } from "@/api/base/oauth";
import { ElMessage, ElNotification } from "element-plus";
import type { LoginRequest } from "@/rpc/base/v1/login";
import type { OauthProvider } from "@/rpc/base/v1/oauth";
import { getOauthProviderIcon, withOauthProviderDisplay, type OauthProviderDisplay } from "@/utils/oauthProvider";
import { useUserStore } from "@/stores/modules/user";
import { useDictStore } from "@/stores/modules/dict";
import { useTabsStore } from "@/stores/modules/tabs";
import { useKeepAliveStore } from "@/stores/modules/keepAlive";
import { initDynamicRouter } from "@/routers/modules/dynamicRouter";
import { isUnmatchedRoute, navigateTo, resolveFrontendRouteURL } from "@/utils/router";
import { CircleClose, UserFilled } from "@element-plus/icons-vue";
import type { ElForm } from "element-plus";
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from "@/utils/passwordCrypto";
import { useConfigStore } from "@/stores/modules/config";
import { Click as GoCaptchaClick, Rotate as GoCaptchaRotate, Slide as GoCaptchaSlide } from "go-captcha-vue";
import "go-captcha-vue/dist/style.css";

const router = useRouter();
const route = useRoute();
const userStore = useUserStore();
const configStore = useConfigStore();
const dictStore = useDictStore();
const tabsStore = useTabsStore();
const keepAliveStore = useKeepAliveStore();

/** 登录表单实例类型。 */
type FormInstance = InstanceType<typeof ElForm>;
const loginFormRef = ref<FormInstance>();
const captcha_base64 = ref("");
const defaultCaptchaImageWidth = 96;
const captchaImageWidth = ref(`${defaultCaptchaImageWidth}px`);
const behaviorDialogVisible = ref(false);
const behaviorLoading = ref(false);
const loginRules = reactive({
  tenant_code: [{ required: true, message: "请输入租户编码", trigger: "blur" }],
  user_name: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
  captcha_code: [{ required: true, message: "请输入验证码", trigger: "blur" }]
});

const loading = ref(false);
const oauthLoadingProvider = ref("");
const oauthTicketLoading = ref(false);

/** 登录页三方登录展示项。 */
type LoginOauthProvider = OauthProvider & OauthProviderDisplay;

const oauthProviders = ref<LoginOauthProvider[]>([]);

/** 登录表单状态。 */
interface LoginFormState {
  /** 租户编码。 */
  tenant_code: string;
  /** 用户名。 */
  user_name: string;
  /** 密码。 */
  password: string;
  /** 验证码。 */
  captcha_code: string;
  /** 验证码 ID。 */
  captcha_id: string;
}

const loginForm = reactive<LoginFormState>({
  tenant_code: "",
  user_name: "",
  password: "",
  captcha_code: "",
  captcha_id: ""
});

/** 行为验证码类型集合。 */
const behaviorCaptchaTypeSet = new Set(["slide", "click", "rotate"]);

/** 行为验证码服务端返回的图片载荷。 */
interface BehaviorCaptchaPayload {
  type?: string;
  image: string;
  thumb: string;
  width?: number;
  height?: number;
  thumbX?: number;
  thumbY?: number;
  thumbWidth?: number;
  thumbHeight?: number;
  thumbSize?: number;
}

/** 行为验证码组件使用的坐标点。 */
interface CaptchaPoint {
  x: number;
  y: number;
}

/** 点击验证码组件提交的坐标点。 */
interface ClickCaptchaPoint extends CaptchaPoint {
  key?: number;
  index?: number;
}

/** 行为验证码组件数据。 */
interface BehaviorCaptchaData {
  image: string;
  thumb: string;
  thumbX?: number;
  thumbY?: number;
  thumbWidth?: number;
  thumbHeight?: number;
  thumbSize?: number;
  angle?: number;
}

const behaviorCaptchaData = reactive<BehaviorCaptchaData>({
  image: "",
  thumb: ""
});
const configuredCaptchaType = computed(() => configStore.captcha.type || "digit");
const currentCaptchaType = ref(configuredCaptchaType.value);
const isBehaviorCaptcha = computed(() => behaviorCaptchaTypeSet.has(currentCaptchaType.value));
const behaviorCaptchaDisplayWidth = 340;
const rotateCaptchaDisplaySize = 300;
const behaviorCaptchaSource = reactive({
  width: 300,
  height: 220,
  thumbWidth: 60,
  thumbHeight: 60,
  thumbSize: 150
});
const behaviorCaptchaDisplayHeight = computed(() => Math.round((behaviorCaptchaSource.height * behaviorCaptchaDisplayWidth) / behaviorCaptchaSource.width));
const behaviorCaptchaScaleX = computed(() => behaviorCaptchaDisplayWidth / behaviorCaptchaSource.width);
const behaviorCaptchaScaleY = computed(() => behaviorCaptchaDisplayHeight.value / behaviorCaptchaSource.height);
const rotateCaptchaScale = computed(() => rotateCaptchaDisplaySize / behaviorCaptchaSource.width);

/** 将服务端原始 X 坐标换算为前端展示坐标。 */
const toDisplayCaptchaX = (value: number) => Math.round(value * behaviorCaptchaScaleX.value);

/** 将服务端原始 Y 坐标换算为前端展示坐标。 */
const toDisplayCaptchaY = (value: number) => Math.round(value * behaviorCaptchaScaleY.value);

/** 将服务端原始旋转内圈尺寸换算为前端展示尺寸。 */
const toDisplayRotateSize = (value: number) => Math.round(value * rotateCaptchaScale.value);

/** 将前端展示 X 坐标换算回服务端原始坐标。 */
const toOriginalCaptchaX = (value: number) => Math.min(behaviorCaptchaSource.width, Math.max(0, Math.round(value / behaviorCaptchaScaleX.value)));

/** 将前端展示 Y 坐标换算回服务端原始坐标。 */
const toOriginalCaptchaY = (value: number) => Math.min(behaviorCaptchaSource.height, Math.max(0, Math.round(value / behaviorCaptchaScaleY.value)));
const slideCaptchaConfig = computed(() => ({
  width: behaviorCaptchaDisplayWidth,
  height: behaviorCaptchaDisplayHeight.value,
  thumbWidth: toDisplayCaptchaX(behaviorCaptchaSource.thumbWidth),
  thumbHeight: toDisplayCaptchaY(behaviorCaptchaSource.thumbHeight),
  showTheme: false,
  verticalPadding: 0,
  horizontalPadding: 0,
  iconSize: 20,
  title: "请拖动滑块完成拼图"
}));
const clickCaptchaConfig = computed(() => ({
  width: behaviorCaptchaDisplayWidth,
  height: behaviorCaptchaDisplayHeight.value,
  thumbWidth: toDisplayCaptchaX(behaviorCaptchaSource.thumbWidth),
  thumbHeight: toDisplayCaptchaY(behaviorCaptchaSource.thumbHeight),
  showTheme: false,
  verticalPadding: 0,
  horizontalPadding: 0,
  buttonText: "确认",
  iconSize: 20,
  dotSize: 20,
  title: "请按顺序点击文字"
}));
const rotateCaptchaConfig = computed(() => ({
  width: behaviorCaptchaDisplayWidth,
  height: rotateCaptchaDisplaySize,
  size: rotateCaptchaDisplaySize,
  showTheme: false,
  verticalPadding: 0,
  horizontalPadding: 0,
  iconSize: 20,
  title: "拖动滑块，将内圈图片转正"
}));
const slideCaptchaEvents = {
  refresh: () => getCaptcha(),
  close: () => closeBehaviorCaptcha(),
  confirm: (point: CaptchaPoint, reset: () => void) => verifyBehaviorCaptcha(String(toOriginalCaptchaX(point.x)), reset)
};
const clickCaptchaEvents = {
  refresh: () => getCaptcha(),
  close: () => closeBehaviorCaptcha(),
  confirm: (dots: ClickCaptchaPoint[], reset: () => void) =>
    verifyBehaviorCaptcha(JSON.stringify(dots.map(dot => ({ x: toOriginalCaptchaX(dot.x), y: toOriginalCaptchaY(dot.y) }))), reset)
};
const rotateCaptchaEvents = {
  refresh: () => getCaptcha(),
  close: () => closeBehaviorCaptcha(),
  confirm: (angle: number, reset: () => void) => verifyBehaviorCaptcha(String(Math.round(angle)), reset)
};

/** 获取登录后的首个可访问路由 */
const getFirstAccessibleRoutePath = () => {
  // 首页仅在真正完成动态路由注册后才视为可达，避免把全局 404 占位路由误判为首页已加载。
  if (!isUnmatchedRoute(router, HOME_URL)) return HOME_URL;

  const systemRouteSet = new Set(["/", "/layout", "/login", "/403", "/404", "/500"]);
  const firstRoute = router.getRoutes().find(item => {
    if (!item.path || systemRouteSet.has(item.path) || item.path.includes(":pathMatch")) return false;
    return item.meta?.hidden !== true;
  });
  return firstRoute?.path ?? HOME_URL;
};

/** 获取登录成功后的回跳地址，优先使用登录前记录的完整页面地址。 */
const getLoginRedirectPath = () => {
  const redirect = route.query.redirect;
  if (typeof redirect === "string" && redirect && redirect !== HOME_URL) {
    return normalizeFrontendRedirectPath(redirect);
  }
  return getFirstAccessibleRoutePath();
};

/** 将 OAuth 回调带回的同源绝对地址还原为 Vue Router 可识别的站内路径。 */
const normalizeFrontendRedirectPath = (redirect: string) => {
  try {
    const redirectURL = new URL(redirect, window.location.origin);
    if (redirectURL.origin !== window.location.origin) return HOME_URL;
    if (redirectURL.hash.startsWith("#/")) return redirectURL.hash.slice(1);
    return `${redirectURL.pathname}${redirectURL.search}`;
  } catch {
    return HOME_URL;
  }
};

/** 获取 OAuth 登录完成后的前端接收地址，并携带账号登录一致的业务回跳目标。 */
const getOauthLoginRedirectURL = () => {
  const query = { ...route.query };
  delete query.oauth_ticket;
  delete query.oauth_error;
  query.redirect = getLoginRedirectPath();
  return resolveFrontendRouteURL(router, { path: route.path, query });
};

/** 查询配置启用的三方登录方式。 */
const loadOauthProviders = async () => {
  const result = await defOauthService.ListOauthProvider({});
  oauthProviders.value = result.providers.map(withOauthProviderDisplay);
};

/** 发起三方登录授权跳转。 */
const handleOauthLogin = async (provider: LoginOauthProvider) => {
  // 授权地址创建期间锁定当前按钮，避免同一个 provider 重复创建 state。
  if (oauthLoadingProvider.value) return;
  oauthLoadingProvider.value = provider.provider;
  try {
    const result = await defOauthService.CreateOauthAuthorization({
      provider: provider.provider,
      redirect_url: getOauthLoginRedirectURL()
    });
    if (result.authorization_url) {
      window.location.href = result.authorization_url;
    }
  } finally {
    oauthLoadingProvider.value = "";
  }
};

/** 完成登录后的用户信息、字典与动态路由初始化。 */
const finishLogin = async () => {
  // 1.获取用户信息
  await userStore.getUserInfo();

  // 2.预加载字典缓存，避免页面首次渲染时字典值为空
  await dictStore.loadDictionaries();

  // 3.添加动态路由
  await initDynamicRouter();

  // 4.清空 tabs、keepAlive 数据
  tabsStore.setTabs([]);
  keepAliveStore.setKeepAliveName([]);

  // 5.优先跳回登录失效前页面，没有记录时再进入首个可访问页面。
  // 统一走动态路由感知跳转，避免首次登录后目标页面尚未完成挂载时直接进入 404。
  await navigateTo(router, getLoginRedirectPath());
  ElNotification({
    title: getTimeState(),
    message: "欢迎登录管理后台",
    type: "success",
    duration: 3000
  });
};

/** 消费 OAuth 回调携带的一次性票据。 */
const consumeOauthTicket = async () => {
  const oauthError = route.query.oauth_error;
  if (typeof oauthError === "string" && oauthError) {
    ElMessage.error(oauthError);
    await router.replace({ path: route.path, query: { ...route.query, oauth_error: undefined, oauth_ticket: undefined } });
    return;
  }

  const ticket = route.query.oauth_ticket;
  if (typeof ticket !== "string" || !ticket) return;

  oauthTicketLoading.value = true;
  try {
    const result = await defOauthService.ExchangeOauthTicket({ ticket });
    userStore.updateTokenAuth(result.access_token, result.refresh_token ?? "", result.token_type ?? "", result.expires_in);
    await finishLogin();
  } finally {
    oauthTicketLoading.value = false;
  }
};

/** 获取验证码 */
const getCaptcha = async () => {
  const requestedCaptchaType = configuredCaptchaType.value;
  const data = await defLoginService.Captcha({ type: requestedCaptchaType });
  currentCaptchaType.value = data.type || requestedCaptchaType;
  if (!isBehaviorCaptcha.value) {
    behaviorDialogVisible.value = false;
  }
  loginForm.captcha_id = data.captcha_id;
  loginForm.captcha_code = "";
  captchaImageWidth.value = `${defaultCaptchaImageWidth}px`;
  captcha_base64.value = isBehaviorCaptcha.value ? "" : data.captcha_base64;
  if (isBehaviorCaptcha.value) {
    applyBehaviorCaptchaPayload(data.captcha_base64);
  }
};

/** 页面加载或普通表单刷新验证码，行为验证码延迟到登录弹窗打开时再请求。 */
const loadPageCaptcha = async () => {
  if (isBehaviorCaptcha.value) {
    loginForm.captcha_id = "";
    loginForm.captcha_code = "";
    captcha_base64.value = "";
    return;
  }
  await getCaptcha();
};

/** 解析行为验证码图片载荷并映射为官方组件数据。 */
const applyBehaviorCaptchaPayload = (payloadText: string) => {
  const payload = JSON.parse(payloadText || "{}") as BehaviorCaptchaPayload;
  const payloadType = payload.type || currentCaptchaType.value;
  behaviorCaptchaSource.width = payload.width || 300;
  behaviorCaptchaSource.height = payload.height || (payloadType === "rotate" ? 300 : 220);
  behaviorCaptchaSource.thumbWidth = payload.thumbWidth || (payloadType === "click" ? 180 : 60);
  behaviorCaptchaSource.thumbHeight = payload.thumbHeight || (payloadType === "click" ? 48 : 60);
  behaviorCaptchaSource.thumbSize = payload.thumbSize || 150;
  behaviorCaptchaData.image = payload.image || "";
  behaviorCaptchaData.thumb = payload.thumb || "";
  behaviorCaptchaData.thumbX = toDisplayCaptchaX(payload.thumbX ?? 0);
  behaviorCaptchaData.thumbY = toDisplayCaptchaY(payload.thumbY ?? 0);
  behaviorCaptchaData.thumbWidth = toDisplayCaptchaX(behaviorCaptchaSource.thumbWidth);
  behaviorCaptchaData.thumbHeight = toDisplayCaptchaY(behaviorCaptchaSource.thumbHeight);
  behaviorCaptchaData.thumbSize = toDisplayRotateSize(behaviorCaptchaSource.thumbSize);
  behaviorCaptchaData.angle = 0;
};

/** 根据验证码图片原始比例更新展示宽度。 */
const handleCaptchaImageLoad = (event: Event) => {
  const image = event.target as HTMLImageElement;
  if (!image.naturalWidth || !image.naturalHeight) return;

  // 验证码固定展示高度，宽度按图片比例自适应，避免算术验证码横向内容被裁剪。
  const width = Math.round((40 * image.naturalWidth) / image.naturalHeight);
  captchaImageWidth.value = `${Math.min(Math.max(width, defaultCaptchaImageWidth), 180)}px`;
};

/** 预校验验证码并返回可用于登录的一次性令牌。 */
const verifyCaptchaToken = async (captchaCode: string) => {
  const result = await defLoginService.VerifyCaptcha({
    captcha_id: loginForm.captcha_id,
    captcha_code: captchaCode
  });
  return result.captcha_token;
};

/** 执行真正的账号登录流程。 */
const submitLogin = async (captchaToken: string) => {
  loading.value = true;
  try {
    const password = await encryptPassword(loginForm.password, PASSWORD_CRYPTO_SCENE.LOGIN);
    const loginRequest: LoginRequest = {
      tenant_code: loginForm.tenant_code,
      user_name: loginForm.user_name,
      password,
      captcha_code: captchaToken,
      captcha_id: loginForm.captcha_id
    };
    // 1.执行登录接口
    await userStore.login(loginRequest);

    await finishLogin();
  } catch (_error) {
    await loadPageCaptcha();
  } finally {
    loading.value = false;
  }
};

/** 验证行为验证码并继续登录。 */
const verifyBehaviorCaptcha = async (captchaCode: string, reset: () => void) => {
  if (behaviorLoading.value) return;
  behaviorLoading.value = true;
  try {
    const captchaToken = await verifyCaptchaToken(captchaCode);
    behaviorDialogVisible.value = false;
    await submitLogin(captchaToken);
  } catch (_error) {
    reset();
    await getCaptcha();
  } finally {
    behaviorLoading.value = false;
  }
};

/** 关闭行为验证码弹窗。 */
const closeBehaviorCaptcha = () => {
  behaviorDialogVisible.value = false;
};

/** 打开行为验证码弹窗。 */
const openBehaviorCaptcha = async () => {
  behaviorDialogVisible.value = true;
  behaviorLoading.value = true;
  try {
    await getCaptcha();
  } finally {
    behaviorLoading.value = false;
  }
};

/** 登录 */
const handleLogin = (formEl: FormInstance | undefined) => {
  // 登录请求执行期间直接拦截重复提交，避免回车或连续点击导致重复登录。
  if (!formEl || loading.value) return;
  formEl.validate(async valid => {
    if (!valid) return;
    if (isBehaviorCaptcha.value) {
      await openBehaviorCaptcha();
      return;
    }
    try {
      const captchaToken = await verifyCaptchaToken(loginForm.captcha_code);
      await submitLogin(captchaToken);
    } catch (_error) {
      await loadPageCaptcha();
    }
  });
};

/** 重置表单 */
const resetForm = (formEl: FormInstance | undefined) => {
  if (!formEl) return;
  formEl.resetFields();
  void loadPageCaptcha();
};

onMounted(() => {
  void loadPageCaptcha();
  void loadOauthProviders();
});

watch(
  () => [route.query.oauth_ticket, route.query.oauth_error],
  () => {
    void consumeOauthTicket();
  },
  { immediate: true }
);
</script>

<style scoped lang="scss">
@use "../index.scss" as *;
.captcha-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;

  .el-input {
    flex: 1;
    min-width: 0;
  }
}
.captcha-image {
  flex: 0 0 auto;
  height: 40px;
  cursor: pointer;
  object-fit: contain;
  border-radius: 6px;
}
.behavior-captcha-body {
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.oauth-login {
  margin-top: 22px;
}

.oauth-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--el-text-color-primary);
  white-space: nowrap;

  &::before,
  &::after {
    flex: 1;
    height: 1px;
    content: "";
    background-color: var(--el-border-color-lighter);
  }
}

.oauth-provider-list {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  padding-bottom: 4px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: thin;
}

.oauth-provider-button {
  display: inline-flex;
  flex: 0 0 36px;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 50%;
  transition:
    color 0.2s ease,
    transform 0.2s ease,
    background-color 0.2s ease;

  &:hover:not(:disabled) {
    background-color: var(--el-fill-color-light);
    transform: translateY(-1px);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }

  svg,
  img {
    width: 26px;
    height: 26px;
    object-fit: contain;
  }
}

.oauth-provider-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  font-size: 14px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-color-primary);
}

:global(.behavior-captcha-dialog) {
  --go-captcha-theme-text-color: var(--el-text-color-primary);
  --go-captcha-theme-bg-color: var(--el-bg-color);
  --go-captcha-theme-btn-bg-color: var(--el-color-primary);
  --go-captcha-theme-btn-border-color: var(--el-color-primary);
  --go-captcha-theme-btn-disabled-color: var(--el-color-primary-light-5);
  --go-captcha-theme-active-color: var(--el-color-primary);
  --go-captcha-theme-border-color: var(--el-border-color-light);
  --go-captcha-theme-icon-color: var(--el-text-color-regular);
  --go-captcha-theme-drag-bar-color: var(--el-fill-color);
  --go-captcha-theme-drag-bg-color: var(--el-color-primary);
  --go-captcha-theme-drag-icon-color: #ffffff;
  --go-captcha-theme-round-color: var(--el-fill-color);
  --go-captcha-theme-loading-icon-color: var(--el-color-primary);
  --go-captcha-theme-body-bg-color: var(--el-fill-color-light);
  --go-captcha-theme-dot-color-color: #ffffff;
  --go-captcha-theme-dot-bg-color: color-mix(in srgb, var(--el-color-primary) 68%, transparent);
  --go-captcha-theme-dot-border-color: var(--el-bg-color);
  border-radius: 10px;
  box-shadow: rgb(0 0 0 / 10%) 0 2px 10px 2px;
}

:global(.behavior-captcha-dialog .el-dialog__header) {
  display: none;
}

:global(.behavior-captcha-dialog .el-dialog__body) {
  padding: 10px 12px 12px;
}

:global(.behavior-captcha-dialog .go-captcha .gc-header) {
  height: auto;
  min-height: 24px;
  margin-bottom: 6px;
}

:global(.behavior-captcha-dialog .go-captcha .gc-header span),
:global(.behavior-captcha-dialog .go-captcha .gc-header .gc-text) {
  font-size: 14px;
  line-height: 22px;
  white-space: nowrap;
}

:global(.behavior-captcha-dialog .go-captcha .gc-body) {
  margin-top: 0;
  border-radius: 6px;
}

:global(.behavior-captcha-dialog .go-captcha .gc-footer) {
  height: 38px;
  padding-top: 8px;
}

:global(.behavior-captcha-dialog .go-captcha .gc-drag-line) {
  height: 12px;
  margin-top: -6px;
  border-radius: 999px;
}

:global(.behavior-captcha-dialog .go-captcha .gc-drag-block) {
  background: linear-gradient(135deg, var(--el-color-primary-light-3) 0%, var(--el-color-primary) 100%);
  box-shadow: 0 8px 18px color-mix(in srgb, var(--el-color-primary) 28%, transparent);
}

:global(.behavior-captcha-dialog .go-captcha .gc-drag-block.disabled) {
  background: var(--el-color-primary-light-5);
  box-shadow: none;
}

:global(.behavior-captcha-dialog .go-captcha .gc-rotate-picture .gc-round) {
  border-width: 4px;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--el-bg-color) 80%, transparent);
}

:global(.behavior-captcha-dialog .go-captcha .gc-rotate-thumb-block) {
  overflow: hidden;
  border: 2px solid var(--el-bg-color);
  border-radius: 50%;
  box-shadow:
    0 8px 18px rgb(0 0 0 / 18%),
    0 0 0 1px color-mix(in srgb, var(--el-border-color) 80%, transparent);
}

:global(.behavior-captcha-dialog .go-captcha .gc-rotate-thumb-block img) {
  border-radius: 50%;
}

:global(.behavior-captcha-dialog .go-captcha .gc-dots .gc-dot) {
  font-size: 12px;
  font-weight: 600;
  border-width: 2px;
  box-shadow: 0 3px 9px rgb(0 0 0 / 16%);
}
</style>
