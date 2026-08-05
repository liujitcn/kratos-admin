<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, ref } from 'vue'
import BootstrapStatus from '../../../components/BootstrapStatus.vue'
import { initializeAppNavigation, launchAppStatus, navigateAppRoute } from '../../../navigation'
import type { BootstrapViewKey } from '../../../module'
import { useI18n } from '../../../locales'

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
  void initializeAppNavigation()
    .then(() => {
      navigateAppRoute(options?.route ? decodeURIComponent(options.route) : 'app/home', {
        replace: true,
      })
    })
    .catch((error) => {
      launchAppStatus('CONFIG_ERROR', error instanceof Error ? error.message : String(error))
    })
})
</script>

<template>
  <BootstrapStatus :title="stateTitle" :detail="detail" />
</template>
