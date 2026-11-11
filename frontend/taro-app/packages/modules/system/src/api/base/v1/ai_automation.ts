import { http } from '@liujitcn/kratos-taro-app-core/utils/http'
import type { AiAutomationRun, GetAiAutomationRunRequest } from '../../../rpc/base/v1/ai_automation'

/** 查询当前用户可访问的执行记录，供站内提醒定位结果。 */
export function getAiAutomationRun(request: GetAiAutomationRunRequest): Promise<AiAutomationRun> {
  return http({ url: `/v1/base/ai/automation-run/${request.id}`, method: 'GET', authMode: 'required', data: request })
}
