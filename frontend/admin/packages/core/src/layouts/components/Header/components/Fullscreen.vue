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

const isFullscreen = ref(screenfull.isFullscreen);
const fullscreenTooltip = computed(() => (isFullscreen.value ? "退出全屏" : "全屏"));

onMounted(() => {
  screenfull.on("change", () => {
    if (screenfull.isFullscreen) isFullscreen.value = true;
    else isFullscreen.value = false;
  });
});

const handleFullScreen = () => {
  if (!screenfull.isEnabled) ElMessage.warning("当前您的浏览器不支持全屏 ❌");
  screenfull.toggle();
};
</script>
