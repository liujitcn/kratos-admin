import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseMessageDetail,
  BaseMessageForm,
  CancelBaseMessageScheduleRequest,
  BaseMessageService,
  CreateBaseMessageRequest,
  CreateBaseMessageResponse,
  DeleteBaseMessageRequest,
  GetBaseMessageRequest,
  PageBaseMessageRequest,
  PageBaseMessageResponse,
  PublishBaseMessageRequest,
  RevokeBaseMessageRequest,
  RetryBaseMessageDispatchRequest,
  UpdateBaseMessageRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const MESSAGE_URL = "/v1/admin/base/message";

/** 站内信管理服务实现。 */
export class BaseMessageServiceImpl implements BaseMessageService {
  /** 分页查询消息。 */
  PageBaseMessage(request: PageBaseMessageRequest): Promise<PageBaseMessageResponse> {
    return service({ url: MESSAGE_URL, method: "get", params: request });
  }

  /** 查询消息详情。 */
  GetBaseMessage(request: GetBaseMessageRequest): Promise<BaseMessageDetail> {
    return service({ url: `${MESSAGE_URL}/${request.id}`, method: "get" });
  }

  /** 创建消息草稿。 */
  CreateBaseMessage(request: CreateBaseMessageRequest): Promise<CreateBaseMessageResponse> {
    return service<BaseMessageForm | undefined, CreateBaseMessageResponse>({ url: MESSAGE_URL, method: "post", data: request.base_message });
  }

  /** 更新消息草稿。 */
  UpdateBaseMessage(request: UpdateBaseMessageRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/${request.base_message?.id ?? ""}`, method: "put", data: request.base_message });
  }

  /** 删除消息草稿。 */
  DeleteBaseMessage(request: DeleteBaseMessageRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/${request.id}`, method: "delete" });
  }

  /** 发布消息。 */
  PublishBaseMessage(request: PublishBaseMessageRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/${request.id}/publish`, method: "put", data: request });
  }

  /** 取消定时发布。 */
  CancelBaseMessageSchedule(request: CancelBaseMessageScheduleRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/${request.id}/schedule/cancel`, method: "put", data: request });
  }

  /** 撤回消息。 */
  RevokeBaseMessage(request: RevokeBaseMessageRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/${request.id}/revoke`, method: "put", data: request });
  }

  /** 重试消息投递任务。 */
  RetryBaseMessageDispatch(request: RetryBaseMessageDispatchRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_URL}/dispatch/${request.id}/retry`, method: "put", data: request });
  }
}

export const defBaseMessageService = new BaseMessageServiceImpl();
