import service from "@liujitcn/kratos-admin-core/request";
import type { BaseRedactPolicyForm, BaseRedactPolicyService, CreateBaseRedactPolicyRequest, DeleteBaseRedactPolicyRequest, GetBaseRedactPolicyRequest, PageBaseRedactPolicyRequest, PageBaseRedactPolicyResponse, SetBaseRedactPolicyStatusRequest, UpdateBaseRedactPolicyRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_policy";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const BASE_REDACT_POLICY_URL = "/v1/admin/base/redact-policy";

/** Admin脱敏策略绑定服务。 */
export class BaseRedactPolicyServiceImpl implements BaseRedactPolicyService {
  /** 查询脱敏策略分页列表。 */
  PageBaseRedactPolicy(request: PageBaseRedactPolicyRequest): Promise<PageBaseRedactPolicyResponse> { return service<PageBaseRedactPolicyRequest, PageBaseRedactPolicyResponse>({ url: BASE_REDACT_POLICY_URL, method: "get", params: request }); }
  /** 查询脱敏策略详情。 */
  GetBaseRedactPolicy(request: GetBaseRedactPolicyRequest): Promise<BaseRedactPolicyForm> { return service<GetBaseRedactPolicyRequest, BaseRedactPolicyForm>({ url: `${BASE_REDACT_POLICY_URL}/${request.id}`, method: "get" }); }
  /** 创建脱敏策略。 */
  CreateBaseRedactPolicy(request: CreateBaseRedactPolicyRequest): Promise<Empty> { return service<BaseRedactPolicyForm | undefined, Empty>({ url: BASE_REDACT_POLICY_URL, method: "post", data: request.base_redact_policy }); }
  /** 更新脱敏策略。 */
  UpdateBaseRedactPolicy(request: UpdateBaseRedactPolicyRequest): Promise<Empty> { return service<BaseRedactPolicyForm | undefined, Empty>({ url: `${BASE_REDACT_POLICY_URL}/${request.base_redact_policy?.id ?? ""}`, method: "put", data: request.base_redact_policy }); }
  /** 删除脱敏策略。 */
  DeleteBaseRedactPolicy(request: DeleteBaseRedactPolicyRequest): Promise<Empty> { return service<DeleteBaseRedactPolicyRequest, Empty>({ url: `${BASE_REDACT_POLICY_URL}/${request.id}`, method: "delete" }); }
  /** 设置脱敏策略状态。 */
  SetBaseRedactPolicyStatus(request: SetBaseRedactPolicyStatusRequest): Promise<Empty> { return service<SetBaseRedactPolicyStatusRequest, Empty>({ url: `${BASE_REDACT_POLICY_URL}/${request.id}/status`, method: "put", data: request }); }
}

/** defBaseRedactPolicyService 脱敏策略绑定服务实例。 */
export const defBaseRedactPolicyService = new BaseRedactPolicyServiceImpl();
