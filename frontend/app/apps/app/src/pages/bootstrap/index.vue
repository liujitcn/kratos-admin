<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { resolveStaticView } from '@liujitcn/kratos-app-core'

onLoad((options) => {
  const route = resolveStaticView('BOOTSTRAP_LOADING') ?? 'pages/status/index'
  const target = options?.route ? decodeURIComponent(options.route) : 'app/home'
  uni.reLaunch({
    url: `/${route}?state=BOOTSTRAP_LOADING&bootstrap=1&route=${encodeURIComponent(target)}`,
  })
})
</script>

<template>
  <view />
</template>
