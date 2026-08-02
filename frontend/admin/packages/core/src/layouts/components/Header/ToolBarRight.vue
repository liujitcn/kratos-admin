<template>
  <div class="tool-bar-ri">
    <div class="header-icon">
      <component :is="tool.component" v-for="tool in headerTools" :id="tool.name" :key="tool.name" />
      <LocaleSwitch id="localeSwitch" />
      <AssemblySize id="assemblySize" />
      <SearchMenu id="searchMenu" />
      <ThemeSetting id="themeSetting" />
      <Fullscreen id="fullscreen" />
    </div>
    <Avatar />
  </div>
</template>

<script setup lang="ts">
import { getAdminHeaderTools } from "@/modules";
import AssemblySize from "./components/AssemblySize.vue";
import SearchMenu from "./components/SearchMenu.vue";
import ThemeSetting from "./components/ThemeSetting.vue";
import Fullscreen from "./components/Fullscreen.vue";
import Avatar from "./components/Avatar.vue";
import LocaleSwitch from "@/components/LocaleSwitch/index.vue";

const headerTools = getAdminHeaderTools();
</script>

<style scoped lang="scss">
.tool-bar-ri {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  .header-icon {
    display: flex;
    align-items: center;
    & > * {
      margin-left: 21px;
      color: var(--el-header-text-color);
    }
  }

  // 头像入口与工具组之间保留呼吸感，避免顶部右侧元素粘连。
  :deep(.el-dropdown) {
    margin-left: 18px;
  }
}
</style>
