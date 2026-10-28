<script setup lang="ts">
import KratosSettingsPage from '@liujitcn/kratos-uni-app-core/components/KratosSettingsPage.vue'
// #ifdef H5
import ChangePasswordEntry from './components/ChangePasswordEntry.vue'
// #endif
</script>

<template>
  <KratosSettingsPage>
    <!-- #ifdef H5 -->
    <template #security>
      <ChangePasswordEntry />
    </template>
    <!-- #endif -->
  </KratosSettingsPage>
</template>
