import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseLoginPolicy,
  GetBaseLoginPolicyRequest,
  UpdateBaseLoginPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_policy";

const LOGIN_POLICY_URL = "/v1/admin/base/login-policy";

/** BaseLoginPolicyService Admin登录来源策略服务。 */
export class BaseLoginPolicyServiceImpl {
  /** 查询登录来源策略。 */
  GetBaseLoginPolicy(request: GetBaseLoginPolicyRequest): Promise<BaseLoginPolicy> {
    return service<GetBaseLoginPolicyRequest, BaseLoginPolicy>({
      url: LOGIN_POLICY_URL,
      method: "get",
      params: request
    });
  }

  /** 更新登录来源策略。 */
  UpdateBaseLoginPolicy(request: UpdateBaseLoginPolicyRequest): Promise<BaseLoginPolicy> {
    return service<UpdateBaseLoginPolicyRequest["policy"], BaseLoginPolicy>({
      url: LOGIN_POLICY_URL,
      method: "put",
      data: request.policy
    });
  }
}

/** defBaseLoginPolicyService Admin登录来源策略服务实例。 */
export const defBaseLoginPolicyService = new BaseLoginPolicyServiceImpl();
