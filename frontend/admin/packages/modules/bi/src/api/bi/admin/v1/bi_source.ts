import service from "@liujitcn/kratos-admin-core/request";
import type {
  BiSource,
  BiSourceForm,
  BiSourceService,
  ListBiSourceTablesRequest,
  ListBiSourceTablesResponse,
  CreateBiSourceRequest,
  DeleteBiSourceRequest,
  DeleteBiSourceResponse,
  GetBiSourceRequest,
  OptionBiSourceDatabaseRequest,
  OptionBiSourceDatabaseResponse,
  PageBiSourceRequest,
  PageBiSourceResponse,
  SetBiSourceStatusRequest,
  SaveBiSourceModelRequest,
  TestBiSourceRequest,
  TestBiSourceResponse,
  UpdateBiSourceRequest
} from "../../../../rpc/bi/admin/v1/bi_source";

const BI_SOURCE_URL = "/v1/admin/bi/source";

/** BI 数据源配置服务实现。 */
export class BiSourceServiceImpl implements BiSourceService {
  /** 分页查询当前租户的数据源配置。 */
  PageBiSource(request: PageBiSourceRequest): Promise<PageBiSourceResponse> {
    return service({url: BI_SOURCE_URL, method: "get", params: request});
  }

  /** 查询数据源配置详情。 */
  GetBiSource(request: GetBiSourceRequest): Promise<BiSource> {
    return service({url: `${BI_SOURCE_URL}/${request.id}`, method: "get", params: request});
  }

  /** 创建数据源配置。 */
  CreateBiSource(request: CreateBiSourceRequest): Promise<BiSource> {
    return service({url: BI_SOURCE_URL, method: "post", data: request.source});
  }

  /** 更新数据源配置。 */
  UpdateBiSource(request: UpdateBiSourceRequest): Promise<BiSource> {
    return service({url: `${BI_SOURCE_URL}/${request.source?.id ?? 0}`, method: "put", data: request.source});
  }

  /** 删除数据源配置。 */
  DeleteBiSource(request: DeleteBiSourceRequest): Promise<DeleteBiSourceResponse> {
    return service({url: `${BI_SOURCE_URL}/${request.id}`, method: "delete", data: request});
  }

  /** 修改数据源启用状态。 */
  SetBiSourceStatus(request: SetBiSourceStatusRequest): Promise<BiSource> {
    return service({url: `${BI_SOURCE_URL}/${request.id}/status`, method: "put", data: {status: request.status}});
  }

  /** 查询服务端已配置的数据库客户端选项。 */
  OptionBiSourceDatabase(request: OptionBiSourceDatabaseRequest): Promise<OptionBiSourceDatabaseResponse> {
    return service({url: `${BI_SOURCE_URL}/options/databases`, method: "get", params: request});
  }

  /** 测试数据源连接和表配置。 */
  TestBiSource(request: TestBiSourceRequest): Promise<TestBiSourceResponse> {
    return service({url: `${BI_SOURCE_URL}/test`, method: "post", data: request.source});
  }

  /** 查询数据源中的表和字段结构。 */
  ListBiSourceTables(request: ListBiSourceTablesRequest): Promise<ListBiSourceTablesResponse> {
    return service({url: `${BI_SOURCE_URL}/${request.id}/tables`, method: "get", params: request});
  }

  /** 保存数据源语义模型。 */
  SaveBiSourceModel(request: SaveBiSourceModelRequest): Promise<BiSource> {
    return service({url: `${BI_SOURCE_URL}/${request.id}/model`, method: "put", data: request.model});
  }
}

export const defBiSourceService = new BiSourceServiceImpl();

export type { BiSourceForm };
