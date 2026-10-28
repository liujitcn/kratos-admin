import service from "@/utils/request";
import type {
  AccountService,
  CurrentPasswordPolicy,
  GetCurrentPasswordPolicyRequest,
  UpdateUserPasswordRequest
} from "@/rpc/base/v1/account";
import type { Empty } from "@/rpc/google/protobuf/empty";

const ACCOUNT_URL = "/v1/base/account";

/** AccountService 账户安全公共服务。 */
export class AccountServiceImpl implements AccountService {
  /** 获取当前用户生效的密码策略。 */
  GetCurrentPasswordPolicy(request: GetCurrentPasswordPolicyRequest): Promise<CurrentPasswordPolicy> {
    return service<GetCurrentPasswordPolicyRequest, CurrentPasswordPolicy>({
      url: `${ACCOUNT_URL}/password-policy`,
      method: "get",
      params: request
    });
  }

  /** 修改当前用户密码。 */
  UpdateUserPassword(request: UpdateUserPasswordRequest): Promise<Empty> {
    return service<UpdateUserPasswordRequest["user_password"], Empty>({
      url: `${ACCOUNT_URL}/password`,
      method: "put",
      data: request.user_password
    });
  }
}

/** 当前用户账户安全公共服务实例。 */
export const defAccountService = new AccountServiceImpl();
