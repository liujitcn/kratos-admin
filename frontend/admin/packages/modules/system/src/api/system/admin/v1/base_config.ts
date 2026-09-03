import service from "@liujitcn/kratos-admin-core/request";
import {
  type BaseConfigForm,
  type BaseConfigService,
  type CreateBaseConfigRequest,
  type DeleteBaseConfigRequest,
  type GetBaseConfigByKeyRequest,
  type BaseConfigValue,
  type GetBaseConfigRequest,
  type PageBaseConfigRequest,
  type PageBaseConfigResponse,
  type RefreshBaseConfigCacheRequest,
  type SetBaseConfigStatusRequest,
  type UpdateBaseConfigByKeyRequest,
  type UpdateBaseConfigRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_config";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";

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

  /** 按配置键查询隐藏系统配置。 */
  GetBaseConfigByKey(request: GetBaseConfigByKeyRequest): Promise<BaseConfigValue> {
    return service<GetBaseConfigByKeyRequest, BaseConfigValue>({
      url: `${BASE_CONFIG_URL}/key/${request.key}`,
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

  /** 按配置键更新隐藏系统配置。 */
  UpdateBaseConfigByKey(request: UpdateBaseConfigByKeyRequest): Promise<Empty> {
    return service<Pick<UpdateBaseConfigByKeyRequest, "value_json">, Empty>({
      url: `${BASE_CONFIG_URL}/key/${request.key}`,
      method: "put",
      data: { value_json: request.value_json }
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
