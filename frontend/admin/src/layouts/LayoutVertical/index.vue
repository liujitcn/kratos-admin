<!-- 纵向布局 -->
<template>
  <el-container class="layout">
    <el-aside>
      <div class="aside-box" :style="{ width: isCollapse ? '65px' : '240px' }">
        <div ref="logoWrapperRef" :class="['logo', { 'logo--text-only': !showLogoIcon && !isCollapse }]">
          <img v-show="isCollapse || showLogoIcon" ref="logoIconRef" class="logo-img" :src="logoUrl" alt="logo" />
          <span
            v-show="!isCollapse"
            ref="logoTextRef"
            :class="['logo-text', { 'logo-text--with-icon': showLogoIcon, 'logo-text--full': !showLogoIcon && !isCollapse }]"
          >
            {{ title }}
          </span>
        </div>
        <el-scrollbar>
          <el-menu
            :router="false"
            :default-active="activeMenu"
            :collapse="isCollapse"
            :unique-opened="accordion"
            :collapse-transition="false"
          >
            <SubMenu :menu-list="menuList" />
          </el-menu>
        </el-scrollbar>
      </div>
    </el-aside>
    <el-container>
      <el-header>
        <ToolBarLeft />
        <ToolBarRight />
      </el-header>
      <Main />
    </el-container>
  </el-container>
</template>

<script setup lang="ts" name="layoutVertical">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/modules/auth";
import { useConfigStore } from "@/stores/modules/config";
import { useGlobalStore } from "@/stores/modules/global";
import Main from "@/layouts/components/Main/index.vue";
import ToolBarLeft from "@/layouts/components/Header/ToolBarLeft.vue";
import ToolBarRight from "@/layouts/components/Header/ToolBarRight.vue";
import SubMenu from "@/layouts/components/Menu/SubMenu.vue";

const route = useRoute();
const authStore = useAuthStore();
const configStore = useConfigStore();
const globalStore = useGlobalStore();
const accordion = computed(() => globalStore.accordion);
const isCollapse = computed(() => globalStore.isCollapse);
const menuList = computed(() => authStore.showMenuListGet);
const activeMenu = computed(() => route.path as string);
const title = computed(() => configStore.display.sysName || import.meta.env.VITE_GLOB_APP_TITLE);
const logoUrl = computed(() => configStore.display.adminLogo);
const logoWrapperRef = ref<HTMLElement>();
const logoIconRef = ref<HTMLImageElement>();
const logoTextRef = ref<HTMLElement>();
const showLogoIcon = ref(true);

let logoResizeObserver: ResizeObserver | null = null;

/**
 * 同步左侧标题区布局。
 * 优先展示 Logo 和文字，若长标题在单行下放不下，则自动隐藏 Logo，仅保留文字。
 */
async function updateLogoLayout() {
  await nextTick();

  if (isCollapse.value) {
    showLogoIcon.value = true;
    return;
  }

  const wrapperElement = logoWrapperRef.value;
  const iconElement = logoIconRef.value;
  const textElement = logoTextRef.value;
  if (!wrapperElement || !textElement) return;

  showLogoIcon.value = true;
  await nextTick();

  const iconWidth = iconElement?.getBoundingClientRect().width ?? 0;
  const availableWidth = Math.max(wrapperElement.clientWidth - iconWidth - 8 - 24, 0);

  // 标题单行溢出时与登录页保持一致，直接隐藏 Logo，让完整名称优先展示。
  if (textElement.scrollWidth > availableWidth) {
    showLogoIcon.value = false;
  }
}

onMounted(() => {
  updateLogoLayout();
  logoResizeObserver = new ResizeObserver(() => {
    updateLogoLayout();
  });

  if (logoWrapperRef.value) {
    logoResizeObserver.observe(logoWrapperRef.value);
  }
});

onBeforeUnmount(() => {
  logoResizeObserver?.disconnect();
  logoResizeObserver = null;
});

watch([title, logoUrl, isCollapse], () => {
  updateLogoLayout();
});
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
