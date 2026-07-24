import service from "@/utils/request";
import {
  type BaseUserForm,
  type BaseUserService,
  type CreateBaseUserRequest,
  type DeleteBaseUserRequest,
  type GetBaseUserRequest,
  type PageBaseUserRequest,
  type PageBaseUserResponse,
  type OptionBaseUserRequest,
  type ResetBaseUserPasswordRequest,
  type SetBaseUserStatusRequest,
  type UpdateBaseUserRequest
} from "@/rpc/system/admin/v1/base_user";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@/rpc/common/v1/common";

const BASE_USER_URL = "/v1/admin/base/user";

/** Admin用户服务 */
export class BaseUserServiceImpl implements BaseUserService {
  /** 查询用户下拉选择 */
  OptionBaseUser(request: OptionBaseUserRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseUserRequest, SelectOptionResponse>({
      url: `${BASE_USER_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询用户分页列表 */
  PageBaseUser(request: PageBaseUserRequest): Promise<PageBaseUserResponse> {
    return service<PageBaseUserRequest, PageBaseUserResponse>({
      url: `${BASE_USER_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询用户 */
  GetBaseUser(request: GetBaseUserRequest): Promise<BaseUserForm> {
    return service<GetBaseUserRequest, BaseUserForm>({
      url: `${BASE_USER_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建用户 */
  CreateBaseUser(request: CreateBaseUserRequest): Promise<Empty> {
    return service<BaseUserForm | undefined, Empty>({
      url: `${BASE_USER_URL}`,
      method: "post",
      data: request.base_user
    });
  }

  /** 更新用户 */
  UpdateBaseUser(request: UpdateBaseUserRequest): Promise<Empty> {
    return service<BaseUserForm | undefined, Empty>({
      url: `${BASE_USER_URL}/${request.base_user?.id ?? ""}`,
      method: "put",
      data: request.base_user
    });
  }

  /** 删除用户 */
  DeleteBaseUser(request: DeleteBaseUserRequest): Promise<Empty> {
    return service<DeleteBaseUserRequest, Empty>({
      url: `${BASE_USER_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseUserStatus(request: SetBaseUserStatusRequest): Promise<Empty> {
    return service<SetBaseUserStatusRequest, Empty>({
      url: `${BASE_USER_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }

  /** 重置密码 */
  ResetBaseUserPassword(request: ResetBaseUserPasswordRequest): Promise<Empty> {
    return service<ResetBaseUserPasswordRequest, Empty>({
      url: `${BASE_USER_URL}/${request.id}/password`,
      method: "put",
      data: request
    });
  }
}

export const defBaseUserService = new BaseUserServiceImpl();
