import service from "@liujitcn/kratos-admin-core/request";
import type {
  BaseMessageTemplateForm,
  BaseMessageTemplateService,
  CreateBaseMessageTemplateRequest,
  DeleteBaseMessageTemplateRequest,
  GetBaseMessageTemplateRequest,
  PageBaseMessageTemplateRequest,
  PageBaseMessageTemplateResponse,
  UpdateBaseMessageTemplateRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_message_template";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

const MESSAGE_TEMPLATE_URL = "/v1/admin/base/message-template";

/** 消息 Provider 模板管理服务实现。 */
export class BaseMessageTemplateServiceImpl implements BaseMessageTemplateService {
  /** 分页查询消息 Provider 模板。 */
  PageBaseMessageTemplate(request: PageBaseMessageTemplateRequest): Promise<PageBaseMessageTemplateResponse> {
    return service({ url: MESSAGE_TEMPLATE_URL, method: "get", params: request });
  }

  /** 查询消息 Provider 模板详情。 */
  GetBaseMessageTemplate(request: GetBaseMessageTemplateRequest): Promise<BaseMessageTemplateForm> {
    return service({ url: `${MESSAGE_TEMPLATE_URL}/${request.id}`, method: "get" });
  }

  /** 创建消息 Provider 模板。 */
  CreateBaseMessageTemplate(request: CreateBaseMessageTemplateRequest): Promise<Empty> {
    return service({ url: MESSAGE_TEMPLATE_URL, method: "post", data: request.base_message_template });
  }

  /** 更新消息 Provider 模板。 */
  UpdateBaseMessageTemplate(request: UpdateBaseMessageTemplateRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_TEMPLATE_URL}/${request.base_message_template?.id ?? ""}`, method: "put", data: request.base_message_template });
  }

  /** 删除消息 Provider 模板。 */
  DeleteBaseMessageTemplate(request: DeleteBaseMessageTemplateRequest): Promise<Empty> {
    return service({ url: `${MESSAGE_TEMPLATE_URL}/${request.id}`, method: "delete" });
  }
}

export const defBaseMessageTemplateService = new BaseMessageTemplateServiceImpl();
