import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiAutomationService,
  AiAutomation,
  AiAutomationRun,
  ListAiAutomationRequest,
  ListAiAutomationResponse,
  GetAiAutomationRequest,
  ListAiAutomationRunRequest,
  ListAiAutomationRunResponse,
  GetAiAutomationRunRequest,
} from '../../../rpc/base/v1/ai_automation'

/** 本人定时任务与执行记录只读服务。 */
class AiAutomationReadService
  implements
    Pick<
      AiAutomationService,
      'ListAiAutomation' | 'GetAiAutomation' | 'ListAiAutomationRun' | 'GetAiAutomationRun'
    >
{
  /** 查询当前用户的任务。 */
  ListAiAutomation(request: ListAiAutomationRequest): Promise<ListAiAutomationResponse> {
    return http({
      url: '/v1/base/ai/automation',
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询当前用户的任务详情。 */
  GetAiAutomation(request: GetAiAutomationRequest): Promise<AiAutomation> {
    return http({
      url: `/v1/base/ai/automation/${request.id}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 只查询指定任务的执行记录。 */
  ListAiAutomationRun(request: ListAiAutomationRunRequest): Promise<ListAiAutomationRunResponse> {
    return http({
      url: '/v1/base/ai/automation-run',
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询可访问执行记录的冻结配置。 */
  GetAiAutomationRun(request: GetAiAutomationRunRequest): Promise<AiAutomationRun> {
    return getAiAutomationRun(request)
  }
}

/** 查询当前用户可访问的执行记录，供站内提醒定位结果。 */
export function getAiAutomationRun(request: GetAiAutomationRunRequest): Promise<AiAutomationRun> {
  return http({
    url: `/v1/base/ai/automation-run/${request.id}`,
    method: 'GET',
    authMode: 'required',
    data: request,
  })
}

/** 默认定时任务只读服务。 */
export const defAiAutomationService = new AiAutomationReadService()
