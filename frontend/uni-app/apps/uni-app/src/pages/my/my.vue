<script setup lang="ts">
import KratosPage from '@liujitcn/kratos-uni-app-system/views/pages/my/my.vue'
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.vue'

defineOptions({ inheritAttrs: false })
</script>

<template>
  <KratosPage />
  <KratosTabBar route="pages/my/my" />
</template>
