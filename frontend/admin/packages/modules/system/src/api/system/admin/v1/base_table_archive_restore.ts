import service from "@liujitcn/kratos-admin-core/request";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import type { BaseTableArchiveRestoreService } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_restore";
import type {
  BaseTableArchiveRestore,
  ExecuteBaseTableArchiveRestoreRequest,
  GetBaseTableArchiveRestoreRequest,
  PageBaseTableArchiveRestoreRequest,
  PageBaseTableArchiveRestoreResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_table_archive_restore";

const BASE_TABLE_ARCHIVE_RESTORE_URL = "/v1/admin/base/table-archive-restore";

/** 表归档恢复服务实现。 */
export class BaseTableArchiveRestoreServiceImpl implements BaseTableArchiveRestoreService {
  /** 分页查询归档恢复记录。 */
  PageBaseTableArchiveRestore(request: PageBaseTableArchiveRestoreRequest): Promise<PageBaseTableArchiveRestoreResponse> {
    return service<PageBaseTableArchiveRestoreRequest, PageBaseTableArchiveRestoreResponse>({
      url: BASE_TABLE_ARCHIVE_RESTORE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询归档恢复记录。 */
  GetBaseTableArchiveRestore(request: GetBaseTableArchiveRestoreRequest): Promise<BaseTableArchiveRestore> {
    return service<GetBaseTableArchiveRestoreRequest, BaseTableArchiveRestore>({
      url: `${BASE_TABLE_ARCHIVE_RESTORE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 手工执行归档恢复。 */
  ExecuteBaseTableArchiveRestore(request: ExecuteBaseTableArchiveRestoreRequest): Promise<Empty> {
    return service<BaseTableArchiveRestore | undefined, Empty>({
      url: BASE_TABLE_ARCHIVE_RESTORE_URL,
      method: "post",
      data: request.base_table_archive_restore
    });
  }
}

/** 表归档恢复服务实例。 */
export const defBaseTableArchiveRestoreService = new BaseTableArchiveRestoreServiceImpl();
