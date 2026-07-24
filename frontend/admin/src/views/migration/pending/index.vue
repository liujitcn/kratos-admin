<script setup lang="ts">
// 动态菜单组件暂未落地时展示统一提示，避免路由被静默丢弃。
</script>

<template>
  <div class="migration-pending page-card">
    <el-empty description="该页面暂不可用，请联系管理员检查菜单组件配置" />
  </div>
</template>

<style scoped lang="scss">
.migration-pending {
  display: flex;
  min-height: 360px;
  align-items: center;
  justify-content: center;
}
</style>
