import service from "@liujitcn/kratos-admin-core/request";
import type {
  GetMqttMessageRequest,
  MqttMessageDetail,
  MqttMessageService,
  PageMqttMessageRequest,
  PageMqttMessageResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/mqtt_message";

const MQTT_MESSAGE_URL = "/v1/admin/mqtt-message";

/** Admin MQTT 消息记录查询服务。 */
export class MqttMessageServiceImpl implements MqttMessageService {
  /** 分页查询 MQTT 消息记录。 */
  PageMqttMessage(request: PageMqttMessageRequest): Promise<PageMqttMessageResponse> {
    return service<PageMqttMessageRequest, PageMqttMessageResponse>({
      url: MQTT_MESSAGE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询 MQTT 消息记录详情。 */
  GetMqttMessage(request: GetMqttMessageRequest): Promise<MqttMessageDetail> {
    return service<GetMqttMessageRequest, MqttMessageDetail>({
      url: `${MQTT_MESSAGE_URL}/${request.id}`,
      method: "get"
    });
  }
}

/** 默认 MQTT 消息记录查询服务。 */
export const defMqttMessageService = new MqttMessageServiceImpl();
