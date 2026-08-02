<template>
  <div class="theme-setting">
    <el-tooltip effect="dark" :content="t('core.header.theme')" placement="bottom" :show-after="200">
      <i :class="'iconfont icon-zhuti'" class="toolBar-icon" @click="openDrawer"></i>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import mittBus from "@/utils/mittBus";
import { useLocaleStore } from "@/locales";
const { t } = useLocaleStore();
const openDrawer = () => {
  mittBus.emit("openThemeDrawer");
};
</script>
