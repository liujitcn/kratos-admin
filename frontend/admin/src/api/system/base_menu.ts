import service from "@/utils/request";
import {
  type BaseMenuForm,
  type BaseMenuService,
  type CreateBaseMenuRequest,
  type DeleteBaseMenuRequest,
  type GetBaseMenuRequest,
  type OptionBaseMenuRequest,
  type SetBaseMenuStatusRequest,
  type TreeBaseMenuRequest,
  type TreeBaseMenuResponse,
  type UpdateBaseMenuRequest
} from "@/rpc/system/admin/v1/base_menu";
import { type Empty } from "@/rpc/google/protobuf/empty";
import { type TreeOptionResponse } from "@/rpc/common/v1/common";

const BASE_MENU_URL = "/v1/admin/base/menu";

/** Admin菜单管理服务 */
export class BaseMenuServiceImpl implements BaseMenuService {
  /** 查询菜单树形列表 */
  TreeBaseMenu(request: TreeBaseMenuRequest): Promise<TreeBaseMenuResponse> {
    return service<TreeBaseMenuRequest, TreeBaseMenuResponse>({
      url: `${BASE_MENU_URL}/tree`,
      method: "get",
      params: request
    });
  }

  /** 查询菜单树形选择 */
  OptionBaseMenu(request: OptionBaseMenuRequest): Promise<TreeOptionResponse> {
    return service<OptionBaseMenuRequest, TreeOptionResponse>({
      url: `${BASE_MENU_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询菜单 */
  GetBaseMenu(request: GetBaseMenuRequest): Promise<BaseMenuForm> {
    return service<GetBaseMenuRequest, BaseMenuForm>({
      url: `${BASE_MENU_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建菜单 */
  CreateBaseMenu(request: CreateBaseMenuRequest): Promise<Empty> {
    return service<BaseMenuForm | undefined, Empty>({
      url: `${BASE_MENU_URL}`,
      method: "post",
      data: request.base_menu
    });
  }

  /** 更新菜单 */
  UpdateBaseMenu(request: UpdateBaseMenuRequest): Promise<Empty> {
    return service<BaseMenuForm | undefined, Empty>({
      url: `${BASE_MENU_URL}/${request.base_menu?.id ?? ""}`,
      method: "put",
      data: request.base_menu
    });
  }

  /** 删除菜单 */
  DeleteBaseMenu(request: DeleteBaseMenuRequest): Promise<Empty> {
    return service<DeleteBaseMenuRequest, Empty>({
      url: `${BASE_MENU_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseMenuStatus(request: SetBaseMenuStatusRequest): Promise<Empty> {
    return service<SetBaseMenuStatusRequest, Empty>({
      url: `${BASE_MENU_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseMenuService = new BaseMenuServiceImpl();
