<template>
  <div class="footer flx-center">
    <a :href="icpLink" target="_blank" rel="noopener noreferrer">{{ footerText }}</a>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useConfigStore } from "@/stores/modules/config";

const configStore = useConfigStore();
const footerText = computed(() => {
  const footerTextList = [configStore.display.copyright, configStore.display.icp].filter(Boolean);
  return footerTextList.join(" ");
});
const icpLink = computed(() => (configStore.display.icp ? "https://beian.miit.gov.cn/" : "/"));
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
