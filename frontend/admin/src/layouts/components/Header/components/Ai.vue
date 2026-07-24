<template>
  <el-tooltip v-if="showAi" effect="dark" content="AI助手" placement="bottom" :show-after="200">
    <button class="ai" type="button" aria-label="打开AI助手" @click="openAi">
      <el-icon><ChatDotRound /></el-icon>
    </button>
  </el-tooltip>
</template>

<script setup lang="ts">
import { ChatDotRound } from "@element-plus/icons-vue";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/modules/auth";
import { navigateTo } from "@/utils/router";

const router = useRouter();
const authStore = useAuthStore();
const showAi = computed(() => {
  return authStore.flatMenuListGet.some(item => item.path === "/ai/chat" || item.name === "AiChat");
});

/** 打开隐藏的 AI 助手页面。 */
function openAi() {
  navigateTo(router, "/ai/chat");
}
</script>

<style scoped lang="scss">
.ai {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  color: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
  transition:
    color 0.16s ease,
    transform 0.16s ease;
  .el-icon {
    font-size: 24px;
  }
  &:hover {
    color: var(--el-color-primary);
    transform: translateY(-1px);
  }
  &:focus-visible {
    outline: 2px solid var(--el-color-primary-light-5);
    outline-offset: 4px;
    border-radius: 4px;
  }
}
</style>
