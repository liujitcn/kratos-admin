import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseMessageCategoryForm,
  BaseMessageCategoryService,
  CreateBaseMessageCategoryRequest,
  DeleteBaseMessageCategoryRequest,
  GetBaseMessageCategoryRequest,
  OptionBaseMessageCategoryRequest,
  PageBaseMessageCategoryRequest,
  PageBaseMessageCategoryResponse,
  SetBaseMessageCategoryStatusRequest,
  UpdateBaseMessageCategoryRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_category";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const MESSAGE_CATEGORY_URL = "/v1/admin/base/message-category";

/** 消息分类管理服务实现。 */
export class BaseMessageCategoryServiceImpl implements BaseMessageCategoryService {
  /** 查询消息分类选项。 */
  OptionBaseMessageCategory(request: OptionBaseMessageCategoryRequest): Promise<SelectOptionResponse> {
    return service({ url: `${MESSAGE_CATEGORY_URL}/option`, method: "get", params: request });
  }

  /** 分页查询消息分类。 */
  PageBaseMessageCategory(request: PageBaseMessageCategoryRequest): Promise<PageBaseMessageCategoryResponse> {
    return service({ url: MESSAGE_CATEGORY_URL, method: "get", params: request });
  }

  /** 查询消息分类详情。 */
  GetBaseMessageCategory(request: GetBaseMessageCategoryRequest): Promise<BaseMessageCategoryForm> {
    return service({ url: `${MESSAGE_CATEGORY_URL}/${request.id}`, method: "get" });
  }

  /** 创建消息分类。 */
  CreateBaseMessageCategory(request: CreateBaseMessageCategoryRequest): Promise<Empty> {
    return service({ url: MESSAGE_CATEGORY_URL, method: "post", data: request.base_message_category });
  }

  /** 更新消息分类。 */
  UpdateBaseMessageCategory(request: UpdateBaseMessageCategoryRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_CATEGORY_URL}/${request.base_message_category?.id ?? ""}`, method: "put", data: request.base_message_category });
  }

  /** 删除消息分类。 */
  DeleteBaseMessageCategory(request: DeleteBaseMessageCategoryRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_CATEGORY_URL}/${request.id}`, method: "delete" });
  }

  /** 设置消息分类状态。 */
  SetBaseMessageCategoryStatus(request: SetBaseMessageCategoryStatusRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_CATEGORY_URL}/${request.id}/status`, method: "put", data: request });
  }
}

export const defBaseMessageCategoryService = new BaseMessageCategoryServiceImpl();
