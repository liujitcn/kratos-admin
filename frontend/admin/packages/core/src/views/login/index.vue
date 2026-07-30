<template>
  <div class="login-container flx-center">
    <div class="login-box">
      <SwitchDark class="dark" />
      <div class="login-left">
        <img class="login-left-img" :src="backgroundUrl" alt="login" />
      </div>
      <div class="login-form">
        <div ref="loginLogoRef" class="login-logo">
          <img v-show="showLogoIcon" ref="logoIconRef" class="login-icon" :src="logoUrl" alt="" />
          <h2 ref="logoTextRef" class="logo-text" :style="{ fontSize: `${logoFontSize}px` }">{{ projectName }}</h2>
        </div>
        <LoginForm />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts" name="Login">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import LoginForm from "./components/LoginForm.vue";
import SwitchDark from "@/components/SwitchDark/index.vue";
import { useConfigStore } from "@/stores/modules/config";

const configStore = useConfigStore();
const projectName = computed(() => configStore.display.sysName || import.meta.env.VITE_GLOB_APP_TITLE);
const logoUrl = computed(() => configStore.display.adminLogo);
const backgroundUrl = computed(() => configStore.display.background);

const MAX_LOGO_FONT_SIZE = 42;
const PREFERRED_MIN_LOGO_FONT_SIZE = 30;
const MIN_LOGO_FONT_SIZE = 8;
const LOGO_TEXT_GAP = 16;

const loginLogoRef = ref<HTMLElement>();
const logoIconRef = ref<HTMLImageElement>();
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
  await fitLogoText(true, PREFERRED_MIN_LOGO_FONT_SIZE);

  if (logoTextElement.scrollWidth <= logoTextElement.clientWidth) return;

  showLogoIcon.value = false;
  await fitLogoText(false, MIN_LOGO_FONT_SIZE);
}

/**
 * 在当前图标显示状态下，按可用宽度逐步收缩标题字号。
 */
async function fitLogoText(withIcon: boolean, minFontSize: number) {
  await nextTick();

  const loginLogoElement = loginLogoRef.value;
  const logoIconElement = logoIconRef.value;
  const logoTextElement = logoTextRef.value;
  if (!loginLogoElement || !logoTextElement) return;

  const iconWidth = withIcon && logoIconElement ? logoIconElement.getBoundingClientRect().width : 0;
  const availableWidth = Math.max(loginLogoElement.clientWidth - iconWidth - (withIcon ? LOGO_TEXT_GAP : 0), 0);

  let nextFontSize = MAX_LOGO_FONT_SIZE;
  logoFontSize.value = nextFontSize;
  await nextTick();

  while (nextFontSize > minFontSize && logoTextElement.scrollWidth > availableWidth) {
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

watch([projectName, logoUrl], () => {
  updateLogoLayout();
});
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
