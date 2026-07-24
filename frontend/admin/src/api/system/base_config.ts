import service from "@/utils/request";
import {
  type BaseConfigForm,
  type BaseConfigService,
  type CreateBaseConfigRequest,
  type DeleteBaseConfigRequest,
  type GetBaseConfigRequest,
  type PageBaseConfigRequest,
  type PageBaseConfigResponse,
  type RefreshBaseConfigCacheRequest,
  type SetBaseConfigStatusRequest,
  type UpdateBaseConfigRequest
} from "@/rpc/system/admin/v1/base_config";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_CONFIG_URL = "/v1/admin/base/config";

/** Admin系统配置服务 */
export class BaseConfigServiceImpl implements BaseConfigService {
  /** 刷新缓存 */
  RefreshBaseConfigCache(request: RefreshBaseConfigCacheRequest): Promise<Empty> {
    return service<RefreshBaseConfigCacheRequest, Empty>({
      url: `${BASE_CONFIG_URL}/cache`,
      method: "put",
      data: request
    });
  }

  /** 查询系统配置分页列表 */
  PageBaseConfig(request: PageBaseConfigRequest): Promise<PageBaseConfigResponse> {
    return service<PageBaseConfigRequest, PageBaseConfigResponse>({
      url: `${BASE_CONFIG_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询系统配置 */
  GetBaseConfig(request: GetBaseConfigRequest): Promise<BaseConfigForm> {
    return service<GetBaseConfigRequest, BaseConfigForm>({
      url: `${BASE_CONFIG_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建系统配置 */
  CreateBaseConfig(request: CreateBaseConfigRequest): Promise<Empty> {
    return service<BaseConfigForm | undefined, Empty>({
      url: `${BASE_CONFIG_URL}`,
      method: "post",
      data: request.base_config
    });
  }

  /** 更新系统配置 */
  UpdateBaseConfig(request: UpdateBaseConfigRequest): Promise<Empty> {
    return service<BaseConfigForm | undefined, Empty>({
      url: `${BASE_CONFIG_URL}/${request.base_config?.id ?? ""}`,
      method: "put",
      data: request.base_config
    });
  }

  /** 删除系统配置 */
  DeleteBaseConfig(request: DeleteBaseConfigRequest): Promise<Empty> {
    return service<DeleteBaseConfigRequest, Empty>({
      url: `${BASE_CONFIG_URL}/${request.id}`,
      method: "delete"
    });
  }

  /** 设置状态 */
  SetBaseConfigStatus(request: SetBaseConfigStatusRequest): Promise<Empty> {
    return service<SetBaseConfigStatusRequest, Empty>({
      url: `${BASE_CONFIG_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseConfigService = new BaseConfigServiceImpl();
