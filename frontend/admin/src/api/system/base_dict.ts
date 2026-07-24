import service from "@/utils/request";
import {
  type BaseDictForm,
  type BaseDictItemForm,
  type BaseDictService,
  type CreateBaseDictItemRequest,
  type CreateBaseDictRequest,
  type DeleteBaseDictItemRequest,
  type DeleteBaseDictRequest,
  type GetBaseDictItemRequest,
  type GetBaseDictRequest,
  type PageBaseDictItemRequest,
  type PageBaseDictItemResponse,
  type PageBaseDictRequest,
  type PageBaseDictResponse,
  type OptionBaseDictRequest,
  type OptionBaseDictResponse,
  type SetBaseDictItemStatusRequest,
  type SetBaseDictStatusRequest,
  type UpdateBaseDictItemRequest,
  type UpdateBaseDictRequest
} from "@/rpc/system/admin/v1/base_dict";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_DICT_URL = "/v1/admin/base/dict";
const BASE_DICT_ITEM_URL = "/v1/admin/base/dict-item";

/** Admin字典服务 */
export class BaseDictServiceImpl implements BaseDictService {
  /** 查询字典下拉选择 */
  OptionBaseDict(request: OptionBaseDictRequest): Promise<OptionBaseDictResponse> {
    return service<OptionBaseDictRequest, OptionBaseDictResponse>({
      url: `${BASE_DICT_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询字典分页列表 */
  PageBaseDict(request: PageBaseDictRequest): Promise<PageBaseDictResponse> {
    return service<PageBaseDictRequest, PageBaseDictResponse>({
      url: `${BASE_DICT_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询字典 */
  GetBaseDict(request: GetBaseDictRequest): Promise<BaseDictForm> {
    return service<GetBaseDictRequest, BaseDictForm>({
      url: `${BASE_DICT_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建字典 */
  CreateBaseDict(request: CreateBaseDictRequest): Promise<Empty> {
    return service<BaseDictForm | undefined, Empty>({
      url: `${BASE_DICT_URL}`,
      method: "post",
      data: request.base_dict
    });
  }

  /** 更新字典 */
  UpdateBaseDict(request: UpdateBaseDictRequest): Promise<Empty> {
    return service<BaseDictForm | undefined, Empty>({
      url: `${BASE_DICT_URL}/${request.base_dict?.id ?? ""}`,
      method: "put",
      data: request.base_dict
    });
  }

  /** 删除字典 */
  DeleteBaseDict(request: DeleteBaseDictRequest): Promise<Empty> {
    return service<DeleteBaseDictRequest, Empty>({
      url: `${BASE_DICT_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseDictStatus(request: SetBaseDictStatusRequest): Promise<Empty> {
    return service<SetBaseDictStatusRequest, Empty>({
      url: `${BASE_DICT_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }

  /** 查询字典属性分页列表 */
  PageBaseDictItem(request: PageBaseDictItemRequest): Promise<PageBaseDictItemResponse> {
    return service<PageBaseDictItemRequest, PageBaseDictItemResponse>({
      url: `${BASE_DICT_ITEM_URL}`,
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
      url: `${BASE_DICT_ITEM_URL}`,
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

export const defBaseDictService = new BaseDictServiceImpl();
