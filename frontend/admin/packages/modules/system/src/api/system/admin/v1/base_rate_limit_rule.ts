import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseRateLimitRuleForm,
  BaseRateLimitRuleService,
  CreateBaseRateLimitRuleRequest,
  DeleteBaseRateLimitRuleRequest,
  GetBaseRateLimitRuleRequest,
  OptionBaseRateLimitRuleResponse,
  OptionBaseRateLimitRuleRequest,
  PageBaseRateLimitRuleRequest,
  PageBaseRateLimitRuleResponse,
  SetBaseRateLimitRuleStatusRequest,
  UpdateBaseRateLimitRuleRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_RATE_LIMIT_RULE_URL = "/v1/admin/base/rate-limit-rule";

/** 管理端限流规则模板服务。 */
export class BaseRateLimitRuleServiceImpl implements BaseRateLimitRuleService {
  /** 查询限流规则选项。 */
  OptionBaseRateLimitRule(request: OptionBaseRateLimitRuleRequest = { keyword: "" }): Promise<OptionBaseRateLimitRuleResponse> {
    return service<OptionBaseRateLimitRuleRequest, OptionBaseRateLimitRuleResponse>({ url: `${BASE_RATE_LIMIT_RULE_URL}/option`, method: "get", params: request });
  }

  /** 查询限流规则分页列表。 */
  PageBaseRateLimitRule(request: PageBaseRateLimitRuleRequest): Promise<PageBaseRateLimitRuleResponse> {
    return service<PageBaseRateLimitRuleRequest, PageBaseRateLimitRuleResponse>({ url: BASE_RATE_LIMIT_RULE_URL, method: "get", params: request });
  }

  /** 查询限流规则详情。 */
  GetBaseRateLimitRule(request: GetBaseRateLimitRuleRequest): Promise<BaseRateLimitRuleForm> {
    return service<GetBaseRateLimitRuleRequest, BaseRateLimitRuleForm>({ url: `${BASE_RATE_LIMIT_RULE_URL}/${request.id}`, method: "get" });
  }

  /** 创建限流规则。 */
  CreateBaseRateLimitRule(request: CreateBaseRateLimitRuleRequest): Promise<Empty> {
    return service<BaseRateLimitRuleForm | undefined, Empty>({ url: BASE_RATE_LIMIT_RULE_URL, method: "post", data: request.base_rate_limit_rule });
  }

  /** 更新限流规则。 */
  UpdateBaseRateLimitRule(request: UpdateBaseRateLimitRuleRequest): Promise<Empty> {
    return service<BaseRateLimitRuleForm | undefined, Empty>({ url: `${BASE_RATE_LIMIT_RULE_URL}/${request.base_rate_limit_rule?.id ?? ""}`, method: "put", data: request.base_rate_limit_rule });
  }

  /** 删除限流规则。 */
  DeleteBaseRateLimitRule(request: DeleteBaseRateLimitRuleRequest): Promise<Empty> {
    return service<DeleteBaseRateLimitRuleRequest, Empty>({ url: `${BASE_RATE_LIMIT_RULE_URL}/${request.id}`, method: "delete" });
  }

  /** 设置限流规则状态。 */
  SetBaseRateLimitRuleStatus(request: SetBaseRateLimitRuleStatusRequest): Promise<Empty> {
    return service<SetBaseRateLimitRuleStatusRequest, Empty>({ url: `${BASE_RATE_LIMIT_RULE_URL}/${request.id}/status`, method: "put", data: request });
  }
}

/** 限流规则模板服务实例。 */
export const defBaseRateLimitRuleService = new BaseRateLimitRuleServiceImpl();
