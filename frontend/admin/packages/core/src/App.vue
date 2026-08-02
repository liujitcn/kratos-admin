<template>
  <el-config-provider :locale="elementLocale" :size="assemblySize" :button="buttonConfig">
    <router-view></router-view>
  </el-config-provider>
</template>

<script setup lang="ts">
import { reactive, computed } from "vue";
import { useTheme } from "@/hooks/useTheme";
import { ElConfigProvider } from "element-plus";
import { useGlobalStore } from "@/stores/modules/global";
import { useLocaleStore } from "@/locales";
import en from "element-plus/es/locale/lang/en";
import ja from "element-plus/es/locale/lang/ja";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import zhTw from "element-plus/es/locale/lang/zh-tw";
import ko from "element-plus/es/locale/lang/ko";
import fr from "element-plus/es/locale/lang/fr";
import es from "element-plus/es/locale/lang/es";

const globalStore = useGlobalStore();
const { locale } = useLocaleStore();
const elementLocale = computed(() => ({ "zh-CN": zhCn, "zh-TW": zhTw, "en-US": en, "ja-JP": ja, "ko-KR": ko, "fr-FR": fr, "es-ES": es })[locale.value]);

// init theme
const { initTheme } = useTheme();
initTheme();

// element assemblySize
const assemblySize = computed(() => globalStore.assemblySize);

// element button config
const buttonConfig = reactive({ autoInsertSpace: false });
</script>
