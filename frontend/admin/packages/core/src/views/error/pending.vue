<script setup lang="ts" name="Pending">
import { useLocaleStore } from "@/locales";
const { t } = useLocaleStore();
</script>

<template>
  <div class="route-pending page-card">
    <el-empty :description="t('core.error.pending.detail')" />
  </div>
</template>

<style scoped lang="scss">
.route-pending {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 360px;
}
</style>
