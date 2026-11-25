<template>
  <el-tooltip
    :content="title"
    :disabled="!title"
    placement="top"
    :popper-style="{ maxWidth: 'min(360px, calc(100vw - 24px))', overflowWrap: 'anywhere' }"
  >
    <slot />
  </el-tooltip>
</template>

<script setup lang="ts">
defineProps<{
  /** 数据卡片的完整名称。 */
  title?: string;
}>();
</script>
