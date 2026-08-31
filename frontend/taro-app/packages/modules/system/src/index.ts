import { registerUserStoreExtension } from '@liujitcn/kratos-taro-app-core/stores'
import { systemModule } from './module'
import {
  pauseNotificationPolling,
  resumeNotificationPolling,
  startNotificationPolling,
  stopNotificationPolling,
} from './notification'

registerUserStoreExtension({
  onLogin: startNotificationPolling,
  onLogout: stopNotificationPolling,
  onSilentLogout: stopNotificationPolling,
})

export { pauseNotificationPolling, resumeNotificationPolling }
export { systemModule }
export default systemModule
