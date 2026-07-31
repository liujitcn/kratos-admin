<script setup lang="ts">
import KratosPage from "@liujitcn/kratos-uni-app-core/views/pages/webview/webview.vue"
import KratosTabBar from '@liujitcn/kratos-uni-app-core/components/KratosTabBar.vue'

defineOptions({ inheritAttrs: false })
</script>

<template>
  <KratosPage />
  <KratosTabBar route="pages/webview/webview" />
</template>
