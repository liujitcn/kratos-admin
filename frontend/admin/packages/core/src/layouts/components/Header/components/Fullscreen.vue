<template>
  <div class="fullscreen">
    <el-tooltip effect="dark" :content="fullscreenTooltip" placement="bottom" :show-after="200">
      <i :class="['iconfont', isFullscreen ? 'icon-suoxiao' : 'icon-fangda']" class="toolBar-icon" @click="handleFullScreen"></i>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import screenfull from "screenfull";
import { useLocaleStore } from "@/locales";

const isFullscreen = ref(screenfull.isFullscreen);
const { t } = useLocaleStore();
const fullscreenTooltip = computed(() => (isFullscreen.value ? t("core.header.fullscreen_exit") : t("core.header.fullscreen")));

onMounted(() => {
  screenfull.on("change", () => {
    if (screenfull.isFullscreen) isFullscreen.value = true;
    else isFullscreen.value = false;
  });
});

const handleFullScreen = () => {
  if (!screenfull.isEnabled) ElMessage.warning(t("core.header.fullscreen_unsupported"));
  screenfull.toggle();
};
</script>
