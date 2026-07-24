import service from "@/utils/request";
import {
  type BasePostForm,
  type BasePostService,
  type CreateBasePostRequest,
  type DeleteBasePostRequest,
  type GetBasePostRequest,
  type OptionBasePostRequest,
  type PageBasePostRequest,
  type PageBasePostResponse,
  type SetBasePostStatusRequest,
  type UpdateBasePostRequest
} from "@/rpc/system/admin/v1/base_post";
import type { Empty } from "@/rpc/google/protobuf/empty";
import type { SelectOptionResponse } from "@/rpc/common/v1/common";

const BASE_POST_URL = "/v1/admin/base/post";

/** Admin岗位服务。 */
export class BasePostServiceImpl implements BasePostService {
  /** 查询岗位下拉选择。 */
  OptionBasePost(request: OptionBasePostRequest): Promise<SelectOptionResponse> {
    return service<OptionBasePostRequest, SelectOptionResponse>({
      url: `${BASE_POST_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询岗位分页列表。 */
  PageBasePost(request: PageBasePostRequest): Promise<PageBasePostResponse> {
    return service<PageBasePostRequest, PageBasePostResponse>({
      url: BASE_POST_URL,
      method: "get",
      params: request
    });
  }

  /** 查询岗位。 */
  GetBasePost(request: GetBasePostRequest): Promise<BasePostForm> {
    return service<GetBasePostRequest, BasePostForm>({
      url: `${BASE_POST_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建岗位。 */
  CreateBasePost(request: CreateBasePostRequest): Promise<Empty> {
    return service<BasePostForm | undefined, Empty>({
      url: BASE_POST_URL,
      method: "post",
      data: request.base_post
    });
  }

  /** 更新岗位。 */
  UpdateBasePost(request: UpdateBasePostRequest): Promise<Empty> {
    return service<BasePostForm | undefined, Empty>({
      url: `${BASE_POST_URL}/${request.base_post?.id ?? ""}`,
      method: "put",
      data: request.base_post
    });
  }

  /** 删除岗位。 */
  DeleteBasePost(request: DeleteBasePostRequest): Promise<Empty> {
    return service<DeleteBasePostRequest, Empty>({
      url: `${BASE_POST_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置岗位状态。 */
  SetBasePostStatus(request: SetBasePostStatusRequest): Promise<Empty> {
    return service<SetBasePostStatusRequest, Empty>({
      url: `${BASE_POST_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBasePostService = new BasePostServiceImpl();
