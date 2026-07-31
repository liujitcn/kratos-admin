import { coreModule } from '@liujitcn/kratos-taro-app-core'
import { systemModule } from '@liujitcn/kratos-taro-app-system'

/** 宿主唯一模块清单，顺序决定静态视图覆盖优先级。 */
export const moduleManifest = [coreModule, systemModule]
