import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
import type {
  AiAutomationService,
  AiAutomation,
  AiAutomationRun,
  ListAiAutomationRequest,
  ListAiAutomationResponse,
  GetAiAutomationRequest,
  RunAiAutomationRequest,
  ListAiAutomationRunRequest,
  ListAiAutomationRunResponse,
  GetAiAutomationRunRequest,
  StopAiAutomationRunRequest,
  ContinueAiAutomationRunRequest,
} from '../../../rpc/base/v1/ai_automation'
import type { CreateAiSessionResponse } from '../../../rpc/base/v1/ai_session'

/** 移动端查看与使用已有任务，不提供计划配置操作。 */
class AiAutomationUseService
  implements
    Pick<
      AiAutomationService,
      | 'ListAiAutomation'
      | 'GetAiAutomation'
      | 'RunAiAutomation'
      | 'ListAiAutomationRun'
      | 'GetAiAutomationRun'
      | 'StopAiAutomationRun'
      | 'ContinueAiAutomationRun'
    >
{
  /** 查询本人任务。 */
  ListAiAutomation(request: ListAiAutomationRequest): Promise<ListAiAutomationResponse> {
    return http({
      url: '/v1/base/ai/automation',
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询任务详情。 */
  GetAiAutomation(request: GetAiAutomationRequest): Promise<AiAutomation> {
    return http({
      url: `/v1/base/ai/automation/${request.id}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 运行一次已有任务。 */
  RunAiAutomation(request: RunAiAutomationRequest): Promise<AiAutomationRun> {
    return http({
      url: `/v1/base/ai/automation/${request.id}/run`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询执行记录。 */
  ListAiAutomationRun(request: ListAiAutomationRunRequest): Promise<ListAiAutomationRunResponse> {
    return http({
      url: '/v1/base/ai/automation-run',
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询执行详情。 */
  GetAiAutomationRun(request: GetAiAutomationRunRequest): Promise<AiAutomationRun> {
    return http({
      url: `/v1/base/ai/automation-run/${request.id}`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 请求停止本次，不改变后续计划。 */
  StopAiAutomationRun(request: StopAiAutomationRunRequest): Promise<AiAutomationRun> {
    return http({
      url: `/v1/base/ai/automation-run/${request.id}/stop`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
  /** 从冻结结果创建普通讨论会话。 */
  ContinueAiAutomationRun(
    request: ContinueAiAutomationRunRequest,
  ): Promise<CreateAiSessionResponse> {
    return http({
      url: `/v1/base/ai/automation-run/${request.id}/discussion`,
      method: 'POST',
      authMode: 'required',
      data: request,
    })
  }
}

export const defAiAutomationService = new AiAutomationUseService()
