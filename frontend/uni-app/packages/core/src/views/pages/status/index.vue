<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import BootstrapStatus from '../../../components/BootstrapStatus.vue'
import { initializeAppNavigation, launchAppStatus, navigateAppRoute } from '../../../navigation'
import type { BootstrapViewKey } from '../../../module'
import { useI18n } from '../../../locales'
import { useSettingStore } from '../../../stores'

const state = ref<BootstrapViewKey>('BOOTSTRAP_LOADING')
const detail = ref('')
const { t } = useI18n()
const stateTitle = computed(() =>
  state.value === 'BOOTSTRAP_LOADING' ? t('core.status.loading') : t(`core.status.${state.value}`),
)

onLoad((options) => {
  state.value = (options?.state as BootstrapViewKey | undefined) ?? 'BOOTSTRAP_LOADING'
  detail.value = options?.detail ? decodeURIComponent(options.detail) : ''
  if (options?.bootstrap !== '1') return
  void useSettingStore()
    .loadData()
    .then(() => initializeAppNavigation())
    .then(() => {
      navigateAppRoute(options?.route ? decodeURIComponent(options.route) : 'app/home', {
        replace: true,
      })
    })
    .catch((error) => {
      launchAppStatus('CONFIG_ERROR', resolveErrorMessage(error))
    })
})

/** 将请求错误转换为可展示的文本，避免直接显示 Object。 */
function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message || '')
  }
  return ''
}
</script>

<template>
  <BootstrapStatus :title="stateTitle" :detail="detail" />
</template>
