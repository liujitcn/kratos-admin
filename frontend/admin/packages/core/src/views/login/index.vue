<template>
  <div class="login-container flx-center">
    <div class="login-box">
      <SwitchDark class="dark" />
      <LocaleSwitch v-if="languageOptions.length > 1" class="locale" />
      <div class="login-left">
        <img class="login-left-img" :src="backgroundUrl" :alt="t('core.login.background_alt')" @error="handleBackgroundError" />
      </div>
      <div ref="loginFormRef" class="login-form">
        <div ref="loginLogoRef" class="login-logo">
          <img v-show="showLogoIcon" class="login-icon" :src="logoUrl" alt="" @error="handleLogoError" />
          <h2 ref="logoTextRef" class="logo-text" :style="{ fontSize: `${logoFontSize}px` }">
            <span>{{ projectName }}</span>
            <span v-if="configStore.version" class="login-version">v{{ configStore.version }}</span>
          </h2>
        </div>
        <LoginForm :dialog-anchor="loginFormRef" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts" name="Login">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import LoginForm from "./components/LoginForm.vue";
import SwitchDark from "@/components/SwitchDark/index.vue";
import LocaleSwitch from "@/components/LocaleSwitch/index.vue";
import { useConfigStore } from "@/stores/modules/config";
import { useLocaleStore } from "@/locales";
import defaultBackgroundUrl from "@/assets/images/login_left.png";
import { useLogoUrl } from "@/hooks/useLogoUrl";
import { useImageUrl } from "@/hooks/useImageUrl";

const configStore = useConfigStore();
const { t, languageOptions } = useLocaleStore();
const projectName = computed(() => configStore.display.sysName || import.meta.env.VITE_GLOB_APP_TITLE);
const configuredLogoUrl = computed(() => configStore.display.adminLogo);
const { logoUrl, handleLogoError } = useLogoUrl(configuredLogoUrl);
const configuredBackgroundUrl = computed(() => configStore.display.background);
const { imageUrl: backgroundUrl, handleImageError: handleBackgroundError } = useImageUrl(
  configuredBackgroundUrl,
  defaultBackgroundUrl
);

const MAX_LOGO_FONT_SIZE = 42;
const PREFERRED_MIN_LOGO_FONT_SIZE = 26;
const MIN_LOGO_FONT_SIZE = 8;

const loginFormRef = ref<HTMLElement>();
const loginLogoRef = ref<HTMLElement>();
const logoTextRef = ref<HTMLElement>();
const logoFontSize = ref(MAX_LOGO_FONT_SIZE);
const showLogoIcon = ref(true);

let logoResizeObserver: ResizeObserver | null = null;

/**
 * 根据表单头部可用宽度动态缩放标题字号，优先保留文字大小，不够时先隐藏图标。
 */
async function updateLogoLayout() {
  await nextTick();

  const logoTextElement = logoTextRef.value;
  if (!logoTextElement) return;

  showLogoIcon.value = true;
  await fitLogoText(PREFERRED_MIN_LOGO_FONT_SIZE);

  if (logoTextElement.scrollWidth <= logoTextElement.clientWidth) return;

  showLogoIcon.value = false;
  await fitLogoText(MIN_LOGO_FONT_SIZE);
}

/**
 * 在当前图标显示状态下，按可用宽度逐步收缩标题字号。
 */
async function fitLogoText(minFontSize: number) {
  await nextTick();

  const logoTextElement = logoTextRef.value;
  if (!logoTextElement) return;

  let nextFontSize = MAX_LOGO_FONT_SIZE;
  logoFontSize.value = nextFontSize;
  await nextTick();

  while (nextFontSize > minFontSize && logoTextElement.scrollWidth > logoTextElement.clientWidth) {
    nextFontSize -= 1;
    logoFontSize.value = nextFontSize;
    await nextTick();
  }
}

onMounted(() => {
  updateLogoLayout();
  logoResizeObserver = new ResizeObserver(() => {
    updateLogoLayout();
  });
  if (loginLogoRef.value) {
    logoResizeObserver.observe(loginLogoRef.value);
  }
});

onBeforeUnmount(() => {
  logoResizeObserver?.disconnect();
  logoResizeObserver = null;
});

watch([projectName, logoUrl, () => configStore.version], () => {
  updateLogoLayout();
});
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
