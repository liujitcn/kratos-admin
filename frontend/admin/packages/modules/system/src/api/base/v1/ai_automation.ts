import type { CreateAiSessionResponse } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import service from "@liujitcn/kratos-admin-core/request";
import type {
  AiAutomation,
  AiAutomationRun,
  AiAutomationService,
  ContinueAiAutomationRunRequest,
  CreateAiAutomationRequest,
  GetAiAutomationRequest,
  GetAiAutomationRunRequest,
  ListAiAutomationRequest,
  ListAiAutomationResponse,
  ListAiAutomationRunRequest,
  ListAiAutomationRunResponse,
  PreviewAiAutomationScheduleRequest,
  PreviewAiAutomationScheduleResponse,
  RunAiAutomationRequest,
  SetAiAutomationStatusRequest,
  StopAiAutomationRunRequest,
  UpdateAiAutomationRequest
} from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_automation";

/** AI 助手自动化服务。 */
export class AiAutomationServiceImpl implements AiAutomationService {
  /** ListAiAutomation 调用服务端自动化能力。 */
  ListAiAutomation(request: ListAiAutomationRequest): Promise<ListAiAutomationResponse> {
    return service<ListAiAutomationRequest, ListAiAutomationResponse>({
      url: `/v1/base/ai/automation`,
      method: "get",
      params: request
    });
  }
  /** GetAiAutomation 调用服务端自动化能力。 */
  GetAiAutomation(request: GetAiAutomationRequest): Promise<AiAutomation> {
    return service<GetAiAutomationRequest, AiAutomation>({
      url: `/v1/base/ai/automation/${request.id}`,
      method: "get",
      params: request
    });
  }
  /** PreviewAiAutomationSchedule 调用服务端自动化能力。 */
  PreviewAiAutomationSchedule(request: PreviewAiAutomationScheduleRequest): Promise<PreviewAiAutomationScheduleResponse> {
    return service<PreviewAiAutomationScheduleRequest, PreviewAiAutomationScheduleResponse>({
      url: `/v1/base/ai/automation-preview`,
      method: "post",
      data: request
    });
  }
  /** CreateAiAutomation 调用服务端自动化能力。 */
  CreateAiAutomation(request: CreateAiAutomationRequest): Promise<AiAutomation> {
    return service<CreateAiAutomationRequest, AiAutomation>({ url: `/v1/base/ai/automation`, method: "post", data: request });
  }
  /** UpdateAiAutomation 调用服务端自动化能力。 */
  UpdateAiAutomation(request: UpdateAiAutomationRequest): Promise<AiAutomation> {
    return service<UpdateAiAutomationRequest, AiAutomation>({
      url: `/v1/base/ai/automation/${request.id}`,
      method: "put",
      data: request
    });
  }
  /** SetAiAutomationStatus 调用服务端自动化能力。 */
  SetAiAutomationStatus(request: SetAiAutomationStatusRequest): Promise<AiAutomation> {
    return service<SetAiAutomationStatusRequest, AiAutomation>({
      url: `/v1/base/ai/automation/${request.id}/status`,
      method: "put",
      data: request
    });
  }
  /** RunAiAutomation 调用服务端自动化能力。 */
  RunAiAutomation(request: RunAiAutomationRequest): Promise<AiAutomationRun> {
    return service<RunAiAutomationRequest, AiAutomationRun>({
      url: `/v1/base/ai/automation/${request.id}/run`,
      method: "post",
      data: request
    });
  }
  /** ListAiAutomationRun 调用服务端自动化能力。 */
  ListAiAutomationRun(request: ListAiAutomationRunRequest): Promise<ListAiAutomationRunResponse> {
    return service<ListAiAutomationRunRequest, ListAiAutomationRunResponse>({
      url: `/v1/base/ai/automation-run`,
      method: "get",
      params: request
    });
  }
  /** GetAiAutomationRun 调用服务端自动化能力。 */
  GetAiAutomationRun(request: GetAiAutomationRunRequest): Promise<AiAutomationRun> {
    return service<GetAiAutomationRunRequest, AiAutomationRun>({
      url: `/v1/base/ai/automation-run/${request.id}`,
      method: "get",
      params: request
    });
  }
  /** StopAiAutomationRun 调用服务端自动化能力。 */
  StopAiAutomationRun(request: StopAiAutomationRunRequest): Promise<AiAutomationRun> {
    return service<StopAiAutomationRunRequest, AiAutomationRun>({
      url: `/v1/base/ai/automation-run/${request.id}/stop`,
      method: "post",
      data: request
    });
  }
  /** ContinueAiAutomationRun 调用服务端自动化能力。 */
  ContinueAiAutomationRun(request: ContinueAiAutomationRunRequest): Promise<CreateAiSessionResponse> {
    return service<ContinueAiAutomationRunRequest, CreateAiSessionResponse>({
      url: `/v1/base/ai/automation-run/${request.id}/discussion`,
      method: "post",
      data: request
    });
  }
}
export const defAiAutomationService = new AiAutomationServiceImpl();
