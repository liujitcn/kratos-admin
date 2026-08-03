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
import { ELEMENT_LOCALES } from "@/locales/generated";
import { useLocaleStore } from "@/locales";

const globalStore = useGlobalStore();
const { locale } = useLocaleStore();
const elementLocale = computed(() => ELEMENT_LOCALES[locale.value]);

// init theme
const { initTheme } = useTheme();
initTheme();

// element assemblySize
const assemblySize = computed(() => globalStore.assemblySize);

// element button config
const buttonConfig = reactive({ autoInsertSpace: false });
</script>
