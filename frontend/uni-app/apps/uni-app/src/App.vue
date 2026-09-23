<script setup lang="ts">
import { onHide, onLaunch, onShow } from '@dcloudio/uni-app'
import {
  pauseNotificationPolling,
  resumeNotificationPolling,
  startNotificationPolling,
} from '@liujitcn/kratos-uni-app-system'

onLaunch(() => startNotificationPolling())
onShow(() => resumeNotificationPolling())
onHide(() => pauseNotificationPolling())
</script>

<style lang="scss">
@use '../../../packages/core/src/styles/base.scss';
</style>
