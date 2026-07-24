import service from "@/utils/request";
import {
  type CreateBaseAreaRequest,
  type DeleteBaseAreaRequest,
  type GetBaseAreaRequest,
  type TreeBaseAreaRequest,
  type TreeBaseAreaResponse,
  type OptionBaseAreaRequest,
  type BaseAreaForm,
  type BaseAreaService,
  type UpdateBaseAreaRequest
} from "@/rpc/system/admin/v1/base_area";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { TreeOptionResponse } from "@/rpc/common/v1/common";

const BASE_AREA_URL = "/v1/admin/base/area";

/** 行政区域服务。 */
export class BaseAreaServiceImpl implements BaseAreaService {
  /** 查询树形选择 */
  OptionBaseArea(request?: OptionBaseAreaRequest): Promise<TreeOptionResponse> {
    return service<OptionBaseAreaRequest, TreeOptionResponse>({
      url: BASE_AREA_URL + "/option",
      method: "get",
      params: request
    });
  }

  /** 查询树形列表 */
  TreeBaseArea(request: TreeBaseAreaRequest): Promise<TreeBaseAreaResponse> {
    return service<TreeBaseAreaRequest, TreeBaseAreaResponse>({
      url: BASE_AREA_URL + "/tree",
      method: "get",
      params: request
    });
  }

  /** 查询详情 */
  GetBaseArea(request: GetBaseAreaRequest): Promise<BaseAreaForm> {
    return service<GetBaseAreaRequest, BaseAreaForm>({
      url: BASE_AREA_URL + "/" + request.id,
      method: "get"
    });
  }

  /** 创建 */
  CreateBaseArea(request: CreateBaseAreaRequest): Promise<Empty> {
    return service<BaseAreaForm | undefined, Empty>({
      url: BASE_AREA_URL,
      method: "post",
      data: request.base_area
    });
  }

  /** 更新 */
  UpdateBaseArea(request: UpdateBaseAreaRequest): Promise<Empty> {
    return service<BaseAreaForm | undefined, Empty>({
      url: BASE_AREA_URL + "/" + request.id,
      method: "put",
      data: request.base_area
    });
  }

  /** 删除 */
  DeleteBaseArea(request: DeleteBaseAreaRequest): Promise<Empty> {
    return service<DeleteBaseAreaRequest, Empty>({
      url: BASE_AREA_URL + "/" + request.ids,
      method: "delete"
    });
  }
}

export const defBaseAreaService = new BaseAreaServiceImpl();
