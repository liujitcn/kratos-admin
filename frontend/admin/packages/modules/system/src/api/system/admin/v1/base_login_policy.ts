import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseLoginPolicyForm,
  BaseLoginPolicyService,
  CreateBaseLoginPolicyRequest,
  DeleteBaseLoginPolicyRequest,
  GetBaseLoginPolicyRequest,
  PageBaseLoginPolicyRequest,
  PageBaseLoginPolicyResponse,
  SetBaseLoginPolicyStatusRequest,
  UpdateBaseLoginPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_policy";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const LOGIN_POLICY_URL = "/v1/admin/base/login-policy";

/** BaseLoginPolicyService 登录策略管理服务实现。 */
export class BaseLoginPolicyServiceImpl implements BaseLoginPolicyService {
  /** 分页查询登录策略。 */
  PageBaseLoginPolicy(request: PageBaseLoginPolicyRequest): Promise<PageBaseLoginPolicyResponse> {
    return service<PageBaseLoginPolicyRequest, PageBaseLoginPolicyResponse>({
      url: LOGIN_POLICY_URL,
      method: "get",
      params: request
    });
  }

  /** 查询登录策略详情。 */
  GetBaseLoginPolicy(request: GetBaseLoginPolicyRequest): Promise<BaseLoginPolicyForm> {
    return service<GetBaseLoginPolicyRequest, BaseLoginPolicyForm>({ url: `${LOGIN_POLICY_URL}/${request.id}`, method: "get" });
  }

  /** 创建登录策略。 */
  CreateBaseLoginPolicy(request: CreateBaseLoginPolicyRequest): Promise<Empty> {
    return service<BaseLoginPolicyForm | undefined, Empty>({
      url: LOGIN_POLICY_URL,
      method: "post",
      data: request.base_login_policy
    });
  }

  /** 更新登录策略。 */
  UpdateBaseLoginPolicy(request: UpdateBaseLoginPolicyRequest): Promise<Empty> {
    return service<BaseLoginPolicyForm | undefined, Empty>({
      url: `${LOGIN_POLICY_URL}/${request.base_login_policy?.id ?? ""}`,
      method: "put",
      data: request.base_login_policy
    });
  }

  /** 删除登录策略。 */
  DeleteBaseLoginPolicy(request: DeleteBaseLoginPolicyRequest): Promise<Empty> {
    return service<DeleteBaseLoginPolicyRequest, Empty>({ url: `${LOGIN_POLICY_URL}/${request.id}`, method: "delete" });
  }

  /** 设置登录策略状态。 */
  SetBaseLoginPolicyStatus(request: SetBaseLoginPolicyStatusRequest): Promise<Empty> {
    return service<SetBaseLoginPolicyStatusRequest, Empty>({
      url: `${LOGIN_POLICY_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

/** defBaseLoginPolicyService 登录策略管理服务实例。 */
export const defBaseLoginPolicyService = new BaseLoginPolicyServiceImpl();
