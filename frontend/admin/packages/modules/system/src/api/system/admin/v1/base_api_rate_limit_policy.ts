import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseApiRateLimitPolicyForm,
  BaseApiRateLimitPolicyService,
  CreateBaseApiRateLimitPolicyRequest,
  DeleteBaseApiRateLimitPolicyRequest,
  GetBaseApiRateLimitPolicyRequest,
  PageBaseApiRateLimitPolicyRequest,
  PageBaseApiRateLimitPolicyResponse,
  SetBaseApiRateLimitPolicyStatusRequest,
  UpdateBaseApiRateLimitPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_rate_limit_policy";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_API_RATE_LIMIT_POLICY_URL = "/v1/admin/base/api-rate-limit-policy";

/** 管理端接口限流策略服务。 */
export class BaseApiRateLimitPolicyServiceImpl implements BaseApiRateLimitPolicyService {
  /** 查询接口限流策略分页列表。 */
  PageBaseApiRateLimitPolicy(request: PageBaseApiRateLimitPolicyRequest): Promise<PageBaseApiRateLimitPolicyResponse> {
    return service<PageBaseApiRateLimitPolicyRequest, PageBaseApiRateLimitPolicyResponse>({ url: BASE_API_RATE_LIMIT_POLICY_URL, method: "get", params: request });
  }

  /** 查询接口限流策略详情。 */
  GetBaseApiRateLimitPolicy(request: GetBaseApiRateLimitPolicyRequest): Promise<BaseApiRateLimitPolicyForm> {
    return service<GetBaseApiRateLimitPolicyRequest, BaseApiRateLimitPolicyForm>({ url: `${BASE_API_RATE_LIMIT_POLICY_URL}/${request.id}`, method: "get" });
  }

  /** 创建接口限流策略。 */
  CreateBaseApiRateLimitPolicy(request: CreateBaseApiRateLimitPolicyRequest): Promise<Empty> {
    return service<BaseApiRateLimitPolicyForm | undefined, Empty>({ url: BASE_API_RATE_LIMIT_POLICY_URL, method: "post", data: request.base_api_rate_limit_policy });
  }

  /** 更新接口限流策略。 */
  UpdateBaseApiRateLimitPolicy(request: UpdateBaseApiRateLimitPolicyRequest): Promise<Empty> {
    return service<BaseApiRateLimitPolicyForm | undefined, Empty>({ url: `${BASE_API_RATE_LIMIT_POLICY_URL}/${request.base_api_rate_limit_policy?.id ?? ""}`, method: "put", data: request.base_api_rate_limit_policy });
  }

  /** 删除接口限流策略。 */
  DeleteBaseApiRateLimitPolicy(request: DeleteBaseApiRateLimitPolicyRequest): Promise<Empty> {
    return service<DeleteBaseApiRateLimitPolicyRequest, Empty>({ url: `${BASE_API_RATE_LIMIT_POLICY_URL}/${request.id}`, method: "delete" });
  }

  /** 设置接口限流策略状态。 */
  SetBaseApiRateLimitPolicyStatus(request: SetBaseApiRateLimitPolicyStatusRequest): Promise<Empty> {
    return service<SetBaseApiRateLimitPolicyStatusRequest, Empty>({ url: `${BASE_API_RATE_LIMIT_POLICY_URL}/${request.id}/status`, method: "put", data: request });
  }
}

/** 接口限流策略服务实例。 */
export const defBaseApiRateLimitPolicyService = new BaseApiRateLimitPolicyServiceImpl();
