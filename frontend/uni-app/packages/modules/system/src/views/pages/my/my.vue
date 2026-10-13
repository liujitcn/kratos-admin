<script setup lang="ts">
import KratosMyPage from '@liujitcn/kratos-uni-app-system/components/KratosMyPage.vue'
import MyExtensions from './components/MyExtensions.vue'
</script>

<template>
  <KratosMyPage>
    <template #extensions>
      <MyExtensions />
    </template>
  </KratosMyPage>
</template>
