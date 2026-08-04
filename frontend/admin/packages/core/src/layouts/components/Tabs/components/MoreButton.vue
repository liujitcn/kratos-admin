<template>
  <el-dropdown trigger="click" :teleported="false">
    <div class="more-button">
      <i :class="'iconfont icon-xiala'"></i>
    </div>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item @click="refresh">
          <el-icon><Refresh /></el-icon>{{ t("core.tabs.refresh") }}
        </el-dropdown-item>
        <el-dropdown-item @click="maximize">
          <el-icon><FullScreen /></el-icon>{{ t("core.tabs.maximize") }}
        </el-dropdown-item>
        <el-dropdown-item divided @click="closeCurrentTab">
          <el-icon><Remove /></el-icon>{{ t("core.tabs.close_current") }}
        </el-dropdown-item>
        <el-dropdown-item @click="tabStore.closeTabsOnSide(currentTabPath, 'left')">
          <el-icon><DArrowLeft /></el-icon>{{ t("core.tabs.close_left") }}
        </el-dropdown-item>
        <el-dropdown-item @click="tabStore.closeTabsOnSide(currentTabPath, 'right')">
          <el-icon><DArrowRight /></el-icon>{{ t("core.tabs.close_right") }}
        </el-dropdown-item>
        <el-dropdown-item divided @click="tabStore.closeMultipleTab(currentTabPath)">
          <el-icon><CircleClose /></el-icon>{{ t("core.tabs.close_other") }}
        </el-dropdown-item>
        <el-dropdown-item @click="closeAllTab">
          <el-icon><FolderDelete /></el-icon>{{ t("core.tabs.close_all") }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed, inject, nextTick } from "vue";
import { HOME_URL } from "@/config";
import { getAdminTabPath } from "@/modules";
import { useTabsStore } from "@/stores/modules/tabs";
import { useGlobalStore } from "@/stores/modules/global";
import { useKeepAliveStore } from "@/stores/modules/keepAlive";
import { useRoute, useRouter } from "vue-router";
import { useLocaleStore } from "@/locales";

const route = useRoute();
const router = useRouter();
const tabStore = useTabsStore();
const globalStore = useGlobalStore();
const keepAliveStore = useKeepAliveStore();
const { t } = useLocaleStore();
const currentTabPath = computed(() => getAdminTabPath(route));

// refresh current page
const refreshCurrentPage: Function = inject("refresh") as Function;
const refresh = () => {
  setTimeout(() => {
    route.meta.keepAlive && keepAliveStore.removeKeepAliveName(currentTabPath.value);
    refreshCurrentPage(false);
    nextTick(() => {
      route.meta.keepAlive && keepAliveStore.addKeepAliveName(currentTabPath.value);
      refreshCurrentPage(true);
    });
  }, 0);
};

// maximize current page
const maximize = () => {
  globalStore.setGlobalState("maximize", true);
};

// Close Current
const closeCurrentTab = () => {
  if (route.meta.affix) return;
  tabStore.removeTabs(currentTabPath.value);
};

// Close All
const closeAllTab = () => {
  tabStore.closeMultipleTab();
  router.push(HOME_URL);
};
</script>

<style scoped lang="scss">
@use "../index.scss" as *;
</style>
