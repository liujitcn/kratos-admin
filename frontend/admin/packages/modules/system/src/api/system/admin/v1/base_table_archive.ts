import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { BaseTableArchiveService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive";
import type {
  BaseTableArchiveForm,
  CreateBaseTableArchiveRequest,
  DeleteBaseTableArchiveRequest,
  GetBaseTableArchiveRequest,
  PageBaseTableArchiveRequest,
  PageBaseTableArchiveResponse,
  SetBaseTableArchiveStatusRequest,
  UpdateBaseTableArchiveRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive";

const BASE_TABLE_ARCHIVE_URL = "/v1/admin/base/table-archive";

/** 表归档配置服务实现。 */
export class BaseTableArchiveServiceImpl implements BaseTableArchiveService {
  /** 分页查询归档配置。 */
  PageBaseTableArchive(request: PageBaseTableArchiveRequest): Promise<PageBaseTableArchiveResponse> {
    return service<PageBaseTableArchiveRequest, PageBaseTableArchiveResponse>({
      url: BASE_TABLE_ARCHIVE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询归档配置。 */
  GetBaseTableArchive(request: GetBaseTableArchiveRequest): Promise<BaseTableArchiveForm> {
    return service<GetBaseTableArchiveRequest, BaseTableArchiveForm>({
      url: `${BASE_TABLE_ARCHIVE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建归档配置。 */
  CreateBaseTableArchive(request: CreateBaseTableArchiveRequest): Promise<Empty> {
    return service<BaseTableArchiveForm | undefined, Empty>({
      url: BASE_TABLE_ARCHIVE_URL,
      method: "post",
      data: request.base_table_archive
    });
  }

  /** 更新归档配置。 */
  UpdateBaseTableArchive(request: UpdateBaseTableArchiveRequest): Promise<Empty> {
    return service<BaseTableArchiveForm | undefined, Empty>({
      url: `${BASE_TABLE_ARCHIVE_URL}/${request.base_table_archive?.id ?? ""}`,
      method: "put",
      data: request.base_table_archive
    });
  }

  /** 删除归档配置。 */
  DeleteBaseTableArchive(request: DeleteBaseTableArchiveRequest): Promise<Empty> {
    return service<DeleteBaseTableArchiveRequest, Empty>({ url: `${BASE_TABLE_ARCHIVE_URL}/${request.id}`, method: "delete" });
  }

  /** 设置归档配置状态。 */
  SetBaseTableArchiveStatus(request: SetBaseTableArchiveStatusRequest): Promise<Empty> {
    return service<SetBaseTableArchiveStatusRequest, Empty>({
      url: `${BASE_TABLE_ARCHIVE_URL}/${request.id}/status`,
      method: "put",
      data: request
    });
  }
}

/** 表归档配置服务实例。 */
export const defBaseTableArchiveService = new BaseTableArchiveServiceImpl();
