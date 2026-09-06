import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseRedactOutputPolicyForm,
  BaseRedactOutputPolicyService,
  CreateBaseRedactOutputPolicyRequest,
  DeleteBaseRedactOutputPolicyRequest,
  GetBaseRedactOutputPolicyRequest,
  PageBaseRedactOutputPolicyRequest,
  PageBaseRedactOutputPolicyResponse,
  SetBaseRedactOutputPolicyStatusRequest,
  UpdateBaseRedactOutputPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_output_policy";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_REDACT_OUTPUT_POLICY_URL = "/v1/admin/base/redact-output-policy";

/** 管理端出库脱敏策略服务。 */
export class BaseRedactOutputPolicyServiceImpl implements BaseRedactOutputPolicyService {
  /** 查询出库脱敏策略分页列表。 */
  PageBaseRedactOutputPolicy(request: PageBaseRedactOutputPolicyRequest): Promise<PageBaseRedactOutputPolicyResponse> {
    return service<PageBaseRedactOutputPolicyRequest, PageBaseRedactOutputPolicyResponse>({ url: BASE_REDACT_OUTPUT_POLICY_URL, method: "get", params: request });
  }

  /** 查询出库脱敏策略详情。 */
  GetBaseRedactOutputPolicy(request: GetBaseRedactOutputPolicyRequest): Promise<BaseRedactOutputPolicyForm> {
    return service<GetBaseRedactOutputPolicyRequest, BaseRedactOutputPolicyForm>({ url: `${BASE_REDACT_OUTPUT_POLICY_URL}/${request.id}`, method: "get" });
  }

  /** 批量创建出库脱敏策略。 */
  CreateBaseRedactOutputPolicy(request: CreateBaseRedactOutputPolicyRequest): Promise<Empty> {
    return service<CreateBaseRedactOutputPolicyRequest, Empty>({ url: BASE_REDACT_OUTPUT_POLICY_URL, method: "post", data: request });
  }

  /** 批量更新出库脱敏策略。 */
  UpdateBaseRedactOutputPolicy(request: UpdateBaseRedactOutputPolicyRequest): Promise<Empty> {
    return service<UpdateBaseRedactOutputPolicyRequest, Empty>({ url: BASE_REDACT_OUTPUT_POLICY_URL, method: "put", data: request });
  }

  /** 删除出库脱敏策略。 */
  DeleteBaseRedactOutputPolicy(request: DeleteBaseRedactOutputPolicyRequest): Promise<Empty> {
    return service<DeleteBaseRedactOutputPolicyRequest, Empty>({ url: `${BASE_REDACT_OUTPUT_POLICY_URL}/${request.id}`, method: "delete" });
  }

  /** 设置出库脱敏策略状态。 */
  SetBaseRedactOutputPolicyStatus(request: SetBaseRedactOutputPolicyStatusRequest): Promise<Empty> {
    return service<SetBaseRedactOutputPolicyStatusRequest, Empty>({ url: `${BASE_REDACT_OUTPUT_POLICY_URL}/${request.id}/status`, method: "put", data: request });
  }
}

/** defBaseRedactOutputPolicyService 出库脱敏策略服务实例。 */
export const defBaseRedactOutputPolicyService = new BaseRedactOutputPolicyServiceImpl();
