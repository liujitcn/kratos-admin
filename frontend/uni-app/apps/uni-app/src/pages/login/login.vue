<script setup lang="ts">
import KratosPage from "@liujitcn/kratos-uni-app-core/views/pages/login/login.vue"
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.vue'

defineOptions({ inheritAttrs: false })
</script>

<template>
  <KratosPage />
  <KratosTabBar route="pages/login/login" />
</template>
