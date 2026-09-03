import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type {
  BaseDictItemForm,
  BaseDictItemService,
  CreateBaseDictItemRequest,
  DeleteBaseDictItemRequest,
  GetBaseDictItemRequest,
  PageBaseDictItemRequest,
  PageBaseDictItemResponse,
  SetBaseDictItemStatusRequest,
  UpdateBaseDictItemRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dict_item";

const BASE_DICT_ITEM_URL = "/v1/admin/base/dict-item";

/** Admin字典项服务 */
export class BaseDictItemServiceImpl implements BaseDictItemService {
  /** 查询字典属性分页列表 */
  PageBaseDictItem(request: PageBaseDictItemRequest): Promise<PageBaseDictItemResponse> {
    return service<PageBaseDictItemRequest, PageBaseDictItemResponse>({
      url: BASE_DICT_ITEM_URL,
      method: "get",
      params: request
    });
  }

  /** 查询字典属性 */
  GetBaseDictItem(request: GetBaseDictItemRequest): Promise<BaseDictItemForm> {
    return service<GetBaseDictItemRequest, BaseDictItemForm>({
      url: `${BASE_DICT_ITEM_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建字典属性 */
  CreateBaseDictItem(request: CreateBaseDictItemRequest): Promise<Empty> {
    return service<BaseDictItemForm | undefined, Empty>({
      url: BASE_DICT_ITEM_URL,
      method: "post",
      data: request.base_dict_item
    });
  }

  /** 更新字典属性 */
  UpdateBaseDictItem(request: UpdateBaseDictItemRequest): Promise<Empty> {
    return service<BaseDictItemForm | undefined, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.base_dict_item?.id ?? ""}`,
      method: "put",
      data: request.base_dict_item
    });
  }

  /** 删除字典属性 */
  DeleteBaseDictItem(request: DeleteBaseDictItemRequest): Promise<Empty> {
    return service<DeleteBaseDictItemRequest, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseDictItemStatus(request: SetBaseDictItemStatusRequest): Promise<Empty> {
    return service<SetBaseDictItemStatusRequest, Empty>({
      url: `${BASE_DICT_ITEM_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseDictItemService = new BaseDictItemServiceImpl();
