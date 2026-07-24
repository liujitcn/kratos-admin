<template>
  <div class="theme-setting">
    <el-tooltip effect="dark" content="主题设置" placement="bottom" :show-after="200">
      <i :class="'iconfont icon-zhuti'" class="toolBar-icon" @click="openDrawer"></i>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import mittBus from "@/utils/mittBus";
const openDrawer = () => {
  mittBus.emit("openThemeDrawer");
};
</script>
