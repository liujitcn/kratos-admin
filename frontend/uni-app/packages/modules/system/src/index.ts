import { registerUserStoreExtension } from '@liujitcn/kratos-uni-app-core/stores'
import {
  pauseNotificationPolling,
  resumeNotificationPolling,
  startNotificationPolling,
  stopNotificationPolling,
} from './notification'

import { systemModule } from './module'

registerUserStoreExtension({
  onLogin: startNotificationPolling,
  onLogout: stopNotificationPolling,
  onSilentLogout: stopNotificationPolling,
})

export { pauseNotificationPolling, resumeNotificationPolling }

export default systemModule
