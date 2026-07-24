import service from "@/utils/request";
import {
  type BaseRoleForm,
  type BaseRoleService,
  type CreateBaseRoleRequest,
  type DeleteBaseRoleRequest,
  type GetBaseRoleRequest,
  type PageBaseRoleRequest,
  type PageBaseRoleResponse,
  type OptionBaseRoleRequest,
  type SetBaseRoleMenuRequest,
  type SetBaseRoleStatusRequest,
  type UpdateBaseRoleRequest
} from "@/rpc/system/admin/v1/base_role";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@/rpc/common/v1/common";

const BASE_ROLE_URL = "/v1/admin/base/role";

/** Admin角色服务 */
export class BaseRoleServiceImpl implements BaseRoleService {
  /** 查询角色下拉选择 */
  OptionBaseRole(request: OptionBaseRoleRequest): Promise<SelectOptionResponse> {
    return service<OptionBaseRoleRequest, SelectOptionResponse>({
      url: `${BASE_ROLE_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询角色分页列表 */
  PageBaseRole(request: PageBaseRoleRequest): Promise<PageBaseRoleResponse> {
    return service<PageBaseRoleRequest, PageBaseRoleResponse>({
      url: `${BASE_ROLE_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询角色 */
  GetBaseRole(request: GetBaseRoleRequest): Promise<BaseRoleForm> {
    return service<GetBaseRoleRequest, BaseRoleForm>({
      url: `${BASE_ROLE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建角色 */
  CreateBaseRole(request: CreateBaseRoleRequest): Promise<Empty> {
    return service<BaseRoleForm | undefined, Empty>({
      url: `${BASE_ROLE_URL}`,
      method: "post",
      data: request.base_role
    });
  }

  /** 更新角色 */
  UpdateBaseRole(request: UpdateBaseRoleRequest): Promise<Empty> {
    return service<BaseRoleForm | undefined, Empty>({
      url: `${BASE_ROLE_URL}/${request.base_role?.id ?? ""}`,
      method: "put",
      data: request.base_role
    });
  }

  /** 删除角色 */
  DeleteBaseRole(request: DeleteBaseRoleRequest): Promise<Empty> {
    return service<DeleteBaseRoleRequest, Empty>({
      url: `${BASE_ROLE_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseRoleStatus(request: SetBaseRoleStatusRequest): Promise<Empty> {
    return service<SetBaseRoleStatusRequest, Empty>({
      url: `${BASE_ROLE_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }

  /** 设置角色菜单权限 */
  SetBaseRoleMenu(request: SetBaseRoleMenuRequest): Promise<Empty> {
    return service<SetBaseRoleMenuRequest, Empty>({
      url: `${BASE_ROLE_URL}/${request.id}/menu`,
      method: "put",
      data: request
    });
  }
}

export const defBaseRoleService = new BaseRoleServiceImpl();
