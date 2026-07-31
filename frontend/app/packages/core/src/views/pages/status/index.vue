<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import BootstrapStatus from '../../../components/BootstrapStatus.vue'
import { initializeAppNavigation, launchAppStatus, navigateAppRoute } from '../../../navigation'
import type { BootstrapViewKey } from '../../../module'

const state = ref<BootstrapViewKey>('BOOTSTRAP_LOADING')
const detail = ref('')
const stateTitles: Record<BootstrapViewKey, string> = {
  BOOTSTRAP_LOADING: '正在加载',
  FORBIDDEN: '暂无访问权限',
  NOT_FOUND: '页面不存在',
  OFFLINE: '网络不可用',
  CONFIG_ERROR: '导航配置无效',
  PAGE_UNAVAILABLE: '页面暂不可用',
}

onLoad(async (options) => {
  state.value = (options?.state as BootstrapViewKey | undefined) ?? 'BOOTSTRAP_LOADING'
  detail.value = options?.detail ? decodeURIComponent(options.detail) : ''
  if (options?.bootstrap !== '1') return
  try {
    await initializeAppNavigation()
    navigateAppRoute(options?.route ? decodeURIComponent(options.route) : 'app/home', {
      replace: true,
    })
  } catch (error) {
    launchAppStatus('CONFIG_ERROR', error instanceof Error ? error.message : String(error))
  }
})
</script>

<template>
  <BootstrapStatus :title="stateTitles[state]" :detail="detail" />
</template>
