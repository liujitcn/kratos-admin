import service from "@/utils/request";
import {
  type BaseApi,
  type BaseApiDoc,
  type BaseApiService,
  type GetBaseApiDocRequest,
  type GetBaseApiRequest,
  type OptionBaseApiRequest,
  type OptionBaseApiResponse,
  type PageBaseApiRequest,
  type PageBaseApiResponse,
  type SetBaseApiAgentStatusRequest,
  type SetBaseApiMcpStatusRequest,
  type UpdateBaseApiRequest
} from "@/rpc/system/admin/v1/base_api";
import type { Empty } from "@/rpc/google/protobuf/empty";

const BASE_API_URL = "/v1/admin/base/api";

/** AdminAPI服务 */
export class BaseApiServiceImpl implements BaseApiService {
  /** 查询API选项列表 */
  OptionBaseApi(request: OptionBaseApiRequest): Promise<OptionBaseApiResponse> {
    return service<OptionBaseApiRequest, OptionBaseApiResponse>({
      url: `${BASE_API_URL}/option`,
      method: "get",
      params: request
    });
  }

  /** 查询API分页列表 */
  PageBaseApi(request: PageBaseApiRequest): Promise<PageBaseApiResponse> {
    return service<PageBaseApiRequest, PageBaseApiResponse>({
      url: `${BASE_API_URL}`,
      method: "get",
      params: request
    });
  }

  /** 查询API详情 */
  GetBaseApi(request: GetBaseApiRequest): Promise<BaseApi> {
    return service<GetBaseApiRequest, BaseApi>({
      url: `${BASE_API_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 查询API文档 */
  GetBaseApiDoc(request: GetBaseApiDocRequest): Promise<BaseApiDoc> {
    return service<GetBaseApiDocRequest, BaseApiDoc>({
      url: `${BASE_API_URL}/${request.id}/doc`,
      method: "get"
    });
  }

  /** 更新API配置 */
  UpdateBaseApi(request: UpdateBaseApiRequest): Promise<Empty> {
    return service<UpdateBaseApiRequest, Empty>({
      url: `${BASE_API_URL}/${request.id}`,
      method: "put",
      data: request
    });
  }

  /** 设置API Agent工具状态 */
  SetBaseApiAgentStatus(request: SetBaseApiAgentStatusRequest): Promise<Empty> {
    return service<SetBaseApiAgentStatusRequest, Empty>({
      url: `${BASE_API_URL}/${request.id}/agent-status`,
      method: "put",
      data: request
    });
  }

  /** 设置API MCP工具状态 */
  SetBaseApiMcpStatus(request: SetBaseApiMcpStatusRequest): Promise<Empty> {
    return service<SetBaseApiMcpStatusRequest, Empty>({
      url: `${BASE_API_URL}/${request.id}/mcp-status`,
      method: "put",
      data: request
    });
  }
}

export const defBaseApiService = new BaseApiServiceImpl();
