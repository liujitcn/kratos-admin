import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseMessageProviderForm,
  BaseMessageProviderService,
  CreateBaseMessageProviderRequest,
  DeleteBaseMessageProviderRequest,
  GetBaseMessageProviderRequest,
  OptionBaseMessageProviderRequest,
  PageBaseMessageProviderRequest,
  PageBaseMessageProviderResponse,
  SetBaseMessageProviderStatusRequest,
  UpdateBaseMessageProviderRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_provider";
import type { SelectOptionResponse } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const MESSAGE_PROVIDER_URL = "/v1/admin/base/message-provider";

/** 消息发送 Provider 管理服务实现。 */
export class BaseMessageProviderServiceImpl implements BaseMessageProviderService {
  /** 查询消息发送 Provider 选项。 */
  OptionBaseMessageProvider(request: OptionBaseMessageProviderRequest): Promise<SelectOptionResponse> {
    return service({ url: `${MESSAGE_PROVIDER_URL}/option`, method: "get", params: request });
  }

  /** 分页查询消息发送 Provider。 */
  PageBaseMessageProvider(request: PageBaseMessageProviderRequest): Promise<PageBaseMessageProviderResponse> {
    return service({ url: MESSAGE_PROVIDER_URL, method: "get", params: request });
  }

  /** 查询消息发送 Provider 详情。 */
  GetBaseMessageProvider(request: GetBaseMessageProviderRequest): Promise<BaseMessageProviderForm> {
    return service({ url: `${MESSAGE_PROVIDER_URL}/${request.id}`, method: "get" });
  }

  /** 创建消息发送 Provider。 */
  CreateBaseMessageProvider(request: CreateBaseMessageProviderRequest): Promise<Empty> {
    return service({ url: MESSAGE_PROVIDER_URL, method: "post", data: request.base_message_provider });
  }

  /** 更新消息发送 Provider。 */
  UpdateBaseMessageProvider(request: UpdateBaseMessageProviderRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_PROVIDER_URL}/${request.base_message_provider?.id ?? ""}`, method: "put", data: request.base_message_provider });
  }

  /** 删除消息发送 Provider。 */
  DeleteBaseMessageProvider(request: DeleteBaseMessageProviderRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_PROVIDER_URL}/${request.id}`, method: "delete" });
  }

  /** 设置消息发送 Provider 状态。 */
  SetBaseMessageProviderStatus(request: SetBaseMessageProviderStatusRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_PROVIDER_URL}/${request.id}/status`, method: "put", data: request });
  }
}

export const defBaseMessageProviderService = new BaseMessageProviderServiceImpl();
