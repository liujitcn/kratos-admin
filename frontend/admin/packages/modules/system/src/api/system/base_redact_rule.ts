import service from "@liujitcn/kratos-admin-core/request";
import type { BaseRedactRuleForm, BaseRedactRuleService, CreateBaseRedactRuleRequest, DeleteBaseRedactRuleRequest, GetBaseRedactRuleRequest, OptionBaseRedactRuleRequest, PageBaseRedactRuleRequest, PageBaseRedactRuleResponse, SetBaseRedactRuleStatusRequest, UpdateBaseRedactRuleRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_rule";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_REDACT_RULE_URL = "/v1/admin/base/redact-rule";

/** Admin脱敏规则模板服务。 */
export class BaseRedactRuleServiceImpl implements BaseRedactRuleService {
  /** 查询脱敏规则选项。 */
  OptionBaseRedactRule(request: OptionBaseRedactRuleRequest = { keyword: "" }): Promise<SelectOptionResponse> { return service<OptionBaseRedactRuleRequest, SelectOptionResponse>({ url: `${BASE_REDACT_RULE_URL}/option`, method: "get", params: request }); }
  /** 查询脱敏规则分页列表。 */
  PageBaseRedactRule(request: PageBaseRedactRuleRequest): Promise<PageBaseRedactRuleResponse> { return service<PageBaseRedactRuleRequest, PageBaseRedactRuleResponse>({ url: BASE_REDACT_RULE_URL, method: "get", params: request }); }
  /** 查询脱敏规则详情。 */
  GetBaseRedactRule(request: GetBaseRedactRuleRequest): Promise<BaseRedactRuleForm> { return service<GetBaseRedactRuleRequest, BaseRedactRuleForm>({ url: `${BASE_REDACT_RULE_URL}/${request.id}`, method: "get" }); }
  /** 创建脱敏规则。 */
  CreateBaseRedactRule(request: CreateBaseRedactRuleRequest): Promise<Empty> { return service<BaseRedactRuleForm | undefined, Empty>({ url: BASE_REDACT_RULE_URL, method: "post", data: request.base_redact_rule }); }
  /** 更新脱敏规则。 */
  UpdateBaseRedactRule(request: UpdateBaseRedactRuleRequest): Promise<Empty> { return service<BaseRedactRuleForm | undefined, Empty>({ url: `${BASE_REDACT_RULE_URL}/${request.base_redact_rule?.id ?? ""}`, method: "put", data: request.base_redact_rule }); }
  /** 删除脱敏规则。 */
  DeleteBaseRedactRule(request: DeleteBaseRedactRuleRequest): Promise<Empty> { return service<DeleteBaseRedactRuleRequest, Empty>({ url: `${BASE_REDACT_RULE_URL}/${request.id}`, method: "delete" }); }
  /** 设置脱敏规则状态。 */
  SetBaseRedactRuleStatus(request: SetBaseRedactRuleStatusRequest): Promise<Empty> { return service<SetBaseRedactRuleStatusRequest, Empty>({ url: `${BASE_REDACT_RULE_URL}/${request.id}/status`, method: "put", data: request }); }
}

/** defBaseRedactRuleService 脱敏规则模板服务实例。 */
export const defBaseRedactRuleService = new BaseRedactRuleServiceImpl();
