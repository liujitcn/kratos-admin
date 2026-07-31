<script setup lang="ts">
import KratosPage from '@liujitcn/kratos-uni-app-core/views/pages/index/index.vue'
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.vue'

defineOptions({ inheritAttrs: false })
</script>

<template>
  <KratosPage />
  <KratosTabBar route="pages/index/index" />
</template>
